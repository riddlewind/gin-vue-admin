// ============================================================================
// 【文件导读】路由层 —— 把 URL 和处理函数绑起来。
//
// 这一层只做三件事：分组、挂中间件、绑定 URL 到 API 方法。不写任何业务逻辑。
//
// 路由最终是怎么生效的？记住这条链：
//
//	router/<模块>/xxx.go        定义 InitXxxRouter
//	  -> router/<模块>/enter.go  把 XxxRouter 塞进 RouterGroup
//	    -> router/enter.go       把 RouterGroup 塞进 RouterGroupApp
//	      -> initialize/router_biz.go 调用 InitXxxRouter(privateGroup)
//
// 【本文件涉及的 Go 语法】
//   - 空结构体做"方法集合容器"（type XxxRouter struct{}）
//   - 指针接收者
//   - 大括号 {} 只是代码块的视觉分组，不产生作用域
//
// ============================================================================
package example

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

// CustomerRouter 路由注册器
//
// [Go] 为什么是空结构体 struct{}？
// 因为它不需要存任何状态，只需要有个"类型"来承载方法集合。
// 零内存占用，而且方法接收者用指针 (e *CustomerRouter) 也不会真的分配内存。
// 这是 Go 里常见的"命名空间"手法——把一组相关函数挂在一个类型上。
type CustomerRouter struct{}

// InitCustomerRouter 注册客户模块的所有路由
//
// [Go] 参数 Router *gin.RouterGroup 是从上层传进来的路由组
// （实际是 initialize/router.go 里的 PrivateGroup，已经挂上了 JWT + Casbin 中间件）。
// 用指针是因为 RouterGroup 内部持有 engine 和中间件列表，必须操作同一份。
func (e *CustomerRouter) InitCustomerRouter(Router *gin.RouterGroup) {
	// [注意] 这里刻意分成两个组：
	//   customerRouter            -> 挂了 OperationRecord 中间件，用于"写操作"留痕
	//   customerRouterWithoutRecord -> 不挂，用于"读操作"，避免刷爆操作日志表
	// 增删改写操作日志，查询不写——这是很实用的设计取舍，抄的时候别把两个组合并了。
	//
	// [Go] Gin 的 Router.Group() 返回一个新组，.Use() 继续返回该组，
	// 所以可以链式调用。链式调用的前提是每个方法都返回自身类型。
	customerRouter := Router.Group("customer").Use(middleware.OperationRecord())
	customerRouterWithoutRecord := Router.Group("customer")

	// [Go] 下面的大括号 {} 纯粹是为了让代码视觉上分组，Go 不会因此产生新的作用域，
	// 也不需要加分号。这是一种被广泛使用的风格约定。
	{
		// [注意] 第二个参数直接传方法值 exaCustomerApi.CreateExaCustomer（不用加括号！）。
		// 加括号就变成"立刻调用"，类型不匹配会编译报错。
		//
		// exaCustomerApi 这个变量定义在 router/example/enter.go 里，
		// 从 api.ApiGroupApp 取出来的——路由层不自己 new API 对象。
		customerRouter.POST("customer", exaCustomerApi.CreateExaCustomer)   // 创建客户
		customerRouter.PUT("customer", exaCustomerApi.UpdateExaCustomer)    // 更新客户
		customerRouter.DELETE("customer", exaCustomerApi.DeleteExaCustomer) // 删除客户
	}
	{
		// [注意] PUT/POST/DELETE 用的是同一个路径 "customer"，
		// 靠 HTTP Method 区分。所以 RESTful 风格下"路径相同、方法不同"是正常的，
		// 在「API 管理」里登记权限时也必须连 Method 一起登记。
		customerRouterWithoutRecord.GET("customer", exaCustomerApi.GetExaCustomer)         // 获取单一客户信息
		customerRouterWithoutRecord.GET("customerList", exaCustomerApi.GetExaCustomerList) // 获取客户列表
	}
}
