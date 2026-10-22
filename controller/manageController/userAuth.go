package manageController

import (
	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/provider"
)

// 后台「认证管理」：会员实名 / 企业认证的列表与审核。
// 界面是独立页 /system/user-auth.html（后台 SPA 是编译产物，无法直接加路由）。

// PluginUserAuthList 认证列表（支持状态、关键词筛选）
func PluginUserAuthList(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("pageSize", 20)
	keyword := ctx.URLParam("keyword")
	// status: all（默认，全部）/ 0 待审核 / 1 已通过 / -1 已驳回
	status := provider.UserAuthStatusAll
	if raw := ctx.URLParam("status"); raw != "" && raw != "all" {
		status = ctx.URLParamIntDefault("status", provider.UserAuthStatusAll)
	}

	list, total := currentSite.GetUserAuthList(status, keyword, page, pageSize)
	pending, passed, rejected := currentSite.GetUserAuthStat()

	ctx.JSON(iris.Map{
		"code": config.StatusOK,
		"msg":  "",
		"data": iris.Map{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"pending":   pending,
			"passed":    passed,
			"rejected":  rejected,
		},
	})
}

// PluginUserAuthDetail 认证详情
func PluginUserAuthDetail(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	id := uint(ctx.URLParamIntDefault("id", 0))
	item, err := currentSite.GetUserAuthById(id)
	if err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": err.Error()})
		return
	}
	ctx.JSON(iris.Map{"code": config.StatusOK, "msg": "", "data": item})
}

// PluginUserAuthAudit 审核：status=1 通过，-1 拒绝（拒绝必须填原因）
func PluginUserAuthAudit(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Id     uint   `json:"id"`
		Status int    `json:"status"`
		Remark string `json:"remark"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Id = uint(ctx.PostValueIntDefault("id", 0))
		req.Status = ctx.PostValueIntDefault("status", 0)
		req.Remark = ctx.PostValueTrim("remark")
	}
	adminId := ctx.Values().GetUintDefault("adminId", 0)
	item, err := currentSite.AuditUserAuth(req.Id, req.Status, req.Remark, adminId)
	if err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": err.Error()})
		return
	}
	currentSite.AddAdminLog(ctx, ctx.Tr("AuditUserAuth")+" "+item.UserName)

	ctx.JSON(iris.Map{"code": config.StatusOK, "msg": ctx.Tr("AuditCompleted"), "data": item})
}
