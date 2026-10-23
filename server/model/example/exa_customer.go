// ============================================================================
// 【文件导读】GORM 数据模型 —— "客户"表。
//
// 这是"新增一个业务模块"时你要写的第一个文件。
// Model 层的职责只有一件事：描述数据长什么样（表结构 + 关联关系），
// 不写任何业务逻辑，也不碰 HTTP。
//
// 【本文件涉及的 Go 语法】
//   - 结构体嵌入（匿名成员）实现"字段提升"
//   - 一个字段挂三套 tag（json / form / gorm）
//   - 结构体类型字段（表达外键关联）
//
// ============================================================================
package example

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

// ExaCustomer 客户表模型
//
// [注意] 表名不需要你指定。GORM 的默认规则是"结构体名转蛇形并复数化"，
// ExaCustomer -> exa_customers。想自定义表名，就在文件里加一个方法：
//
//	func (ExaCustomer) TableName() string { return "my_customers" }
//
// （可参考 model/system/sys_base_menu.go 的写法）
type ExaCustomer struct {
	// [Go] 匿名嵌入：只写类型不写字段名。
	// 效果是 ID / CreatedAt / UpdatedAt / DeletedAt 四个字段被"提升"到 ExaCustomer 上，
	// 你可以直接 customer.ID 访问；同时 GORM 建表时会把它们当成 exa_customers 表的列。
	//
	// [注意] 这不是继承。Go 没有继承，嵌入只是"字段和方法的借用 + 平铺到同一张表"。
	global.GVA_MODEL

	// [Go] 三套 tag 各司其职，同一个字段能同时适配三个方向：
	//   json:"customerName" -> 接口返回给前端时叫 customerName
	//   form:"customerName" -> 接收 GET query / 表单时按这个名字绑定
	//   gorm:"comment:客户名" -> 建表时把注释写进表结构（方便 DBA 看）
	//
	// [注意] 大小写必须严格一致。前端传 customerName，这里就不能写 customer_name，
	// 否则字段是空的，而且不会报错——静默失败，排查起来很痛苦。
	CustomerName      string `json:"customerName" form:"customerName" gorm:"comment:客户名"`             // 客户名
	CustomerPhoneData string `json:"customerPhoneData" form:"customerPhoneData" gorm:"comment:客户手机号"` // 客户手机号

	// 这两个字段是"冗余存的外键"：为了避免查列表时每行都去 join 用户表，
	// 直接把创建人的 ID 和角色 ID 写死在客户记录里。
	SysUserID          uint `json:"sysUserId" form:"sysUserId" gorm:"comment:管理ID"`                     // 管理ID
	SysUserAuthorityID uint `json:"sysUserAuthorityID" form:"sysUserAuthorityID" gorm:"comment:管理角色ID"` // 管理角色ID

	// [Go] 这个字段的类型是另一个包的结构体。它表达的是"多对一"关联：
	// 一个客户属于一个系统用户。
	//
	// [注意] 它默认是"零值"的，查询时不会自动填充。想拿到真实数据，
	// 必须在查询时显式 Preload：db.Preload("SysUser").Find(&list)
	// 不 Preload 的话 SysUser 里全是空字符串，前端会显示空——又一个静默失败点。
	SysUser system.SysUser `json:"sysUser" form:"sysUser" gorm:"comment:管理详情"` // 管理详情
}
