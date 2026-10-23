// ============================================================================
// 【文件导读】Service 层的装配入口。
//
// 这里只做一件事：提供构造函数，让上层能拿到一个"可以直接用"的
// InspectionRecordService 实例。
//
// 完整链条（背下来，缺一环就是空指针）：
//
//	service/enter.go
//	  NewServiceGroup()                                     ← 构造 ServiceGroup
//	    InspectionRecordService                             ← 字段名
//	      = inspectionrecord.NewInspectionRecordService()    ← 本文件
//	        service.ServiceGroupApp                          ← 上层 api 通过它取
//
// [Go] 为什么不在这里写 var XxxApp = new(InspectionRecordService)？
// 那是"包级全局单例"，谁 import 这个包都能直接改它，依赖关系在代码里完全看不出来。
// 构造函数把"谁创建、何时创建"交给调用方，依赖链一眼可查。
// ============================================================================
package inspectionrecord

// NewInspectionRecordService 构造一个巡检记录 Service 实例
//
// [Go] 返回**值**而不是指针：
//   - 结构体只有一个 *gorm.DB 字段，值拷贝成本几乎为零；
//   - 上层 ServiceGroup 里存的就是值类型，返回指针反而要多解一层引用；
//   - 方法集挂在 *InspectionRecordService 上，对值取地址照样能调用，不受影响。
//
// [注意] 这里**不**传 DB。Service 内部每个方法都通过 db() 取连接，
// 注入为空时自动回落到 global.GVA_DB；需要事务时再手动给 s.DB 赋值即可。
func NewInspectionRecordService() InspectionRecordService {
	return InspectionRecordService{}
}
