// ============================================================================
// 【文件导读】第四步：API 层 —— HTTP 接线员。
//
// 固定四步，顺序不能变：
//
//	绑定参数 -> 校验参数 -> 调用 Service -> 统一响应
//
// 这一层可以有 gin.Context（Service 层绝对不能有）。
// 它不写业务逻辑，只做翻译：把 HTTP 请求翻译成 Go 参数，
// 把 Go 返回值翻译成 HTTP 响应。
//
// 【本文件涉及的 Go 语法】
//   - 结构体字段持有依赖（依赖注入），比包级全局变量更好测
//   - 方法接收者 (a *InspectionRecordApi)
//   - 匿名结构体（只为一个请求体造一个用完就扔的类型）
//   - strconv.Atoi 的 (值, error) 双返回
//   - errors.Is 判断"是不是某类错误"（不是比字符串）
//
// 【和原始需求不一致的 3 处，都是会直接出错的】
//  1. 不存在 reply.OkWithMessage / reply.FailWithMessage，本项目叫 response.*
//  2. response.OkWithMessage 只收 (message, c) 两个参数，塞不进数据；
//     带数据要用 response.OkWithDetailed(data, message, c)
//  3. 不存在 service_service 这个包；Service 类型在 service/inspectionrecord 里
//
// ============================================================================
package inspectionrecord

import (
	"errors"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/inspectionrecord"
	// [Go] 包名冲突的处理：model/inspectionrecord 和 service/inspectionrecord 的
	// 包名都叫 inspectionrecord（目录名相同）。同一个文件里导入两个同名包，
	// 必须给其中一个起别名，Go 不允许两个标识符同时叫 inspectionrecord。
	inspectionrecordService "github.com/flipped-aurora/gin-vue-admin/server/service/inspectionrecord"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InspectionRecordApi 巡检记录接口集合
//
// [Go] 这里用的是"结构体字段持有依赖"，而不是项目里 demo/system 包那种包级变量：
//
//	var bookService = service.ServiceGroupApp.DemoServiceGroup.BookService
//
// 字段注入的好处是**可测试**：测试时可以直接 new 一个 Api 塞进去一个连着测试库的
// Service，完全不用碰全局单例。包级变量做不到这一点（全局只有一个）。
//
// [注意] Service 字段是**值类型**不是指针。值为零时它内部的 DB 字段是 nil，
// 而 Service 内部每个方法都通过 db() 兜底回落到 global.GVA_DB，所以
// ServiceGroupApp 用 new(ServiceGroup) 自动装配出来的零值 Api 也能正常工作。
// 这也正是 Service 层要写 db() 而不是直接写 s.DB 的原因。
type InspectionRecordApi struct {
	Service inspectionrecordService.InspectionRecordService
}

// inspectionRecordPayload 只用于解析 JSON 入参，不参与业务
//
// [为什么需要它] 标准库 encoding/json 解析 time.Time 时**只接受 RFC3339**
// （2026-10-23T09:30:00+08:00）。而国内前端日期选择器最常用的是
// "2026-10-23 09:30:00"（中间是空格），直接绑到 model 的 time.Time 字段上会报：
//
//	parsing time "2026-10-23 09:30:00" as "2006-01-02T15:04:05Z07:00":
//	cannot parse " 09:30:00" as "T"
//
// 这不是前端写错了——Element Plus 的 value-format="YYYY-MM-DD HH:mm:ss" 就是这个格式。
//
// [Go] 解决办法是"嵌入 + 同名字段覆盖"：
// 外层这个显式声明的 InspectionTime 深度是 0，内嵌结构体里那个深度是 1，
// encoding/json 遇到同名的 JSON key 时**优先选层级更浅的那个**，
// 所以解析时走的是这里的 string 版本，其它字段照旧从内嵌结构体提升。
// 解析完再自己调用 Service 的 ParseInspectionTime 转换，兼容多种格式。
type inspectionRecordPayload struct {
	inspectionrecord.InspectionRecord
	InspectionTime string `json:"inspectionTime" form:"inspectionTime"`
}

// toModel 转成真正的模型，顺便把时间字符串解析成 time.Time
func (p *inspectionRecordPayload) toModel() (inspectionrecord.InspectionRecord, error) {
	record := p.InspectionRecord
	if p.InspectionTime == "" {
		return record, nil
	}
	// [Go] 这里复用了 Service 包导出的 ParseInspectionTime，
	// 保证"新增/更新"和"列表筛选"用的是同一套格式规则，不会两边不一致。
	t, _, err := inspectionrecordService.ParseInspectionTime(p.InspectionTime)
	if err != nil {
		return record, err
	}
	record.InspectionTime = t
	return record, nil
}

// CreateInspectionRecord
// @Tags     InspectionRecord
// @Summary  新增设备巡检记录
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data  body      inspectionrecord.InspectionRecord  true  "巡检记录"
// @Success  200   {object}  response.Response{data=inspectionrecord.InspectionRecord,msg=string}  "创建成功"
// @Router   /inspectionRecord [post]
func (a *InspectionRecordApi) CreateInspectionRecord(c *gin.Context) {
	var payload inspectionRecordPayload

	// [Go] ShouldBindJSON 按 json tag 把请求体解析进结构体。
	// [注意] 用 Should 前缀而不是 Bind：Bind 系列失败时会自己写 400 并中止请求，
	// 那样就绕过了项目统一的 {code,msg,data} 格式，前端拿到裸 400 一脸懵。
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.FailWithMessage("参数解析失败："+err.Error(), c)
		return
	}

	record, err := payload.toModel()
	if err != nil {
		// 时间格式错误属于参数问题，原样把提示给前端
		response.FailWithMessage(err.Error(), c)
		return
	}

	// [注意] 这里传 &record（指针）：Service 创建成功后要把自增主键回填到 record.ID，
	// 传值的话回填的是副本，下面返回给前端的 ID 就是 0。
	if err = a.Service.CreateInspectionRecord(&record); err != nil {
		// 完整错误进服务端日志，给前端只回模糊提示。
		// 把数据库报错原文吐给前端，等于免费给攻击者做信息收集。
		global.GVA_LOG.Error("新增巡检记录失败", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}

	// [注意] 用的是 OkWithDetailed 而不是需求里写的 OkWithMessage：
	// OkWithMessage 的签名是 (message string, c *gin.Context)，压根没有数据参数，
	// 第二个位置传 gin.H 会直接编译失败（类型不对）。
	// OkWithDetailed(data, message, c) 产出的结构是 {code, data, msg}，
	// 前端读 res.data 就能拿到记录，不会多套一层 data.data。
	response.OkWithDetailed(record, "创建成功", c)
}

// DeleteInspectionRecord
// @Tags     InspectionRecord
// @Summary  删除设备巡检记录（软删除）
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data  body      object  true  "巡检记录ID"
// @Success  200   {object}  response.Response{msg=string}  "删除成功"
// @Router   /inspectionRecord [delete]
func (a *InspectionRecordApi) DeleteInspectionRecord(c *gin.Context) {
	// [Go] 匿名结构体：只为一个请求体造一个"用完就扔"的类型，不必污染包级命名空间。
	// 项目自带的 request.GetById 也能用，但它多一个 import，这里就地定义更省事。
	var req struct {
		ID int `json:"id" form:"id"`
	}

	// [注意] DELETE 请求带 JSON body 时这么绑；但如果前端把 id 放在 URL 上
	// （?id=5），ShouldBindJSON 会因为 body 为空而返回 EOF 错误。
	// 所以这里不能一失败就 return，要留一条 query 的退路。
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		if id := queryID(c, "id"); id > 0 {
			req.ID = id
		}
	}

	// [注意] int 的零值是 0，"没传 id" 和 "传了 id=0" 无法区分，必须显式挡。
	// 不挡的话会执行 DELETE ... WHERE id=0，删不到东西也不报错，前端以为删成功了。
	if req.ID <= 0 {
		response.FailWithMessage("巡检记录ID不能为空", c)
		return
	}

	if err := a.Service.DeleteInspectionRecord(req.ID); err != nil {
		global.GVA_LOG.Error("删除巡检记录失败", zap.Error(err), zap.Int("id", req.ID))
		response.FailWithMessage("删除失败", c)
		return
	}
	response.OkWithMessage("删除成功", c)
}

// GetInspectionRecord
// @Tags     InspectionRecord
// @Summary  按ID获取巡检记录
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    id  query  int  true  "巡检记录ID"
// @Success  200  {object}  response.Response{data=inspectionrecord.InspectionRecord,msg=string}  "查询成功"
// @Router   /inspectionRecord [get]
func (a *InspectionRecordApi) GetInspectionRecord(c *gin.Context) {
	// [Go] 先路径参数 /inspectionRecord/:id，再退到 query ?id=1。
	// 这样同一个 handler 既能挂 REST 风格路由，也能挂 query 风格路由。
	id := queryID(c, "id")
	if id <= 0 {
		id = pathID(c, "id")
	}
	if id <= 0 {
		response.FailWithMessage("巡检记录ID不能为空", c)
		return
	}

	record, err := a.Service.GetInspectionRecord(id)
	if err != nil {
		// [Go] errors.Is 判断错误链里有没有目标错误。
		// [注意] 不要写 err.Error() == "record not found" —— 字符串比对一改版本就失效。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.FailWithMessage("记录不存在或已被删除", c)
			return
		}
		global.GVA_LOG.Error("获取巡检记录失败", zap.Error(err), zap.Int("id", id))
		response.FailWithMessage("查询失败", c)
		return
	}

	response.OkWithDetailed(record, "查询成功", c)
}

// GetInspectionRecordList
// @Tags     InspectionRecord
// @Summary  分页获取巡检记录列表
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data  body  inspectionrecord.InspectRecordInfo  false  "分页与筛选条件"
// @Success  200   {object}  response.Response{data=response.PageResult,msg=string}  "查询成功"
// @Router   /inspectionRecordList [post]
func (a *InspectionRecordApi) GetInspectionRecordList(c *gin.Context) {
	var info inspectionrecord.InspectRecordInfo

	// [Go] 同一个接口要同时吃两种传参方式，靠请求方法分流：
	//   POST/PUT -> 参数在 body 里（JSON），用 ShouldBindJSON
	//   GET      -> 参数在 URL 上（?page=1），用 ShouldBindQuery
	//
	// [注意] 这两种不能混用：GET 请求没有 body，调 ShouldBindJSON 只会拿到 EOF 错误；
	// 反过来 POST 用 ShouldBindQuery 则永远读不到 body 里的条件，筛选会静默失效。
	if c.Request.Method == "POST" || c.Request.Method == "PUT" {
		if err := c.ShouldBindJSON(&info); err != nil {
			response.FailWithMessage("参数解析失败："+err.Error(), c)
			return
		}
	} else {
		if err := c.ShouldBindQuery(&info); err != nil {
			response.FailWithMessage("参数解析失败："+err.Error(), c)
			return
		}
	}

	// 分页兜底。Service 层虽然也会修正一次，但这里先补上是为了让响应里的
	// page / pageSize 回显的是"真正生效的值"，前端分页组件才不会显示成 0。
	if info.Page <= 0 {
		info.Page = 1
	}
	if info.PageSize <= 0 {
		info.PageSize = 10
	}

	list, total, err := a.Service.GetInspectionRecordInfo(&info)
	if err != nil {
		// [Go] 用 errors.Is 区分错误来源：参数类错误（时间格式）可以原样给用户看，
		// 数据库错误则只记日志、给一句模糊提示。
		// 注意这里判断的是"错误链里有没有这个哨兵值"，所以 Service 里用 %w 包装过也认得出来。
		if errors.Is(err, inspectionrecordService.ErrInvalidInspectionTime) {
			response.FailWithMessage(err.Error(), c)
			return
		}
		global.GVA_LOG.Error("获取巡检记录列表失败", zap.Error(err))
		response.FailWithMessage("查询失败", c)
		return
	}

	// [Go] response.PageResult 的字段是 List / Total / Page / PageSize，
	// 前端表格读的是 res.data.list 和 res.data.total——和需求里的
	// gin.H{"list": ..., "total": ...} 结构一致，只是多了 page/pageSize 两个回显字段。
	// 用结构体而不是 gin.H，是为了让 Swagger 能生成准确的返回模型。
	response.OkWithDetailed(
		response.PageResult{
			List:     list,
			Total:    total,
			Page:     info.Page,
			PageSize: info.PageSize,
		},
		"查询成功", c)
}

// UpdateInspectionRecord
// @Tags     InspectionRecord
// @Summary  更新设备巡检记录
// @Security ApiKeyAuth
// @accept   application/json
// @Produce  application/json
// @Param    data  body      inspectionrecord.InspectionRecord  true  "巡检记录（必须带ID）"
// @Success  200   {object}  response.Response{msg=string}  "更新成功"
// @Router   /inspectionRecord [put]
func (a *InspectionRecordApi) UpdateInspectionRecord(c *gin.Context) {
	var payload inspectionRecordPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.FailWithMessage("参数解析失败："+err.Error(), c)
		return
	}

	record, err := payload.toModel()
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 更新必须先有 ID，否则不知道改哪条。
	// [注意] 这一句和 Service 里的判断重复了，但**不能省**：
	// API 层挡掉的是"用户没传 ID"（这是参数问题，该给人话提示），
	// Service 层挡的是"内部调用方传了 ID=0"（这是编程错误，该报错）。
	// 两层的守卫目的不同，不是冗余。
	if record.ID == 0 {
		response.FailWithMessage("巡检记录ID不能为空", c)
		return
	}

	if err = a.Service.UpdateInspectionRecord(&record); err != nil {
		global.GVA_LOG.Error("更新巡检记录失败", zap.Error(err), zap.Uint("id", record.ID))
		response.FailWithMessage("更新失败", c)
		return
	}
	response.OkWithMessage("更新成功", c)
}

// ============================================================================
// 内部辅助函数（小写开头 = 只在本包内可见）
// ============================================================================

// queryID 从 URL 查询串里取一个正整数 ID
//
// [Go] strconv.Atoi 返回 (int, error)：
// 转换失败时返回 0 和一个 error，所以必须两个都接，不能只接值。
// [注意] 这里刻意不把 error 往上传——"传了个 abc 当 ID"和"没传 ID"
// 对用户来说都是"ID 不对"，用同一句提示就够，不必细分。
func queryID(c *gin.Context, key string) int {
	raw := c.Query(key)
	if raw == "" {
		// [坑] gin 的 c.Query 是**区分大小写**的，而本项目前端 request 封装
		// 是直接拿模型字段名当参数名传的，所以删除/详情实际发的是 ?ID=1（大写）。
		// 官方模块里写的就是 c.Query("ID")。这里做一次大小写回退，
		// 让 ?ID=1 和 ?id=1 两种写法都能用，避免前端一改就 404/报"ID不能为空"。
		if alt := strings.ToUpper(key); alt != key {
			raw = c.Query(alt)
		} else if alt := strings.ToLower(key); alt != key {
			raw = c.Query(alt)
		}
	}
	if raw == "" {
		return 0
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// pathID 从路径参数里取一个正整数 ID（对应路由 /xxx/:id）
func pathID(c *gin.Context, key string) int {
	raw := c.Param(key)
	if raw == "" {
		return 0
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}
