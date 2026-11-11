package manageController

import (
	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/provider"
)

// 后台「认证管理」：会员实名 / 企业认证的列表与审核。
// 界面是独立页 /system/user-auth.html（后台 SPA 是编译产物，无法直接加路由）。

// PluginUserAuthList 认证列表（支持状态、关键词筛选）
// 参数说明：
//   - 查询参数 "page": 当前页码，默认 1。
//   - 查询参数 "pageSize": 每页条数，默认 20。
//   - 查询参数 "keyword": 按姓名/手机号等关键词筛选，可空。
//   - 查询参数 "status": all（默认，全部）/ 0 待审核 / 1 已通过 / -1 已驳回。
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
// 参数说明：
//   - 查询参数 "id": 认证记录 ID，必填。
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
	// JSON 解析失败时回退读表单值（独立页面以表单提交）；req 不会是零值落库
	err := ctx.ReadJSON(&req)
	if err != nil {
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
