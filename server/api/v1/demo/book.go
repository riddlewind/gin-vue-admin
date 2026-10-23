// ============================================================================
// 【文件导读】第三步：API 层 —— HTTP 接线员。
//
// 固定四步，顺序不能变：
//
//	绑定参数 -> 校验参数 -> 调用 Service -> 统一响应
//
// 这一层可以有 gin.Context（Service 层绝对不能有）。
// 它不写业务逻辑，只做"翻译"：把 HTTP 请求翻译成 Go 参数，
// 把 Go 的返回值翻译成 HTTP 响应。
//
// 【本文件涉及的 Go 语法】
//   - 方法接收者 (a *BookApi)
//   - 自定义校验规则 map（utils.Rules）
//   - 多返回值接收
//   - 结构体字面量"字段名: 值"初始化
//
// ============================================================================
package demo

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/demo"
	demoReq "github.com/flipped-aurora/gin-vue-admin/server/model/demo/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// bookVerify 本模块的校验规则
//
// [Go] 类型是 utils.Rules，实际定义是 map[string][]string：
// key = 结构体字段名（注意是字段名 Title，不是 json 名 title）
// value = 规则列表
//
// [注意] 字段名必须写 Go 结构体里的名字，大小写敏感。
// 写成 "title" 永远不会生效，而且不报错——静默失效，最难查的那类 bug。
//
// 可用规则见 utils/validator.go：notEmpty、gt=、ge=、lt=、le=、eq=、ne=、regexp=
var bookVerify = utils.Rules{
	"Title":  {"notEmpty"},
	"Author": {"notEmpty"},
	"Price":  {"ge=0"},
	"Stock":  {"ge=0"},
}

// BookApi 图书接口集合
type BookApi struct{}

// CreateBook
// @Tags     DemoBook
// @Summary  新增图书
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data  body      demo.Book                  true  "书名, 作者, 价格, 库存"
// @Success  200   {object}  response.Response{msg=string}  "新增图书"
// @Router   /demo/book [post]
func (a *BookApi) CreateBook(c *gin.Context) {
	var book demo.Book

	// [Go] ShouldBindJSON 按 json tag 把请求体解析进结构体。
	// [注意] 用 Should 前缀而不是 Bind：Bind 系列失败时会自己写 400 并中止，
	// 那样就绕过了我们统一的 {code,msg} 响应格式，前端会拿到裸 400 一脸懵。
	if err := c.ShouldBindJSON(&book); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 校验：注意传的是 book 整体，utils.Verify 内部会用反射按字段名去 bookVerify 里找规则
	if err := utils.Verify(book, bookVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 交给 Service。这里传 &book 是因为 Service 签名要的是指针。
	if err := bookService.CreateBook(&book); err != nil {
		// [注意] 完整错误进服务端日志，给前端只返回模糊提示——
		// 不要把数据库报错原文吐给前端，那等于免费给攻击者做信息收集。
		global.GVA_LOG.Error("新增图书失败", zap.Error(err))
		response.FailWithMessage("新增失败", c)
		return
	}

	response.OkWithMessage("新增成功", c)
}

// DeleteBook
// @Tags     DemoBook
// @Summary  删除图书
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data  body      request.GetById             true  "图书ID"
// @Success  200   {object}  response.Response{msg=string}  "删除图书"
// @Router   /demo/book [delete]
func (a *BookApi) DeleteBook(c *gin.Context) {
	// [Go] 用通用的 GetById 结构体接收 {id: 1}，不必为了一个 ID 再造一个结构体。
	var req struct {
		ID uint `json:"id" form:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	// [注意] uint 的零值是 0，所以"没传 id"和"传了 id=0"无法区分。
	// 必须显式判断 == 0，否则会执行 DELETE WHERE id=0（虽然删不到东西，但逻辑是错的）。
	if req.ID == 0 {
		response.FailWithMessage("图书ID不能为空", c)
		return
	}

	if err := bookService.DeleteBook(req.ID); err != nil {
		global.GVA_LOG.Error("删除图书失败", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// UpdateBook
// @Tags     DemoBook
// @Summary  更新图书
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data  body      demo.Book                  true  "图书ID及要更新的字段"
// @Success  200   {object}  response.Response{msg=string}  "更新图书"
// @Router   /demo/book [put]
func (a *BookApi) UpdateBook(c *gin.Context) {
	var book demo.Book
	if err := c.ShouldBindJSON(&book); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 更新必须先有 ID，否则不知道改哪条
	if book.ID == 0 {
		response.FailWithMessage("图书ID不能为空", c)
		return
	}
	if err := utils.Verify(book, bookVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if err := bookService.UpdateBook(&book); err != nil {
		global.GVA_LOG.Error("更新图书失败", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// GetBook
// @Tags     DemoBook
// @Summary  按ID获取图书
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id  query  int  true  "图书ID"
// @Success  200  {object}  response.Response{data=demo.Book,msg=string}  "图书详情"
// @Router   /demo/book [get]
func (a *BookApi) GetBook(c *gin.Context) {
	// [Go] GET 请求的参数在 URL 上，所以用 ShouldBindQuery 而不是 ShouldBindJSON。
	// [注意] 这一步依赖模型上的 `form:"id"` 标签；只写 json tag 的话这里永远绑不上。
	var q struct {
		ID uint `json:"id" form:"id"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	if q.ID == 0 {
		response.FailWithMessage("图书ID不能为空", c)
		return
	}

	book, err := bookService.GetBook(q.ID)
	if err != nil {
		global.GVA_LOG.Error("获取图书失败", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}
	response.OkWithDetailed(book, "获取成功", c)
}

// GetBookList
// @Tags     DemoBook
// @Summary  分页获取图书列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    page      query  int     false  "页码"
// @Param    pageSize  query  int     false  "每页数量"
// @Param    title     query  string  false  "书名模糊搜索"
// @Success  200  {object}  response.Response{data=response.PageResult,msg=string}  "列表,总数,页码,每页数量"
// @Router   /demo/bookList [get]
func (a *BookApi) GetBookList(c *gin.Context) {
	var search demoReq.BookSearch
	if err := c.ShouldBindQuery(&search); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// [Go] 三个返回值一次接收：数据、总数、错误。
	// [注意] 这里没有校验分页参数，因为 Paginate() 内部已经把非法值修正了
	// （page<=0 归 1，pageSize 超过 100 截断到 100），这是它比手写 limit/offset 强的地方。
	list, total, err := bookService.GetBookList(search)
	if err != nil {
		global.GVA_LOG.Error("获取图书列表失败", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	// [Go] 注意这里用的字段名：List / Total / Page / PageSize，
	// 前端读的是 res.data.list 和 res.data.total，别改。
	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     search.Page,
		PageSize: search.PageSize,
	}, "获取成功", c)
}
