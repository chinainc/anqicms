package controller

import (
	"strings"

	"github.com/kataras/iris/v12"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/provider"
)

// 会员实名 / 企业认证（提交 + 状态查询）

// memberAuthView 认证页展示用
type memberAuthView struct {
	*model.UserAuth
	StatusText      string
	StatusClass     string
	TypeText        string
	CreatedText     string
	AuditText       string
	IdCardMasked    string
	CompanyNoMasked string
	IdCardFrontUrl  string
	IdCardBackUrl   string
	IdCardHandUrl   string
	LicenceUrl      string
}

// maskSerial 证件号打码：保留前 4 位与后 4 位
func maskSerial(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

// memberVerifyPage 认证页：未提交为空，已提交展示状态与资料
func memberVerifyPage(ctx iris.Context, currentSite *provider.Website, userInfo *model.User) {
	auth := currentSite.GetUserAuthByUserId(userInfo.Id)
	var view *memberAuthView
	if auth != nil {
		view = &memberAuthView{
			UserAuth:        auth,
			StatusText:      memberAuthStatusText(currentSite, auth.Status),
			StatusClass:     memberAuthStatusClass(auth.Status),
			TypeText:        memberAuthTypeText(currentSite, auth.Type),
			CreatedText:     memberTimeText(auth.CreatedTime),
			AuditText:       memberTimeText(auth.AuditTime),
			IdCardMasked:    maskSerial(auth.IdCard),
			CompanyNoMasked: maskSerial(auth.CompanyNo),
			IdCardFrontUrl:  currentSite.StorageFullUrl(auth.IdCardFront),
			IdCardBackUrl:   currentSite.StorageFullUrl(auth.IdCardBack),
			IdCardHandUrl:   currentSite.StorageFullUrl(auth.IdCardHand),
			LicenceUrl:      currentSite.StorageFullUrl(auth.CompanyLicence),
		}
	}
	ctx.ViewData("userAuth", view)
	ctx.ViewData("verifyRealName", userInfo.RealName)
	// 只传可编辑的值，已通过时前端展示只读
	ctx.ViewData("authTypePerson", provider.UserAuthTypePerson)
	ctx.ViewData("authTypeCompany", provider.UserAuthTypeCompany)
	ctx.ViewData("authPending", provider.UserAuthStatusPending)
	ctx.ViewData("authPassed", provider.UserAuthStatusPassed)
	ctx.ViewData("authRejected", provider.UserAuthStatusRejected)
}

func memberAuthStatusText(currentSite *provider.Website, status int) string {
	switch status {
	case provider.UserAuthStatusPassed:
		return currentSite.TplTr("AuthPassed")
	case provider.UserAuthStatusRejected:
		return currentSite.TplTr("AuthRejected")
	}
	return currentSite.TplTr("AuthPending")
}

func memberAuthStatusClass(status int) string {
	switch status {
	case provider.UserAuthStatusPassed:
		return "completed"
	case provider.UserAuthStatusRejected:
		return "closed"
	}
	return "waiting"
}

func memberAuthTypeText(currentSite *provider.Website, authType int) string {
	if authType == provider.UserAuthTypeCompany {
		return currentSite.TplTr("AuthTypeCompany")
	}
	return currentSite.TplTr("AuthTypePerson")
}

// ApiUserAuthApply 提交 / 重新提交认证
func ApiUserAuthApply(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	var req struct {
		Type           int    `json:"type"`
		RealName       string `json:"real_name"`
		IdCard         string `json:"id_card"`
		IdCardFront    string `json:"id_card_front"`
		IdCardBack     string `json:"id_card_back"`
		IdCardHand     string `json:"id_card_hand"`
		CompanyName    string `json:"company_name"`
		CompanyNo      string `json:"company_no"`
		CompanyLegal   string `json:"company_legal"`
		CompanyLicence string `json:"company_licence"`
		Phone          string `json:"phone"`
	}
	if err := ctx.ReadJSON(&req); err != nil {
		req.Type = ctx.PostValueIntDefault("type", provider.UserAuthTypePerson)
		req.RealName = memberPostValue(ctx, "real_name")
		req.IdCard = memberPostValue(ctx, "id_card")
		req.IdCardFront = memberPostValue(ctx, "id_card_front")
		req.IdCardBack = memberPostValue(ctx, "id_card_back")
		req.IdCardHand = memberPostValue(ctx, "id_card_hand")
		req.CompanyName = memberPostValue(ctx, "company_name")
		req.CompanyNo = memberPostValue(ctx, "company_no")
		req.CompanyLegal = memberPostValue(ctx, "company_legal")
		req.CompanyLicence = memberPostValue(ctx, "company_licence")
		req.Phone = memberPostValue(ctx, "phone")
	}
	item, err := currentSite.ApplyUserAuth(apiCommerceUser(ctx), &provider.UserAuthRequest{
		Type:           req.Type,
		RealName:       req.RealName,
		IdCard:         req.IdCard,
		IdCardFront:    req.IdCardFront,
		IdCardBack:     req.IdCardBack,
		IdCardHand:     req.IdCardHand,
		CompanyName:    req.CompanyName,
		CompanyNo:      req.CompanyNo,
		CompanyLegal:   req.CompanyLegal,
		CompanyLicence: req.CompanyLicence,
		Phone:          req.Phone,
	})
	if err != nil {
		apiCommerceFail(ctx, err.Error())
		return
	}
	apiCommerceOK(ctx, item)
}

// ApiUserAuthInfo 当前认证状态（给前端轮询 / 其它端复用）
func ApiUserAuthInfo(ctx iris.Context) {
	currentSite := provider.CurrentSite(ctx)
	auth := currentSite.GetUserAuthByUserId(apiCommerceUser(ctx))
	if auth == nil {
		apiCommerceOK(ctx, iris.Map{"status": -2})
		return
	}
	apiCommerceOK(ctx, iris.Map{
		"status":       auth.Status,
		"type":         auth.Type,
		"status_text":  memberAuthStatusText(currentSite, auth.Status),
		"remark":       auth.Remark,
		"audit_time":   auth.AuditTime,
		"real_name":    strings.TrimSpace(auth.RealName),
		"company_name": strings.TrimSpace(auth.CompanyName),
		"id":           auth.Id,
	})
}
