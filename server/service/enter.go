package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service/customer"
	"github.com/flipped-aurora/gin-vue-admin/server/service/demo"
	"github.com/flipped-aurora/gin-vue-admin/server/service/example"
	"github.com/flipped-aurora/gin-vue-admin/server/service/inspectionrecord"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
)

// ServiceGroupApp 全局唯一的 Service 聚合实例
//
// [注意] 全项目只能有这一份。别处再 new(ServiceGroup) 出来的副本，
// 里面每个 Service 的 DB 字段都是零值，注入就形同虚设（接口能跑但换了数据源不生效）。
var ServiceGroupApp = NewServiceGroup()

type ServiceGroup struct {
	SystemServiceGroup  system.ServiceGroup
	ExampleServiceGroup example.ServiceGroup
	// [注册点 1/5] 暴露业务模块的 ServiceGroup
	DemoServiceGroup demo.ServiceGroup
	// [注册点 1/5] 巡检记录模块
	InspectionRecordService inspectionrecord.InspectionRecordService
	CustomerServiceGroup    customer.

		// NewServiceGroup 构造 Service 聚合，按需初始化各业务模块的 Service
		//
		// [Go] 返回 *ServiceGroup 而不是值：上层 ServiceGroupApp 是包级变量，
		// 指针能保证全局拿到的是同一个实例（值拷贝会让每个调用方各持一份副本）。
		ServiceGroup
}

func NewServiceGroup() *ServiceGroup {
	return &ServiceGroup{
		InspectionRecordService: inspectionrecord.NewInspectionRecordService(),
	}
}
