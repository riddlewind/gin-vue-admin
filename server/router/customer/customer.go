package customer

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type CustomerRouter struct {}

// InitCustomerRouter 初始化 客户 路由信息
func (s *CustomerRouter) InitCustomerRouter(Router *gin.RouterGroup,PublicRouter *gin.RouterGroup) {
	customerInfoRouter := Router.Group("customerInfo").Use(middleware.OperationRecord())
	customerInfoRouterWithoutRecord := Router.Group("customerInfo")
	customerInfoRouterWithoutAuth := PublicRouter.Group("customerInfo")
	{
		customerInfoRouter.POST("createCustomer", customerInfoApi.CreateCustomer)   // 新建客户
		customerInfoRouter.DELETE("deleteCustomer", customerInfoApi.DeleteCustomer) // 删除客户
		customerInfoRouter.DELETE("deleteCustomerByIds", customerInfoApi.DeleteCustomerByIds) // 批量删除客户
		customerInfoRouter.PUT("updateCustomer", customerInfoApi.UpdateCustomer)    // 更新客户
	}
	{
		customerInfoRouterWithoutRecord.GET("findCustomer", customerInfoApi.FindCustomer)        // 根据ID获取客户
		customerInfoRouterWithoutRecord.GET("getCustomerList", customerInfoApi.GetCustomerList)  // 获取客户列表
	}
	{
	    customerInfoRouterWithoutAuth.GET("getCustomerPublic", customerInfoApi.GetCustomerPublic)  // 客户开放接口
	}
}
