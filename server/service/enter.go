package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service/demo"
	"github.com/flipped-aurora/gin-vue-admin/server/service/example"
	"github.com/flipped-aurora/gin-vue-admin/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	SystemServiceGroup  system.ServiceGroup
	ExampleServiceGroup example.ServiceGroup
	// [注册点 1/5] 暴露业务模块的 ServiceGroup
	DemoServiceGroup demo.ServiceGroup
}
