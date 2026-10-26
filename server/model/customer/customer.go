
// 自动生成模板Customer
package customer
import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// 客户 结构体  Customer
type Customer struct {
    global.GVA_MODEL
  FirstName  *string `json:"firstName" form:"firstName" gorm:"comment:名字;column:first_name;" binding:"required"`  //名字
  LastName  *string `json:"lastName" form:"lastName" gorm:"comment:姓氏;column:last_name;" binding:"required"`  //姓氏
  Address  *string `json:"address" form:"address" gorm:"comment:地址;column:address;" binding:"required"`  //地址
  // 电话用 string 而不是 int64：号码可能有前导零、区号、"+"号或分机号，
  // 存数字会把 010-8888 这类内容吃掉，变成 -8888。
  Telephone  *string `json:"telephone" form:"telephone" gorm:"index;comment:电话;column:telephone;type:varchar(32);" binding:"required"`  //电话
  // 原生成结果是 enum('')，即"一个候选值都没有的枚举"，任何非空值都插不进去。
  // 改成 varchar(32)，取值由前端下拉约束（男/女/未知、启用/停用）。
  Gender  string `json:"gender" form:"gender" gorm:"comment:性别;column:gender;type:varchar(32);" binding:"required"`  //性别
  Active  string `json:"active" form:"active" gorm:"comment:启用状态;column:active;type:varchar(32);" binding:"required"`  //启用状态
  Email  *string `json:"email" form:"email" gorm:"index;comment:邮箱;column:email;" binding:"required"`  //邮箱
}


// TableName 客户 Customer自定义表名 customers
func (Customer) TableName() string {
    return "customers"
}





