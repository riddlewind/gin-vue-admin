// ============================================================================
// 【文件导读】API 层的构造函数 + 依赖注入点。
//
// 这里只做一件事：把 Service 实例"喂"给 Api 结构体的字段。
//
// 完整链条（背下来，缺一环就是空指针）：
//
//	service.ServiceGroupApp.InspectionRecordService
//	  ├─ ServiceGroupApp           service/enter.go 的全局实例
//	  └─ InspectionRecordService   service/enter.go 里的字段，
//	                               由 NewInspectionRecordService() 构造出来
//
// [Go] 为什么用构造函数而不是包级变量初始化？
// 因为构造时机是**调用方决定**的：上层 api/v1/enter.go 的 NewApiGroup 里调用它，
// 依赖关系一眼可见。包级变量初始化的顺序由编译器按 import 图决定，
// 读代码时很难一眼看出"谁在什么时候注入了什么"。
// ============================================================================
package inspectionrecord

import "github.com/flipped-aurora/gin-vue-admin/server/service"

// NewInspectionRecordApi 构造一个已完成依赖注入的 API 实例
//
// [Go] 返回**值**而不是指针：InspectionRecordApi 内部只有 Service 一个字段，
// 值拷贝很便宜；而且上层 ApiGroup 里存的就是值类型，返回指针反而要多解一层。
func NewInspectionRecordApi() InspectionRecordApi {
	return InspectionRecordApi{
		Service: service.ServiceGroupApp.InspectionRecordService,
	}
}
