package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/request"
	"github.com/goccy/go-json"
	"go.uber.org/zap"
)

// [Go] AI 上游有两种形态，靠 autocode.ai-provider 切换：
//   - gva   （默认）：官方托管的 AiPath，Dify 风格请求体 {mode,query,inputs,user,...}
//   - 其它任意值     ：一律视为 OpenAI 兼容中转站，走 {base-url}/chat/completions
//     例如 openai / deepseek / sensenova / oneapi / openrouter，名字随意，只用于日志识别
//
// [注意] 官方 AiPath 里 {FUNC} 对应的 prompt 模板保存在人家服务端，本地拿不到。
// 所以切到兼容通道时必须由本文件补一整套 system prompt，否则中转站不知道 mode 是什么意思。
const (
	AIProviderGVA = "gva"
)

// IsOpenAIProvider 判断当前是否走 OpenAI 兼容通道。
// 语义：留空或写 gva 走官方通道（向后兼容老配置）；其它任何非空值都走兼容通道，
// 这样换中转站时不需要改代码，配什么名字都行。
func IsOpenAIProvider() bool {
	p := strings.ToLower(strings.TrimSpace(global.GVA_CONFIG.AutoCode.AiProvider))
	return p != "" && p != AIProviderGVA
}

// providerName 取当前配置的 provider 名，仅用于报错与日志展示
func providerName() string {
	p := strings.TrimSpace(global.GVA_CONFIG.AutoCode.AiProvider)
	if p == "" {
		return AIProviderGVA
	}
	return p
}

// openAIEndpoint 拼出 chat/completions 完整地址，并校验三项必填配置
func openAIEndpoint() (string, error) {
	cfg := global.GVA_CONFIG.AutoCode
	base := strings.TrimSpace(cfg.AiBaseURL)
	if base == "" {
		return "", fmt.Errorf("ai-provider=%s 时必须配置 autocode.ai-base-url（例如 https://api.xxx.com/v1）", providerName())
	}
	if strings.TrimSpace(cfg.AiKey) == "" {
		return "", fmt.Errorf("ai-provider=%s 时必须配置 autocode.ai-key", providerName())
	}
	if strings.TrimSpace(cfg.AiModel) == "" {
		return "", fmt.Errorf("ai-provider=%s 时必须配置 autocode.ai-model", providerName())
	}
	base = strings.TrimRight(base, "/")
	// 允许用户直接填到 /chat/completions，避免出现 /v1/chat/completions/chat/completions
	if strings.HasSuffix(base, "/chat/completions") {
		return base, nil
	}
	return base + "/chat/completions", nil
}

func openAIHeaders() map[string]string {
	return map[string]string{
		"Authorization":   "Bearer " + strings.TrimSpace(global.GVA_CONFIG.AutoCode.AiKey),
		"Content-Type":    "application/json",
		"Accept":          "text/event-stream",
		"Accept-Encoding": "identity", // 同官方通道：禁 gzip，否则 SSE 会被缓冲住
		"Cache-Control":   "no-cache",
	}
}

// codeGenModes 是「代码生成器」页面用到的 mode。
// 这两个 mode 的返回结果会被前端直接 JSON.parse / 直接当描述文本用，
// 与聊天类的 analysis / workflow 在输出约束上不同，所以单独归类。
var codeGenModes = map[string]bool{
	"ai":  true, // 中文描述 -> 结构体 + 字段定义
	"eye": true, // 图片（建表截图 / 纸质表单照片）-> 中文描述
}

// modelForMode 取该 mode 应使用的模型。
//
// 三种页面各自可配：
//   - ai   ：ai-codegen-model（吐 JSON 要稳），留空继承 ai-model
//   - eye  ：ai-vision-model（必须支持 image 输入），留空继承 ai-model
//   - 其它 ：ai-model（长文本推理）
//
// [坑] eye 绝对不能跟 ai 共用一个模型。代码生成最稳的模型常常是纯文本模型
// （deepseek-v4-flash 的 input_modalities 只有 text），网关会静默丢掉图片内容，
// 模型只看到「请识别这张图片」这句话，于是回一句「我无法从图片中识别字段」，
// 前端再把这句话当需求描述发给 ai，最后报的是 JSON 解析失败——离真正的原因隔了三层。
func modelForMode(mode string) string {
	cfg := global.GVA_CONFIG.AutoCode
	switch mode {
	case "eye":
		if m := strings.TrimSpace(cfg.AiVisionModel); m != "" {
			return m
		}
	case "ai":
		if m := strings.TrimSpace(cfg.AiCodegenModel); m != "" {
			return m
		}
	}
	return strings.TrimSpace(cfg.AiModel)
}

// jsonMapString 安全取 JSONMap 里的字符串值。
// [注意] 不能直接 fmt.Sprintf("%v", m[key])：键不存在时 m[key] 是 nil，
// Sprintf 会得到字符串 "<nil>"，既非空串也能通过 !="" 判断，会被当成真值发出去。
func jsonMapString(m common.JSONMap, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "<nil>" {
		return ""
	}
	return s
}

// buildOpenAIMessages 把 Dify 风格请求翻译成 chat messages。
//
// 三类请求体对应三种形态：
//   - analysisChat / workflowPromptChat：{query, inputs}
//   - ai（代码生成器）：{prompt}
//   - eye（图片识别）：{_file_path}，值是 data:image/png;base64,... 的 data URL，要发多模态消息
func buildOpenAIMessages(llm common.JSONMap) ([]map[string]any, error) {
	mode := jsonMapString(llm, "mode")
	prompt, ok := openAISystemPrompts[mode]
	if !ok {
		return nil, fmt.Errorf("mode=%q 在 openai 通道下没有对应提示词，支持：analysisChat / workflowPromptChat / ai / eye", mode)
	}
	// 输出约束按场景分三类：
	//   ai  —— 前端整段 JSON.parse，必须只吐 JSON 本体（不许围栏）
	//   eye —— 前端把结果直接当描述文本用，必须纯文本，不能加 JSON 约束
	//   其余 —— 聊天类，允许正文 + 一个 json 代码块
	switch mode {
	case "ai":
		prompt += jsonOnlyRules
	case "eye":
		// 不加任何 JSON 约束
	default:
		prompt += jsonOutputRules
	}

	// eye：图片转描述，走多模态消息
	if mode == "eye" {
		image := jsonMapString(llm, "_file_path")
		if image == "" {
			return nil, errors.New("mode=eye 需要 _file_path（图片 data URL），请重新选择图片")
		}
		if !strings.HasPrefix(image, "data:") {
			return nil, errors.New("mode=eye 的 _file_path 必须是 data:image/...;base64, 开头的 data URL")
		}
		parts := []map[string]any{
			{"type": "text", "text": "请识别这张图片里的业务表结构，按要求输出中文描述。"},
			{"type": "image_url", "image_url": map[string]string{"url": image}},
		}
		return []map[string]any{
			{"role": "system", "content": prompt},
			{"role": "user", "content": parts},
		}, nil
	}

	parts := make([]string, 0, 2)
	// 代码生成器用的是 prompt 字段，聊天类用的是 query
	if q := jsonMapString(llm, "prompt"); q != "" {
		parts = append(parts, q)
	}
	if q := jsonMapString(llm, "query"); q != "" {
		parts = append(parts, q)
	}
	// inputs 是页面上那一堆表单（包类型、业务场景、额外约束、是否含客户端页面等）
	if inputs := llm["inputs"]; inputs != nil {
		if raw, err := json.MarshalIndent(inputs, "", "  "); err == nil {
			if s := strings.TrimSpace(string(raw)); s != "" && s != "{}" && s != "null" && s != "[]" {
				parts = append(parts, "结构化输入：\n```json\n"+s+"\n```")
			}
		}
	}
	if len(parts) == 0 {
		return nil, errors.New("请求缺少 prompt / query / inputs，没有内容可发送")
	}

	return []map[string]any{
		{"role": "system", "content": prompt},
		{"role": "user", "content": strings.Join(parts, "\n\n")},
	}, nil
}

func buildOpenAIPayload(llm common.JSONMap, stream bool) (common.JSONMap, error) {
	msgs, err := buildOpenAIMessages(llm)
	if err != nil {
		return nil, err
	}

	mode := jsonMapString(llm, "mode")
	payload := common.JSONMap{
		"model":    modelForMode(mode),
		"messages": msgs,
		"stream":   stream,
	}
	// 允许前端通过 extraPayload 覆盖采样参数
	if v, ok := llm["temperature"]; ok && v != nil {
		payload["temperature"] = v
	}
	return payload, nil
}

// openAI限流重试相关。
// [背景] 免费额度的 key 额度极小：实测连续 2 次请求后第 3 次必 429，
// 之后约 20~60 秒才恢复一个名额。上游**不返回 Retry-After**，
// 也没有 x-ratelimit-* 头，只能按固定退避盲试。
// 三次退避合计约 50 秒，正好能覆盖一个恢复窗口；再久用户就等不下去了。
const openAIRateLimitRetries = 2

var openAIRateLimitBackoff = []time.Duration{
	8 * time.Second,
	25 * time.Second,
	40 * time.Second,
}

// upstreamRateLimited 表示上游 429，属于可重试错误。
// 单独定义类型，是为了让上层能区分「可以等一等再试」和「试了也没用」。
type upstreamRateLimited struct {
	status int
	body   string
}

func (e *upstreamRateLimited) Error() string {
	return fmt.Sprintf("上游限流: status=%d body=%s", e.status, e.body)
}

// rateLimitMessage 把上游 429 翻译成用户能照做的提示。
// retried 为已自动重试次数；流式请求没法重试，传 0。
func rateLimitMessage(model string, retried int, body string) string {
	head := fmt.Sprintf("上游限流（429，模型=%s）", model)
	if retried > 0 {
		head += fmt.Sprintf("，已自动重试 %d 次仍未恢复", retried)
	} else {
		head += "，流式请求不便自动重试"
	}
	return head + "。\n" +
		"这个 key 的额度很小（实测连续 2~3 次请求就会打满，约 20~60 秒恢复一个名额），建议：\n" +
		"1. 等 1 分钟左右再点一次；\n" +
		"2. 在 config.yaml 把 autocode.ai-codegen-model / ai-vision-model 换成额度更宽的模型；\n" +
		"3. 或者换一个额度更大的 key。\n" +
		"原始返回：" + body
}

// openAIChat 非流式调用，供 LLMAuto 使用
func (s *AutoCodeService) openAIChat(ctx context.Context, llm common.JSONMap) (interface{}, error) {
	path, err := openAIEndpoint()
	if err != nil {
		return nil, err
	}
	payload, err := buildOpenAIPayload(llm, false)
	if err != nil {
		return nil, err
	}

	var content string
	for attempt := 0; ; attempt++ {
		content, err = s.openAIChatOnce(ctx, path, payload)
		if err == nil {
			break
		}

		var limited *upstreamRateLimited
		if !errors.As(err, &limited) || attempt >= openAIRateLimitRetries {
			if errors.As(err, &limited) {
				// 重试次数用完，换一句人话，别把上游原始 JSON 直接抛给用户
				return nil, errors.New(rateLimitMessage(modelForMode(modeOf(llm)), openAIRateLimitRetries, limited.body))
			}
			return nil, err
		}

		wait := openAIRateLimitBackoff[attempt]
		if wait > openAIRateLimitBackoff[len(openAIRateLimitBackoff)-1] {
			wait = openAIRateLimitBackoff[len(openAIRateLimitBackoff)-1]
		}
		// 测试里 GVA_LOG 可能没初始化
		if global.GVA_LOG != nil {
			global.GVA_LOG.Warn("上游限流，等待后重试",
				zap.String("model", modelForMode(modeOf(llm))),
				zap.Int("retry", attempt+1),
				zap.Duration("wait", wait))
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("等待上游限流恢复时被取消: %w", ctx.Err())
		case <-time.After(wait):
		}
	}

	// 代码生成器（mode=ai）拿到的 text 会被前端直接 JSON.parse，这里先兜一层：
	// 有些模型习惯性加 ```json 围栏，剥掉；剥完还不是合法 JSON 就直接报错，
	// 否则前端会抛未捕获异常、页面上什么都不显示，很难排查。
	if modeOf(llm) == "ai" {
		content = stripJSONFence(content)
		if !json.Valid([]byte(content)) {
			return nil, fmt.Errorf("模型返回的不是合法 JSON（已尝试剥掉代码围栏）。model=%s 开头 200 字：%s",
				modelForMode("ai"), previewResponseBody([]byte(content)))
		}
		// 实测模型经常把 dataType 的取值（text / varchar / datetime）写进 fieldType。
		// 前端 fieldType 是固定下拉框，非法值会让该行选不中、生成模板时类型映射也失效，
		// 所以在服务端做一次归一化，而不是指望提示词一次说对。
		content = normalizeCodegenFieldTypes(content)
		// package 与 abbreviation 撞名会让生成的代码编译不过（变量遮蔽同名包），
		// 模型又特别爱把两者填成同一个词，这里兜一层。
		content = normalizeCodegenNames(content)
	}
	// 图片识别（mode=eye）要额外验一下：模型「没看见图」时也会正常返回 200 + 一段道歉，
	// 前端拿到这种文本会当成需求描述继续往下走，最后报一个跟图片毫无关系的 JSON 错误。
	// 在这里拦住，错误信息就能指到真正的毛病（模型不支持图片）。
	if modeOf(llm) == "eye" {
		if err := checkEyeDescription(content, modelForMode("eye")); err != nil {
			return nil, err
		}
	}
	// 统一成前端 pickStreamText 认识的字段：answer / text
	return common.JSONMap{"answer": content, "text": content, "provider": providerName()}, nil
}

// openAIChatOnce 发一次请求并取出 content。429 单独包装成可重试错误。
func (s *AutoCodeService) openAIChatOnce(ctx context.Context, path string, payload common.JSONMap) (string, error) {
	res, err := request.HttpRequestWithContextAndTimeout(ctx, path, http.MethodPost, openAIHeaders(), nil, payload, -1)
	if err != nil {
		return "", fmt.Errorf("调用 OpenAI 兼容中转站失败: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("读取中转站响应失败: %w", err)
	}
	if res.StatusCode == http.StatusTooManyRequests {
		return "", &upstreamRateLimited{status: res.StatusCode, body: previewResponseBody(body)}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("中转站返回非 2xx: status=%d body=%s", res.StatusCode, previewResponseBody(body))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("解析中转站响应失败: body=%s err=%w", previewResponseBody(body), err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("中转站响应缺少 choices: body=%s", previewResponseBody(body))
	}
	return parsed.Choices[0].Message.Content, nil
}

// modeOf 取请求里的 mode，去掉首尾空白
func modeOf(llm common.JSONMap) string {
	return jsonMapString(llm, "mode")
}

// stripJSONFence 去掉模型加的 ```json ... ``` 围栏，返回纯 JSON 文本。
// 没有围栏时原样返回。
func stripJSONFence(s string) string {
	t := strings.TrimSpace(s)
	if !strings.HasPrefix(t, "```") {
		return t
	}
	// 去掉第一行围栏（``` 或 ```json）
	if idx := strings.IndexByte(t, '\n'); idx >= 0 {
		t = t[idx+1:]
	} else {
		return t
	}
	// 去掉结尾的 ```
	if idx := strings.LastIndex(t, "```"); idx >= 0 {
		t = t[:idx]
	}
	return strings.TrimSpace(t)
}

// eyeRefusalMarkers 是「模型其实没看到图」时的典型措辞。
// 纯文本模型收到带 image_url 的请求时，网关通常不报错、只把图片丢掉，
// 模型于是用这些说法反过来要用户提供文字内容。
var eyeRefusalMarkers = []string{
	"无法从图片", "无法识别", "无法辨认", "无法看清", "看不清",
	"图片为空", "空白图片", "空白或无效", "无效图片", "无法提取", "无法读取",
	"请提供", "抱歉", "没有图片", "未检测到图片", "不是图片",
	"unable to", "cannot identify", "can't identify", "no image",
}

// checkEyeDescription 校验 mode=eye 的返回是否真的是一段表结构描述。
// 返回 nil 表示可用；否则给出能落到配置上的错误，而不是让调用方去猜。
func checkEyeDescription(content, model string) error {
	t := strings.TrimSpace(content)
	if t == "" {
		return fmt.Errorf("图片识别返回了空内容（model=%s）。该模型很可能不接受图片输入，"+
			"请把 autocode.ai-vision-model 配成支持 image 的模型（例如 sensenova-6.8-flash-lite）", model)
	}
	lower := strings.ToLower(t)
	for _, marker := range eyeRefusalMarkers {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("图片识别没拿到表结构描述（model=%s），模型回的是：「%s」。\n"+
				"这通常说明该模型不接受图片输入（input_modalities 只有 text），"+
				"请把 autocode.ai-vision-model 配成支持 image 的模型（例如 sensenova-6.8-flash-lite），"+
				"或者换一张更清晰的建表截图/表单照片重试",
				model, previewResponseBody([]byte(t)))
		}
	}
	return nil
}

// codegenFieldTypeCanonical 是 fieldType 的合法取值（大小写不敏感 -> 规范写法）。
// 与前端 autoCode/index.vue 的 typeOptions 一一对应，改一边必须同步改另一边。
var codegenFieldTypeCanonical = map[string]string{
	"string":    "string",
	"richtext":  "richtext",
	"int":       "int",
	"bool":      "bool",
	"float64":   "float64",
	"time.time": "time.Time",
	"enum":      "enum",
	"picture":   "picture",
	"pictures":  "pictures",
	"video":     "video",
	"file":      "file",
	"json":      "json",
	"array":     "array",
}

// codegenFieldTypeAlias 是模型常见的误写 -> 合法值。
// 模型很容易把数据库类型（dataType 那一列）塞进 fieldType，这里按语义往回映射。
var codegenFieldTypeAlias = map[string]string{
	"text":       "richtext", // 长文本
	"longtext":   "richtext",
	"mediumtext": "richtext",
	"tinytext":   "richtext",
	"varchar":    "string",
	"char":       "string",
	"nvarchar":   "string",
	"uuid":       "string",
	"email":      "string",
	"url":        "string",
	"password":   "string",
	"bigint":     "int",
	"smallint":   "int",
	"mediumint":  "int",
	"tinyint":    "int",
	"integer":    "int",
	"number":     "int",
	"decimal":    "float64",
	"numeric":    "float64",
	"float":      "float64",
	"double":     "float64",
	"money":      "float64",
	"datetime":   "time.Time",
	"date":       "time.Time",
	"timestamp":  "time.Time",
	"time":       "time.Time",
	"boolean":    "bool",
	"bit":        "bool",
	"jsonb":      "json",
	"image":      "picture",
	"img":        "picture",
	"images":     "pictures",
}

// normalizeCodegenFieldType 把模型给的 fieldType 归一化成前端能识别的枚举值。
// 认不出来的按 string 处理——宁可变字符串，也不要留一个前端选不中的值。
func normalizeCodegenFieldType(raw string) string {
	k := strings.ToLower(strings.TrimSpace(raw))
	// "varchar(255)" / "tinyint(1)" 这类带长度的，先摘掉括号部分
	if idx := strings.IndexByte(k, '('); idx > 0 {
		p := strings.TrimSpace(k[:idx])
		if p == "tinyint" {
			// tinyint(1) 在 MySQL 里就是布尔
			if strings.Contains(k, "(1)") {
				return "bool"
			}
		}
		k = p
	}
	if v, ok := codegenFieldTypeCanonical[k]; ok {
		return v
	}
	if v, ok := codegenFieldTypeAlias[k]; ok {
		return v
	}
	return "string"
}

// normalizeCodegenFieldTypes 就地修正 fields[].fieldType，返回新的 JSON 文本。
// 只在 mode=ai 且内容已是合法 JSON 时调用，解析失败就原样返回，不掩盖原始输出。
func normalizeCodegenFieldTypes(content string) string {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		return content
	}
	rawFields, ok := root["fields"]
	if !ok {
		return content
	}
	var fields []map[string]json.RawMessage
	if err := json.Unmarshal(rawFields, &fields); err != nil {
		return content
	}

	changed := false
	for _, f := range fields {
		raw, ok := f["fieldType"]
		if !ok {
			continue
		}
		var ft string
		if err := json.Unmarshal(raw, &ft); err != nil {
			continue
		}
		norm := normalizeCodegenFieldType(ft)
		if norm == ft {
			continue
		}
		if b, err := json.Marshal(norm); err == nil {
			f["fieldType"] = b
			changed = true
		}
	}
	if !changed {
		return content
	}

	nb, err := json.Marshal(fields)
	if err != nil {
		return content
	}
	root["fields"] = nb
	out, err := json.Marshal(root)
	if err != nil {
		return content
	}
	return string(out)
}

// normalizeCodegenNames 保证 package / abbreviation / structName 三者不撞名。
//
// 模板里渲染的是 `var {{.Abbreviation}} {{.Package}}.{{.StructName}}`，
// package 与 abbreviation 同名会变成 `var book book.Book` —— 变量遮蔽了同名包，
// 生成的代码直接编译失败；前端提交前也会拦下来（"package和结构体简称不可同名"）。
// abbreviation 与 structName 同名同理（`var Book Book.Book`）。
//
// 这里只改 abbreviation（自由文本），不动 package：package 由页面下拉框从模板包列表里选，
// 改成任意值会让下拉选不中，反而更麻烦。
// 只在 mode=ai 且内容已是合法 JSON 时调用，解析失败原样返回。
func normalizeCodegenNames(content string) string {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &root); err != nil {
		return content
	}
	str := func(key string) string {
		raw, ok := root[key]
		if !ok {
			return ""
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return ""
		}
		return strings.TrimSpace(s)
	}

	structName := str("structName")
	pkg := str("package")
	abbr := str("abbreviation")

	// 合法条件：非空，且既不是包名也不是结构体名
	valid := func(v string) bool {
		return v != "" && v != pkg && v != structName
	}
	if valid(abbr) {
		return content
	}

	// 依次尝试：structName 小驼峰 → 包名+Obj → structName 全小写
	var candidates []string
	if c := lowerFirstASCII(structName); c != "" {
		candidates = append(candidates, c)
	}
	if pkg != "" {
		candidates = append(candidates, pkg+"Obj")
	}
	if c := strings.ToLower(structName); c != "" {
		candidates = append(candidates, c)
	}
	chosen := ""
	for _, c := range candidates {
		if valid(c) {
			chosen = c
			break
		}
	}
	if chosen == "" {
		// 实在没法安全改名就原样返回，让前端按自己的规则报错，不擅自编造
		return content
	}

	nb, err := json.Marshal(chosen)
	if err != nil {
		return content
	}
	root["abbreviation"] = nb
	out, err := json.Marshal(root)
	if err != nil {
		return content
	}
	return string(out)
}

// lowerFirstASCII 把首字母转小写（只处理 ASCII，够用于 Go 标识符）。
func lowerFirstASCII(s string) string {
	if s == "" {
		return ""
	}
	b := []byte(s)
	if b[0] >= 'A' && b[0] <= 'Z' {
		b[0] += 'a' - 'A'
	}
	return string(b)
}

// openAIStream 流式调用，供 LLMAutoStream 使用。
// 返回的 http.Response.Body 是原始 OpenAI SSE，按帧翻译的活交给 api 层做。
func (s *AutoCodeService) openAIStream(ctx context.Context, llm common.JSONMap) (*http.Response, error) {
	path, err := openAIEndpoint()
	if err != nil {
		return nil, err
	}
	payload, err := buildOpenAIPayload(llm, true)
	if err != nil {
		return nil, err
	}

	res, err := request.HttpRequestWithContextAndTimeout(ctx, path, http.MethodPost, openAIHeaders(), nil, payload, -1)
	if err != nil {
		return nil, fmt.Errorf("调用 OpenAI 兼容中转站流式接口失败: %w", err)
	}
	// 流式已经开了头就没法重试，429 只能在这里翻译成人话交给上层
	if res.StatusCode == http.StatusTooManyRequests {
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return nil, errors.New(rateLimitMessage(modelForMode(modeOf(llm)), 0, previewResponseBody(body)))
	}
	return res, nil
}

// extractOpenAIDelta 从单个 SSE 分片里取增量文本。
// 第二个返回值 false 表示这一帧没有可展示文本（首帧 role、结束帧、心跳等），直接跳过。
func ExtractOpenAIDelta(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[DONE]" {
		return "", false
	}

	var chunk struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			Text string `json:"text"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
		return "", false
	}
	if len(chunk.Choices) == 0 {
		return "", false
	}

	if text := chunk.Choices[0].Delta.Content; text != "" {
		return text, true
	}
	if text := chunk.Choices[0].Message.Content; text != "" {
		return text, true
	}
	if text := chunk.Choices[0].Text; text != "" {
		return text, true
	}
	return "", false
}

// openAIChunkError 部分中转站会边流边报错（额度不足、模型不存在），
// 这种帧里没有 choices，只有 error，需要单独捞出来推给前端。
func OpenAIChunkError(raw string) string {
	var chunk struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &chunk); err != nil {
		return ""
	}
	return strings.TrimSpace(chunk.Error.Message)
}

// openAISystemPrompts 各 mode 的系统提示词。
// [注意] 输出契约必须和 web/src/view/systemTools/aiWrokflow/index.vue 里的
// normalizeAnalysis / normalizeWorkflow 对齐，字段名写错前端就渲染成空。
// jsonOutputRules 追加到每个 prompt 末尾，约束模型输出能被 JSON.parse 直接解析的内容。
// 起因：模型在 prompt 字段里写 Go 代码时把反引号转义成 \`，导致整份 JSON 解析失败，
// 前端只能退化成"步骤 1"。
const jsonOutputRules = `

【输出格式硬性要求】
1. 全文只允许出现一个 json 代码块，且 JSON 字符串内部不要再嵌套任何围栏。
2. 字符串里的文件路径一律用正斜杠 /，禁止出现 Windows 反斜杠路径。
3. 不要对反引号、单引号、括号做反斜杠转义，直接写字面字符。
4. 所有换行必须写成 \n 转义，不能直接敲回车。
5. 最终输出的 JSON 必须能被标准 JSON 解析器一次性解析成功。`

// jsonOnlyRules 用于「前端会把整段返回直接 JSON.parse」的场景（代码生成器）。
// 与 jsonOutputRules 的区别：这里连 json 代码块都不允许，必须从第一个字符就是 { 或 [。
const jsonOnlyRules = `

【输出格式硬性要求】
1. 直接输出 JSON 本体，从 { 开始、到 } 结束，禁止任何解释性文字。
2. 禁止使用 markdown 代码围栏（不要写三个反引号），禁止在 JSON 前后加说明。
3. 字符串里的文件路径一律用正斜杠 /。
4. 不要对反引号、单引号、括号做反斜杠转义，直接写字面字符。
5. 所有换行必须写成 \n 转义，不能直接敲回车。
6. 布尔值写 true / false，不要写成字符串；不确定的数字字段给 0。`

var openAISystemPrompts = map[string]string{
	// ai：代码生成器「用中文描述建表」-> 直接填进页面表单的结构化结果。
	// [注意] 这份 JSON 会被前端 JSON.parse 后整对象摊到 form 上，少一个键就是空值，
	// 所以下面把 form 的全部键都列进模板里，让模型照着填，而不是自由发挥。
	"ai": `你是 gin-vue-admin（GVA）框架的代码生成器。用户会用中文描述一张业务表，你要把它转换成 GVA 自动代码需要的结构体与字段定义。

严格按下面的 JSON 结构输出（键名一字不差，不要增删）：
{
  "structName": "结构体名，PascalCase，例如 Book",
  "tableName": "表名，snake_case 复数，例如 books",
  "packageName": "生成文件名，小驼峰，例如 book",
  "package": "Go 包名（模板包名），全小写，例如 book；必须与 abbreviation 不同",
  "abbreviation": "结构简称，小驼峰，作为变量名和路由 group，例如 bookInfo；必须与 package 不同",
  "description": "结构体中文说明，例如 图书",
  "businessDB": "",
  "autoCreateApiToSql": true,
  "autoCreateMenuToSql": true,
  "autoCreateBtnAuth": false,
  "autoMigrate": true,
  "gvaModel": true,
  "autoCreateResource": false,
  "onlyTemplate": false,
  "isTree": false,
  "generateWeb": true,
  "generateServer": true,
  "treeJson": "",
  "fields": [
    {
      "fieldName": "字段名，PascalCase，例如 Title",
      "fieldDesc": "字段中文名，例如 书名",
      "fieldType": "只能取这些值之一：string / richtext / int / bool / float64 / time.Time / enum / picture / pictures / video / file / json / array",
      "dataType": "数据库类型，例如 varchar(255)、int、bigint、tinyint(1)、decimal(10,2)、datetime、text、longtext",
      "fieldJson": "前端 JSON 里的键名，camelCase，例如 title",
      "columnName": "数据库列名，snake_case，例如 title",
      "dataTypeLong": "",
      "comment": "数据库注释，一般与 fieldDesc 相同",
      "defaultValue": "",
      "require": true,
      "sort": false,
      "form": true,
      "desc": true,
      "table": true,
      "excel": false,
      "errorText": "",
      "primaryKey": false,
      "clearable": true,
      "fieldSearchType": "取 = / <> / > / < / LIKE / BETWEEN / NOT BETWEEN 之一，不需要搜索就留空字符串",
      "fieldIndexType": "取 index / uniqueIndex 之一，普通字段留空字符串",
      "dictType": "",
      "dataSource": { "dbName": "", "association": 1, "table": "", "label": "", "value": "", "hasDeletedAt": false }
    }
  ]
}

要求：
- 只输出上面这一个 JSON 对象，不要解释、不要代码围栏。
- fieldType 只能取那 13 个值之一，绝对不能写数据库类型。text / longtext / varchar / datetime / decimal / tinyint 这些都是 dataType，不是 fieldType：长文本必须写 fieldType=richtext 且 dataType=text。
- 不要生成 id / created_at / updated_at / deleted_at / created_by / updated_by / deleted_by，这些由 gvaModel 自动带上。
- 一张表通常 4-12 个业务字段。用户描述里提到的字段必须全部包含，用户没提但业务必需的（如状态、排序、备注）可以补，但不要臆造大量无关字段。
- 是否必填按业务判断：名称、编号一类必填；备注、描述一类不必填。
- 字符串类字段 default varchar(255)，长文本用 text；金额用 decimal(10,2) 且 fieldType 为 float64；时间用 datetime 且 fieldType 为 time.Time；开关用 tinyint(1) 且 fieldType 为 bool。
- 枚举类（状态、类型）fieldType 用 enum，dictType 留空字符串（后续由用户自己配字典）。
- 中文描述里如果提到"树形/层级/上下级"，isTree 置 true；否则 false。
- structName、packageName、abbreviation 必须是合法 Go 标识符，不能含数字开头或中文。
- **package 和 abbreviation 绝对不能相同**。模板会生成 var {{abbreviation}} {{package}}.{{structName}}，
  两者同名时变量会遮蔽同名包，生成的代码直接编译失败。例如 package=book 时 abbreviation 要写成 bookInfo 或 bk。
- abbreviation 也不能与 structName 相同（structName 是 PascalCase 大写开头，abbreviation 用小驼峰）。`,

	// eye：把建表截图 / 纸质表单照片转成一段中文描述，前端会把它当 prompt 再喂给 ai。
	"eye": `你是 gin-vue-admin（GVA）代码生成器的需求识别助手。用户会发一张图片，内容通常是数据库设计图、Excel 表结构、纸质表单或系统界面截图。

请把它整理成一段可直接用于代码生成的中文需求描述，格式如下：

这是一张【业务名称】相关的表，用于【这张表解决什么问题】。
需要以下字段：
1. 字段中文名（是否必填）：含义与取值范围，例如 状态（必填）：1 启用 0 停用
2. ……

要求：
- 只输出描述正文，不要输出 JSON，不要输出 markdown 代码围栏，不要加"以下是"之类的客套话。
- 字段顺序按图片里出现的顺序。
- 图片里能看出的枚举值、长度限制、默认值都要写进描述，看不出来就不要编。
- 识别不出业务含义时，如实说明"图片中部分字段无法辨识"，不要猜测具体内容。
- 不要写 id、创建时间、更新时间这类 GVA 自带的字段。`,

	"analysisChat": `你是 gin-vue-admin（GVA）框架的资深架构师，负责把业务需求拆解成可落地的后端模块与前端页面。

请先通读用户需求与结构化输入，然后：
1. 用 2-5 句中文概述你对需求的理解、技术方案与风险点。
2. 在最后给出一个 json 代码块，严格使用下面的字段（不要增删顶层字段）：

` + "```json" + `
{
  "summary": "一句话总结这次要做什么",
  "recommendedPackageType": "single 或 plugin（普通业务模块填 single，需要独立安装/卸载的填 plugin）",
  "missingInfo": ["还需要用户确认的问题，没有就留空数组"],
  "suggestions": ["实现建议、注意事项"],
  "modules": [
    {
      "name": "英文小写模块名，例如 book",
      "label": "中文模块名，例如 图书",
      "description": "这个模块负责什么",
      "fields": [
        {
          "name": "英文小写字段名，例如 title",
          "label": "中文名，例如 书名",
          "type": "string / int / float / bool / time",
          "required": true,
          "description": "字段含义与约束"
        }
      ]
    }
  ],
  "clientPages": [
    {
      "name": "英文小写页面名，例如 bookList",
      "label": "中文页面名，例如 图书列表",
      "description": "页面目标",
      "pageType": "list / form / detail / dashboard",
      "targetModules": ["book"],
      "fields": [
        {
          "name": "页面字段名",
          "label": "中文名",
          "sourceModule": "来源模块名",
          "sourceField": "来源字段名",
          "displayType": "text / input / select / date / number",
          "required": true,
          "description": "展示或录入要求"
        }
      ]
    }
  ]
}
` + "```" + `

要求：
- 字段名一律英文小写，可用下划线；label 一律中文。
- 每个模块至少给出主键以外的关键业务字段，不要写 id / created_at / updated_at 这类 GVA 自带字段。
- 用户没提到前端页面时，clientPages 给空数组。
- 除概述和这个 json 块之外，不要输出其他内容。`,

	"workflowPromptChat": `你是 gin-vue-admin（GVA）框架的交付专家，负责把需求拆成一份可以逐步交给 AI 编码工具执行的 Prompt 工作流。

GVA 后端固定分层是 Router -> API -> Service -> Model，新增一个业务模块必须按顺序处理这几处注册点：
model 定义、service、api、router、service/enter.go、api/v1/enter.go、router/enter.go、initialize/gorm_biz.go、initialize/router_biz.go，最后是前端 api 封装与页面、菜单与权限。

请先通读用户需求与结构化输入，然后：
1. 用 2-4 句中文说明这条工作流的执行顺序与关键风险。
2. 在最后给出一个 json 代码块，严格使用下面的字段：

` + "```json" + `
{
  "summary": "一句话总结这份工作流的目标",
  "steps": [
    {
      "title": "步骤标题，例如 定义 Book 模型",
      "goal": "这一步要达成什么",
      "prompt": "可以直接复制给 AI 编码工具的完整指令，必须写清文件路径、分层位置、字段定义与注意事项",
      "expectedOutput": "这一步应该产出什么文件或结果",
      "suggestedTool": "建议使用的工具，例如 代码编辑器 / 终端 / 数据库客户端",
      "autoExecutable": true
    }
  ]
}
` + "```" + `

要求：
- steps 顺序必须符合 GVA 的真实依赖关系，不要跳过 enter.go 与 initialize 下的注册点。
- 步骤要"粗"不要"碎"：同一个文件的改动合并成一步（例如 model + 迁移、service + enter 注册、router + router_biz 注册），
  总步数控制在 6-10 步为宜。不要把某个注册点单独拆成一步，那样用户要复制十几次才能做完。
- 每个 prompt 都要自包含、可直接粘贴执行：写清文件路径、分层位置、字段定义、方法签名与关键注意事项，
  让 AI 不看上文也能照着把这一步做完。
- 除说明和这个 json 块之外，不要输出其他内容。`,
}
