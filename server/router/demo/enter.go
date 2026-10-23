// ============================================================================
// 【文件导读】路由层的分组聚合。
//
// 从 api.ApiGroupApp 里取出本模块的 Api 实例，变成包级变量给 book.go 用。
// ============================================================================
package demo

import (
	api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
)

type RouterGroup struct {
	BookRouter
}

var (
	// [Go] 注意别名 api：导入路径最后一段是 v1，但包声明是 package v1，
	// GVA 在 router 层统一起别名 api，读起来更符合语义。
	bookApi = api.ApiGroupApp.DemoApiGroup.BookApi
)
