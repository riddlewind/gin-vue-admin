
package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"time"
)

type CustomerSearch struct{
    CreatedAtRange []time.Time `json:"createdAtRange" form:"createdAtRange[]"`
      FirstName  *string `json:"firstName" form:"firstName"` 
      LastName  *string `json:"lastName" form:"lastName"` 
      Address  *string `json:"address" form:"address"` 
      // Telephone 跟着模型一起从 *int 改成 *string：号码含区号/前导零时，
      // 绑定到 int 会直接报错，搜索框里填 "010-8888" 就查不了。
      Telephone  *string `json:"telephone" form:"telephone"` 
      Gender  string `json:"gender" form:"gender"` 
      Active  string `json:"active" form:"active"` 
      Email  *string `json:"email" form:"email"` 
    request.PageInfo
}
