// ============================================================================
// 【文件导读】第二步：Service 层 —— 业务逻辑。
//
// 铁律（AGENT.MD 明确规定）：**这里绝对不能出现 gin.Context**。
// 参数只收纯数据，返回只给数据和 error。这样这层才能被定时任务、
// 消息队列、gRPC、MCP 工具复用，而不是只能被 HTTP 调。
//
// 【本文件涉及的 Go 语法】
//   - 指针接收者（要拿到调用方传进来的对象，并让 GORM 回填主键）
//   - 命名返回值 + 裸 return
//   - GORM Scopes 复用分页逻辑
//   - 字符串拼接构造 LIKE 参数
//
// ============================================================================
package demo

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	// [Go] 这里导入的是 model/demo，它的包名也叫 demo，
	// 和本文件所在的 service/demo 包同名——这是允许的。
	// 包自身的名字不会在包里形成标识符，所以不会冲突。
	// 下层代码里写 demo.Book 指的就是 model/demo 包里的类型。
	"github.com/flipped-aurora/gin-vue-admin/server/model/demo"
	demoReq "github.com/flipped-aurora/gin-vue-admin/server/model/demo/request"
)

// BookService 图书业务服务
//
// [Go] 空结构体：不存状态。数据库句柄来自 global.GVA_DB 这个全局单例。
type BookService struct{}

// CreateBook 新增图书
//
// [Go] 参数用 *demo.Book（指针）而不是 demo.Book（值）：
//   - GORM 创建成功后要把数据库自增的 ID **回填**到对象里；
//   - 传值的话，回填的是副本，调用方拿到的 b.ID 永远是 0。
func (s *BookService) CreateBook(b *demo.Book) error {
	return global.GVA_DB.Create(b).Error
}

// DeleteBook 删除图书（软删除）
//
// [注意] 因为 Book 嵌入了 GVA_MODEL，所以这里是软删除：
// GORM 实际执行的是 UPDATE demo_books SET deleted_at=now() WHERE id=?
// 记录还在表里，只是查询时会被自动过滤。
// 真要物理删除得写 Unscoped().Delete(...)。
//
// [Go] Delete(&demo.Book{}, id) 第二个参数是主键条件。
// 这种写法比先查再删少一次查询，但要确认 id 来自可信来源（前端传的可能造假，
// 不过删自己的数据问题不大；涉及越权时要先查归属）。
func (s *BookService) DeleteBook(id uint) error {
	return global.GVA_DB.Delete(&demo.Book{}, id).Error
}

// UpdateBook 更新图书
//
// [踩坑实录] 这里我一开始写的是 global.GVA_DB.Save(b)，结果接口报：
//
//	Error 1292: Incorrect datetime value: '0000-00-00' for column 'created_at'
//
// 原因：Save 会按主键做**全字段覆盖**，而前端传来的 JSON 里没有 CreatedAt，
// 它的零值被当成"要写入的新值"，于是往 datetime 列写了 0000-00-00，MySQL 严格模式直接拒绝。
//
// 正确做法：用 Select 显式列出"允许被更新的列"，再交给 Updates。
//   - Select 里的列即使传的是零值（比如 status=0 下架）也会被强制写入；
//   - 不列出来的列（ID / CreatedAt / DeletedAt）完全不受影响。
//
// [注意] 只写 Updates(b) 而不加 Select 是另一个坑：
// Updates 默认**跳过所有零值字段**，前端把状态从 1 改成 0 时静默失效。
// 这两个坑方向相反，所以"Select + Updates"才是既安全又能更新零值的组合。
func (s *BookService) UpdateBook(b *demo.Book) error {
	return global.GVA_DB.Model(&demo.Book{}).
		Where("id = ?", b.ID).
		Select("title", "author", "price", "stock", "status").
		Updates(b).Error
}

// GetBook 按 ID 查单条
func (s *BookService) GetBook(id uint) (book demo.Book, err error) {
	// [Go] Where 里用 ? 占位符，由驱动做参数化，杜绝 SQL 注入。
	err = global.GVA_DB.Where("id = ?", id).First(&book).Error
	return
}

// GetBookList 分页 + 按书名模糊搜索
//
// [Go] 返回值 (list []demo.Book, total int64, err error) 用了命名返回值，
// 所以下面出错时可以裸 return，三个值会按当前状态原样返回。
func (s *BookService) GetBookList(info demoReq.BookSearch) (list []demo.Book, total int64, err error) {
	db := global.GVA_DB.Model(&demo.Book{})

	// [Go] 这里是"链式条件按需叠加"的标准写法：
	// 每一步都把结果赋回 db，因为每个方法都返回新的 *gorm.DB。
	if info.Title != "" {
		// [注意] 必须自己加 %，GORM 不会帮你补。只写 info.Title 就变成精确匹配了。
		db = db.Where("title LIKE ?", "%"+info.Title+"%")
	}

	// 先查总数（分页组件需要），再查当前页数据。两次查询共用同一组过滤条件。
	if err = db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// [Go] Scopes 的用法：把"对查询的改造动作"当参数传进去。
	// info.Paginate() 返回的是 func(*gorm.DB) *gorm.DB（见 model/common/request/common.go），
	// 它在内部算好 offset/limit，还顺便修正了非法页码。
	err = db.Scopes(info.Paginate()).Order("id DESC").Find(&list).Error
	return list, total, err
}
