
package customer

import (
	"context"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/customer"
    customerReq "github.com/flipped-aurora/gin-vue-admin/server/model/customer/request"
	"gorm.io/gorm/clause"
)

type CustomerService struct {}
// CreateCustomer 创建客户记录
// Author [yourname](https://github.com/yourname)
func (customerInfoService *CustomerService) CreateCustomer(ctx context.Context, customerInfo *customer.Customer) (err error) {
	err = global.GVA_DB.WithContext(ctx).Create(customerInfo).Error
	return err
}

// DeleteCustomer 删除客户记录
// Author [yourname](https://github.com/yourname)
func (customerInfoService *CustomerService)DeleteCustomer(ctx context.Context, ID string) (err error) {
	err = global.GVA_DB.WithContext(ctx).Delete(&customer.Customer{},"id = ?",ID).Error
	return err
}

// DeleteCustomerByIds 批量删除客户记录
// Author [yourname](https://github.com/yourname)
func (customerInfoService *CustomerService)DeleteCustomerByIds(ctx context.Context, IDs []string) (err error) {
	err = global.GVA_DB.WithContext(ctx).Delete(&[]customer.Customer{},"id in ?",IDs).Error
	return err
}

// UpdateCustomer 更新客户记录
// Author [yourname](https://github.com/yourname)
func (customerInfoService *CustomerService)UpdateCustomer(ctx context.Context, customerInfo customer.Customer) (err error) {
	err = global.GVA_DB.WithContext(ctx).Model(&customer.Customer{}).Where("id = ?",customerInfo.ID).Updates(&customerInfo).Error
	return err
}

// GetCustomer 根据ID获取客户记录
// Author [yourname](https://github.com/yourname)
func (customerInfoService *CustomerService)GetCustomer(ctx context.Context, ID string) (customerInfo customer.Customer, err error) {
	err = global.GVA_DB.WithContext(ctx).Where("id = ?", ID).First(&customerInfo).Error
	return
}
// GetCustomerInfoList 分页获取客户记录
// Author [yourname](https://github.com/yourname)
func (customerInfoService *CustomerService)GetCustomerInfoList(ctx context.Context, info customerReq.CustomerSearch) (list []customer.Customer, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
    // 创建db
	db := global.GVA_DB.WithContext(ctx).Model(&customer.Customer{})
    var customerInfos []customer.Customer
    // 如果有条件搜索 下方会自动创建搜索语句
    if len(info.CreatedAtRange) == 2 {
     db = db.Where("created_at BETWEEN ? AND ?", info.CreatedAtRange[0], info.CreatedAtRange[1])
    }
    
    if info.FirstName != nil && *info.FirstName != "" {
        db = db.Where(clause.Expr{
            SQL: "? LIKE ?",
            Vars: []any{clause.Column{Name: "first_name"}, "%" + *info.FirstName + "%"},
        })
    }
    if info.LastName != nil && *info.LastName != "" {
        db = db.Where(clause.Expr{
            SQL: "? LIKE ?",
            Vars: []any{clause.Column{Name: "last_name"}, "%" + *info.LastName + "%"},
        })
    }
    if info.Address != nil && *info.Address != "" {
        db = db.Where(clause.Expr{
            SQL: "? LIKE ?",
            Vars: []any{clause.Column{Name: "address"}, "%" + *info.Address + "%"},
        })
    }
    if info.Telephone != nil && *info.Telephone != "" {
        // 电话改成字符串后按模糊匹配：用户往往只记得区号或尾号，
        // 精确匹配基本搜不到东西。
        db = db.Where(clause.Expr{
            SQL: "? LIKE ?",
            Vars: []any{clause.Column{Name: "telephone"}, "%" + *info.Telephone + "%"},
        })
    }
    if info.Gender != "" {
        db = db.Where(clause.Expr{
            SQL: "? = ?",
            Vars: []any{clause.Column{Name: "gender"}, info.Gender},
        })
    }
    if info.Active != "" {
        db = db.Where(clause.Expr{
            SQL: "? = ?",
            Vars: []any{clause.Column{Name: "active"}, info.Active},
        })
    }
    if info.Email != nil && *info.Email != "" {
        db = db.Where(clause.Expr{
            SQL: "? LIKE ?",
            Vars: []any{clause.Column{Name: "email"}, "%" + *info.Email + "%"},
        })
    }
	err = db.Count(&total).Error
	if err!=nil {
    	return
    }

	if limit != 0 {
       db = db.Limit(limit).Offset(offset)
    }

	err = db.Find(&customerInfos).Error
	return  customerInfos, total, err
}
func (customerInfoService *CustomerService)GetCustomerPublic(ctx context.Context) {
    // 此方法为获取数据源定义的数据
    // 请自行实现
}
