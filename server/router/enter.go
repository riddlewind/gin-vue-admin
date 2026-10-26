package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router/demo"
	"github.com/flipped-aurora/gin-vue-admin/server/router/example"
	// [Go] 起别名 router_inspectionrecord：这个包名和下面 RouterGroup 里的字段名
	// InspectionRecordRouter 长得太像，用带前缀的别名读起来能立刻分清
	// "这是包" 还是 "这是字段"。
	"github.com/flipped-aurora/gin-vue-admin/server/router/customer"
	router_inspectionrecord "github.com/flipped-aurora/gin-vue-admin/server/router/inspectionrecord"
	"github.com/flipped-aurora/gin-vue-admin/server/router/system"
)

var RouterGroupApp = NewRouterGroup()

type RouterGroup struct {
	System  system.RouterGroup
	Example example.RouterGroup
	// [注册点 3/5] 暴露业务模块的 RouterGroup
	Demo demo.RouterGroup

	// [注册点 3/5] 巡检记录模块
	//
	// [注意] 字段名和类型名相同（InspectionRecordRouter），Go 允许这么写。
	// 调用点：router.RouterGroupApp.InspectionRecordRouter.SetInspectionRecordRouter(...)
	// 读的时候要靠位置区分哪半边是名字、哪半边是类型，别当成重复声明。
	InspectionRecordRouter router_inspectionrecord.InspectionRecordRouter
	Customer               customer.

		// NewRouterGroup 构造路由聚合，按需初始化各业务模块的路由对象
		//
		// [Go] 返回 *RouterGroup：RouterGroupApp 是包级变量，指针能保证全局只有一个实例。
		// 换成值返回的话，每次赋值都是一次完整拷贝，模块间就不共享同一份状态了。
		//
		// [注意] 这里只显式初始化了 InspectionRecordRouter。System / Example / Demo 的
		// 路由对象都是空结构体，零值即"可用"，不需要在这里赋值。
		// 以后哪个模块的路由对象也开始持有状态字段，就往这个 return 里加一行。
		RouterGroup
}

func NewRouterGroup() *RouterGroup {
	return &RouterGroup{
		InspectionRecordRouter: router_inspectionrecord.NewInspectionRecordRouter(),
	}
}
