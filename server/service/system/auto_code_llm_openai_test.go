package system

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"github.com/goccy/go-json"
)

// OpenAI 兼容通道的回归测试。
// 前四个是纯离线断言（模型选择、nil 处理、围栏剥离、fieldType 归一化）；
// TestLiveCodegenAI 会真实请求上游，需要设置 GVA_TEST_AI_KEY，未设置时自动跳过。
// [注意] 本包已有测试文件带 vet 报错，跑测试要加 -vet=off。

func setupLiveCfg(t *testing.T) {
	t.Helper()
	global.GVA_CONFIG.AutoCode.AiProvider = "sensenova"
	global.GVA_CONFIG.AutoCode.AiBaseURL = "https://token.sensenova.cn/v1"
	global.GVA_CONFIG.AutoCode.AiKey = os.Getenv("GVA_TEST_AI_KEY")
	global.GVA_CONFIG.AutoCode.AiModel = "sensenova-6.8-flash-lite"
	global.GVA_CONFIG.AutoCode.AiCodegenModel = "deepseek-v4-flash"
	global.GVA_CONFIG.AutoCode.AiVisionModel = "sensenova-6.8-flash-lite"
}

func TestRateLimitMessage(t *testing.T) {
	msg := rateLimitMessage("deepseek-v4-flash", openAIRateLimitRetries, `{"error":{"code":"429003"}}`)

	if !strings.Contains(msg, "429") {
		t.Errorf("应点明 429：%s", msg)
	}
	if !strings.Contains(msg, "deepseek-v4-flash") {
		t.Errorf("应带上模型名，便于判断换哪个：%s", msg)
	}
	if !strings.Contains(msg, "重试") {
		t.Errorf("应说明已经重试过：%s", msg)
	}
	// 必须给出可操作的建议，而不是丢了上游 JSON 就完事
	for _, want := range []string{"等 1 分钟", "ai-codegen-model", "换一个额度"} {
		if !strings.Contains(msg, want) {
			t.Errorf("缺少建议片段 %q：%s", want, msg)
		}
	}
	// 原始返回要保留，方便排查是不是别的 429
	if !strings.Contains(msg, "429003") {
		t.Errorf("应保留上游原始返回：%s", msg)
	}

	// 流式路径没法重试，文案不能谎称"已自动重试"
	streamMsg := rateLimitMessage("deepseek-v4-flash", 0, "x")
	if strings.Contains(streamMsg, "已自动重试") {
		t.Errorf("流式路径不该说已重试：%s", streamMsg)
	}
	if !strings.Contains(streamMsg, "不便自动重试") {
		t.Errorf("流式路径应说明不能重试：%s", streamMsg)
	}
}

func TestUpstreamRateLimitedIsRetryable(t *testing.T) {
	var err error = &upstreamRateLimited{status: 429, body: "x"}
	var target *upstreamRateLimited
	if !errors.As(err, &target) {
		t.Fatal("errorAs 应能匹配到 upstreamRateLimited")
	}
	// 包装一层包装仍要能被识别，否则外面加 fmt.Errorf 就退化成不可重试
	wrapped := fmt.Errorf("外层包装: %w", err)
	if !errors.As(wrapped, &target) {
		t.Fatal("经过 fmt.Errorf 包装后应仍可识别为可重试")
	}
}

func TestModelForMode(t *testing.T) {
	setupLiveCfg(t)
	cases := map[string]string{
		"ai":                 "deepseek-v4-flash",
		"eye":                "sensenova-6.8-flash-lite",
		"analysisChat":       "sensenova-6.8-flash-lite",
		"workflowPromptChat": "sensenova-6.8-flash-lite",
	}
	for mode, want := range cases {
		if got := modelForMode(mode); got != want {
			t.Fatalf("mode=%s 期望 %s，实际 %s", mode, want, got)
		}
	}

	// 留空时各回落到 ai-model
	global.GVA_CONFIG.AutoCode.AiCodegenModel = ""
	global.GVA_CONFIG.AutoCode.AiVisionModel = ""
	for _, mode := range []string{"ai", "eye", "analysisChat"} {
		if got := modelForMode(mode); got != "sensenova-6.8-flash-lite" {
			t.Fatalf("mode=%s 留空时应继承 ai-model，实际 %s", mode, got)
		}
	}

	// eye 不能跟着 ai-codegen-model 走：纯文本模型识不了图
	global.GVA_CONFIG.AutoCode.AiCodegenModel = "deepseek-v4-flash"
	if got := modelForMode("eye"); got == "deepseek-v4-flash" {
		t.Fatalf("eye 不应使用 ai-codegen-model，实际 %s", got)
	}
}

func TestCheckEyeDescription(t *testing.T) {
	model := "deepseek-v4-flash"
	bad := []string{
		"",
		"   ",
		"抱歉，我无法从图片中识别字段。请直接提供业务表的中文描述。",
		"图片为空白或无效图片，未能辨识到任何业务表结构信息。",
		"Sorry, I cannot identify the table structure from this image.",
	}
	for _, s := range bad {
		if err := checkEyeDescription(s, model); err == nil {
			t.Errorf("应判为无效识图结果，实际通过：%q", s)
		}
	}
	good := "这是一张设备巡检记录表，用于记录每次巡检结果。需要以下字段：设备编号（varchar）、巡检时间（datetime）、巡检人（varchar）、巡检结果（varchar）、备注（text）。"
	if err := checkEyeDescription(good, model); err != nil {
		t.Errorf("正常描述被误判：%v", err)
	}
}

func TestJsonMapStringNil(t *testing.T) {
	m := common.JSONMap{"ok": "v"}
	if got := jsonMapString(m, "missing"); got != "" {
		t.Fatalf("缺键应返回空串，实际 %q", got)
	}
	m["nilv"] = nil
	if got := jsonMapString(m, "nilv"); got != "" {
		t.Fatalf("显式 nil 应返回空串，实际 %q", got)
	}
	if got := jsonMapString(m, "ok"); got != "v" {
		t.Fatalf("正常值应返回 v，实际 %q", got)
	}
}

func TestStripJSONFence(t *testing.T) {
	cases := map[string]string{
		"{\"a\":1}":                      "{\"a\":1}",
		"```json\n{\"a\":1}\n```":        "{\"a\":1}",
		"```\n{\"a\":1}\n```":            "{\"a\":1}",
		"  \n```json {\"a\":1}```  \n\n": "```json {\"a\":1}```",
	}
	for in, want := range cases {
		if got := stripJSONFence(in); got != want {
			t.Fatalf("输入 %q 期望 %q，实际 %q", in, want, got)
		}
	}
}

func TestNormalizeCodegenFieldType(t *testing.T) {
	cases := map[string]string{
		"string":        "string",
		"richtext":      "richtext",
		"time.Time":     "time.Time",
		"time.time":     "time.Time",
		"text":          "richtext", // 本次实测踩到的误写
		"longtext":      "richtext",
		"varchar(255)":  "string",
		"datetime":      "time.Time",
		"decimal(10,2)": "float64",
		"tinyint(1)":    "bool",
		"tinyint":       "int",
		"bigint":        "int",
		"boolean":       "bool",
		"jsonb":         "json",
		"":              "string",
		"whatever":      "string",
	}
	for in, want := range cases {
		if got := normalizeCodegenFieldType(in); got != want {
			t.Errorf("输入 %q 期望 %q，实际 %q", in, want, got)
		}
	}
}

func TestNormalizeCodegenFieldTypesJSON(t *testing.T) {
	in := `{"structName":"Book","fields":[{"fieldName":"A","fieldType":"text"},{"fieldName":"B","fieldType":"string"},{"fieldName":"C","fieldType":"datetime"}]}`
	out := normalizeCodegenFieldTypes(in)
	var m struct {
		StructName string `json:"structName"`
		Fields     []struct {
			FieldName string `json:"fieldName"`
			FieldType string `json:"fieldType"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("归一化后不是合法 JSON: %v\n%s", err, out)
	}
	if m.StructName != "Book" {
		t.Errorf("顶层字段丢了: %s", out)
	}
	want := []string{"richtext", "string", "time.Time"}
	for i, f := range m.Fields {
		if f.FieldType != want[i] {
			t.Errorf("第 %d 个字段期望 %s，实际 %s", i+1, want[i], f.FieldType)
		}
	}
	if got := normalizeCodegenFieldTypes("not json"); got != "not json" {
		t.Errorf("非 JSON 应原样返回，实际 %q", got)
	}
}

// TestNormalizeCodegenNames 守住「package 和结构体简称不可同名」。
// 模板渲染 var {{abbreviation}} {{package}}.{{structName}}，同名会遮蔽包名，编译不过。
func TestNormalizeCodegenNames(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		wantAbbr   string
		wantUnchan bool
	}{
		{
			name:     "package 与 abbreviation 同名",
			in:       `{"structName":"Book","package":"book","abbreviation":"book","fields":[]}`,
			wantAbbr: "bookObj",
		},
		{
			name:     "同名但与 structName 小驼峰不冲突",
			in:       `{"structName":"BookRecord","package":"record","abbreviation":"record"}`,
			wantAbbr: "bookRecord",
		},
		{
			name:     "abbreviation 为空",
			in:       `{"structName":"Book","package":"record","abbreviation":""}`,
			wantAbbr: "book",
		},
		{
			name:     "abbreviation 与 structName 同名",
			in:       `{"structName":"Book","package":"record","abbreviation":"Book"}`,
			wantAbbr: "book",
		},
		{
			name:       "本来就不冲突",
			in:         `{"structName":"Book","package":"book","abbreviation":"bookInfo"}`,
			wantAbbr:   "bookInfo",
			wantUnchan: true,
		},
	}

	for _, c := range cases {
		out := normalizeCodegenNames(c.in)
		var m struct {
			StructName   string `json:"structName"`
			Package      string `json:"package"`
			Abbreviation string `json:"abbreviation"`
			Fields       []any  `json:"fields"`
		}
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("%s: 归一化后不是合法 JSON: %v\n%s", c.name, err, out)
		}
		if m.Abbreviation != c.wantAbbr {
			t.Errorf("%s: 期望 abbreviation=%s，实际 %s（完整：%s）", c.name, c.wantAbbr, m.Abbreviation, out)
		}
		if m.Abbreviation == m.Package {
			t.Errorf("%s: 输出仍然同名（package=%s），前端会拒绝提交", c.name, m.Package)
		}
		if c.wantUnchan && out != c.in {
			t.Errorf("%s: 无需修改时不应改动内容\n原：%s\n后：%s", c.name, c.in, out)
		}
	}

	if got := normalizeCodegenNames("not json"); got != "not json" {
		t.Errorf("非 JSON 应原样返回，实际 %q", got)
	}
}

// TestLiveCodegenEye 用真实表结构图验证 mode=eye：
// 需要 GVA_TEST_AI_KEY + GVA_TEST_EYE_IMG（PNG 路径），两者缺一即跳过。
// [注意] 这个用例专门守住「eye 用了纯文本模型」这个坑：那种情况下模型会正常返回
// 一段「我无法从图片中识别字段」，看起来像成功，实际整条链路已经废了。
func TestLiveCodegenEye(t *testing.T) {
	setupLiveCfg(t)
	key := global.GVA_CONFIG.AutoCode.AiKey
	imgPath := os.Getenv("GVA_TEST_EYE_IMG")
	if key == "" || imgPath == "" {
		t.Skip("未设置 GVA_TEST_AI_KEY / GVA_TEST_EYE_IMG")
	}

	raw, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatalf("读图片失败: %v", err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)

	svc := AutoCodeService{}
	got, err := svc.LLMAuto(context.Background(), common.JSONMap{
		"mode":       "eye",
		"_file_path": dataURL,
	})
	if err != nil {
		t.Fatalf("识图失败（model=%s）: %v", modelForMode("eye"), err)
	}
	m, ok := got.(common.JSONMap)
	if !ok {
		t.Fatalf("返回类型异常: %T", got)
	}
	text, _ := m["text"].(string)
	if strings.TrimSpace(text) == "" {
		t.Fatal("识图结果为空")
	}
	t.Logf("识图模型=%s 输出：\n%s", modelForMode("eye"), text)

	// 图片里的类型字面量必须体现出来，否则说明模型根本没看图
	// （模型会把列名翻译成中文，所以拿 varchar(64) / datetime 这类原样保留的当锚点）
	for _, want := range []string{"varchar(64)", "varchar(128)", "datetime", "varchar(32)"} {
		if !strings.Contains(text, want) {
			t.Errorf("识图结果里没出现 %q，模型可能没真正读到图片内容", want)
		}
	}
}

// TestLiveCodegenAI 真实调用上游，验证 mode=ai 能拿到可直接 JSON.parse 的结果
func TestLiveCodegenAI(t *testing.T) {
	setupLiveCfg(t)
	if global.GVA_CONFIG.AutoCode.AiKey == "" {
		t.Skip("未设置 GVA_TEST_AI_KEY")
	}

	svc := AutoCodeService{}
	llm := common.JSONMap{
		"mode":   "ai",
		"prompt": "我要做一张图书表，字段有：书名、作者、价格、出版日期、是否上架、库存、简介。",
	}

	raw, err := svc.LLMAuto(context.Background(), llm)
	if err != nil {
		t.Fatalf("调用失败: %v", err)
	}
	m, ok := raw.(common.JSONMap)
	if !ok {
		t.Fatalf("返回类型异常: %T", raw)
	}
	text, _ := m["text"].(string)
	if text == "" {
		t.Fatalf("text 为空，原始返回: %#v", raw)
	}
	t.Logf("模型输出长度 %d", len(text))

	if !json.Valid([]byte(text)) {
		t.Fatalf("不是合法 JSON:\n%s", text)
	}
	if strings.Contains(text, "```") {
		t.Fatalf("残留 markdown 围栏:\n%s", text)
	}

	var out struct {
		StructName   string `json:"structName"`
		TableName    string `json:"tableName"`
		PackageName  string `json:"packageName"`
		Package      string `json:"package"`
		Abbreviation string `json:"abbreviation"`
		Description  string `json:"description"`
		GvaModel     bool   `json:"gvaModel"`
		Fields       []struct {
			FieldName  string `json:"fieldName"`
			FieldType  string `json:"fieldType"`
			ColumnName string `json:"columnName"`
			FieldJson  string `json:"fieldJson"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("二次解析失败: %v", err)
	}

	// package 与 abbreviation 同名会让生成的代码编译失败（变量遮蔽同名包），
	// 前端提交时也会直接拦住，所以必须在服务端就保证两者不同。
	if out.Abbreviation == "" {
		t.Errorf("abbreviation 为空，提交后路由 group 会变成空字符串:\n%s", text)
	}
	if out.Package != "" && out.Package == out.Abbreviation {
		t.Errorf("package 与 abbreviation 同名（都是 %q），归一化没生效", out.Package)
	}
	t.Logf("package=%q abbreviation=%q structName=%q", out.Package, out.Abbreviation, out.StructName)

	if out.StructName != "Book" {
		t.Errorf("structName 期望 Book，实际 %q", out.StructName)
	}
	if out.TableName != "books" {
		t.Errorf("tableName 期望 books，实际 %q", out.TableName)
	}
	if out.PackageName != "book" {
		t.Errorf("packageName 期望 book，实际 %q", out.PackageName)
	}
	if len(out.Fields) < 7 {
		t.Errorf("字段数 %d < 7，用户提到的 7 个字段没全覆盖:\n%s", len(out.Fields), text)
	}

	allowed := map[string]bool{
		"string": true, "richtext": true, "int": true, "bool": true, "float64": true,
		"time.Time": true, "enum": true, "picture": true, "pictures": true,
		"video": true, "file": true, "json": true, "array": true,
	}
	reserved := map[string]bool{
		"id": true, "created_at": true, "updated_at": true, "deleted_at": true,
		"created_by": true, "updated_by": true, "deleted_by": true,
	}
	for _, f := range out.Fields {
		if !allowed[f.FieldType] {
			t.Errorf("字段 %s 的 fieldType=%q 不在允许集合内", f.FieldName, f.FieldType)
		}
		if reserved[f.ColumnName] {
			t.Errorf("字段 %s 生成了 GVA 自动带的主键/审计列 %q", f.FieldName, f.ColumnName)
		}
		if f.FieldJson == "" || f.ColumnName == "" {
			t.Errorf("字段 %s 缺少 fieldJson/columnName", f.FieldName)
		}
	}

	t.Logf("最终返回：\n%s", text)
}
