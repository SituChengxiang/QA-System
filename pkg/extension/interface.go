package extension

// PluginMetadata 插件元数据
type PluginMetadata struct {
	Name        string // 插件名称
	Version     string // 插件版本
	Author      string // 插件作者
	Description string // 插件描述
}

// Plugin 定义插件接口
type Plugin interface {
	GetMetadata() PluginMetadata         // GetMetadata 获取插件元数据
	Execute(params map[string]any) error // Execute 执行插件功能，接收参数
}

// PluginHealthChecker 定义插件健康检查接口
type PluginHealthChecker interface {
	IsHealthy() bool   // IsHealthy 检查插件是否健康可用
	GetStatus() string // GetStatus 获取插件状态描述
}
