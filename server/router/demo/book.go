// ============================================================================
// 【文件导读】第四步：路由层 —— 把 URL 绑到处理函数上。
//
// 只做三件事：分组、挂中间件、绑定。不写业务。
//
// 【本文件涉及的 Go 语法】
//   - 空结构体承载方法集合
//   - 指针接收者
//   - 方法值（不带括号地传函数）
//
// ============================================================================
package demo

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

// BookRouter 图书路由
type BookRouter struct{}

// InitBookRouter 注册图书模块路由
//
// [Go] Router 是上层传进来的 PrivateGroup（已挂 JWT + Casbin）。
// 这里再 Group("demo") 得到 /demo 前缀，里面再绑 book / bookList。
// 最终路径：/demo/book、/demo/bookList
func (r *BookRouter) InitBookRouter(Router *gin.RouterGroup) {
	// [Note] 写操作单独一组，挂 OperationRecord 中间件记录操作日志；
	// 查询不挂，避免日志表被列表接口刷爆。
	demoRouter := Router.Group("demo").Use(middleware.OperationRecord())
	demoRouterWithoutRecord := Router.Group("demo")

	{
		// [Go] 第二个参数是"方法值"：bookApi.CreateBook 不带括号，
		// 表示"把这个函数本身交出去"，而不是"立刻调用它"。
		// 加了括号就变成传返回值，类型对不上，编译报错。
		demoRouter.POST("book", bookApi.CreateBook)   // 新增图书
		demoRouter.PUT("book", bookApi.UpdateBook)    // 更新图书
		demoRouter.DELETE("book", bookApi.DeleteBook) // 删除图书
	}
	{
		demoRouterWithoutRecord.GET("book", bookApi.GetBook)         // 按ID查单条
		demoRouterWithoutRecord.GET("bookList", bookApi.GetBookList) // 分页列表
	}
}
