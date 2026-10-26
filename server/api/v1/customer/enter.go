package customer

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	CustomerApi
}

var customerInfoService = service.ServiceGroupApp.CustomerServiceGroup.CustomerService
