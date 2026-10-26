// ============================================================================
// 【文件导读】业务表迁移钩子 —— 二次开发的第一个必改文件。
//
// 这是框架给你预留的"空插槽"。系统表（用户、菜单、权限…）的迁移写在
// initialize/gorm.go 的 RegisterTables() 里，而**你自己的业务表必须写在这里**。
//
// 不改这里会发生什么？
//
//	代码能编译、服务能启动、接口能访问，但一查数据库就报
//	"Table 'xxx' doesn't exist" —— 因为根本没人帮你建表。
//
// 【本文件涉及的 Go 语法】
//   - 可变参数 ...interface{} 与"零参数调用"
//   - 错误向上抛（不吞错）
//
// ============================================================================
package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	// [注册点 4/5] 导入你的业务模型包
	"github.com/flipped-aurora/gin-vue-admin/server/model/customer"
	"github.com/flipped-aurora/gin-vue-admin/server/model/demo"
	"github.com/flipped-aurora/gin-vue-admin/server/model/inspectionrecord"
)

func bizModel() error {
	db := global.GVA_DB

	// [Go] AutoMigrate 的签名是 AutoMigrate(dst ...interface{}) error，
	// 可变参数允许传 0 个、1 个或多个。所以现在这样调用是合法的（什么也不做）。
	//
	// 用法：把你的模型实例的指针按顺序传进去，例如
	//
	//	err := db.AutoMigrate(
	//	    &demo.Book{},
	//	    &demo.Order{},
	//	)
	//
	// [注意] AutoMigrate 只做"增量加"：会建表、加字段、加索引，
	// 但**不会删除**已存在但模型里没有的字段，也不会改字段类型。
	// 想删列或改类型，必须自己写 SQL 或引入 migration 工具（如 goose、atlas）。
	//
	// [注意] 传指针 &demo.Book{} 而不是 demo.Book{}。
	// 传值也能跑，但 GORM 需要通过指针反射拿到类型信息，指针更规范。
	err := db.AutoMigrate(
		&demo.Book{},                         // 图书示例表 demo_books
		&inspectionrecord.InspectionRecord{}, // 巡检记录表 inspection_records
		&customer.Customer{},                 // 客户表 customers
	)
	if err != nil {
		// [Go] 这里没有 log 也没有 panic，而是把错误返回给调用方。
		// 调用方是 initialize/gorm.go 的 RegisterTables()，它拿到 err 后会
		// 记录日志并 os.Exit(1)。职责分离：这里只负责"做事并报告结果"。
		return err
	}
	return nil
}
