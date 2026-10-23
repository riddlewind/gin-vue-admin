package inspectionrecord

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// InspectionRecord 设备巡检记录表
//
// [Go] 命名约定：结构体名大驼峰单数（InspectionRecord），表名蛇形复数（inspection_records）。
//
// [注意] 这里内嵌的是 global.GVA_MODEL，不是 "server/model" 包里的 Model。
// server/model 目录下没有任何 .go 文件，只有 common/demo/example/system 四个子目录，
// 所以不存在 model.Model 这个类型。基础字段（ID/CreatedAt/UpdatedAt/DeletedAt）
// 全部来自 global/model.go 的 GVA_MODEL。
type InspectionRecord struct {
	// [Go] 嵌入 GVA_MODEL 后自动获得四列：ID、CreatedAt、UpdatedAt、DeletedAt。
	// 可以直接 rec.ID 访问（字段提升），建表时 GORM 会把它们平铺进 inspection_records。
	global.GVA_MODEL

	// [注意] gorm tag 里的分号是选项分隔符，不是 SQL 语句分隔符。
	// 用逗号分隔 GORM 不报错，但会静默忽略后面所有选项——很阴的坑。
	// column:xxx 显式指定列名，避免 GORM 把 DeviceName 推导成 device_name（虽然结果相同，
	// 但显式写出来更利于和 DBA、SQL 脚本对齐）。
	DeviceName string `json:"deviceName" form:"deviceName" gorm:"column:device_name;type:varchar(100);comment:设备名称;"`

	DeviceCode string `json:"deviceCode" form:"deviceCode" gorm:"column:device_code;type:varchar(50);comment:设备编号;"`

	DeviceLocation string `json:"deviceLocation" form:"deviceLocation" gorm:"column:device_location;type:varchar(200);comment:设备位置;"`

	Inspector string `json:"inspector" form:"inspector" gorm:"column:inspector;type:varchar(100);comment:巡检人;"`

	// [注意] 这是本表唯一必须 import "time" 的字段，同时也是查询时最容易踩坑的字段：
	// 前端传来的是字符串（如 "2026-10-23 09:00:00"），不能直接塞进 time.Time，
	// 必须先在 Service 层用 time.Parse 转换，否则 GORM 会报类型错误。
	InspectionTime time.Time `json:"inspectionTime" form:"inspectionTime" gorm:"column:inspection_time;type:datetime;comment:巡检时间;"`

	InspectionType string `json:"inspectionType" form:"inspectionType" gorm:"column:inspection_type;type:varchar(50);comment:巡检类型;"`

	InspectionResult string `json:"inspectionResult" form:"inspectionResult" gorm:"column:inspection_result;type:varchar(50);comment:巡检结果;"`

	// [注意] type:text 不支持默认值也不支持索引，适合长文本。
	// MySQL 的 text 上限 64KB，正常的问题描述足够用。
	ProblemDesc string `json:"problemDesc" form:"problemDesc" gorm:"column:problem_desc;type:text;comment:问题描述;"`

	// [注意] 照片以 JSON 数组字符串形式存在一列里（如 ["/upload/a.jpg","/upload/b.jpg"]）。
	// 这种写法读写简单，但无法对单张照片建索引、也无法按照片查询。
	// 如果以后要按图片检索，得拆成独立子表（一对多）。
	Photos string `json:"photos" form:"photos" gorm:"column:photos;type:text;comment:现场照片JSON数组;"`

	HandlingSuggestion string `json:"handlingSuggestion" form:"handlingSuggestion" gorm:"column:handling_suggestion;type:varchar(1000);comment:处理建议;"`

	Remark string `json:"remark" form:"remark" gorm:"column:remark;type:varchar(500);comment:备注;"`
}

// TableName 自定义表名
//
// [Go] 值接收者 func (InspectionRecord)，因为方法体不碰任何字段。
//
// [注意] 不写这个方法的话，GORM 会按"结构体名转蛇形复数"自动推导成
// inspection_records——恰好和这里想要的一致。显式写出来是为了防止以后
// 改结构体名（比如改成 InspectRecord）时表名悄悄跟着变，导致老数据读不到。
func (InspectionRecord) TableName() string {
	return "inspection_records"
}
