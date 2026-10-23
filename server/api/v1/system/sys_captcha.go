package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"go.uber.org/zap"
)

// 当开启多服务器部署时，替换下面的配置，使用redis共享存储验证码
// var store = captcha.NewDefaultRedisStore()
var store = base64Captcha.DefaultMemStore

type BaseApi struct{}

// Captcha 验证码
// @Tags      Base
// @Summary   生成验证码
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Success   200  {object}  response.Response{data=systemRes.SysCaptchaResponse,msg=string}  "生成验证码,返回包括随机数id,base64,验证码长度,是否开启验证码"
// @Router    /base/captcha [post]
func (b *BaseApi) Captcha(c *gin.Context) {
	// 判断验证码是否开启
	oc := captchaRequired(c)
	// 字符,公式,验证码配置
	// 生成默认数字的driver
	driver := base64Captcha.NewDriverDigit(global.GVA_CONFIG.Captcha.ImgHeight, global.GVA_CONFIG.Captcha.ImgWidth, global.GVA_CONFIG.Captcha.KeyLong, 0.7, 80)
	// cp := base64Captcha.NewCaptcha(driver, store.UseWithCtx(c))   // v8下使用redis
	cp := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := cp.Generate()
	if err != nil {
		global.GVA_LOG.Error("验证码获取失败!", zap.Error(err))
		response.FailWithMessage("验证码获取失败", c)
		return
	}
	response.OkWithDetailed(systemRes.SysCaptchaResponse{
		CaptchaId:     id,
		PicPath:       b64s,
		CaptchaLength: global.GVA_CONFIG.Captcha.KeyLong,
		OpenCaptcha:   oc,
	}, "验证码获取成功", c)
}

// captchaRequired 判断当前请求是否需要填写验证码。
//
// 规则由 config.yaml 的 captcha.open-captcha 决定：
//   - 负数（推荐 -1）：永久关闭，任何情况下都不要求验证码；
//   - 0：每次登录都必须填，最严格；
//   - 正数 N：同一 IP 密码错误累计达到 N 次后开始要求，即"防爆破次数"。
//
// [注意] 本函数同时被登录接口（sys_user.go 的 Login）和验证码接口使用。
// 两边必须共用同一套判断，否则会出现"前端不显示验证码、后端却仍在校验"
// 或者"前端显示了、后端根本不看"的错位。
//
// [Go] 这里用 switch 而不是 if-else 链，是为了让三种取值语义在视觉上一目了然。
func captchaRequired(c *gin.Context) bool {
	openCaptcha := global.GVA_CONFIG.Captcha.OpenCaptcha
	key := c.ClientIP()

	// 首次访问时初始化该 IP 的失败计数，后续失败由各接口 Increment 累加
	v, ok := global.BlackCache.Get(key)
	if !ok {
		global.BlackCache.Set(key, 1, time.Second*time.Duration(global.GVA_CONFIG.Captcha.OpenCaptchaTimeOut))
	}

	switch {
	case openCaptcha < 0:
		return false
	case openCaptcha == 0:
		return true
	default:
		return openCaptcha < interfaceToInt(v)
	}
}

// 类型转换
func interfaceToInt(v interface{}) (i int) {
	switch v := v.(type) {
	case int:
		i = v
	default:
		i = 0
	}
	return
}
