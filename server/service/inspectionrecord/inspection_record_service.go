// ============================================================================
// 【文件导读】第三步：Service 层 —— 巡检记录的业务逻辑。
//
// 铁律（AGENT.MD）：**这一层绝对不能出现 gin.Context**。
// 只收纯数据、只返回数据 + error，这样定时任务、消息队列、CLI 都能复用它。
//
// 【本文件涉及的 Go 语法】
//   - 结构体持有一个"依赖"（DB 字段）—— 最朴素的依赖注入
//   - 指针接收者 (s *InspectionRecordService)
//   - 命名返回值 + 裸 return
//   - 可变参数 time.Parse 的多布局尝试（循环 + 提前 return）
//   - 指针与 nil 判断（"有没有传这个条件"）
//
// 【本文件修正了原始需求里的 3 处会出错的地方】
//  1. s.DB 恒为 nil —— 加了 db() 兜底，见下方注释
//  2. Find(&list).Limit(...) 顺序反了 —— Limit/Offset 必须在 Find 之前
//  3. Updates(结构体) 跳过零值 —— 必须 Select 指定列，否则清空备注会静默失败
//
// ============================================================================
package inspectionrecord

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/inspectionrecord"
	"gorm.io/gorm"
)

// ErrInvalidInspectionTime 时间格式错误的哨兵错误（sentinel error）
//
// [Go] 这是 Go 里区分错误类型的标准做法：定义一个包级变量当"标记"，
// 上层用 errors.Is(err, ErrInvalidInspectionTime) 判断，而不是比对错误字符串。
//
// 为什么需要它？因为 API 层要做区分：
//   - 这种错误是用户传参问题，可以原样透给前端（"时间格式不对"）
//   - 数据库错误不能透（避免泄露表结构、SQL 片段）
//
// [注意] 用 errors.New 而不是 fmt.Errorf 定义，保证它是稳定可比较的同一个值。
var ErrInvalidInspectionTime = errors.New("巡检时间格式不正确，请使用 2006-01-02 15:04:05 或 2006-01-02")

// InspectionRecordService 巡检记录服务
//
// [Go] 这里和项目里其它 Service（比如 demo.BookService 那种空结构体）不一样：
// 它显式持有一个 *gorm.DB 字段，也就是把"数据库连接"当成构造参数注入进来。
//
// 好处：可以塞不同的 DB 进来（主库/从库、带事务的 tx、测试用的内存库），
// 不用全局变量也能换数据源。
//
// [注意] 但光有字段是不够的 —— 上层 service/enter.go 的 NewServiceGroup() 调用的是
// NewInspectionRecordService()，它刻意**不**注入 DB（项目统一走 global.GVA_DB）。
// 也就是说自动装配出来的实例里 DB 永远是 nil。
// 所以下面每个方法都通过 db() 取连接，而不是直接写 s.DB.xxx，
// 否则第一次调用就是 nil pointer panic。
type InspectionRecordService struct {
	DB *gorm.DB
}

// db 取出本次操作要用的数据库连接
//
// [Go] 这个方法就是"FIELD 为空则回退到全局单例"的兜底写法：
//   - 有人显式 new(InspectionRecordService{DB: tx}) 时，用注入的那个（比如事务）；
//   - 项目统一用 ServiceGroupApp 自动装配时，DB 是 nil，回退到 global.GVA_DB。
//
// 只写 s.DB 的话，走 ServiceGroupApp 这条路 100% 会崩。
func (s *InspectionRecordService) db() *gorm.DB {
	if s.DB != nil {
		return s.DB
	}
	// [Go] 每次调用 db() 都返回同一个 *gorm.DB 原型，
	// GORM 的链式方法（Where/Order/...）都是"返回新实例"，不会污染这个原型。
	return global.GVA_DB
}

// CreateInspectionRecord 新增一条巡检记录
//
// [Go] 参数用指针 *InspectionRecord 而不是值：
// GORM 插入成功后要把自增主键**回填**到对象里，传值的话回填的是副本，
// 调用方拿到的 ID 永远是 0（拿不到新记录的 ID 就没法接着做后续操作）。
//
// [注意] 这里不校验业务字段（必填、长度等）。校验属于参数层（API 层）的职责，
// Service 层假设"进来的一定是合法的"。如果想让 Service 也能被内部任务安全调用，
// 可以在这里补最基础的守卫，但不要写"给前端看的提示文案"——那一层不该知道 HTTP。
func (s *InspectionRecordService) CreateInspectionRecord(inspectionRecord *inspectionrecord.InspectionRecord) error {
	if inspectionRecord == nil {
		return errors.New("巡检记录不能为空")
	}
	return s.db().Create(inspectionRecord).Error
}

// DeleteInspectionRecord 按 ID 删除（软删除）
//
// [Go] Delete(&Model{}, ID) 的第二个参数是"主键条件"。
// ID 传 int 也可以（GORM 内部是 interface{}），但表主键是 uint，
// 调用方如果传负数会被当成"主键为 -1"去查，删不到东西也不报错。
//
// [注意] 因为 InspectionRecord 内嵌了 global.GVA_MODEL（含 DeletedAt），
// 这里执行的是 UPDATE inspection_records SET deleted_at=now() WHERE id=?
// 数据还在表里，只是后续查询被自动过滤。
// 真要物理删除：s.db().Unscoped().Delete(&inspectionrecord.InspectionRecord{}, ID)
func (s *InspectionRecordService) DeleteInspectionRecord(ID int) error {
	if ID <= 0 {
		return errors.New("巡检记录 ID 不合法")
	}
	return s.db().Delete(&inspectionrecord.InspectionRecord{}, ID).Error
}

// GetInspectionRecord 按 ID 查单条
//
// [Go] 返回值是 (值, error) 两个。查不到时 First 会返回 gorm.ErrRecordNotFound，
// 调用方要用 errors.Is(err, gorm.ErrRecordNotFound) 判断，而不是比字符串。
//
// [注意] 这里用的是 First(&result, ID)，GORM 会自动拼上 "ORDER BY id LIMIT 1"，
// 并且自动带 deleted_at IS NULL 条件。如果记录已被软删除，这里会返回 ErrRecordNotFound。
func (s *InspectionRecordService) GetInspectionRecord(ID int) (result inspectionrecord.InspectionRecord, err error) {
	if ID <= 0 {
		return result, errors.New("巡检记录 ID 不合法")
	}
	err = s.db().First(&result, ID).Error
	return result, err
}

// GetInspectionRecordInfo 分页 + 条件查询
//
// [Go] 命名返回值 (list, total, err)：出错时直接裸 return，
// 已赋值的 total / list 会按当前状态返回，不会因为中途 return 而丢失。
func (s *InspectionRecordService) GetInspectionRecordInfo(info *inspectionrecord.InspectRecordInfo) (list []inspectionrecord.InspectionRecord, total int64, err error) {
	if info == nil {
		return nil, 0, errors.New("查询条件不能为空")
	}

	// [Go] db 是"查询构造器"，不是已经发出的 SQL。
	// 每一步 db = db.Where(...) 都在往同一个构造器上叠加条件，
	// 直到 Find / Count 这类"终结方法"被调用才会真正打到数据库。
	db := s.db().Model(&inspectionrecord.InspectionRecord{})

	// [注意] LIKE 的 % 必须自己加，GORM 不会补。
	// 只写 db.Where("device_name LIKE ?", info.DeviceName) 就变成了精确匹配。
	// 用的是参数化占位符 ?，所以拼进去的 % 不会引发 SQL 注入。
	if keyword := strings.TrimSpace(info.DeviceName); keyword != "" {
		db = db.Where("device_name LIKE ?", "%"+keyword+"%")
	}
	if keyword := strings.TrimSpace(info.Inspector); keyword != "" {
		db = db.Where("inspector LIKE ?", "%"+keyword+"%")
	}
	// 类型和结果是枚举值，用等值匹配（走索引）；模糊匹配会让索引失效。
	if keyword := strings.TrimSpace(info.InspectionType); keyword != "" {
		db = db.Where("inspection_type = ?", keyword)
	}
	if keyword := strings.TrimSpace(info.InspectionResult); keyword != "" {
		db = db.Where("inspection_result = ?", keyword)
	}

	// [注意] 时间范围是两个字符串，必须转成 time.Time 再比较。
	// 直接用字符串比："2026-10-23" <= "2026-10-23 18:00:00" 在 MySQL 里
	// 会把前者当 '2026-10-23 00:00:00' 比较，看着没问题，
	// 但一旦前端传来 "2026/10/23" 这种格式，MySQL 会静默按 0 处理，
	// 既不报错也查不出数据 —— 最难排查的那类 bug。
	start, end, err := parseInspectionRange(info.InspectionTimeStart, info.InspectionTimeEnd)
	if err != nil {
		return nil, 0, err
	}
	if start != nil {
		db = db.Where("inspection_time >= ?", *start)
	}
	if end != nil {
		db = db.Where("inspection_time <= ?", *end)
	}

	// 先查总数：前端分页组件需要 total 才能算总页数。
	// [注意] Count 会临时改写 SELECT 子句为 count(*)，执行完自动还原，
	// 所以下面可以继续用同一个 db 查列表。
	if err = db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// [踩坑实录] 原需求写的是：
	//     db.Order(...).Find(&list).Limit(info.PageSize).Offset(...)
	// 这是错的。GORM v2 里 Find 是**终结方法**，调用它的那一刻 SQL 就已经发出去了，
	// 后面再挂 Limit / Offset 只会改一个已经执行完的构造器，完全没有任何效果 ——
	// 结果就是"分页参数传了但永远返回全表"，而且不报错。
	//
	// 正确顺序：先 Order / Offset / Limit 把构造器装配好，最后一步才 Find。
	page, pageSize := normalizePage(info.Page, info.PageSize)
	err = db.Order("inspection_time DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

// UpdateInspectionRecord 更新巡检记录
//
// [踩坑实录 1] 不能用 Save：
//
//	s.db().Save(inspectionRecord)
//
// Save 按主键做**全字段覆盖**，而前端传来的 JSON 里没有 CreatedAt，
// 它的零值会被当成"要写入的新值"，往 datetime 列写 0000-00-00，
// MySQL 严格模式直接报 Error 1292 Incorrect datetime value。
//
// [踩坑实录 2] 也不能只写 Updates(结构体)：
//
//	s.db().Model(&X{}).Where("id = ?", id).Updates(inspectionRecord)
//
// Updates 传结构体时**默认跳过所有零值字段**，
// 所以"把备注清空"（空字符串）、"把处理建议删掉"这类操作会静默失败 ——
// 接口返回成功，数据库里一个字没变。这两个坑方向正好相反。
//
// 正确姿势：Select 显式列出允许更新的列 + Updates。
// Select 里的列即使传的是零值也会被强制写入；
// 没列出来的列（ID / CreatedAt / UpdatedAt / DeletedAt）一个字节都不会动。
//
// [注意] UpdatedAt 不用自己维护，GORM 在 Updates 时会自动写 now()。
func (s *InspectionRecordService) UpdateInspectionRecord(inspectionRecord *inspectionrecord.InspectionRecord) error {
	if inspectionRecord == nil {
		return errors.New("巡检记录不能为空")
	}
	// [注意] 必须挡住 ID=0：Where("id = ?", 0) 会更新 0 行，
	// 而 GORM 不把"影响行数为 0"当错误，接口会假装成功。
	if inspectionRecord.ID == 0 {
		return errors.New("更新时必须提供巡检记录 ID")
	}
	// [注意] Select 里列的列**一定会被写进 SQL**，哪怕结构体里它是零值。
	// 这就是本函数刻意不用 db.Save 的原因：Save 会把 CreatedAt 一起覆盖成零值。
	// 但反过来说，Select 也得小心——如果调用方只传了部分字段（比如只想改
	// inspection_result），而 inspection_time 在结构体里还是零值 time.Time{}
	// （Go 零值是 0001-01-01，GORM 拼出来是 '0000-00-00'），
	// MySQL 严格模式会直接报 1292 Incorrect datetime value。
	// 所以时间列单独判断：传了才更新，没传就跳过（保持库里原值）。
	columns := []string{
		"device_name",
		"device_code",
		"device_location",
		"inspector",
		"inspection_type",
		"inspection_result",
		"problem_desc",
		"photos",
		"handling_suggestion",
		"remark",
	}
	if !inspectionRecord.InspectionTime.IsZero() {
		columns = append(columns, "inspection_time")
	}
	return s.db().Model(&inspectionrecord.InspectionRecord{}).
		Where("id = ?", inspectionRecord.ID).
		Select(columns).
		Updates(inspectionRecord).Error
}

// ============================================================================
// 内部辅助函数（小写开头 = 只在本包内可见）
// ============================================================================

// inspectionTimeLayouts 允许的时间格式，按"越精确越靠前"排列。
//
// [Go] 切片字面量 + 包级变量：这里只是一组常量字符串，
// 不是可变状态，所以没必要每次调用都重新构造。
var inspectionTimeLayouts = []string{
	// [注意] Go 的时间布局是"用 2006-01-02 15:04:05 这个参考时间的**字面样子**当模板"，
	// 不是像 C 语言那样写 %Y-%m-%d。这是 Go 最反直觉的设计之一：
	// 想解析 "2026-10-23 09:30:00"，模板就写成 "2006-01-02 15:04:05"。
	time.DateTime, // 2006-01-02 15:04:05
	"2006-01-02T15:04:05",
	time.RFC3339,  // 2026-10-23T09:30:00+08:00
	time.DateOnly, // 2006-01-02（只有日期）
}

// ParseInspectionTime 把一个时间字符串按几种常见格式解析
//
// [Go] 返回三个值，第二个 bool 表示"这个字符串里只有日期，没有时分秒"。
// 这个信息很关键：只给日期时，结束时间要补到当天 23:59:59，
// 否则 "结束日期 = 2026-10-23" 会被当成 00:00:00，把当天的数据全漏掉。
//
// [注意] 这个函数是**导出**的（大写开头），因为 API 层要用它：
// 标准库 encoding/json 解析 time.Time 时只认 RFC3339，
// 前端传 "2026-10-23 09:30:00"（中间是空格）会直接报
// "cannot parse \" 09:30:00\" as \"T\""，所以 API 层必须自己接管这一步。
func ParseInspectionTime(raw string) (t time.Time, dateOnly bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false, nil
	}
	for _, layout := range inspectionTimeLayouts {
		// [Go] time.Parse(layout, value)：第一个参数是模板，第二个才是待解析的字符串。
		// 顺序反了是新手最常见的手误，报错信息还特别费解。
		parsed, parseErr := time.ParseInLocation(layout, raw, time.Local)
		if parseErr == nil {
			return parsed, layout == time.DateOnly, nil
		}
		// [注意] 这里不能直接 return err。循环本身就是"挨个试"，
		// 单个格式失败是预期内的，全部失败才算真错。
	}
	// [Go] fmt.Errorf 的 %w 是"包装"（wrap）：把哨兵错误包进去，
	// 外层用 errors.Is 仍能识别出 ErrInvalidInspectionTime，
	// 同时又能带上"用户到底传了什么"这种细节。
	return time.Time{}, false, fmt.Errorf("%w（收到的是 %q）", ErrInvalidInspectionTime, raw)
}

// parseInspectionRange 解析 [start, end] 闭区间
//
// [Go] 返回 *time.Time（指针）而不是 time.Time，因为"没传这个条件"和
// "传了零值时间"必须能区分开。指针的 nil 天然能表达"没有"，
// 而 time.Time 的零值是 0001-01-01，没法当"没传"用。
func parseInspectionRange(rawStart, rawEnd string) (start, end *time.Time, err error) {
	if t, _, parseErr := ParseInspectionTime(rawStart); parseErr != nil {
		return nil, nil, parseErr
	} else if !t.IsZero() {
		start = &t
	}

	if t, dateOnly, parseErr := ParseInspectionTime(rawEnd); parseErr != nil {
		return nil, nil, parseErr
	} else if !t.IsZero() {
		// 只给了日期时，把右端点补到当天最后一刻，做成闭区间 [00:00:00, 23:59:59.999999999]。
		// 不加这一步的话，前端选"截至 10-23"会查不到 10-23 当天的任何记录。
		if dateOnly {
			t = t.Add(24*time.Hour - time.Nanosecond)
		}
		end = &t
	}
	return start, end, nil
}

// normalizePage 修正非法分页参数
//
// [Go] 这里没有直接复用 request.PageInfo.Paginate()，原因是
// InspectRecordInfo 自带 Page / PageSize 两个字段（没有内嵌 PageInfo），
// 类型对不上，不能直接传。与其为了复用去改模型结构，不如就地写清楚。
func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	switch {
	case pageSize <= 0:
		pageSize = 10
	case pageSize > 100: // 防止前端传 999999 把数据库拖死
		pageSize = 100
	}
	return page, pageSize
}
