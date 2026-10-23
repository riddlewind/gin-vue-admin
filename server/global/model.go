// ============================================================================
// 【文件导读】所有业务模型的"公共字段基类"。
//
// 每个业务表结构体都会内嵌它，从而自动获得 ID / 创建时间 / 更新时间 / 软删除。
// 这是 Go 的"嵌入（embedding）"语法撑起来的伪继承。
//
// 【本文件涉及的 Go 语法】
//   - 结构体嵌入（匿名成员）：把字段和方法"提升"到外层
//   - struct tag：告诉 GORM 和 encoding/json 该怎么处理这个字段
//
// ============================================================================
package global

import (
	"time"

	"gorm.io/gorm"
)

// GVA_MODEL 所有业务模型的基类
//
// [Go] 业务模型使用方式（注意是"内嵌"而不是"继承"）：
//
//	type Book struct {
//	    global.GVA_MODEL          // <-- 只写类型名，不写字段名，这叫匿名/嵌入成员
//	    Title string `json:"title"`
//	}
//
// 内嵌之后，book.ID、book.CreatedAt 都能直接访问，就像 Book 自己声明的一样，
// 这被称为字段"提升（promoted）"。区别在于：
//   - 它不是真正的继承，没有多态，也没有虚函数；
//   - 但 GORM 的 AutoMigrate 会把嵌入结构体的字段当作本表字段一起建出来。
//
// [注意] 正因为是"平铺进同一张表"，内嵌的结构体不要带 gorm:"embedded" 之外
// 的冲突设定，也不要在两层里出现同名字段，否则 GORM 会报字段冲突。
type GVA_MODEL struct {
	// [Go] struct tag 用反引号包裹，本质是一串给反射读取的元数据。
	// 这里一个字段同时带了两套 tag：
	//   gorm:"primarykey" -> GORM 建表时把它当主键
	//   json:"ID"         -> 序列化成 JSON 时字段名叫 ID（大写！）
	// 前端代码里到处出现的 row.ID 就是这么来的。改成 json:"id" 前端会全部取不到值。
	ID uint `gorm:"primarykey" json:"ID"` // 主键ID

	// [注意] 下面这两个字段没有写 tag，靠 GORM 的"约定优于配置"自动识别：
	// 名为 CreatedAt / UpdatedAt 的 time.Time 字段，
	// GORM 会在创建、更新时自动填值，不需要你手动赋值。
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间

	// [Go] gorm.DeletedAt 是 GORM 的软删除类型（内部含一个 *time.Time 和 Valid 标记）。
	// 带这个字段的模型执行 db.Delete() 时不会真的 DELETE，
	// 而是 UPDATE deleted_at 为当前时间；之后所有查询都会自动追加
	// WHERE deleted_at IS NULL，把"已删除"的记录过滤掉。
	//
	// [注意] json:"-" 表示序列化成 JSON 时完全忽略这个字段，
	// 前端永远看不到它。因为把软删除时间暴露给前端既没用又容易引起误解。
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间
}
