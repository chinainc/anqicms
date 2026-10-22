package controller

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/skip2/go-qrcode"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/provider"
)

// 会员中心：评论管理 / 留言管理 / 分销邀请 / 余额提现 / 账户充值
//
// 页面都是服务端渲染（与会员中心其它页一致），按钮操作走 /api/user/* 接口，
// 成功后前端整页刷新即可（数据量小，不做局部渲染）。

// ---------------------------------------------------------------- 展示用结构

// memberCommentView 评论 + 状态文案
type memberCommentView struct {
	*provider.MemberCommentItem
	StatusText  string
	StatusClass string
	CreatedText string
}

// memberMessageView 留言 + 状态文案
type memberMessageView struct {
	*model.Guestbook
	StatusText  string
	StatusClass string
	CreatedText string
}

// memberWithdrawView 提现记录 + 状态文案
type memberWithdrawView struct {
	*model.UserWithdraw
	StatusText  string
	StatusClass string
	CreatedText string
	WayText     string
}

// memberRechargeView 充值申请 + 状态文案
type memberRechargeView struct {
	*model.Recharge
	StatusText  string
	StatusClass string
	CreatedText string
}

func memberTimeText(stamp int64) string {
	if stamp <= 0 {
		return ""
	}
	return time.Unix(stamp, 0).Format("2006-01-02 15:04")
}

// memberCommentStatus 评论状态：0 未处理 1 正常 2 疑似垃圾 3 垃圾
func memberCommentStatus(currentSite *provider.Website, status uint) (string, string) {
	switch status {
	case 1:
		return currentSite.TplTr("CommentApproved"), "completed"
	case 3:
		return currentSite.TplTr("CommentHidden"), "closed"
	case 2:
		return currentSite.TplTr("CommentSpam"), "closed"
	}
	return currentSite.TplTr("CommentPending"), "waiting"
}

// memberMessageStatus 留言状态：0 未处理 1 正常 2 垃圾
func memberMessageStatus(currentSite *provider.Website, status int) (string, string) {
	switch status {
	case 1:
		return currentSite.TplTr("CommentApproved"), "completed"
	case 2:
		return currentSite.TplTr("CommentSpam"), "closed"
	}
	return currentSite.TplTr("CommentPending"), "waiting"
}

// memberWithdrawStatus 提现状态：0 等待处理 1 已同意 2 已提现 -1 提现错误
func memberWithdrawStatus(currentSite *provider.Website, status int) (string, string) {
	switch status {
	case config.WithdrawStatusAgree:
		return currentSite.TplTr("WithdrawAgreed"), "waiting"
	case config.WithdrawStatusFinished:
		return currentSite.TplTr("WithdrawFinished"), "completed"
	case config.WithdrawStatusCanceled:
		return currentSite.TplTr("WithdrawCanceled"), "closed"
	}
	return currentSite.TplTr("WithdrawWaiting"), "waiting"
}

func memberRechargeStatus(currentSite *provider.Website, status int) (string, string) {
	switch status {
	case 1:
		return currentSite.TplTr("RechargeConfirmed"), "completed"
	case -1:
		return currentSite.TplTr("WithdrawCanceled"), "closed"
	}
	return currentSite.TplTr("RechargePending"), "waiting"
}

func memberWithdrawWayText(currentSite *provider.Website, way int) string {
	switch way {
	case provider.WithdrawWayBank:
		return currentSite.TplTr("WithdrawWayBank")
	case provider.WithdrawWayWechat:
		return currentSite.TplTr("WithdrawWayWechat")
	}
	return currentSite.TplTr("WithdrawWayAlipay")
}

// ---------------------------------------------------------------- 页面数据

func memberCommentsPage(ctx iris.Context, currentSite *provider.Website, userInfo *model.User) {
	status := ctx.URLParamIntDefault("status", -1)
	if status > 3 {
		status = -1
	}
	page := ctx.URLParamIntDefault("page", 1)
	list, total := currentSite.GetUserArchiveComments(userInfo.Id, status, page, memberPageSize)
	pending, all := currentSite.GetUserCommentStat(userInfo.Id)

	views := make([]*memberCommentView, 0, len(list))
	for _, item := range list {
		statusText, statusClass := memberCommentStatus(currentSite, item.Status)
		views = append(views, &memberCommentView{
			MemberCommentItem: item,
			StatusText:        statusText,
			StatusClass:       statusClass,
			CreatedText:       memberTimeText(item.CreatedTime),
		})
	}

	query := url.Values{}
	if status >= 0 {
		query.Set("status", strconv.Itoa(status))
	}
	ctx.ViewData("comments", views)
	ctx.ViewData("commentStatus", status)
	ctx.ViewData("pendingCount", pending)
	ctx.ViewData("commentTotal", all)
	ctx.ViewData("pager", newMemberPager("/member/comments", query, page, memberPageSize, total))
}

func memberMessagesPage(ctx iris.Context, currentSite *provider.Website, userInfo *model.User) {
	page := ctx.URLParamIntDefault("page", 1)
	list, total := currentSite.GetUserGuestbooks(userInfo.Id, page, memberPageSize)
	views := make([]*memberMessageView, 0, len(list))
	for _, item := range list {
		statusText, statusClass := memberMessageStatus(currentSite, item.Status)
		views = append(views, &memberMessageView{
			Guestbook:   item,
			StatusText:  statusText,
			StatusClass: statusClass,
			CreatedText: memberTimeText(item.CreatedTime),
		})
	}
	ctx.ViewData("messages", views)
	ctx.ViewData("messageTotal", total)
	ctx.ViewData("pager", newMemberPager("/member/messages", url.Values{}, page, memberPageSize, total))
}

func memberInvitePage(ctx iris.Context, currentSite *provider.Website, userInfo *model.User) {
	info := currentSite.GetMemberInviteInfo(userInfo.Id)
	members := currentSite.GetMemberInviteMembers(userInfo.Id, 20)
	ctx.ViewData("invite", info)
	ctx.ViewData("inviteMembers", members)
}

func memberWithdrawPage(ctx iris.Context, currentSite *provider.Website, userInfo *model.User) {
	page := ctx.URLParamIntDefault("page", 1)
	list, total := currentSite.GetUserWithdraws(userInfo.Id, page, memberPageSize)
	views := make([]*memberWithdrawView, 0, len(list))
	for _, item := range list {
		statusText, statusClass := memberWithdrawStatus(currentSite, item.Status)
		views = append(views, &memberWithdrawView{
			UserWithdraw: item,
			StatusText:   statusText,
			StatusClass:  statusClass,
			CreatedText:  memberTimeText(item.CreatedTime),
			WayText:      memberWithdrawWayText(currentSite, item.WithdrawWay),
		})
	}
	ctx.ViewData("withdraws", views)
	ctx.ViewData("withdrawTotal", total)
	ctx.ViewData("pager", newMemberPager("/member/withdraw", url.Values{}, page, memberPageSize, total))
}

func memberRechargePage(ctx iris.Context, currentSite *provider.Website, userInfo *model.User) {
	page := ctx.URLParamIntDefault("page", 1)
	list, total := currentSite.GetUserRecharges(userInfo.Id, page, memberPageSize)
	views := make([]*memberRechargeView, 0, len(list))
	for _, item := range list {
		statusText, statusClass := memberRechargeStatus(currentSite, item.Status)
		views = append(views, &memberRechargeView{
			Recharge:    item,
			StatusText:  statusText,
			StatusClass: statusClass,
			CreatedText: memberTimeText(item.CreatedTime),
		})
	}
	ctx.ViewData("recharges", views)
	ctx.ViewData("rechargeTotal", total)
	ctx.ViewData("pager", newMemberPager("/member/recharge", url.Values{}, page, memberPageSize, total))
}

// memberInviteQrcode 输出邀请链接二维码（PNG）
func memberInviteQrcode(ctx iris.Context, currentSite *provider.Website) {
	userId := ctx.Values().GetUintDefault("userId", 0)
	if userId == 0 {
		ctx.StatusCode(iris.StatusNotFound)
		return
	}
	info := currentSite.GetMemberInviteInfo(userId)
	if info.InviteLink == "" {
		ctx.StatusCode(iris.StatusNotFound)
		return
	}
	png, err := qrcode.Encode(info.InviteLink, qrcode.Medium, 360)
	if err != nil {
		ctx.StatusCode(iris.StatusInternalServerError)
		return
	}
	ctx.Header("Content-Type", "image/png")
	ctx.Header("Cache-Control", "no-store")
	_, _ = ctx.Write(png)
}

// ---------------------------------------------------------------- 接口

// memberPostValue 同时兼容 JSON 与表单提交
func memberPostValue(ctx iris.Context, key string) string {
	if val := ctx.PostValueTrim(key); val != "" {
		return val
	}
	return ""
}

// ApiUserCommentStatus 会员审核/隐藏自己文档下的评论
func ApiUserCommentStatus(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Id     uint `json:"id"`
		Status int  `json:"status"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Id = uint(ctx.PostValueIntDefault("id", 0))
		req.Status = ctx.PostValueIntDefault("status", 0)
	}
	if req.Status != 0 && req.Status != 1 && req.Status != 3 {
		req.Status = 1
	}
	comment, err := currentSite.GetUserArchiveComment(apiCommerceUser(ctx), req.Id)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	comment.Status = uint(req.Status)
	if err = comment.Save(currentSite.DB); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{"id": comment.Id, "status": comment.Status})
}

// ApiUserCommentDelete 会员删除自己文档下的评论
func ApiUserCommentDelete(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Id uint `json:"id"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Id = uint(ctx.PostValueIntDefault("id", 0))
	}
	comment, err := currentSite.GetUserArchiveComment(apiCommerceUser(ctx), req.Id)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	if err = comment.Delete(currentSite.DB); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{"id": comment.Id})
}

// ApiUserCommentReply 会员回复自己文档下的评论
func ApiUserCommentReply(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Id      uint   `json:"id"`
		Content string `json:"content"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Id = uint(ctx.PostValueIntDefault("id", 0))
		req.Content = memberPostValue(ctx, "content")
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		apiCommerceFail(ctx, currentSite.TplTr("PleaseEnterTheContent"))
		return
	}
	comment, err := currentSite.GetUserArchiveComment(apiCommerceUser(ctx), req.Id)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	userInfo, _ := ctx.Values().Get("userInfo").(*model.User)
	if userInfo == nil {
		apiCommerceFail(ctx, currentSite.TplTr("PleaseLogin"))
		return
	}
	reply := &model.Comment{
		ArchiveId: comment.ArchiveId,
		UserId:    userInfo.Id,
		UserName:  userInfo.UserName,
		Content:   req.Content,
		ParentId:  comment.Id,
		ToUid:     comment.UserId,
		Status:    1,
		Ip:        ctx.RemoteAddr(),
	}
	if err = currentSite.DB.Create(reply).Error; err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	// 回复也要让文档评论数保持准确
	reply.UpdateCommentCount(currentSite.DB)
	apiCommerceOK(ctx, iris.Map{"id": reply.Id})
}

// ApiUserArchiveDetail 我的一条内容详情（含正文；草稿也能取，仅限本人）
func ApiUserArchiveDetail(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	id := ctx.URLParamInt64Default("id", 0)
	userId := apiCommerceUser(ctx)
	owner, exist := currentSite.GetArchiveOwner(id)
	if !exist {
		apiCommerceFail(ctx, currentSite.TplTr("DocumentDoesNotExist"))
		return
	}
	if owner != userId {
		apiCommerceFail(ctx, currentSite.TplTr("NoPermissionToOperateThisContent"))
		return
	}
	out := iris.Map{"id": id, "draft": false}
	if archive, err := currentSite.GetArchiveById(id); err == nil && archive != nil {
		out["title"] = archive.Title
		out["keywords"] = archive.Keywords
		out["description"] = archive.Description
		out["category_id"] = archive.CategoryId
		out["module_id"] = archive.ModuleId
		out["url_token"] = archive.UrlToken
		out["created_time"] = archive.CreatedTime
		out["link"] = currentSite.GetUrl("archive", archive, 0)
	} else if draft, err := currentSite.GetArchiveDraftById(id); err == nil && draft != nil {
		out["draft"] = true
		out["title"] = draft.Title
		out["keywords"] = draft.Keywords
		out["description"] = draft.Description
		out["category_id"] = draft.CategoryId
		out["module_id"] = draft.ModuleId
		out["url_token"] = draft.UrlToken
		out["created_time"] = draft.CreatedTime
	}
	if data, err := currentSite.GetArchiveDataById(id); err == nil && data != nil {
		out["content"] = data.Content
	}
	if tags := currentSite.GetTagsByItemId(id); len(tags) > 0 {
		names := make([]string, 0, len(tags))
		for _, tag := range tags {
			if tag != nil && tag.Title != "" {
				names = append(names, tag.Title)
			}
		}
		out["tags"] = names
	}
	apiCommerceOK(ctx, out)
}

// ApiUserComments 我的内容收到的评论（开放接口 / CLI 使用）
func ApiUserComments(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("page_size", 20)
	if pageSize > 50 {
		pageSize = 50
	}
	status := -1
	switch ctx.URLParamDefault("status", "all") {
	case "pending":
		status = 0
	case "approved":
		status = 1
	case "hidden":
		status = 3
	}
	list, total := currentSite.GetUserArchiveComments(apiCommerceUser(ctx), status, page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// ApiUserGuestbooks 我的留言列表（开放接口 / CLI 使用）
func ApiUserGuestbooks(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	page := ctx.URLParamIntDefault("page", 1)
	pageSize := ctx.URLParamIntDefault("page_size", 20)
	if pageSize > 50 {
		pageSize = 50
	}
	list, total := currentSite.GetUserGuestbooks(apiCommerceUser(ctx), page, pageSize)
	apiCommerceOK(ctx, iris.Map{"list": list, "total": total, "page": page, "page_size": pageSize})
}

// ApiUserGuestbookDelete 删除自己的留言
func ApiUserGuestbookDelete(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Id uint `json:"id"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Id = uint(ctx.PostValueIntDefault("id", 0))
	}
	if err := currentSite.DeleteUserGuestbook(apiCommerceUser(ctx), req.Id); err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, iris.Map{"id": req.Id})
}

// ApiUserWithdrawApply 申请余额提现（金额单位：分）
func ApiUserWithdrawApply(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Amount   int64  `json:"amount"`
		Way      int    `json:"way"`
		Account  string `json:"account"`
		RealName string `json:"real_name"`
		Remark   string `json:"remark"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Amount = ctx.PostValueInt64Default("amount", 0)
		req.Way = ctx.PostValueIntDefault("way", 0)
		req.Account = memberPostValue(ctx, "account")
		req.RealName = memberPostValue(ctx, "real_name")
		req.Remark = memberPostValue(ctx, "remark")
	}
	withdraw, err := currentSite.ApplyBalanceWithdraw(apiCommerceUser(ctx), req.Amount, req.Way, req.Account, req.RealName, req.Remark)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, withdraw)
}

// ApiUserRechargeApply 提交充值申请（金额单位：分）
func ApiUserRechargeApply(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Amount int64  `json:"amount"`
		PayWay string `json:"pay_way"`
		Remark string `json:"remark"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Amount = ctx.PostValueInt64Default("amount", 0)
		req.PayWay = memberPostValue(ctx, "pay_way")
		req.Remark = memberPostValue(ctx, "remark")
	}
	recharge, err := currentSite.ApplyRecharge(apiCommerceUser(ctx), req.Amount, req.PayWay, req.Remark)
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, recharge)
}
