// ============================================================================
// 【文件导读】API 层的"分组装配"文件 —— 全项目最容易被忽略、但删了就跑不起来的文件。
//
// 它解决两个问题：
//  1. 把本模块所有 Api 类型聚合成一个 ApiGroup，供上层（api/v1/enter.go）引用；
//  2. 把本模块需要的 Service 实例在这里取出来，变成包级变量供各 Api 复用。
//
// 完整的三级注册链（一定要串起来记）：
//
//	service/example/enter.go     ServiceGroup{CustomerService}
//	  -> api/v1/example/enter.go  ApiGroup{CustomerApi} + 包级 service 变量
//	    -> api/v1/enter.go        ApiGroupApp{ExampleApiGroup}
//	      -> router/example/enter.go  exaCustomerApi = api.ApiGroupApp...
//
// 【本文件涉及的 Go 语法】
//   - 结构体嵌入（这里嵌入的是类型，用于聚合方法集合）
//   - 包级变量组 var (...)
//   - 点号链式取字段：service.ServiceGroupApp.ExampleServiceGroup.CustomerService
//
// ============================================================================
package example

import "github.com/flipped-aurora/gin-vue-admin/server/service"

// ApiGroup 本模块的 API 聚合体
//
// [Go] 注意这里是"嵌入类型"而不是"命名字段"：
//
//	type ApiGroup struct {
//	    CustomerApi            // 嵌入：ApiGroup 自动拥有 CustomerApi 的所有方法
//	    AttachmentCategoryApi  // 嵌入
//	}
//
// 因为 ApiGroup 只是被当作"容器"来索引（ApiGroupApp.ExampleApiGroup.CustomerApi），
// 嵌入之后就能用 .CustomerApi 这个"提升的字段名"访问到它，
// 所以能写成 api.ApiGroupApp.ExampleApiGroup.CustomerApi。
//
// [注意] 新增模块时，必须在这里加一行你的 XxxApi，
// 否则 router 层根本拿不到你的 API 对象，编译报错 undefined。
type ApiGroup struct {
	CustomerApi

	AttachmentCategoryApi
	FileUploadAndDownloadApi
}

// [Go] var (...) 是包级变量组。这些变量在包初始化时就被赋值，
// 之后所有请求共用同一份 Service 实例。
//
// [注意] 等号右边是一条"字段链"：
//
//	service.ServiceGroupApp                 全局唯一的 ServiceGroup 指针
//	  .ExampleServiceGroup                  example 这一组
//	    .CustomerService                    CustomerService 的嵌入字段（值类型）
//
// 因为 CustomerService 里面没有状态（空结构体），
// 复制一份也不影响功能，所以这里拿到的是"值拷贝"而不是指针。
// 如果 Service 内部要存状态（比如连接池、缓存），就必须改成取指针，
// 否则每个 API 拿到的是各自的副本——这是很容易踩的隐形 bug。
var (
	customerService = service.ServiceGroupApp.ExampleServiceGroup.CustomerService

	attachmentCategoryService    = service.ServiceGroupApp.ExampleServiceGroup.AttachmentCategoryService
	fileUploadAndDownloadService = service.ServiceGroupApp.ExampleServiceGroup.FileUploadAndDownloadService
)
