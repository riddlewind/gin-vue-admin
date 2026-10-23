// ============================================================================
// 【文件导读】业务逻辑层 —— 客户模块。
//
// Service 层是整个后端的核心。它的铁律（AGENT.MD 里明确写了）：
//
//	"只承载业务逻辑，不处理 HTTP 语义，不要依赖 gin.Context"
//
// 为什么不能碰 gin.Context？因为这一层要能被复用：
// 今天是被 HTTP 接口调用，明天可能被定时任务、gRPC、MCP 工具调用。
// 一旦把 *gin.Context 当参数传进来，这层代码就永远绑死在 HTTP 上了。
//
// 对照记忆：
//
//	API 层管"怎么收、怎么回"        -> 可以有 gin.Context
//	Service 层管"业务到底做什么"    -> 绝对不能有 gin.Context
//	Model 层管"数据长什么样"        -> 不关心谁在调用
//
// 【本文件涉及的 Go 语法】
//   - 包级变量做单例：var CustomerServiceApp = new(CustomerService)
//   - 命名返回值 + 裸 return
//   - 指针接收者 vs 值接收者（这里两种都有，注意对比）
//   - for range 遍历 + append 构建切片
//   - GORM 链式调用与 Preload
//
// ============================================================================
package example

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	// [Go] 给导入的包起别名 systemService。
	// 因为本文件已经导入了 model/system（包名也叫 system），
	// 同一个文件里不能有两个同名的包标识符，所以必须起别名区分。
	// 这是 Go 项目里非常常见的处理方式。
	systemService "github.com/flipped-aurora/gin-vue-admin/server/service/system"
)

// CustomerService 客户业务服务
//
// [Go] 空结构体承载方法集合，不存状态。所有状态都来自 global.GVA_DB。
type CustomerService struct{}

// [Go] 包级变量 + new()：在包被初始化时创建唯一实例。
// 这就是这个项目的"依赖注入"——实际上是"全局单例 + 包级变量"的土办法。
// 上层通过 service.ServiceGroupApp.ExampleServiceGroup.CustomerService 拿到它，
// 永远只有一份，不会重复分配。
var CustomerServiceApp = new(CustomerService)

//@author: [piexlmax](https://github.com/piexlmax)
//@function: CreateExaCustomer
//@description: 创建客户
//@param: e model.ExaCustomer
//@return: err error

// CreateExaCustomer 创建客户
//
// [Go] 注意返回值写法：`(err error)` 是"命名返回值"。
// 声明了名字的返回值相当于函数开头就自动 `var err error`，
// 所以函数体最后可以直接写 `return`（叫"裸返回"），不用写 `return err`。
//
// [注意] 裸返回在短函数里还行，长函数里会让读者搞不清到底返回了什么。
// 社区普遍认为函数超过 20 行就不该用裸返回。这个项目里到处都是，
// 你抄的时候可以改成显式 `return err`，更清楚。
func (exa *CustomerService) CreateExaCustomer(e example.ExaCustomer) (err error) {
	// [Go] global.GVA_DB 是全局唯一的 *gorm.DB 连接池。
	// Create 的返回值本身是 *gorm.DB，所以要用 .Error 取错误。
	//
	// [Go] &e 取地址：GORM 需要指针才能把数据库自增生成的 ID 回填到 e 里。
	// 传值的话，创建成功后 e.ID 依然是 0。
	err = global.GVA_DB.Create(&e).Error
	return err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: DeleteFileChunk
//@description: 删除客户
//@param: e model.ExaCustomer
//@return: err error

// DeleteExaCustomer 删除客户
//
// [注意] 上面那行 @function: DeleteFileChunk 是**错的**——明显是从文件块上传那段
// 复制过来忘了改。这是开源项目里常见的注释腐化，也是给你的提醒：
// 读代码要相信代码本身，不要相信注释。遇到注释和行为矛盾时，以代码为准。
func (exa *CustomerService) DeleteExaCustomer(e example.ExaCustomer) (err error) {
	// [注意] 这是"软删除"：因为模型里嵌了 GVA_MODEL，
	// GORM 不会真的 DELETE，而是 UPDATE deleted_at = now()。
	// 记录还在表里，只是之后的查询会自动过滤掉。
	// 如果确实要物理删除，得用 Unscoped().Delete(&e)。
	err = global.GVA_DB.Delete(&e).Error
	return err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: UpdateExaCustomer
//@description: 更新客户
//@param: e *model.ExaCustomer
//@return: err error

// UpdateExaCustomer 更新客户
//
// [Go] 这里参数是 *example.ExaCustomer（指针），而上面 Create 是值类型。
// 差别在于：Save 需要拿到带主键的完整对象做全字段覆盖更新，
// 传指针避免了大结构体的拷贝，也保证 GORM 能正确识别主键。
func (exa *CustomerService) UpdateExaCustomer(e *example.ExaCustomer) (err error) {
	// [注意] Save 是"全字段更新"：即使某个字段你传的是零值（空字符串、0），
	// 也会被写进数据库。想只更新部分字段，应该用 Updates(map) 或 Select 指定字段。
	// 这是新手最容易踩的坑——只传了要改的两个字段，结果把其他字段全清空了。
	err = global.GVA_DB.Save(e).Error
	return err
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: GetExaCustomer
//@description: 获取客户信息
//@param: id uint
//@return: customer model.ExaCustomer, err error

// GetExaCustomer 按 ID 查单个客户
func (exa *CustomerService) GetExaCustomer(id uint) (customer example.ExaCustomer, err error) {
	// [Go] Where("id = ?", id) 里的 ? 是占位符，GORM 会用参数化查询替换它。
	// [注意] 千万不要用字符串拼接写 SQL（fmt.Sprintf("id = %d", id)），
	// 那是 SQL 注入的经典入口。用 ? 占位是底线。
	//
	// [Go] err = ...First(&customer).Error 然后裸 return，
	// 是因为命名返回值 customer 已经被 First 填充过了。
	err = global.GVA_DB.Where("id = ?", id).First(&customer).Error
	return
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: GetCustomerInfoList
//@description: 分页获取客户列表
//@param: sysUserAuthorityID string, info request.PageInfo
//@return: list interface{}, total int64, err error

// GetCustomerInfoList 分页查询客户列表（带数据权限过滤）
//
// [Go] 重点看返回值 `(list interface{}, total int64, err error)`：
// list 用 interface{} 是因为不同模块返回的切片类型不同，
// 上层 API 拿到后原样塞进 response 里，不需要关心具体类型。
// 这是"用空接口换取通用性"的典型场景（代价是丢失类型安全）。
func (exa *CustomerService) GetCustomerInfoList(sysUserAuthorityID uint, info request.PageInfo) (list interface{}, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	// [Go] 这里的 db 是一个"查询构造器"，还没真正执行 SQL。
	// Model(&example.ExaCustomer{}) 只告诉 GORM "接下来操作这张表"，
	// 传空结构体指针是为了取表名，不需要有值。
	db := global.GVA_DB.Model(&example.ExaCustomer{})

	// ---- 下面是这个方法的业务核心：数据权限过滤 ----
	// 逻辑：当前登录人的角色，只能看到"被授权可见的那几个角色"创建的数据。
	// 你在「角色管理 -> 数据权限」里配的就是这个 DataAuthorityId。
	var a system.SysAuthority
	a.AuthorityId = sysUserAuthorityID
	// [Go] 直接调用别的模块的 Service：本模块并不关心权限怎么算的，
	// 复用 systemService 就好。这是分层带来的好处——跨模块复用业务逻辑。
	auth, err := systemService.AuthorityServiceApp.GetAuthorityInfo(a)
	if err != nil {
		// [Go] 裸 return 会原样返回当前的 list(nil)、total(0)、err。
		// 出错时直接返回到此为止，不会继续往下查。
		return
	}

	// [Go] var dataId []uint 声明一个 nil 切片（不是空切片，长度 0 容量 0）。
	// append 会自动处理扩容，所以不用预先 make。
	// [注意] nil 切片可以直接 append，但对 nil map 赋值会 panic。这两者别搞混。
	var dataId []uint
	for _, v := range auth.DataAuthorityId {
		// [Go] for range 返回索引和值。这里不需要索引，所以第一个用 _ 丢弃。
		// [注意] range 拿到的是元素的**副本**，想修改原切片必须用索引下标赋值。
		dataId = append(dataId, v.AuthorityId)
	}

	var CustomerList []example.ExaCustomer

	// [注意] 这里执行了两次查询：
	//   1. Count(&total) 只查总数，不取数据；
	//   2. Find(&CustomerList) 取当前页的数据。
	// 两次查询都带着同样的权限过滤条件。
	//
	// [Go] 关键点：db 是一个"可复用的构造器"，在它上面继续 .Where() 不会互相污染，
	// 因为 GORM 的链式方法每次都会派生新的语句对象。
	// 但如果你用了 Session() 或者手动接管，就要小心条件残留。
	err = db.Where("sys_user_authority_id in ?", dataId).Count(&total).Error
	if err != nil {
		return CustomerList, total, err
	} else {
		// [Go] "in ?" 是 GORM 对 IN 子句的简写，dataId 这个切片会被自动展开成
		// IN (1,2,3)。传切片时不需要自己拼括号，也不需要逗号。
		//
		// [注意] Preload("SysUser") 是必须的！
		// ExaCustomer.SysUser 字段平时是零值，只有 Preload 才会真的去查 sys_users 表
		// 并填充进来。少了这一句，接口能正常返回，但前端看到的 sysUser 是空的。
		err = db.Limit(limit).Offset(offset).Preload("SysUser").Where("sys_user_authority_id in ?", dataId).Find(&CustomerList).Error
	}
	return CustomerList, total, err
}
