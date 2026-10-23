package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router/demo"
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = new(RouterGroup)

type RouterGroup struct {
	System  system.RouterGroup
	Example example.RouterGroup
	// [注册点 3/5] 暴露业务模块的 RouterGroup
	Demo demo.RouterGroup
}
