// ============================================================================
// 【文件导读】第一步：模型。整个链路的地基。
//
// 写业务模块永远从 model 开始，因为后面三层（service/api/router）都依赖它。
// 这个文件只回答一个问题：demo_books 这张表有哪些列、什么类型、什么约束。
//
// 【本文件涉及的 Go 语法】
//   - 结构体嵌入 global.GVA_MODEL（拿到 ID/时间戳/软删除）
//   - struct tag 三件套：json / form / gorm
//   - 方法定义 + 值接收者（TableName）
//
// ============================================================================
package demo

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// Book 图书示例表
//
// [Go] 命名约定：结构体名用大驼峰单数（Book），表名用蛇形复数（demo_books）。
// GVA 里所有系统表都以 sys_ 开头，业务表建议加自己的前缀，避免和插件撞名。
type Book struct {
	// [Go] 嵌入基类：自动获得 ID / CreatedAt / UpdatedAt / DeletedAt 四列。
	// 这是 Go 的"字段提升"——你可以直接 book.ID 访问，就像 Book 自己声明的一样。
	global.GVA_MODEL

	// [注意] gorm tag 里的分号是"选项分隔符"，不是 SQL 语句分隔符。
	//   comment:书名      -> 写进表注释，方便 DBA 看懂
	//   size:128          -> varchar(128)
	//   not null          -> 非空约束
	// 写错分隔符（用逗号）GORM 不会报错，会静默忽略后面所有选项——很阴的坑。
	Title string `json:"title" form:"title" gorm:"comment:书名;size:128;not null"`

	Author string `json:"author" form:"author" gorm:"comment:作者;size:64"`

	// [注意] 金额千万别用 float32/float64！（精度问题，0.1+0.2 != 0.3）
	// 这里用 float64 只是为了教学演示方便。
	// 真实项目要么用 decimal 存（如下面的 type:decimal(10,2)），
	// 要么干脆用 int 存"分"，避免浮点误差。
	Price float64 `json:"price" form:"price" gorm:"comment:价格;type:decimal(10,2);default:0"`

	Stock int `json:"stock" form:"stock" gorm:"comment:库存;default:0"`

	// [注意] Status 用 int 而不是 bool，是留给扩展的：
	// 现在是 1上架/0下架，以后可能要加 2=缺货、3=预售，bool 就不够用了。
	Status int `json:"status" form:"status" gorm:"comment:状态(1上架 0下架);default:1"`
}

// TableName 自定义表名
//
// [Go] 这是一个"值接收者"的方法：func (Book) 而不是 func (b *Book)。
// 因为方法体里不用 Book 的任何字段，只返回一个常量，用值接收者更轻量。
//
// [注意] 方法名和作用域：这个方法是 GORM 的约定接口——
// 只要模型上有 TableName() string，GORM 就用它当表名，
// 没有则按"结构体名转蛇形复数"自动推导（Book -> books）。
// 这里显式指定成 demo_books，避免和以后可能的其他 book 表冲突。
func (Book) TableName() string {
	return "demo_books"
}
