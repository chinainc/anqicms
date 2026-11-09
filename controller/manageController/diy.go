package manageController

import (
	"encoding/json"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/provider"
)

// 后台「可视化装修」接口。
// 界面是独立页 /system/diy.html（后台 SPA 是打包产物，无法直接加路由）。

// PluginDiyTypes 区块类型与可装修页面（后台据此生成模块库和表单）
func PluginDiyTypes(ctx iris.Context) {
	ctx.JSON(iris.Map{
		"code": config.StatusOK,
		"msg":  "",
		"data": iris.Map{
			"types": provider.DiyModuleTypes(),
			"pages": provider.DiyPageKeys(),
		},
	})
}

// PluginDiyPage 读取某个页面的装修配置
func PluginDiyPage(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	key := ctx.URLParamDefault("key", "home")

	modules := currentSite.GetDiyModules(key)
	if modules == nil {
		modules = []provider.DiyModule{}
	}
	page := currentSite.GetDiyPage(key)
	title := ""
	updated := int64(0)
	if page != nil {
		title = page.Title
		updated = page.UpdatedTime
	}

	ctx.JSON(iris.Map{
		"code": config.StatusOK,
		"msg":  "",
		"data": iris.Map{
			"key":          key,
			"title":        title,
			"updated_time": updated,
			"modules":      modules,
		},
	})
}

// PluginDiySave 保存装修配置
func PluginDiySave(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)

	var req struct {
		Key     string               `json:"key"`
		Title   string               `json:"title"`
		Modules []provider.DiyModule `json:"modules"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": err.Error()})
		return
	}
	if req.Key == "" {
		req.Key = "home"
	}
	if err := currentSite.SaveDiyPage(req.Key, req.Title, req.Modules); err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": err.Error()})
		return
	}
	currentSite.AddAdminLog(ctx, ctx.Tr("SaveCustomPage")+" "+req.Key)

	raw, _ := json.Marshal(req.Modules)
	ctx.JSON(iris.Map{
		"code": config.StatusOK,
		"msg":  ctx.Tr("SavedSuccessfully"),
		"data": iris.Map{
			"key":   req.Key,
			"count": len(req.Modules),
			"bytes": len(raw),
		},
	})
}

// PluginDiyReset 清空装修（前台回退到模板默认内容）
func PluginDiyReset(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)

	var req struct {
		Key string `json:"key"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Key = ctx.PostValueTrim("key")
	}
	if req.Key == "" {
		req.Key = "home"
	}
	if err := currentSite.ResetDiyPage(req.Key); err != nil {
		ctx.JSON(iris.Map{"code": config.StatusFailed, "msg": err.Error()})
		return
	}
	ctx.JSON(iris.Map{"code": config.StatusOK, "msg": ctx.Tr("SavedSuccessfully")})
}
