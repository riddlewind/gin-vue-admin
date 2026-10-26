package customer

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	CustomerRouter
}

var customerInfoApi = api.ApiGroupApp.CustomerApiGroup.CustomerApi
