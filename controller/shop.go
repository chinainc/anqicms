package controller

import (
	"strconv"
	"strings"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/provider"
	"kandaoni.com/anqicms/response"
)

// 店铺主页：/shop/{userId}.html
// 展示店铺资料、装修配置、公告与店主发布的内容
func ShopPage(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)

	raw := strings.TrimSpace(ctx.Params().Get("id"))
	raw = strings.TrimSuffix(raw, ".html")
	userId, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || userId == 0 {
		NotFound(ctx)
		return
	}

	shop := currentSite.GetShopByUserId(uint(userId))
	if shop == nil {
		NotFound(ctx)
		return
	}

	archives, total := currentSite.GetUserArchives(uint(userId), 1, 12)
	decoration := currentSite.GetShopDecoration(shop)
	// 库里存相对路径，前台展示补成完整地址
	shop.LogoUrl = currentSite.StorageFullUrl(shop.Logo)
	shop.BannerUrl = currentSite.StorageFullUrl(shop.Banner)
	// 头图优先用「店铺装修」里的横幅，没配则退回店铺资料里的店铺横幅
	shopBanner := decoration.Banner
	if shopBanner == "" {
		shopBanner = shop.Banner
	}

	if webInfo, ok := ctx.Value("webInfo").(*response.WebInfo); ok {
		webInfo.Title = shop.Name
		webInfo.Keywords = shop.Name
		webInfo.Description = shop.Intro
		webInfo.PageName = "shop"
		ctx.ViewData("webInfo", webInfo)
	}
	ctx.ViewData("shop", shop)
	ctx.ViewData("decoration", decoration)
	ctx.ViewData("shopBanner", currentSite.StorageFullUrl(shopBanner))
	ctx.ViewData("archives", archives)
	ctx.ViewData("archiveTotal", total)

	if err = ctx.View(GetViewPath(ctx, "shop/show.html")); err != nil {
		ctx.StatusCode(iris.StatusNotFound)
		ctx.Values().Set("message", err.Error())
	}
}
