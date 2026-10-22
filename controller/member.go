package controller

import (
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kataras/iris/v12"
	"gorm.io/gorm"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/provider"
	"kandaoni.com/anqicms/response"
)

// 会员中心（/member）
//
// 页面清单与 tmago 的会员中心保持一致的入口结构（概览 / 资料 / 密码 / 收藏 / 订单），
// 但数据全部取自 AnQiCMS 自身的用户体系与现有 API：
//   - 用户资料：provider.Website.GetUserInfoById / UpdateUserInfo
//   - 头像上传：ApiUpdateUserAvatar（POST /api/user/avatar）
//   - 修改密码：ApiUpdateUserPassword（POST /api/user/password）
//   - 收藏：GetFavoriteList（GET /api/favorite/list 同源）
//   - 订单：GetOrderList（GET /api/orders 同源）
//
// 页面为服务端渲染（与默认模板其它页面一致），表单类操作走前端 JS 请求既有 API。
const memberPageSize = 12

type memberNavEntry struct {
	Route string
	Icon  string
}

// memberNavEntries 侧栏菜单，顺序即展示顺序
var memberNavEntries = []memberNavEntry{
	{Route: "index", Icon: "fa-dashboard"},
	{Route: "profile", Icon: "fa-user-o"},
	{Route: "password", Icon: "fa-lock"},
	{Route: "cart", Icon: "fa-shopping-cart"},
	{Route: "favorites", Icon: "fa-star-o"},
	{Route: "orders", Icon: "fa-list-alt"},
	{Route: "sales", Icon: "fa-truck"},
	{Route: "published", Icon: "fa-file-text-o"},
	{Route: "publish", Icon: "fa-plus-circle"},
	{Route: "reviews", Icon: "fa-commenting-o"},
	{Route: "verify", Icon: "fa-id-card-o"},
	{Route: "openapi", Icon: "fa-plug"},
	{Route: "comments", Icon: "fa-comments-o"},
	{Route: "messages", Icon: "fa-envelope-o"},
	{Route: "finance", Icon: "fa-cny"},
	{Route: "recharge", Icon: "fa-plus-circle"},
	{Route: "withdraw", Icon: "fa-credit-card"},
	{Route: "points", Icon: "fa-diamond"},
	{Route: "invite", Icon: "fa-users"},
	{Route: "shop", Icon: "fa-home"},
}

// memberNavView 已翻译的菜单项，模板直接输出
type memberNavView struct {
	Route string
	Title string
	Icon  string
	Link  string
}

// memberTitle 各页面的标题词条
func memberTitle(currentSite *provider.Website, route string) string {
	switch route {
	case "profile":
		return currentSite.TplTr("MyProfile")
	case "password":
		return currentSite.TplTr("ChangePassword")
	case "favorites":
		return currentSite.TplTr("MyFavorites")
	case "orders":
		return currentSite.TplTr("MyOrders")
	case "cart":
		return currentSite.TplTr("MyCart")
	case "sales":
		return currentSite.TplTr("SoldOrders")
	case "published":
		return currentSite.TplTr("MyPublished")
	case "publish":
		return currentSite.TplTr("PublishContent")
	case "reviews":
		return currentSite.TplTr("ReceivedReviews")
	case "verify":
		return currentSite.TplTr("RealNameVerification")
	case "openapi":
		return currentSite.TplTr("OpenApiAccess")
	case "comments":
		return currentSite.TplTr("MyComments")
	case "messages":
		return currentSite.TplTr("MyMessages")
	case "finance":
		return currentSite.TplTr("MyFinances")
	case "recharge":
		return currentSite.TplTr("AccountRecharge")
	case "withdraw":
		return currentSite.TplTr("BalanceWithdraw")
	case "points":
		return currentSite.TplTr("MyPoints")
	case "invite":
		return currentSite.TplTr("DistributionInvite")
	case "shop":
		return currentSite.TplTr("MyShop")
	}
	return currentSite.TplTr("MemberCenter")
}

func memberNav(currentSite *provider.Website) []memberNavView {
	list := make([]memberNavView, 0, len(memberNavEntries))
	for _, item := range memberNavEntries {
		link := "/member"
		if item.Route != "index" {
			link = "/member/" + item.Route
		}
		title := memberTitle(currentSite, item.Route)
		if item.Route == "index" {
			title = currentSite.TplTr("AccountOverview")
		}
		list = append(list, memberNavView{
			Route: item.Route,
			Title: title,
			Icon:  item.Icon,
			Link:  link,
		})
	}
	return list
}

// memberPages 允许访问的子页面，同时充当路由白名单
var memberPages = map[string]string{
	"index":     "member/index.html",
	"profile":   "member/profile.html",
	"password":  "member/password.html",
	"favorites": "member/favorites.html",
	"orders":    "member/orders.html",
	"cart":      "member/cart.html",
	"sales":     "member/sales.html",
	"published": "member/published.html",
	"publish":   "member/publish.html",
	"reviews":   "member/reviews.html",
	"verify":    "member/verify.html",
	"openapi":   "member/openapi.html",
	"comments":  "member/comments.html",
	"messages":  "member/messages.html",
	"finance":   "member/finance.html",
	"recharge":  "member/recharge.html",
	"withdraw":  "member/withdraw.html",
	"points":    "member/points.html",
	"invite":    "member/invite.html",
	"shop":      "member/shop.html",
}

type memberPagerLink struct {
	Page    int
	Link    string
	Current bool
}

type memberPager struct {
	Page      int
	PageSize  int
	Total     int64
	TotalPage int
	PrevLink  string
	NextLink  string
	Links     []memberPagerLink
}

// memberOrderView 订单 + 展示用状态文案
type memberOrderView struct {
	*model.Order
	StatusText  string
	StatusClass string
}

func newMemberPager(basePath string, query url.Values, page, pageSize int, total int64) *memberPager {
	if pageSize <= 0 {
		pageSize = memberPageSize
	}
	if page <= 0 {
		page = 1
	}
	totalPage := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPage < 1 {
		totalPage = 1
	}
	// 翻页链接保留其它查询条件（如订单状态）
	build := func(p int) string {
		q := url.Values{}
		for k, v := range query {
			q[k] = v
		}
		if p > 1 {
			q.Set("page", strconv.Itoa(p))
		} else {
			q.Del("page")
		}
		if len(q) == 0 {
			return basePath
		}
		return basePath + "?" + q.Encode()
	}

	pager := &memberPager{
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
		TotalPage: totalPage,
	}

	// 页码窗口：当前页前后各 2 页
	start := page - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > totalPage {
		end = totalPage
	}
	if end-start < 4 {
		start = end - 4
		if start < 1 {
			start = 1
		}
	}
	for p := start; p <= end; p++ {
		pager.Links = append(pager.Links, memberPagerLink{
			Page:    p,
			Link:    build(p),
			Current: p == page,
		})
	}
	if page > 1 {
		pager.PrevLink = build(page - 1)
	}
	if page < totalPage {
		pager.NextLink = build(page + 1)
	}

	return pager
}

func memberOrderStatus(currentSite *provider.Website, order *model.Order) (string, string) {
	switch order.Status {
	case config.OrderStatusCanceled:
		return currentSite.TplTr("OrderClosed"), "closed"
	case config.OrderStatusWaiting:
		return currentSite.TplTr("WaitingPayment"), "waiting"
	case config.OrderStatusPaid:
		return currentSite.TplTr("WaitingDelivery"), "paid"
	case config.OrderStatusDelivering:
		return currentSite.TplTr("WaitingReceive"), "delivering"
	case config.OrderStatusCompleted:
		return currentSite.TplTr("Completed"), "completed"
	case config.OrderStatusRefunding:
		return currentSite.TplTr("OrderRefunding"), "refunding"
	case config.OrderStatusRefunded:
		return currentSite.TplTr("OrderRefunded"), "refunded"
	}
	return currentSite.TplTr("OrderStatusText"), "waiting"
}

func memberOrderList(currentSite *provider.Website, userId uint, status string, page, pageSize int) ([]*memberOrderView, int64) {
	orders, total := currentSite.GetOrderList(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("`user_id` = ?", userId)
	}, status, page, pageSize)
	list := make([]*memberOrderView, 0, len(orders))
	for _, item := range orders {
		statusText, statusClass := memberOrderStatus(currentSite, item)
		list = append(list, &memberOrderView{
			Order:       item,
			StatusText:  statusText,
			StatusClass: statusClass,
		})
	}
	return list, total
}

// MemberPage 会员中心页面，route 为空时展示概览
func MemberPage(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)

	route := strings.TrimSpace(ctx.Params().Get("route"))
	route = strings.Trim(route, "/")
	route = strings.TrimSuffix(route, ".html")
	if route == "" {
		route = "index"
	}
	route = filepath.Clean(route)

	// 邀请二维码（图片，不走模板）
	if route == "invite-qrcode" {
		memberInviteQrcode(ctx, currentSite)
		return
	}

	tpl, exist := memberPages[route]
	if !exist {
		NotFound(ctx)
		return
	}

	userId := ctx.Values().GetUintDefault("userId", 0)
	if userId == 0 {
		redirect := "/member"
		if route != "index" {
			redirect = "/member/" + route
		}
		ctx.Redirect("/login?redirect=" + url.QueryEscape(redirect))
		return
	}

	userInfo, _ := ctx.Values().Get("userInfo").(*model.User)
	userGroup, _ := ctx.Values().Get("userGroup").(*model.UserGroup)
	if userInfo == nil {
		ctx.Redirect("/login?redirect=" + url.QueryEscape("/member"))
		return
	}

	if webInfo, ok := ctx.Value("webInfo").(*response.WebInfo); ok {
		webInfo.Title = memberTitle(currentSite, route)
		webInfo.PageName = "member"
		ctx.ViewData("webInfo", webInfo)
	}

	ctx.ViewData("currentRoute", route)
	ctx.ViewData("userInfo", userInfo)
	ctx.ViewData("userGroup", userGroup)
	ctx.ViewData("memberNav", memberNav(currentSite))

	switch route {
	case "index":
		favorites, favoriteTotal := currentSite.GetFavoriteList(int64(userInfo.Id), 1, 4)
		orders, orderTotal := memberOrderList(currentSite, userInfo.Id, "", 1, 3)
		ctx.ViewData("favoriteCount", favoriteTotal)
		ctx.ViewData("orderCount", orderTotal)
		ctx.ViewData("recentFavorites", favorites)
		ctx.ViewData("recentOrders", orders)
	case "favorites":
		page := ctx.URLParamIntDefault("page", 1)
		favorites, total := currentSite.GetFavoriteList(int64(userInfo.Id), page, memberPageSize)
		ctx.ViewData("favorites", favorites)
		ctx.ViewData("favoriteCount", total)
		ctx.ViewData("pager", newMemberPager("/member/favorites", url.Values{}, page, memberPageSize, total))
	case "orders":
		status := ctx.URLParamDefault("status", "")
		if !memberOrderStatusValid(status) {
			status = ""
		}
		page := ctx.URLParamIntDefault("page", 1)
		orders, total := memberOrderList(currentSite, userInfo.Id, status, page, memberPageSize)
		query := url.Values{}
		if status != "" {
			query.Set("status", status)
		}
		ctx.ViewData("orders", orders)
		ctx.ViewData("orderStatus", status)
		ctx.ViewData("pager", newMemberPager("/member/orders", query, page, memberPageSize, total))
	case "verify":
		memberVerifyPage(ctx, currentSite, userInfo)
	case "openapi":
		memberOpenApiPage(ctx, currentSite, userInfo)
	case "comments":
		memberCommentsPage(ctx, currentSite, userInfo)
	case "messages":
		memberMessagesPage(ctx, currentSite, userInfo)
	case "invite":
		memberInvitePage(ctx, currentSite, userInfo)
	case "withdraw":
		memberWithdrawPage(ctx, currentSite, userInfo)
	case "recharge":
		memberRechargePage(ctx, currentSite, userInfo)
	default:
		memberCommercePage(ctx, currentSite, route, userInfo)
	}

	if err := ctx.View(GetViewPath(ctx, tpl)); err != nil {
		ctx.StatusCode(iris.StatusNotFound)
		ctx.Values().Set("message", err.Error())
	}
}

func memberOrderStatusValid(status string) bool {
	switch status {
	case "", "waiting", "paid", "delivery", "finished", "refunding", "closed":
		return true
	}
	return false
}
