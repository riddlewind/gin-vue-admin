// ============================================================================
// 【文件导读】通用请求参数结构体集合。
//
// 这里放的是"到处都要用"的入参模型：分页、按 ID 查、批量 ID、空请求体。
// 它们被各个模块的 XxxSearch 结构体内嵌，避免每个模块重复定义 page/pageSize。
//
// 【本文件涉及的 Go 语法】
//   - struct tag 同时服务多个库（json / form / gorm）
//   - 指针接收者 (r *Type)
//   - 返回函数：把"对查询的改造动作"当值传递（GORM Scopes 模式）
//   - switch 的无表达式写法（相当于 if/else if 链）
//   - 类型转换 uint(x)
//
// ============================================================================
package request

import (
	"gorm.io/gorm"
)

// PageInfo 通用分页入参结构
//
// [Go] 同一个字段挂了两套 tag：
//   - json:"page"  前端用 JSON body 传参时（POST/PUT）匹配这个
//   - form:"page"  前端用 query string 或表单传参时（GET）匹配这个
//
// 少了 form tag 的话，GET 请求的 ?page=1 就绑定不上，永远拿到 0。
// 这是新手最常犯的错之一。
type PageInfo struct {
	Page     int    `json:"page" form:"page"`         // 页码
	PageSize int    `json:"pageSize" form:"pageSize"` // 每页大小
	Keyword  string `json:"keyword" form:"keyword"`   // 关键字
}

// Paginate 返回一个"查询改造函数"，供 GORM 的 Scopes 使用。
//
// [Go] 这个写法第一次见会很懵，拆开看就清楚了：
//
//	func (r *PageInfo) Paginate() func(db *gorm.DB) *gorm.DB
//	└─方法─┘ └─接收者─┘ └─方法名┘ └──────返回值是一个函数──────┘
//
// 方法返回的是"函数"而不是"数据"。这是 Go 里把行为当值传递的标准手法。
// 使用方式：db.Scopes(pageInfo.Paginate()).Find(&list)
//
// 好处：分页的边界处理逻辑被封装成一个可复用的"插件"，
// 任何查询都能挂上它，不用在每个 service 里重复写 limit/offset 计算。
func (r *PageInfo) Paginate() func(db *gorm.DB) *gorm.DB {
	// [Go] 指针接收者 (r *PageInfo)：因为下面要修改 r.Page / r.PageSize 的值，
	// 用值接收者改的只是副本，调用方看不到修正后的页码。
	return func(db *gorm.DB) *gorm.DB {
		if r.Page <= 0 {
			r.Page = 1
		}
		// [Go] switch 不写表达式时，每个 case 就是一个布尔条件，
		// 等价于 if ... else if ...，但可读性更好。
		// 注意 Go 的 switch 默认不穿透（不需要 break），想穿透要显式 fallthrough。
		switch {
		case r.PageSize > 100: // 防止前端传个 99999 把数据库打死
			r.PageSize = 100
		case r.PageSize <= 0:
			r.PageSize = 10
		}
		offset := (r.Page - 1) * r.PageSize
		// [Go] GORM 的链式调用：每个方法都返回 *gorm.DB，所以能一直点下去。
		// 这也是为什么 db 能当参数传来传去——它其实是一个"查询构造器"。
		return db.Offset(offset).Limit(r.PageSize)
	}
}

// GetById 按 ID 查询的入参结构
type GetById struct {
	ID int `json:"id" form:"id"` // 主键ID
}

// Uint 把 int 转成 uint
//
// [Go] 为什么需要这个转换？因为数据库主键是 uint 类型（见 GVA_MODEL.ID），
// 而前端传过来的 ID 习惯用 int。Go 是强类型语言，不允许隐式转换，
// int 和 uint 之间必须显式写 uint(x)，否则编译不通过。
// 这种"为一处类型差异专门写个小方法"的做法，在 Go 项目里非常普遍。
func (r *GetById) Uint() uint {
	return uint(r.ID)
}

// IdsReq 批量操作时的入参（批量删除、批量导出等）
type IdsReq struct {
	// [Go] []int 是切片（slice），可以理解为"动态数组"。
	// 对比数组 [3]int（长度固定、属于类型的一部分），切片用得更多。
	Ids []int `json:"ids" form:"ids"`
}

// GetAuthorityId 按角色 ID 查询的入参结构
type GetAuthorityId struct {
	AuthorityId uint `json:"authorityId" form:"authorityId"` // 角色ID
}

// Empty 空结构体，用于"不需要任何参数"的接口
//
// [Go] struct{} 是零大小类型，不占内存。
// 常见用法还有 map[string]struct{} 表示"集合"，以及 chan struct{} 表示"纯信号"。
type Empty struct{}
