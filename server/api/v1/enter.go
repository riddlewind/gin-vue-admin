// ============================================================================
// 【文件导读】全部 API 的聚合入口。
//
// 这一层是 router 层唯一的入口：路由注册时只认 v1.ApiGroupApp，
// 不需要知道下面有多少个模块。
//
// [Go] 两种初始化写法对比：
//
//	var ApiGroupApp = new(ApiGroup)          // 零值：所有字段都是各自类型的零值
//	var ApiGroupApp = NewApiGroup()          // 构造函数：可以显式注入依赖
//
// 这里换成后者，因为巡检记录模块的 Api 需要拿到 Service 实例。
// 返回类型仍然是 *ApiGroup，所以 router 那边一个字都不用改。
// ============================================================================
package v1

import (
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/demo"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/example"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/inspectionrecord"
	"github.com/flipped-aurora/gin-vue-admin/server/api/v1/system"
)

// ApiGroupApp 全局唯一的 API 聚合实例
//
// [注意] 它等于 NewApiGroup() 的返回值，不要再手动 new(ApiGroup) 一份——
// 那样会拿到一个没有注入依赖的副本，接口能跑但 Service 里是空的，
// 排查起来非常费劲（现象是"有时能跑有时不能"，取决于用的是哪一个实例）。
var ApiGroupApp = NewApiGroup()

type ApiGroup struct {
	SystemApiGroup  system.ApiGroup
	ExampleApiGroup example.ApiGroup
	// [注册点 2/5] 暴露业务模块的 ApiGroup
	DemoApiGroup demo.ApiGroup

	// [注册点 2/5] 巡检记录模块
	//
	// [Go] 注意这个字段和上面几行**写法不同**：
	//   DemoApiGroup       demo.ApiGroup              -> 嵌套一整个子聚合
	//   InspectionRecordApi inspectionrecord.Xxx      -> 直接持有 API 结构体
	//
	// 两种都能用。直接持有更扁平，router 里写 v1.ApiGroupApp.InspectionRecordApi.CreateXxx；
	// 嵌套则要多一层 v1.ApiGroupApp.DemoApiGroup.BookApi.CreateXxx。
	// 字段名和类型名相同是允许的（Go 不禁止，只是读起来容易看花眼）。
	InspectionRecordApi inspectionrecord.InspectionRecordApi
}

// NewApiGroup 构造并装配整个 API 聚合
//
// [Go] 返回 *ApiGroup 而不是 ApiGroup，是为了和原来 new(ApiGroup) 的类型保持一致
// （ApiGroupApp 一直是 *ApiGroup，router 层直接解引用，调用方一个字都不用改）。
//
// [注意] 这里只显式初始化了 InspectionRecordApi。System / Example / Demo 三个模块的
// Api 结构体都是空结构体、或者只依赖各自 enter.go 里的包级变量，零值即"装配完成"，
// 不需要在这里赋值。以后哪个模块也开始用字段注入，就往这个 return 里加一行。
func NewApiGroup() *ApiGroup {
	return &ApiGroup{
		InspectionRecordApi: inspectionrecord.NewInspectionRecordApi(),
	}
}
