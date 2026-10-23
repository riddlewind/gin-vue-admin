package inspectionrecord

// InspectRecordSearch 巡检记录的查询条件
//
// [Go] 这个结构体只带 json/form tag，没有 gorm tag——因为它不对应数据库列，
// 只负责接住前端传来的筛选参数。GVA 里习惯把它叫 "xxxSearch"。
//
// [注意] 时间范围故意用 string 而不是 time.Time，原因是前端传的是
// "2026-10-23 09:00:00" 这种字符串，直接绑到 time.Time 上会因为格式
// 不匹配而报错。用 string 接下来，再在 Service 层用 time.Parse 转换，
// 出错时还能给出人话提示（"时间格式不对"），而不是一个 400 解析错误。
type InspectRecordSearch struct {
	DeviceName string `json:"deviceName" form:"deviceName"`

	Inspector string `json:"inspector" form:"inspector"`

	InspectionType string `json:"inspectionType" form:"inspectionType"`

	InspectionResult string `json:"inspectionResult" form:"inspectionResult"`

	// 时间范围，闭区间 [Start, End]
	InspectionTimeStart string `json:"inspectionTimeStart" form:"inspectionTimeStart"`

	InspectionTimeEnd string `json:"inspectionTimeEnd" form:"inspectionTimeEnd"`
}

// InspectRecordInfo 单条记录的请求体，同时承载分页参数
//
// [Go] 这里用的是"匿名嵌入"：只写类型名 InspectionRecord，不写字段名。
// 嵌入之后，InspectRecordInfo 可以直接 .DeviceName 访问巡检字段（字段提升），
// 序列化成 JSON 时也是平铺的，不会多一层嵌套——前端拿到的仍然是
// {"deviceName":"...", "page":1, "pageSize":10} 这种扁平结构。
//
// [注意] 不要把它写成命名字段（如 Record InspectionRecord），
// 那样 JSON 会多一层 {"Record":{...}}，和前端约定就对不上了。
type InspectRecordInfo struct {
	InspectionRecord

	// [注意] 下面两个时间范围字段是**必须补**的，不是重复定义。
	// InspectRecordInfo 内嵌的是 InspectionRecord（业务模型），
	// 而时间范围属于"查询条件"，原本只写在 InspectRecordSearch 里，
	// 两边是完全独立的结构体 —— 不补的话 Service 里 info.InspectionTimeStart
	// 会直接编译报 undefined。
	//
	// 为什么不在 Info 里再嵌一个 InspectRecordSearch？
	// 因为 Search 里的 DeviceName / Inspector / InspectionType / InspectionResult
	// 和 InspectionRecord 的同名字段会撞车，Go 不允许对"有歧义的提升字段"取值
	// （写 info.DeviceName 会编译失败）。所以只能显式补这两个没有冲突的字段。
	InspectionTimeStart string `json:"inspectionTimeStart" form:"inspectionTimeStart"`
	InspectionTimeEnd   string `json:"inspectionTimeEnd" form:"inspectionTimeEnd"`

	// [注意] GVA 的分页从 1 开始（page=1 是第一页），
	// 计算偏移量时是 (page-1)*pageSize，别漏了 -1。
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}
