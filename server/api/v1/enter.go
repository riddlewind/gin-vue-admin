package v1

import (
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/demo"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/example"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/system"
)

var ApiGroupApp = new(ApiGroup)

type ApiGroup struct {
	SystemApiGroup  system.ApiGroup
	ExampleApiGroup example.ApiGroup
	// [注册点 2/5] 暴露业务模块的 ApiGroup
	DemoApiGroup demo.ApiGroup
}
