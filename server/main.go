// ============================================================================
// 【文件导读】整个程序的唯一入口。
//
// Go 程序的启动顺序是固定的，理解这个顺序你就理解了所有 Go 项目：
//  1. 先初始化所有被 import 的包（被依赖的先初始化）：每个包内的包级变量按
//     依赖顺序赋值，然后执行该包所有的 init() 函数；
//  2. 全部包初始化完毕，才轮到 main() 执行。
//
// 也就是说 import 链上的 init() 一定比 main() 先跑——这正是本项目
// "表初始化自动注册"机制能工作的前提（见 server/source/system/user.go）。
//
// 【本文件涉及的 Go 语法】
//   - package main / func main()：可执行程序的标志
//   - 空白导入（下划线）：只为了触发对方包的 init()，不使用其导出标识符
//   - 包级变量：用 global 包做"服务注册表"，代替依赖注入
//
// ============================================================================
package main

import (
	"github.com/flipped-aurora/gin-vue-admin/server/core"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/initialize"
	// [Go] 空白导入（下划线 _）：这个包没有导出我们直接使用的东西，
	// 导入它唯一的目的就是让它的 init() 执行。
	// automaxprocs 会读容器 CPU 配额自动设置 GOMAXPROCS，
	// 避免在 K8s 里 Go 误以为有整台物理机的核数而开出过多线程。
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// [Go] 上面是 go:generate 指令，执行 `go generate` 时会把每行命令跑一遍。
// 注意 `//go:generate` 中间不能有空格，否则它就退化成普通注释了。
// README 让你先跑 `go generate`，就是为了配好代理并拉取依赖。

// 这部分 @Tag 设置用于排序, 需要排序的接口请按照下面的格式添加
// swag init 对 @Tag 只会从入口文件解析, 默认 main.go
// 也可通过 --generalInfo flag 指定其他文件
//
// [注意] 下面这一整块是给 swag 工具解析的注解，不是给人看的普通注释。
// 位置必须紧贴 func main()，且 `// @` 前缀是语法的一部分。
// 随手改格式或挪位置，会导致 `swag init` 生成的文档缺内容。
// @Tag.Name        Base
// @Tag.Name        SysUser
// @Tag.Description 用户

// @title                       Gin-Vue-Admin Swagger API接口文档
// @version                     v2.9.2
// @description                 使用gin+vue进行极速开发的全栈开发基础平台
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        x-token
// @BasePath                    /
func main() {
	// 初始化系统
	initializeSystem()
	// 运行服务器（内部会阻塞，直到收到退出信号）
	core.RunServer()
}

// initializeSystem 初始化系统所有组件
// 提取为单独函数以便于系统重载时调用
//
// [注意] 下面 8 行是严格的依赖链，顺序不能乱：
//   - 先有配置（Viper），因为日志级别、数据库地址都要从配置里读；
//   - 再有日志（Zap），因为后面任何一步失败都要靠它记录原因；
//   - 然后才连数据库，因为连库时要靠日志器记录慢查询和错误。
//
// 随手调整顺序 = 启动即 panic。
func initializeSystem() {
	global.GVA_VP = core.Viper() // 初始化Viper
	initialize.OtherInit()
	global.GVA_LOG = core.Zap() // 初始化zap日志库

	// [Go] ReplaceGlobals 把我们的日志器替换成 zap 的全局实例。
	// 之后任何地方调用 zap.L() 拿到的都是这个实例，
	// 就不必把 *zap.Logger 一层层当参数传下去了——第三方库提供的一种"全局单例"用法。
	zap.ReplaceGlobals(global.GVA_LOG)

	// [注意] 首次启动时 config.yaml 的库名是空的，
	// GormMysql() 会直接 return nil，所以此刻 global.GVA_DB == nil 属于正常现象。
	// 服务照样能起，等前端进 /init 页面把数据库配好再连。
	global.GVA_DB = initialize.Gorm() // gorm连接数据库

	initialize.Timer()
	initialize.DBList()
	initialize.SetupHandlers() // 注册全局函数

	// [Go] 指针判空再使用，是 Go 里最常见的防御式写法。
	// GVA_DB 是 *gorm.DB 指针，为 nil 时调用它的方法会直接 panic。
	if global.GVA_DB != nil {
		initialize.RegisterTables() // 初始化表
	}
}
