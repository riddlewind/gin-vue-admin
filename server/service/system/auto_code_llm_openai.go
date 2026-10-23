package system

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/request"
	"github.com/goccy/go-json"
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

// buildOpenAIMessages 把 Dify 风格请求翻译成 chat messages
func buildOpenAIMessages(llm common.JSONMap) ([]map[string]string, error) {
	mode := strings.TrimSpace(fmt.Sprintf("%v", llm["mode"]))
	prompt, ok := openAISystemPrompts[mode]
	if !ok {
		return nil, fmt.Errorf("mode=%q 在 openai 通道下没有对应提示词，仅支持 analysisChat / workflowPromptChat", mode)
	}
	// 模型在 JSON 字符串里写代码片段时，很容易把反引号转义成 \`，而 JSON 只认
	// \" \\ \/ \b \f \n \r \t \uXXXX，一个非法转义就会让前端整段解析失败。
	// 这里统一追加输出约束，减少模型输出非法 JSON 的概率。
	prompt += jsonOutputRules

	parts := make([]string, 0, 2)
	if q := strings.TrimSpace(fmt.Sprintf("%v", llm["query"])); q != "" {
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
		return nil, errors.New("请求缺少 query 与 inputs，没有内容可发送")
	}

	return []map[string]string{
		{"role": "system", "content": prompt},
		{"role": "user", "content": strings.Join(parts, "\n\n")},
	}, nil
}

func buildOpenAIPayload(llm common.JSONMap, stream bool) (common.JSONMap, error) {
	msgs, err := buildOpenAIMessages(llm)
	if err != nil {
		return nil, err
	}

	payload := common.JSONMap{
		"model":    strings.TrimSpace(global.GVA_CONFIG.AutoCode.AiModel),
		"messages": msgs,
		"stream":   stream,
	}
	// 允许前端通过 extraPayload 覆盖采样参数
	if v, ok := llm["temperature"]; ok && v != nil {
		payload["temperature"] = v
	}
	return payload, nil
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

	res, err := request.HttpRequestWithContextAndTimeout(ctx, path, http.MethodPost, openAIHeaders(), nil, payload, -1)
	if err != nil {
		return nil, fmt.Errorf("调用 OpenAI 兼容中转站失败: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("读取中转站响应失败: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("中转站返回非 2xx: status=%d body=%s", res.StatusCode, previewResponseBody(body))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err = json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("解析中转站响应失败: body=%s err=%w", previewResponseBody(body), err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("中转站响应缺少 choices: body=%s", previewResponseBody(body))
	}

	content := parsed.Choices[0].Message.Content
	// 统一成前端 pickStreamText 认识的字段：answer / text
	return common.JSONMap{"answer": content, "text": content, "provider": providerName()}, nil
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

var openAISystemPrompts = map[string]string{
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
