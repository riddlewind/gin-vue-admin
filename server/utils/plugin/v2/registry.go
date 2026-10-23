package plugin

import "sync"

var (
	registryMu sync.RWMutex
	registry   []Plugin
)

// Register 登记插件，供后续自动初始化使用。
func Register(p Plugin) {
	if p == nil {
		return
	}
	registryMu.Lock()
	registry = append(registry, p)
	registryMu.Unlock()
}

// Registered 返回当前已注册插件的快照。
func Registered() []Plugin {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Plugin, len(registry))
	copy(out, registry)
	return out
}
