package config

import (
	"path/filepath"
	"strings"
)

type Autocode struct {
	Web    string `mapstructure:"web" json:"web" yaml:"web"`
	Root   string `mapstructure:"root" json:"root" yaml:"root"`
	Server string `mapstructure:"server" json:"server" yaml:"server"`
	Module string `mapstructure:"module" json:"module" yaml:"module"`
	AiPath string `mapstructure:"ai-path" json:"ai-path" yaml:"ai-path"`

	// [注意] 以下四个字段用于把 AI 上游从「官方托管 AiPath」切换为「OpenAI 兼容中转站」。
	// ai-provider 留空或填 gva 时，行为与官方版本完全一致（走 ai-path）。
	AiProvider string `mapstructure:"ai-provider" json:"ai-provider" yaml:"ai-provider"`
	AiBaseURL  string `mapstructure:"ai-base-url" json:"ai-base-url" yaml:"ai-base-url"`
	AiKey      string `mapstructure:"ai-key" json:"ai-key" yaml:"ai-key"`
	AiModel    string `mapstructure:"ai-model" json:"ai-model" yaml:"ai-model"`

	// AiCodegenModel 是「代码生成器」写代码那一步（mode=ai）专用的模型。
	// 留空则继承 ai-model。
	// 用处：AI 工作流那套要长文本推理、代码生成这套要稳定吐 JSON，两者最优模型往往不同；
	// 同一个中转站下就能一个页面用 A 模型、另一个页面用 B 模型。
	AiCodegenModel string `mapstructure:"ai-codegen-model" json:"ai-codegen-model" yaml:"ai-codegen-model"`

	// AiVisionModel 是「图片识别」页面（mode=eye）专用的模型，必须支持 image 输入。
	// 留空则继承 ai-model。
	// [注意] 不能跟 ai-codegen-model 混用：代码生成最稳的模型往往是纯文本模型
	//（例如 deepseek-v4-flash 的 input_modalities 只有 text），拿它去识图，
	// 网关会把图片丢掉，模型只会回一句「我无法从图片中识别字段」。
	AiVisionModel string `mapstructure:"ai-vision-model" json:"ai-vision-model" yaml:"ai-vision-model"`
}

func (a *Autocode) WebRoot() string {
	webs := strings.Split(a.Web, "/")
	if len(webs) == 0 {
		webs = strings.Split(a.Web, "\\")
	}
	return filepath.Join(webs...)
}
