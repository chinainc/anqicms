package controller

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/provider"
)

// 会员中心「商城类」页面：购物车、资金、积分（签到）、店铺（装修/保证金）、
// 已售出订单、收到的评价、我的发布、发布内容。
// 页面渲染在这里，业务逻辑在 provider/commerce_*.go，前端表单请求 /api/* 。

// 可发布内容的模型：1=文章，2=产品（AnQiCMS 默认模块）
const (
	memberModuleArticle = 1
	memberModuleProduct = 2
)

type memberCategoryOption struct {
	Id    uint
	Title string
	Depth int
}

func memberCommercePage(ctx iris.Context, currentSite *provider.Website, route string, userInfo *model.User) {
	switch route {
	case "cart":
		carts, total := currentSite.GetCommerceCartList(userInfo.Id)
		ctx.ViewData("carts", carts)
		ctx.ViewData("cartTotal", total)
		ctx.ViewData("cartCount", len(carts))
	case "finance":
		page := ctx.URLParamIntDefault("page", 1)
		logs, total := currentSite.GetFinanceLogs(userInfo.Id, page, memberPageSize)
		ctx.ViewData("logs", logs)
		ctx.ViewData("pager", newMemberPager("/member/finance", url.Values{}, page, memberPageSize, total))
	case "points":
		page := ctx.URLParamIntDefault("page", 1)
		logs, total := currentSite.GetPointLogs(userInfo.Id, page, memberPageSize)
		checkedToday, days, monthCount, points := currentSite.SignInfo(userInfo.Id)
		ctx.ViewData("logs", logs)
		ctx.ViewData("checkedToday", checkedToday)
		ctx.ViewData("signDays", days)
		ctx.ViewData("monthCount", monthCount)
		ctx.ViewData("points", points)
		ctx.ViewData("basePoints", provider.SignBasePoints)
		ctx.ViewData("pager", newMemberPager("/member/points", url.Values{}, page, memberPageSize, total))
	case "shop":
		shop, err := currentSite.EnsureShop(userInfo.Id)
		if err != nil {
			shop = &model.Shop{UserId: userInfo.Id, Name: userInfo.UserName}
		}
		// 库里存相对路径，表单里的预览需要完整地址
		shop.LogoUrl = currentSite.StorageFullUrl(shop.Logo)
		shop.BannerUrl = currentSite.StorageFullUrl(shop.Banner)
		orderCount, saleAmount, pendingCount, todayAmount := currentSite.GetSellerStats(userInfo.Id)
		page := ctx.URLParamIntDefault("page", 1)
		depositLogs, depositTotal := currentSite.GetDepositLogs(userInfo.Id, page, 10)
		decoration := currentSite.GetShopDecoration(shop)
		ctx.ViewData("shop", shop)
		ctx.ViewData("decoration", decoration)
		// 装修横幅存的是相对路径，表单里预览需要完整地址
		ctx.ViewData("decorationBannerUrl", currentSite.StorageFullUrl(decoration.Banner))
		ctx.ViewData("sellerOrderCount", orderCount)
		ctx.ViewData("sellerSaleAmount", saleAmount)
		ctx.ViewData("sellerPending", pendingCount)
		ctx.ViewData("sellerTodayAmount", todayAmount)
		ctx.ViewData("depositLogs", depositLogs)
		ctx.ViewData("depositLogTotal", depositTotal)
	case "sales":
		status := ctx.URLParamDefault("status", "")
		if !memberOrderStatusValid(status) {
			status = ""
		}
		page := ctx.URLParamIntDefault("page", 1)
		orders, total := currentSite.GetSellerOrders(userInfo.Id, status, page, memberPageSize)
		list := make([]*memberOrderView, 0, len(orders))
		for _, item := range orders {
			statusText, statusClass := memberOrderStatus(currentSite, item)
			list = append(list, &memberOrderView{
				Order:       item,
				StatusText:  statusText,
				StatusClass: statusClass,
			})
		}
		query := url.Values{}
		if status != "" {
			query.Set("status", status)
		}
		ctx.ViewData("orders", list)
		ctx.ViewData("orderStatus", status)
		ctx.ViewData("pager", newMemberPager("/member/sales", query, page, memberPageSize, total))
	case "reviews":
		page := ctx.URLParamIntDefault("page", 1)
		comments, total := currentSite.GetReceivedComments(userInfo.Id, page, memberPageSize)
		var ratingSum, ratingCount int
		for _, item := range comments {
			if item.Rating > 0 {
				ratingSum += item.Rating
				ratingCount++
			}
		}
		avgRating := 0.0
		if ratingCount > 0 {
			avgRating = float64(ratingSum) / float64(ratingCount)
		}
		ctx.ViewData("comments", comments)
		ctx.ViewData("avgRating", avgRating)
		ctx.ViewData("avgRatingText", fmt.Sprintf("%.1f", avgRating))
		ctx.ViewData("ratingCount", ratingCount)
		ctx.ViewData("pager", newMemberPager("/member/reviews", url.Values{}, page, memberPageSize, total))
	case "published":
		page := ctx.URLParamIntDefault("page", 1)
		archives, total := currentSite.GetUserPublished(userInfo.Id, page, memberPageSize)
		ctx.ViewData("archives", archives)
		ctx.ViewData("pager", newMemberPager("/member/published", url.Values{}, page, memberPageSize, total))
	case "publish":
		ctx.ViewData("articleCategories", memberCategoryTree(currentSite, memberModuleArticle))
		ctx.ViewData("productCategories", memberCategoryTree(currentSite, memberModuleProduct))
		ctx.ViewData("articleModuleId", memberModuleArticle)
		ctx.ViewData("productModuleId", memberModuleProduct)
		canDirect := currentSite.Safe.APIPublish == 1
		if group, ok := ctx.Values().Get("userGroup").(*model.UserGroup); ok && group != nil && group.Setting.ContentNoVerify {
			canDirect = true
		}
		ctx.ViewData("canPublishDirectly", canDirect)
		ctx.ViewData("flagOptions", memberFlagOptions())
		// ?id=xx 进入编辑模式，只允许编辑本人内容
		ctx.ViewData("editArchive", memberLoadArchiveEdit(currentSite, userInfo.Id, ctx.URLParamInt64Default("id", 0)))
	}
}

// memberEditImage 编辑模式下已有的图片：Path 用于回传保存，Url 用于预览
type memberEditImage struct {
	Path string
	Url  string
}

// memberArchiveEdit 发布页「编辑已有内容」的数据
type memberArchiveEdit struct {
	Id          int64
	Title       string
	Keywords    string
	Description string
	Content     string
	UrlToken    string
	ModuleId    uint
	CategoryId  uint
	Images      []memberEditImage
	Tags        string
	Flags       []string
	CreatedTime string
	Price       string
	Stock       int64
	Draft       bool
}

// memberLoadArchiveEdit 读取待编辑的内容。
// 正式内容优先，其次草稿；只接受本人内容，且跳过已进回收站（status=99）的记录。
// 返回 nil 表示不可编辑，页面会退化成「发布内容」。
func memberLoadArchiveEdit(currentSite *provider.Website, userId uint, id int64) *memberArchiveEdit {
	if id <= 0 || userId == 0 {
		return nil
	}
	edit := &memberArchiveEdit{Id: id}
	var base *model.Archive

	if archive, err := currentSite.GetArchiveById(id); err == nil && archive != nil && archive.UserId == userId {
		base = archive
	} else if draft, err := currentSite.GetArchiveDraftById(id); err == nil && draft != nil &&
		draft.UserId == userId && draft.Status != config.ContentStatusDelete {
		base = &draft.Archive
		edit.Draft = true
	} else {
		return nil
	}

	edit.Title = base.Title
	edit.Keywords = base.Keywords
	edit.Description = base.Description
	edit.UrlToken = base.UrlToken
	edit.ModuleId = base.ModuleId
	edit.CategoryId = base.CategoryId
	edit.Stock = base.Stock
	edit.Price = fmt.Sprintf("%.2f", float64(base.Price)/100)
	if base.CreatedTime > 0 {
		edit.CreatedTime = time.Unix(base.CreatedTime, 0).Format("2006-01-02T15:04")
	}

	// 图片：库里存相对路径，预览需要完整地址（本地存储时退回站点根）。
	// 注意 GetArchiveByFunc/GetArchiveDraftByFunc 内部会调 GetThumb，
	// 已把 Images 补成完整地址，这里要还原成相对路径再回传，避免把绝对地址存回库里。
	storageUrl := currentSite.PluginStorage.StorageUrl
	for _, raw := range base.Images {
		if raw == "" {
			continue
		}
		// 入库用相对路径（外链保持原样），预览用完整地址
		path := config.StripStorageUrl(raw, storageUrl)
		edit.Images = append(edit.Images, memberEditImage{
			Path: path,
			Url:  config.JoinStorageUrl(storageUrl, path),
		})
	}

	if data, err := currentSite.GetArchiveDataById(id); err == nil && data != nil {
		edit.Content = data.Content
	}
	if flags := currentSite.GetArchiveFlags(id); flags != "" {
		edit.Flags = strings.Split(flags, ",")
	}
	if tags := currentSite.GetTagsByItemId(id); len(tags) > 0 {
		names := make([]string, 0, len(tags))
		for _, tag := range tags {
			if tag != nil && tag.Title != "" {
				names = append(names, tag.Title)
			}
		}
		edit.Tags = strings.Join(names, ",")
	}

	return edit
}

// memberFlagOption 推荐属性（与后台「添加文档」页的勾选项一致）
type memberFlagOption struct {
	Flag  string
	Title string
}

// memberFlagOptions AnQiCMS 的 flag 字母表：h 头条 c 推荐 f 幻灯 a 特荐 s 滚动 b 加粗 p 图片 j 跳转
func memberFlagOptions() []memberFlagOption {
	return []memberFlagOption{
		{Flag: "h", Title: "头条"},
		{Flag: "c", Title: "推荐"},
		{Flag: "f", Title: "幻灯"},
		{Flag: "a", Title: "特荐"},
		{Flag: "s", Title: "滚动"},
		{Flag: "b", Title: "加粗"},
		{Flag: "p", Title: "图片"},
		{Flag: "j", Title: "跳转"},
	}
}

// memberCategoryTree 取某个模型下可用的栏目（顶级在前，子级缩进）
// 只取「列表栏目」（type=1），单页栏目不能发布内容
func memberCategoryTree(currentSite *provider.Website, moduleId uint) []memberCategoryOption {
	categories := currentSite.GetCategoriesFromCache(moduleId, 0, config.CategoryTypeArchive, true)
	if len(categories) == 0 {
		return nil
	}
	children := make(map[uint][]*model.Category)
	for _, item := range categories {
		children[item.ParentId] = append(children[item.ParentId], item)
	}
	for _, list := range children {
		sort.SliceStable(list, func(i, j int) bool {
			return list[i].Id < list[j].Id
		})
	}

	var result []memberCategoryOption
	var walk func(parentId uint, depth int)
	walk = func(parentId uint, depth int) {
		for _, item := range children[parentId] {
			result = append(result, memberCategoryOption{Id: item.Id, Title: item.Title, Depth: depth})
			walk(item.Id, depth+1)
		}
	}
	walk(0, 0)

	return result
}
