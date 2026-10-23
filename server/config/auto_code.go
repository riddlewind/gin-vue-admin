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
}

func (a *Autocode) WebRoot() string {
	webs := strings.Split(a.Web, "/")
	if len(webs) == 0 {
		webs = strings.Split(a.Web, "\\")
	}
	return filepath.Join(webs...)
}
