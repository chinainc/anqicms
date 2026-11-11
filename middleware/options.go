package middleware

import (
	"strings"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/provider"
)

func Cors(ctx iris.Context) {
	origin := strings.TrimSpace(ctx.GetHeader("Origin"))
	if origin != "" {
		// 响应内容随 Origin 变化，必须声明 Vary，否则中间层缓存可能把
		// 一个站点的 CORS 头复用给另一个站点。
		ctx.Header("Vary", "Origin")
		// 只对本站点（含多站点的其它域名、本机调试地址）回写允许来源。
		// 不能回 *：下面带了 Allow-Credentials，且通配会把接口暴露给任意第三方页面。
		if provider.IsAllowedCORSOrigin(origin) {
			ctx.Header("Access-Control-Allow-Origin", origin)
		}
	}
	ctx.Header("Access-Control-Allow-Credentials", "true")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition,Update-Token")
	ctx.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Api, Accept, Authorization, Version, Admin, Token, Key, Cart, Site-Id, Sub-Site-Id, X-Site-Id, X-Pjax, X-Pjax-Container")
	// 预检结果缓存 10 分钟，减少多站点下每个跨域请求都先打一次 OPTIONS
	ctx.Header("Access-Control-Max-Age", "600")
	if ctx.Request().Method == "OPTIONS" {
		ctx.StatusCode(204)
		return
	}
	ctx.Next()
}
