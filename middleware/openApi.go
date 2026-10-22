package middleware

import (
	"strings"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/provider"
)

// OpenApiTokenAuth 会员开放接口鉴权（API Key）。
//
// 与 ParseUserToken 的区别：这里**只认 API Key**，不接受 JWT，
// 用于 /api/open/mcp 这类给程序/AI 客户端长期访问的端点。
// Key 依次从 X-Api-Key、Authorization: Bearer、?token= 读取。
func OpenApiTokenAuth(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)

	token := strings.TrimSpace(ctx.GetHeader("X-Api-Key"))
	if token == "" {
		auth := ctx.GetHeader("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	if token == "" {
		token = strings.TrimSpace(ctx.URLParam("token"))
	}
	if token == "" {
		ctx.StatusCode(iris.StatusUnauthorized)
		ctx.JSON(iris.Map{
			"code": config.StatusNoLogin,
			"msg":  currentSite.TplTr("MemberApiTokenRequired"),
		})
		return
	}

	user := currentSite.GetUserByApiToken(token)
	if user == nil {
		ctx.StatusCode(iris.StatusUnauthorized)
		ctx.JSON(iris.Map{
			"code": config.StatusNoLogin,
			"msg":  currentSite.TplTr("MemberApiTokenInvalid"),
		})
		return
	}

	ctx.Values().Set("userId", user.Id)
	ctx.Values().Set("userInfo", user)
	ctx.Values().Set("apiToken", token)
	userGroup, _ := currentSite.GetUserGroupInfo(user.GroupId)
	ctx.Values().Set("userGroup", userGroup)

	ctx.Next()
}
