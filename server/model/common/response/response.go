// ============================================================================
// 【文件导读】全项目统一的 HTTP 响应封装。
//
// 所有接口最终都调用这里的函数返回，保证前端拿到的永远是同一个结构：
//
//	{ "code": 0, "data": {...}, "msg": "成功" }
//
// 前端 `src/utils/request.js` 里判断的 `response.data.code === 0`，
// 就是靠这里的 SUCCESS 常量对上的。改这两个数字会让整个前端全部报错。
//
// 【本文件涉及的 Go 语法】
//   - interface{}（空接口）：可以装任意类型
//   - 常量组 const (...)
//   - 结构体字面量的"位置初始化"写法
//   - 包内函数互相调用
//
// ============================================================================
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code int `json:"code"` // 业务状态码，0 表示成功
	// [Go] interface{} 是空接口，任何类型都满足它，效果约等于"任意类型"。
	// Go 1.18 之后可以用 any 作为它的别名，两者完全等价。
	// 这里用 interface{} 是为了让 data 既能放列表、也能放单个对象、还能放 nil。
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"` // 提示信息，前端直接拿去弹 ElMessage
}

// [Go] 常量组。Go 的常量只能是编译期能确定的值（数字、字符串、布尔），
// 且未指定类型时是"无类型常量"，用起来会自动适配。
//
// [注意] 这两个数字是前后端契约的一部分，不要随意改动：
//   - SUCCESS = 0：前端 request.js 只认 0 为成功
//   - ERROR   = 7：历史遗留的取值，前端不区分具体错误码，只看非 0 即失败
const (
	ERROR   = 7
	SUCCESS = 0
)

// Result 最底层的响应方法，其余 Ok/Fail 系列全都转调它
func Result(code int, data interface{}, msg string, c *gin.Context) {
	// [Go] c.JSON 的第一个参数是 HTTP 状态码。
	// [注意] 注意这里无论成功失败都返回 http.StatusOK(200)，
	// 业务上的成败靠 body 里的 code 表达，而不是 HTTP 状态码。
	// 这是国内后端常见的做法，前端因此只需要解析 body 不处理非 2xx。
	//
	// [Go] Response{code, data, msg} 是"位置初始化"：按字段声明顺序赋值。
	// 一旦结构体字段顺序改变了，这里就会静默地赋错值——
	// 所以更推荐写成 Response{Code: code, Data: data, Msg: msg}。
	c.JSON(http.StatusOK, Response{
		code,
		data,
		msg,
	})
}

// Ok 操作成功，无数据
func Ok(c *gin.Context) {
	// [Go] map[string]interface{}{} 创建一个空 map 字面量。
	// 这里用空 map 而不是 nil，是为了让前端拿到 data: {} 而不是 data: null，
	// 避免前端做 data.xxx 时类型报错。
	Result(SUCCESS, map[string]interface{}{}, "操作成功", c)
}

// OkWithMessage 操作成功，只返回提示语
func OkWithMessage(message string, c *gin.Context) {
	Result(SUCCESS, map[string]interface{}{}, message, c)
}

// OkWithData 操作成功，返回数据，提示语固定为"成功"
func OkWithData(data interface{}, c *gin.Context) {
	Result(SUCCESS, data, "成功", c)
}

// OkWithDetailed 操作成功，数据和提示语都自定义（最常用）
func OkWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(SUCCESS, data, message, c)
}

// Fail 操作失败，无具体原因
func Fail(c *gin.Context) {
	Result(ERROR, map[string]interface{}{}, "操作失败", c)
}

// FailWithMessage 操作失败，带具体错误原因（API 层最常用）
func FailWithMessage(message string, c *gin.Context) {
	Result(ERROR, map[string]interface{}{}, message, c)
}

// NoAuth 未认证 / token 失效
//
// [注意] 这是全文件唯一一个返回真实 HTTP 401 的方法。
// 前端的 request.js 拦截器专门判断 `error.response.status === 401`，
// 一旦收到就清空本地 token 并跳转登录页。改这里会破坏自动登出逻辑。
func NoAuth(message string, c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Response{
		7,
		nil,
		message,
	})
}

// FailWithDetailed 操作失败，同时返回附加数据（例如让前端重新加载）
func FailWithDetailed(data interface{}, message string, c *gin.Context) {
	Result(ERROR, data, message, c)
}
