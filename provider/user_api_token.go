package provider

import (
	"errors"
	"strings"

	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
)

// 会员开放接口密钥（API Key）。
//
// 一把 Key 同时用于三处：REST（/api/user/* 等既有会员接口）、
// MCP（/api/open/mcp）与 CLI。Key 只代表「这个会员」，权限边界由
// 各接口自身按 userId 收口，不存在越权到后台的可能。

const (
	// MemberApiTokenLength 密钥长度（随机字母数字）
	MemberApiTokenLength = 40
	// MemberOpenApiBasePath 会员开放接口的 REST 前缀（复用既有 /api 会员接口）
	MemberOpenApiBasePath = "/api"
	// MemberMcpPath 会员专属 MCP 端点
	MemberMcpPath = "/api/open/mcp"
)

// GetUserApiToken 取会员当前的 Key（空表示未开通）
func (w *Website) GetUserApiToken(userId uint) string {
	if userId == 0 {
		return ""
	}
	var token string
	w.DB.Model(&model.User{}).Where("`id` = ?", userId).Pluck("api_token", &token)
	return token
}

// ResetUserApiToken 生成（或重新生成）Key，旧 Key 立即失效
func (w *Website) ResetUserApiToken(userId uint) (string, error) {
	if userId == 0 {
		return "", errors.New(w.TplTr("PleaseLogin"))
	}
	for i := 0; i < 5; i++ {
		token := config.GenerateRandString(MemberApiTokenLength)
		var exist int64
		w.DB.Model(&model.User{}).Where("`api_token` = ?", token).Count(&exist)
		if exist > 0 {
			continue
		}
		if err := w.DB.Model(&model.User{}).Where("`id` = ?", userId).
			UpdateColumn("api_token", token).Error; err != nil {
			return "", err
		}
		// 旧 Key 的 MCP 会话缓存作废
		invalidateMemberMcpCache(userId)
		return token, nil
	}
	return "", errors.New(w.TplTr("SystemBusyPleaseTryAgainLater"))
}

// ClearUserApiToken 停用 Key
func (w *Website) ClearUserApiToken(userId uint) error {
	if userId == 0 {
		return errors.New(w.TplTr("PleaseLogin"))
	}
	if err := w.DB.Model(&model.User{}).Where("`id` = ?", userId).
		UpdateColumn("api_token", "").Error; err != nil {
		return err
	}
	invalidateMemberMcpCache(userId)
	return nil
}

// GetUserByApiToken 按 Key 反查会员，Key 无效返回 nil
func (w *Website) GetUserByApiToken(token string) *model.User {
	token = strings.TrimSpace(token)
	if token == "" || len(token) < 16 {
		return nil
	}
	var user model.User
	if err := w.DB.Where("`api_token` = ?", token).Take(&user).Error; err != nil {
		return nil
	}
	if user.Status != config.UserStatusActive {
		return nil
	}
	return &user
}

// MaskApiToken 展示用掩码：abcd****wxyz
func MaskApiToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 10 {
		return token[:2] + "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
