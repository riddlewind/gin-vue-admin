// ============================================================================
// 【文件导读】API 层的分组聚合。
//
// 除了聚合同包的 Api 类型，还要把用到的 Service 实例"取"成包级变量，
// 这样 book.go 里可以直接写 bookService.XXX，不用每次都从全局链上取。
//
// 完整链条（背下来）：
//
//	service.ServiceGroupApp
//	  .DemoServiceGroup          <- service/enter.go 里的字段
//	    .BookService             <- service/demo/enter.go 里嵌入的类型名
//
// ============================================================================
package demo

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	BookApi
}

var (
	// [Go] 这里取到的是 BookService 的**值拷贝**（因为 ServiceGroup 里是嵌入的，
	// 不是指针）。BookService 是空结构体、无状态，拷贝没有代价。
	// [注意] 如果 Service 里有状态（连接池、缓存），就必须改成取指针，
	// 否则每个 API 拿到各自的副本，缓存会失效——这种 bug 极其难查。
	bookService = service.ServiceGroupApp.DemoServiceGroup.BookService
)
