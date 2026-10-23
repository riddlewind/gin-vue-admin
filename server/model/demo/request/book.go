// ============================================================================
// 【文件导读】查询入参模型 —— 专门给"列表页"用的结构体。
//
// 为什么不直接复用 Book 当查询参数？
// 因为查询条件是 Book 的**子集 + 分页信息**，而且以后可能加"价格区间""上架时间范围"
// 这些 Book 里根本没有的字段。所以单独定义一个 XxxSearch 是 GVA 的固定套路。
//
// 【本文件涉及的 Go 语法】
//   - 结构体嵌入（这里嵌入的是分页参数）
//   - 三个标签各管一段：json 管 body、form 管 query
//
// ============================================================================
package request

import (
	common "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// BookSearch 图书列表查询条件
//
// [Go] 嵌入 PageInfo 之后，BookSearch 就自动拥有了 Page / PageSize / Keyword 三个字段，
// 而且**连 PageInfo 的方法也一起继承了**——包括 Paginate()。
// 所以下面 service 层可以直接写 info.Paginate()。
//
// 这正是嵌入和"组合"的区别：组合要写 info.PageInfo.Paginate()，嵌入可以直接 info.Paginate()。
type BookSearch struct {
	common.PageInfo

	// 模糊搜索的书名。前端传空字符串表示"不过滤"。
	Title string `json:"title" form:"title"`
}
