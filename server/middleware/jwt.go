// ============================================================================
// 【文件导读】JWT 鉴权中间件。整个项目"是谁在请求"的答案就在这里。
//
// 这是理解 Gin 中间件最好的入口，因为它同时演示了三件事：
//  1. 中间件就是一个"返回函数的函数"（闭包）；
//  2. c.Abort() 中断链条，c.Next() 放行；
//  3. c.Next() 之后的代码，是在业务处理完之后才执行的。
//
// 请求进来后的完整顺序：
//
//	JWTAuth 前半段 -> c.Next() -> Casbin -> API -> Service -> 返回
//	             <- 回到 JWTAuth 后半段（设置 new-token 响应头）
//
// 【本文件涉及的 Go 语法】
//   - 高阶函数 / 闭包：JWTAuth() 返回 gin.HandlerFunc
//   - 类型断言带 ok：v.(string) 的双返回值写法
//   - 错误哨兵 + errors.Is 判断
//   - 空标识符 _ 丢弃返回值
//
// ============================================================================
package middleware

import (
	"errors"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/golang-jwt/jwt/v5"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

// JWTAuth 返回一个 Gin 中间件
//
// [Go] 注意签名：外层 JWTAuth() 本身不是中间件，它"返回"中间件。
//
//	gin.HandlerFunc 的底层定义就是：type HandlerFunc func(*gin.Context)
//
// 所以中间件本质上就是一个只接收 *gin.Context 的普通函数。
// 之所以要套一层外层函数，是为了能在外层做一次性准备工作
// （比如读配置、建连接池），内层每次请求复用这些结果——这就是闭包的价值。
func JWTAuth() gin.HandlerFunc {
	// 这个内层匿名函数，就是真正参与每次请求的中间件本体。
	return func(c *gin.Context) {
		// 我们这里jwt鉴权取头部信息 x-token 登录时回返回token信息 这里前端需要把token存储到cookie或者本地localStorage中 不过需要跟后端协商过期时间 可以约定刷新令牌或者重新登录
		//
		// [注意] token 的来源是请求头 x-token（也兼容 cookie），
		// 具体取法封装在 utils.GetToken 里。前端 request.js 会统一注入这个 header。
		token := utils.GetToken(c)
		if token == "" {
			response.NoAuth("未登录或非法访问，请登录", c)

			// [Go] c.Abort() 的作用：把当前请求的中间件链标记为"已中止"，
			// 后续所有中间件和最终的业务 handler 都不会再执行。
			// [注意] Abort() 之后必须 return，否则本函数剩下的代码照样会跑完。
			// 这两个是配套的，只写 Abort 不写 return 是最常见的低级错误。
			c.Abort()
			return
		}

		// 检查 token 是否已被拉黑（用户主动登出、被禁用、异地登录时会被塞进黑名单）
		if isBlacklist(token) {
			response.NoAuth("您的帐户异地登陆或令牌失效", c)
			utils.ClearToken(c)
			c.Abort()
			return
		}

		j := utils.NewJWT()
		// parseToken 解析token包含的信息
		claims, err := j.ParseToken(token)
		if err != nil {
			// [Go] errors.Is 用来判断错误链里是否包含某个"哨兵错误"。
			// 因为 ParseToken 内部可能对原始错误做了包装（fmt.Errorf + %w），
			// 直接用 == 比较会失败，必须用 errors.Is 才能穿透包装。
			if errors.Is(err, utils.TokenExpired) {
				response.NoAuth("登录已过期，请重新登录", c)
				utils.ClearToken(c)
				c.Abort()
				return
			}
			response.NoAuth(err.Error(), c)
			utils.ClearToken(c)
			c.Abort()
			return
		}

		// 已登录用户被管理员禁用 需要使该用户的jwt失效 此处比较消耗性能 如果需要 请自行打开
		// 用户被删除的逻辑 需要优化 此处比较消耗性能 如果需要 请自行打开

		//if user, err := userService.FindUserByUuid(claims.UUID.String()); err != nil || user.Enable == 2 {
		//	_ = jwtService.JsonInBlacklist(system.JwtBlacklist{Jwt: token})
		//	response.FailWithDetailed(gin.H{"reload": true}, err.Error(), c)
		//	c.Abort()
		//}

		// [Go] c.Set 往本次请求的上下文里存值，后面任意中间件/handler
		// 都能用 c.Get("claims") 取出来。
		// 这是 Gin 在链上传递数据的标准方式——因为 *gin.Context 是贯穿整个请求的同一份对象。
		//
		// [注意] Context 不是并发安全的容器，不要往里塞大对象或做缓存用途。
		c.Set("claims", claims)

		// ---- 滑动续期：距离过期不足 BufferTime 时，自动签发新 token ----
		// 配置里 expires-time: 7d、buffer-time: 1d，
		// 意思是"剩余有效期不到 1 天就换新令牌"，用户只要在活跃就永远不会被踢下线。
		if claims.ExpiresAt.Unix()-time.Now().Unix() < claims.BufferTime {
			dr, _ := utils.ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)
			claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(dr))
			newToken, _ := j.CreateTokenByOldToken(token, *claims)
			newClaims, _ := j.ParseToken(newToken)
			c.Header("new-token", newToken)
			c.Header("new-expires-at", strconv.FormatInt(newClaims.ExpiresAt.Unix(), 10))
			utils.SetToken(c, newToken, int(dr.Seconds()))
			if global.GVA_CONFIG.System.UseMultipoint {
				// 记录新的活跃jwt
				//
				// [Go] `_ =` 是空标识符，显式表示"我知道这里返回了 error，但故意忽略"。
				// Go 不允许变量声明后不使用，用 _ 是表明态度的正确做法，
				// 比随手删掉返回值更清楚。
				_ = utils.SetRedisJWT(newToken, newClaims.Username)
			}
		}

		// [Go] c.Next() 是理解 Gin 中间件的分水岭。
		// 它不是"跳过"，而是"先把后面的中间件和 handler 全部执行完，再回到我这行继续"。
		// 所以下一行开始的代码，执行时机是在业务处理"之后"。
		c.Next()

		// 下面两段是"后置处理"：业务代码执行完、响应体已经写出去了，
		// 但响应头还可以补。这里把前面算好的新 token 补到响应头上，
		// 前端 request.js 拦到 new-token 就会自动更新本地 token。
		//
		// [Go] `if v, exists := m[key]; exists` 是 map/类型断言的逗号-ok 惯用法。
		// 第一个值是真值，第二个值是"是否存在"的布尔。
		// [注意] 一定要用带 ok 的写法；直接 newToken.(string) 在类型不符时会 panic。
		if newToken, exists := c.Get("new-token"); exists {
			c.Header("new-token", newToken.(string))
		}
		if newExpiresAt, exists := c.Get("new-expires-at"); exists {
			c.Header("new-expires-at", newExpiresAt.(string))
		}
	}
}

//@author: [piexlmax](https://github.com/piexlmax)
//@function: IsBlacklist
//@description: 判断JWT是否在黑名单内部
//@param: jwt string
//@return: bool

// isBlacklist 判断 token 是否已失效
//
// [Go] 小写开头的函数是"包私有"的（unexported），只有 middleware 包内部能调用。
// 大写开头才是对外导出的。这是 Go 唯一的访问控制机制——靠首字母大小写，没有 private 关键字。
func isBlacklist(jwt string) bool {
	// [Go] 这里只关心"取没取到"，不关心取到的值，所以第一个返回值用 _ 丢掉。
	// 本地缓存（local_cache）实现，进程重启即清空，所以拉黑状态不持久——
	// 这也是为什么项目里还有一张 sys_jwt_blacklist 表用于持久化黑名单。
	_, ok := global.BlackCache.Get(jwt)
	return ok
}
