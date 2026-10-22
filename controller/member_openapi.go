package controller

import (
	"strings"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/provider"
)

// 会员开放接口（API / MCP / CLI）。
//
// 一把 API Key 三处通用：
//   - REST：既有的 /api/user/* 等会员接口，带 Authorization: Bearer <key> 即可
//   - MCP ：/api/open/mcp（Streamable HTTP，Key 用同一个）
//   - CLI ：cmd/anqicms-cli，配置 server + token 即可
// 权限边界完全由会员身份决定，不能访问后台能力。

// memberOpenApiPage 会员中心「开放接口」页数据
func memberOpenApiPage(ctx iris.Context, currentSite *provider.Website, userInfo *model.User) {
	token := currentSite.GetUserApiToken(userInfo.Id)
	baseUrl := strings.TrimRight(currentSite.System.BaseUrl, "/")
	if baseUrl == "" {
		baseUrl = strings.TrimRight(currentSite.GetUrl("/", nil, 0), "/")
	}

	ctx.ViewData("apiToken", token)
	ctx.ViewData("apiTokenMasked", provider.MaskApiToken(token))
	ctx.ViewData("openBaseUrl", baseUrl)
	ctx.ViewData("openRestBase", baseUrl+provider.MemberOpenApiBasePath)
	ctx.ViewData("openMcpUrl", baseUrl+provider.MemberMcpPath)
	ctx.ViewData("openEnabled", token != "")
}

// ApiMemberOpenApiInfo 当前开放接口状态
func ApiMemberOpenApiInfo(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	userId := apiCommerceUser(ctx)
	token := currentSite.GetUserApiToken(userId)
	baseUrl := strings.TrimRight(currentSite.System.BaseUrl, "/")
	if baseUrl == "" {
		baseUrl = strings.TrimRight(currentSite.GetUrl("/", nil, 0), "/")
	}
	apiCommerceOK(ctx, iris.Map{
		"enabled":     token != "",
		"token":       token,
		"masked":      provider.MaskApiToken(token),
		"rest_base":   baseUrl + provider.MemberOpenApiBasePath,
		"mcp_url":     baseUrl + provider.MemberMcpPath,
		"cli_package": "anqicms-cli",
	})
}

// ApiMemberOpenApiToken 生成 / 重置 / 停用 API Key
func ApiMemberOpenApiToken(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	userId := apiCommerceUser(ctx)

	var req struct {
		Action string `json:"action"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Action = ctx.PostValueTrim("action")
	}
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))

	switch req.Action {
	case "clear", "disable":
		if err := currentSite.ClearUserApiToken(userId); err != nil {
			apiCommerceFail(ctx, err.Error())
			return
		}
		apiCommerceOK(ctx, iris.Map{"enabled": false, "token": "", "masked": ""})
		return
	case "reset", "generate", "":
		token, err := currentSite.ResetUserApiToken(userId)
		if err != nil {
			apiCommerceFail(ctx, err.Error())
			return
		}
		apiCommerceOK(ctx, iris.Map{
			"enabled": true,
			"token":   token,
			"masked":  provider.MaskApiToken(token),
		})
		return
	default:
		apiCommerceFail(ctx, currentSite.TplTr("InvalidOperation"))
	}
}

// MemberMcpHTTP 会员专属 MCP 端点（Streamable HTTP）
func MemberMcpHTTP(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	user, _ := ctx.Values().Get("userInfo").(*model.User)
	if user == nil {
		ctx.StatusCode(iris.StatusUnauthorized)
		ctx.JSON(iris.Map{"code": config.StatusNoLogin, "msg": currentSite.TplTr("MemberApiTokenInvalid")})
		return
	}
	token, _ := ctx.Values().Get("apiToken").(string)

	handler, err := provider.MemberMcpHandler(currentSite, user, token)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		ctx.JSON(iris.Map{"error": err.Error()})
		return
	}

	handler.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
}
