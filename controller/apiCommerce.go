package controller

import (
	"path/filepath"
	"strings"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/provider"
	"kandaoni.com/anqicms/request"
)

// 会员商城相关 API：购物车、积分、签到、店铺、保证金、已售订单、收到的评价、资金流水

func apiCommerceUser(ctx iris.Context) uint {
	return ctx.Values().GetUintDefault("userId", 0)
}

func apiCommerceOK(ctx iris.Context, data interface{}) {
	ctx.JSON(iris.Map{
		"code": config.StatusOK,
		"msg":  "",
		"data": data,
	})
}

func apiCommerceFail(ctx iris.Context, msg string) {
	ctx.JSON(iris.Map{
		"code": config.StatusFailed,
		"msg":  msg,
	})
}

// ---------------------------------------------------------------- 购物车

func ApiCartList(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	carts, total := currentSite.GetCartList(apiCommerceUser(ctx))
	apiCommerceOK(ctx, iris.Map{
		"list":  carts,
		"total": total,
		"count": len(carts),
	})
}

func ApiCartAdd(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req request.CartRequest
	if err := ctx.ReadJSON(&req); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	cart, err := currentSite.AddCart(apiCommerceUser(ctx), req.ArchiveId, req.Quantity)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{
		"cart":  cart,
		"count": currentSite.CountCart(apiCommerceUser(ctx)),
	})
}

func ApiCartUpdate(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req request.CartRequest
	if err := ctx.ReadJSON(&req); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	if err := currentSite.UpdateCartQuantity(apiCommerceUser(ctx), req.CartId, req.Quantity); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	carts, total := currentSite.GetCartList(apiCommerceUser(ctx))
	apiCommerceOK(ctx, iris.Map{"list": carts, "total": total})
}

func ApiCartDelete(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req request.CartRequest
	if err := ctx.ReadJSON(&req); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	if err := currentSite.DeleteCart(apiCommerceUser(ctx), req.CartId); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	carts, total := currentSite.GetCartList(apiCommerceUser(ctx))
	apiCommerceOK(ctx, iris.Map{"list": carts, "total": total})
}

func ApiCartClear(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	if err := currentSite.ClearCart(apiCommerceUser(ctx)); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{})
}

func ApiCartCheckout(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req request.OrderRequest
	// 允许空 body（只结算购物车）
	_ = ctx.ReadJSON(&req)
	if req.Address == nil && req.AddressId == 0 {
		apiCommerceFail(ctx, currentSite.TplTr("PleaseSelectShippingAddress"))
		return
	}
	order, err := currentSite.CheckoutCart(apiCommerceUser(ctx), &req)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, order)
}

// ---------------------------------------------------------------- 积分与签到

func ApiPointLogs(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("pageSize", 20)
	logs, total := currentSite.GetPointLogs(apiCommerceUser(ctx), page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": logs, "total": total})
}

func ApiSignInfo(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	checkedToday, days, monthCount, points := currentSite.SignInfo(apiCommerceUser(ctx))
	apiCommerceOK(ctx, iris.Map{
		"checked_today": checkedToday,
		"days":          days,
		"month_count":   monthCount,
		"points":        points,
		"base_points":   provider.SignBasePoints,
	})
}

func ApiSignDo(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	sign, err := currentSite.SignToday(apiCommerceUser(ctx))
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	checkedToday, days, monthCount, points := currentSite.SignInfo(apiCommerceUser(ctx))
	apiCommerceOK(ctx, iris.Map{
		"sign":          sign,
		"checked_today": checkedToday,
		"days":          days,
		"month_count":   monthCount,
		"points":        points,
	})
}

// ---------------------------------------------------------------- 店铺 / 保证金

func ApiShopInfo(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	shop, err := currentSite.EnsureShop(apiCommerceUser(ctx))
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	decoration := currentSite.GetShopDecoration(shop)
	orderCount, saleAmount, pendingCount, todayAmount := currentSite.GetSellerStats(shop.UserId)
	apiCommerceOK(ctx, iris.Map{
		"shop":         shop,
		"decoration":   decoration,
		"order_count":  orderCount,
		"sale_amount":  saleAmount,
		"pending":      pendingCount,
		"today_amount": todayAmount,
	})
}

func ApiShopSave(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req request.ShopRequest
	if err := ctx.ReadJSON(&req); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	shop, err := currentSite.SaveShop(apiCommerceUser(ctx), &req)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{"shop": shop})
}

func ApiShopDepositLogs(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("pageSize", 20)
	logs, total := currentSite.GetDepositLogs(apiCommerceUser(ctx), page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": logs, "total": total})
}

func ApiShopDepositPay(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req request.DepositRequest
	if err := ctx.ReadJSON(&req); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	shop, err := currentSite.PayDeposit(apiCommerceUser(ctx), req.Amount)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{"shop": shop})
}

func ApiShopDepositRefund(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req request.DepositRequest
	if err := ctx.ReadJSON(&req); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	shop, err := currentSite.RefundDeposit(apiCommerceUser(ctx), req.Amount)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{"shop": shop})
}

func ApiShopSales(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("pageSize", 20)
	status := ctx.URLParam("status")
	orders, total := currentSite.GetSellerOrders(apiCommerceUser(ctx), status, page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": orders, "total": total})
}

func ApiShopReviews(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("pageSize", 20)
	comments, total := currentSite.GetReceivedComments(apiCommerceUser(ctx), page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": comments, "total": total})
}

// ---------------------------------------------------------------- 我的发布 / 资金

func ApiUserArchives(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("pageSize", 20)
	archives, total := currentSite.GetUserPublished(apiCommerceUser(ctx), page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": archives, "total": total})
}

func ApiUserFinances(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("pageSize", 20)
	logs, total := currentSite.GetFinanceLogs(apiCommerceUser(ctx), page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": logs, "total": total})
}

// ApiUserUploadImage 会员上传图片（发布内容时的封面、图集、正文插图）
//
// 与后台 /system/api/attachment/upload 走同一个 provider 落库逻辑，
// 区别是鉴权用会员令牌，并且只允许图片。
func ApiUserUploadImage(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)

	file, info, err := ctx.FormFile("file")
	if err != nil {
		apiCommerceFail(ctx, "请选择要上传的图片")
		return
	}
	defer file.Close()

	// 限制 20MB，避免会员占用大量磁盘
	if info.Size > 20<<20 {
		apiCommerceFail(ctx, "图片不能超过 20MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(info.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
	default:
		apiCommerceFail(ctx, "只允许上传图片文件")
		return
	}

	attachment, err := currentSite.AttachmentUpload(file, info, 0, 0, apiCommerceUser(ctx))
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	attachment.GetThumb(currentSite.PluginStorage.StorageUrl)

	apiCommerceOK(ctx, iris.Map{
		"id":    attachment.Id,
		"path":  attachment.FileLocation,
		"url":   attachment.Logo,
		"thumb": attachment.Thumb,
		"name":  attachment.FileName,
	})
}
