// ============================================================================
// 【文件导读】Service 层的分组聚合。
//
// 作用：把本包内所有 Service 类型聚成一个 ServiceGroup，
// 供上层 service/enter.go 引用，最终挂到全局的 ServiceGroupApp 上。
//
// [Go] 注意这里用的是"嵌入类型"而不是命名字段：
//
//	嵌入 -> ServiceGroup 自动拥有 BookService 的方法，且字段名就是 BookService
//	命名 -> 要用 sg.BookService 访问，效果类似但语义不同
//
// GVA 全项目都用嵌入，保持一致就行。
// ============================================================================
package demo

type ServiceGroup struct {
	// 以后加新模块（比如 OrderService），就在这里加一行。
	// [注意] 加完之后还要去 service/enter.go 里确认 DemoServiceGroup 已经被暴露。
	BookService
}
