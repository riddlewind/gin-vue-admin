package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinRecovery recover掉项目可能出现的panic，并使用zap记录相关日志
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 先判断是不是"连接已断开"这类错误。
				// 这种情况不值得记录完整的 panic 堆栈——客户端早就跑了，记了也没人看。
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					global.GVA_LOG.Error(c.Request.URL.Path,
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
					// 连接已经断了，此时无法再往响应里写状态码，只能记日志后中断。
					_ = c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}

				if stack {
					form := "后端"
					info := fmt.Sprintf("Panic: %v\nRequest: %s\nStack: %s", err, string(httpRequest), string(debug.Stack()))
					level := "error"
					_ = service.ServiceGroupApp.SystemServiceGroup.SysErrorService.CreateSysError(context.Background(), &system.SysError{
						Form:  &form,
						Info:  &info,
						Level: level,
					})
					global.GVA_LOG.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				} else {
					form := "后端"
					info := fmt.Sprintf("Panic: %v\nRequest: %s", err, string(httpRequest))
					level := "error"
					_ = service.ServiceGroupApp.SystemServiceGroup.SysErrorService.CreateSysError(context.Background(), &system.SysError{
						Form:  &form,
						Info:  &info,
						Level: level,
					})
					global.GVA_LOG.Error("[Recovery from panic]",
						zap.Any("error", err),
						zap.String("request", string(httpRequest)),
					)
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
