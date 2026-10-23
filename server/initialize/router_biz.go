// ============================================================================
// 【文件导读】业务路由注册钩子 —— 二次开发的第二个必改文件。
//
// 框架的系统路由（登录、用户、菜单、权限…）都在 initialize/router.go 的
// Routers() 里一个个手写注册了。而**你自己模块的路由要注册在这里**。
//
// 为什么不直接加在 router.go 里？
// 因为 router.go 是框架代码，将来升级框架时会被覆盖；这个文件是留给你的"自留地"，
// 升级时可以整份保留下来。这是开源框架常见的"hook 文件"设计。
//
// 不改这里会发生什么？
//
//	你的 API 和 Router 都写好了，编译也过了，但请求 404——
//	因为 Route 对象创建了却从来没被挂到 Gin 的路由树上。
//
// 【本文件涉及的 Go 语法】
//   - 可变参数 ...*gin.RouterGroup
//   - 空标识符 _ 的另一种用途：消除"声明未使用"的编译错误
//
// ============================================================================
package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router"
	"github.com/gin-gonic/gin"
)

// 占位方法，保证文件可以正确加载，避免go空变量检测报错，请勿删除。
//
// [Go] 为什么要写这么个"什么也不干"的函数？
// 因为 Go 编译器有两个硬性规定：
//  1. 导入的包必须被使用，否则编译报错 "imported and not used";
//  2. 声明的局部变量必须被使用，否则报 "declared and not used"。
//
// 当你把 initBizRouter 里的路由注册代码删空（或还没开始写）时，
// 上面导入的 router 包就没人用了，编译直接失败。
// 这个 holder 函数配合下面的 `_ = xxx` 就是为了"用一下它们"，
// 让文件在没有业务路由时也能正常编译。
//
// [Go] `_ = routers` 是显式的"我接收了但故意不用"。
// 比删掉参数更能表达意图，也不会触发未使用检查。
func holder(routers ...*gin.RouterGroup) {
	_ = routers
	_ = router.RouterGroupApp
}

// initBizRouter 注册你自己的业务路由
//
// [Go] routers ...*gin.RouterGroup 是可变参数，调用方可以传任意个路由组。
// 实际调用点在 initialize/router.go:108 —— initBizRouter(PrivateGroup, PublicGroup)。
//
// [注意] 因为是可变参数，函数内部拿到的是切片，用下标取：
//
//	routers[0] -> PrivateGroup（已挂 JWT + Casbin，需要登录才能访问）
//	routers[1] -> PublicGroup （公开，无需登录，比如登录接口、初始化接口）
//
// 如果调用方少传了参数，这里会数组越界 panic。所以千万别自己改调用点。
func initBizRouter(routers ...*gin.RouterGroup) {
	privateGroup := routers[0]
	publicGroup := routers[1]

	// [注册点 5/5] 业务路由注册处 —— 真正把 URL 挂上路由树的地方。
	//
	// [Go] 从全局 RouterGroupApp 里取出本模块的路由组，
	// 再把 privateGroup（已挂 JWT + Casbin）传给它。
	demoRouter := router.RouterGroupApp.Demo
	demoRouter.InitBookRouter(privateGroup)

	// [注册点 5/5] 巡检记录模块
	//
	// [注意] 这里**不能**像上面 demo 那样先取到局部变量再调用：
	//
	//	bad := router.RouterGroupApp.InspectionRecordRouter   // 这是一次值拷贝！
	//	bad.SetInspectionRecordRouter(...)                    // 改的是副本，全局那个还是空的
	//
	// 因为 InspectionRecordRouter 里存了 PublicGroup / PrivateGroup 两个字段，
	// 值拷贝之后指针方法的接收者是副本的地址，写进去的字段值不会回到
	// RouterGroupApp 上。路由本身还是能注册成功（那是作用在 gin 引擎上的），
	// 但"以后再补注册路由"时从 RouterGroupApp 取出来的分组会是 nil —— 属于
	// 那种当场看不出来、几个月后才炸的 bug。
	//
	// 正确写法：直接链式调用字段。RouterGroupApp 是 *RouterGroup，
	// 字段可寻址，Go 会自动对字段取地址，改的就是全局那一份。
	//
	// [注意] 传参顺序是 (publicGroup, privateGroup)，
	// 而上面 routers[0] / routers[1] 的顺序是"先私有再公开"，方向相反，别看串。
	// 这个函数内部对 privateGroup 做了 nil 检查，传反了会在启动时立刻 panic。
	router.RouterGroupApp.InspectionRecordRouter.SetInspectionRecordRouter(publicGroup, privateGroup)

	// [注意] 只要 privateGroup 被用到了，holder 其实就可以删；
	// 但如果 publicGroup 没被用，就会报"声明未使用"。所以最省事的是保留 holder。
	holder(publicGroup, privateGroup)
}
