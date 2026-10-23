// ============================================================================
// 【文件导读】第五步：路由层 —— 把 URL 绑到处理函数上。
//
// 只做三件事：分组、挂中间件、绑定。这一层不写任何业务逻辑，
// 甚至连参数校验都不该有（那是 API 层的事）。
//
// 【最终 URL 长这样】
//
//	POST   /inspectionRecord/createInspectionRecord
//	DELETE /inspectionRecord/deleteInspectionRecord
//	GET    /inspectionRecord/getInspectionRecord
//	GET    /inspectionRecord/getInspectionRecordList
//	PUT    /inspectionRecord/updateInspectionRecord
//
// 【本文件涉及的 Go 语法】
//   - 空结构体 + 指针接收者承载"一组方法"
//   - 方法值：api.CreateInspectionRecord 不带括号
//   - 局部变量与包名同名时的遮蔽问题（这里刻意避开）
//
// ============================================================================
package inspectionrecord

import (
	api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

// InspectionRecordRouter 巡检记录路由
//
// [Go] 为什么把两个 *gin.RouterGroup 存进结构体字段，而不是像 GVA 原版那样
// 只写成 InitXxxRouter(Router *gin.RouterGroup) 的参数？
// 因为 gin.RouterGroup 是"一边走一边复制"的对象：
//
//	Router.Group("a")       返回一个新的 *RouterGroup，原对象不受影响
//	Router.Group("a").Use() 也一样，只作用于返回的副本
//
// 存字段的好处是把"注册时用的分组"和"以后可能补注册的路由"都留住，
// 代价是它变成有状态对象 —— 所以只允许在启动时调一次
// SetInspectionRecordRouter，别在请求处理函数里调。
type InspectionRecordRouter struct {
	// PublicGroup 公开分组（不需要登录）。本模块目前没有公开接口，先存着备用。
	PublicGroup *gin.RouterGroup
	// PrivateGroup 私有分组，已经挂好 JWTAuth + CasbinHandler（在 initialize/router.go 里配的）
	PrivateGroup *gin.RouterGroup
}

// NewInspectionRecordRouter 构造一个空的巡检记录路由对象
//
// [注意] 这里**不**接收任何 *gin.RouterGroup 参数：构造函数只负责"造对象"，
// 分组是在启动时由 initialize/router_biz.go 通过 SetInspectionRecordRouter 注入的。
// 两件事分开的好处是：造对象不需要知道路由树的形状，测试里也能随便塞假分组。
//
// [Go] 返回**值**而不是指针，和 NewInspectionRecordApi / NewInspectionRecordService 保持一致：
// 上层 RouterGroup 里存的就是值类型字段，返回指针还得解一层引用。
// 值返回也不会妨碍调用指针方法 —— 字段本身是可寻址的，Go 会自动帮你取地址。
func NewInspectionRecordRouter() InspectionRecordRouter {
	return InspectionRecordRouter{}
}

// SetInspectionRecordRouter 注册巡检记录模块的全部路由
//
// [注意] 参数顺序是 (publicGroup, privateGroup)，和 initialize/router.go 里的调用点
// `initBizRouter(PrivateGroup, PublicGroup)` **正好相反**。
// 那里是 []*gin.RouterGroup 下标取值（[0]=私有，[1]=公开），很容易记混。
// 所以本函数用命名清晰的参数接，调用方按名字传，别按位置猜。
//
// [Go] 方法接收者用指针 *InspectionRecordRouter：要往结构体字段里写值，
// 值接收者改的是副本，外面的 PublicGroup / PrivateGroup 永远是 nil。
func (s *InspectionRecordRouter) SetInspectionRecordRouter(publicGroup, privateGroup *gin.RouterGroup) {
	s.PublicGroup = publicGroup
	s.PrivateGroup = privateGroup

	// [注意] 这里故意 panic 而不是静默 return。
	// 启动阶段参数错了属于编程错误，直接崩掉比"上线后访问 404、排查两小时"好得多。
	if s.PrivateGroup == nil {
		panic("inspectionrecord: SetInspectionRecordRouter 收到的 privateGroup 为 nil，请检查 initialize/router_biz.go 的传参顺序")
	}

	// [Go] publicGroup 目前没用到，但必须显式"消费"一下。
	// 结构体字段赋值不算"使用"，可读性上又不该删参数，所以这里留一行引用。
	_ = s.PublicGroup

	// [注意] Group("inspectionRecord") 的返回值是**新的**分组对象，
	// 在这个副本上挂中间件不会影响 PrivateGroup 本身，也不会影响别的模块。
	inspectionRecordRouter := s.PrivateGroup.Group("inspectionRecord").Use(middleware.OperationRecord())
	inspectionRecordRouterWithoutRecord := s.PrivateGroup.Group("inspectionRecord")

	// [Go] 局部变量不能叫 api，否则会遮蔽上面 import 进来的包名 api，
	// 变成"自己引用自己"的编译错误（invalid operation: api.ApiGroupApp）。
	inspectionRecordApi := api.ApiGroupApp.InspectionRecordApi

	{
		// 写操作：挂 OperationRecord 中间件，自动往 sys_operation_record 记一条操作日志。
		// [注意] 第二个参数是"方法值"——api.CreateInspectionRecord 后面**不能加括号**。
		// 加了括号表示"立刻调用它"，传进去的是返回值（这里没有返回值），类型对不上。
		inspectionRecordRouter.POST("createInspectionRecord", inspectionRecordApi.CreateInspectionRecord)
		inspectionRecordRouter.DELETE("deleteInspectionRecord", inspectionRecordApi.DeleteInspectionRecord)
		inspectionRecordRouter.PUT("updateInspectionRecord", inspectionRecordApi.UpdateInspectionRecord)
	}
	{
		// 查询：不挂 OperationRecord，否则列表接口一刷新就写一条日志，
		// sys_operation_record 表几天就爆了。
		inspectionRecordRouterWithoutRecord.GET("getInspectionRecord", inspectionRecordApi.GetInspectionRecord)
		inspectionRecordRouterWithoutRecord.GET("getInspectionRecordList", inspectionRecordApi.GetInspectionRecordList)
	}
}
