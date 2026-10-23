// ============================================================================
// 【文件导读】API 层 —— HTTP 接口的"接线员"。
//
// 这一层只干四件事，顺序固定：
//  1. 绑定参数（ShouldBindJSON / ShouldBindQuery / FormFile ...）
//  2. 校验参数（utils.Verify + 校验规则）
//  3. 调用 Service（只调，不写业务逻辑）
//  4. 统一响应（response.OkWithXxx / FailWithXxx）
//
// 铁律：API 层不写业务逻辑，Service 层不碰 gin.Context。
// 如果你的接口里出现了 if/else 的业务判断，那说明代码放错层了，该挪到 Service。
//
// 【本文件涉及的 Go 语法】
//   - 方法接收者 (e *CustomerApi)
//   - 结构体指针传参 &customer 与解引用 *claims
//   - 多返回值接收：list, total, err := xxx()
//   - 匿名结构体嵌套调用：response.PageResult{List: ..., Total: ...}
//
// ============================================================================
package example

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	// [Go] 包别名。model/example/response 这个包的包名是 response，
	// 和上面 model/common/response 重名了，必须起别名区分。
	exampleRes "github.com/flipped-aurora/gin-vue-admin/server/model/example/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CustomerApi 客户接口集合
//
// [Go] 又是空结构体。它不需要状态，因为 service 是通过 enter.go 里的
// 包级变量 customerService 拿到的，而不是作为字段存进来的。
type CustomerApi struct{}

// CreateExaCustomer 创建示例客户
// @Tags      ExaCustomer
// @Summary   创建客户
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      example.ExaCustomer            true  "客户用户名, 客户手机号码"
// @Success   200   {object}  response.Response{msg=string}  "创建客户"
// @Router    /customer/customer [post]
//
// [注意] 上面这一整块是 Swagger 注解，swag init 会解析它生成 API 文档。
// 项目规范要求"注解必须和真实行为一致"——参数从 body 来的就写 body，
// 从 query 来的就写 query，写错了生成的文档会误导调用方。
func (e *CustomerApi) CreateExaCustomer(c *gin.Context) {
	var customer example.ExaCustomer

	// [Go] ShouldBindJSON 从请求体（body）里按 json tag 反序列化到结构体。
	//
	// [注意] ShouldBindJSON 与 BindJSON 的区别：
	//   BindJSON    : 失败时自动往响应里写 400 并 Abort，你不用处理错误（也不可定制）
	//   ShouldBindJSON: 只返回 error，由你自己决定怎么响应
	// 项目统一用 Should 系列，因为要返回自己的 {code,msg} 结构而不是裸 400。
	err := c.ShouldBindJSON(&customer)
	if err != nil {
		// [Go] err.Error() 把 error 转成字符串。
		// [注意] 直接把绑定错误原文返回给前端会暴露内部字段名，
		// 生产环境一般要换成友好的提示，这里为了开发方便才这么写。
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 参数校验。utils.CustomerVerify 是一组校验规则（基于 validator 的 tag）。
	// [Go] 校验失败时返回的 err 是多个错误聚合成的，Error() 出来是多行文本。
	err = utils.Verify(customer, utils.CustomerVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// [注意] 这两行是"服务端补充字段"的经典手法：
	// 创建人和创建人角色不由前端传（否则前端可以伪造），
	// 而是从 JWT 里解析出来的当前登录人信息。
	//
	// [Go] utils.GetUserID(c) 内部就是 c.Get("claims") + 类型断言，
	// 而这个 claims 正是 JWTAuth 中间件用 c.Set("claims", claims) 放进去的。
	// ——这就是中间件和业务代码之间传递数据的通道。
	customer.SysUserID = utils.GetUserID(c)
	customer.SysUserAuthorityID = utils.GetUserAuthorityId(c)

	// 只调用，不判断业务规则。业务规则在 Service 里。
	err = customerService.CreateExaCustomer(customer)
	if err != nil {
		// [Go] zap.Error(err) 把 error 转成 zap 的字段。
		// [注意] 这里的分工很重要：
		//   - 完整错误写进服务端日志（排查问题用）
		//   - 只给前端返回模糊的"创建失败"（不泄露数据库细节，防信息泄露）
		// 想调试时看不到具体原因？去看 log/ 目录下的日志文件。
		global.GVA_LOG.Error("创建失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}
	response.OkWithMessage("创建成功", c)
}

// DeleteExaCustomer 删除示例客户
// @Tags      ExaCustomer
// @Summary   删除客户
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      example.ExaCustomer            true  "客户ID"
// @Success   200   {object}  response.Response{msg=string}  "删除客户"
// @Router    /customer/customer [delete]
func (e *CustomerApi) DeleteExaCustomer(c *gin.Context) {
	var customer example.ExaCustomer

	// [注意] DELETE 请求也从 body 取参数，而不是从 URL 路径取。
	// 虽然 RESTful 更推荐 DELETE /customer/:id，但这里统一走 body
	// （前端 axios 的 delete 也支持带 data）。
	// [Go] 如果确实要用路径参数，写法是 c.Param("id")，路由则写成 "customer/:id"。
	err := c.ShouldBindJSON(&customer)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// [注意] 这里校验的是 customer.GVA_MODEL 而不是 customer 整体。
	// 原因：内嵌结构体可以直接当独立参数传（因为字段被提升了），
	// 而 IdVerify 这条规则只需要 ID 非零，不需要校验客户名等字段。
	// [注意] ID 是 uint 类型，零值是 0，所以"没传 ID"和"传了 ID=0"无法区分——
	// 校验规则正是靠"ID 必须大于 0"来兜住这种情况。
	err = utils.Verify(customer.GVA_MODEL, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = customerService.DeleteExaCustomer(customer)
	if err != nil {
		global.GVA_LOG.Error("删除失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// UpdateExaCustomer 更新示例客户
// @Tags      ExaCustomer
// @Summary   更新客户信息
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      example.ExaCustomer            true  "客户ID, 客户信息"
// @Success   200   {object}  response.Response{msg=string}  "更新客户信息"
// @Router    /customer/customer [put]
func (e *CustomerApi) UpdateExaCustomer(c *gin.Context) {
	var customer example.ExaCustomer
	err := c.ShouldBindJSON(&customer)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 更新要比创建多校验一件事：ID 必须有（否则不知道更新谁）
	err = utils.Verify(customer.GVA_MODEL, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// [注意] 这里做了两轮校验。为什么不能合并成一次？
	// 因为两组规则作用在不同结构体上，而且失败提示要区分
	// "缺少ID" 和 "字段格式不对"，合并后用户就分不清了。
	err = utils.Verify(customer, utils.CustomerVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// [Go] 传 &customer（指针）给 Service，因为 Service 的签名要的是 *ExaCustomer。
	// 注意上面 Create 传的是值 customer，这里是 &customer——
	// 因为两个 Service 方法的签名不同，Go 不会自动帮你转换。
	err = customerService.UpdateExaCustomer(&customer)
	if err != nil {
		global.GVA_LOG.Error("更新失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// GetExaCustomer 获取示例客户
// @Tags      ExaCustomer
// @Summary   获取单一客户信息
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  query     example.ExaCustomer                                                true  "客户ID"
// @Success   200   {object}  response.Response{data=exampleRes.ExaCustomerResponse,msg=string}  "获取单一客户信息,返回包括客户详情"
// @Router    /customer/customer [get]
func (e *CustomerApi) GetExaCustomer(c *gin.Context) {
	var customer example.ExaCustomer

	// [Go] ShouldBindQuery：从 URL 的查询参数（?ID=1）绑定，而不是 body。
	// [注意] 这里能绑定成功，靠的是模型上那个 `form:"..."` tag。
	// 如果模型只写了 json tag，这种 GET 请求会永远拿到零值。
	//
	// 判断该用哪个：
	//   GET 查询    -> ShouldBindQuery（参数在 URL 上）
	//   POST/PUT    -> ShouldBindJSON（参数在 body 里）
	//   文件上传     -> c.FormFile
	//   Header 透传  -> c.GetHeader
	err := c.ShouldBindQuery(&customer)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(customer.GVA_MODEL, utils.IdVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// [Go] `data, err :=` 用的是 :=，但 err 在上面已经声明过了，
	// 这里只是给 err 重新赋值（因为 data 是新变量，所以用 := 合法）。
	// [注意] := 要求左边至少有一个是新变量，否则编译报错 "no new variables on left side"。
	data, err := customerService.GetExaCustomer(customer.ID)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	// [注意] 这里没有直接把 data 塞进 data 字段，而是包了一层
	// exampleRes.ExaCustomerResponse{Customer: data}。
	// 这是有意的设计：给响应加一层壳，以后想追加字段（比如关联的订单列表）
	// 不会破坏前端的取数路径（前端一直读 res.data.customer）。
	// [Go] 结构体字面量用"字段名: 值"初始化，比位置初始化安全得多。
	response.OkWithDetailed(exampleRes.ExaCustomerResponse{Customer: data}, "获取成功", c)
}

// GetExaCustomerList 获取示例客户列表
// @Tags      ExaCustomer
// @Summary   分页获取权限客户列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  query     request.PageInfo                                        true  "页码, 每页大小"
// @Success   200   {object}  response.Response{data=response.PageResult,msg=string}  "分页获取权限客户列表,返回包括列表,总数,页码,每页数量"
// @Router    /customer/customerList [get]
func (e *CustomerApi) GetExaCustomerList(c *gin.Context) {
	var pageInfo request.PageInfo
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// 校验页码必须为正，防止前端传 page=-1 导致 offset 算成负数
	err = utils.Verify(pageInfo, utils.PageInfoVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// [Go] 三个返回值用 := 同时接收。
	// [注意] utils.GetUserAuthorityId(c) 就是数据权限的入口：
	// 把当前登录人的角色 ID 传给 Service，Service 再据此过滤数据。
	// 前端无法伪造这个值，因为它来自服务端签名的 JWT。
	customerList, total, err := customerService.GetCustomerInfoList(utils.GetUserAuthorityId(c), pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取失败!", zap.Error(err))
		// [注意] 这里把 err.Error() 拼进了返回给前端的消息里，
		// 和上面 Create 的做法不一致（上面是模糊提示）。
		// 属于框架内的风格不统一，你自己写的时候选一种并保持全项目一致。
		response.FailWithMessage("获取失败"+err.Error(), c)
		return
	}

	// [注意] 分页响应的字段名固定为 list / total / page / pageSize，
	// 前端 customer.vue 里读的是 table.data.list 和 table.data.total，
	// 改字段名会让分页直接失效。
	// [Go] PageResult 是个结构体，这里用"字段名: 值"逐项填充。
	response.OkWithDetailed(response.PageResult{
		List:     customerList,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}
