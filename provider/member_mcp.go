package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gorm.io/gorm"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/pkg/mcp/server"
	"kandaoni.com/anqicms/request"
)

// 会员专属 MCP 服务。
//
// 与后台 MCP（provider.NewMcpServer）完全隔离：后台那套是「意图内核 + 全站管理能力」，
// 这里只暴露**当前会员自己**的数据与操作，handler 里所有查询都强制带 userId，
// 不存在通过参数越权到别人内容的可能。
//
// 一个会员一把 API Key → 一个 MCP Server 实例（工具集随 Key 绑定），
// StreamableHTTPHandler 按 (站点, 会员, Key) 缓存，Key 重置时清理。

const memberMcpMaxCache = 200

var memberMcpHolder = struct {
	sync.Mutex
	handlers map[string]*mcp.StreamableHTTPHandler
}{handlers: map[string]*mcp.StreamableHTTPHandler{}}

// MemberMcpHandler 取（或构建）会员的 MCP HTTP handler
func MemberMcpHandler(w *Website, user *model.User, token string) (http.Handler, error) {
	if w == nil || user == nil {
		return nil, errors.New(w.TplTr("MemberMcpErrSiteMissing"))
	}
	suffix := token
	if len(suffix) > 8 {
		suffix = suffix[len(suffix)-8:]
	}
	key := fmt.Sprintf("%d|%d|%s", w.Id, user.Id, suffix)

	memberMcpHolder.Lock()
	if h, ok := memberMcpHolder.handlers[key]; ok {
		memberMcpHolder.Unlock()
		return h, nil
	}
	memberMcpHolder.Unlock()

	handler, err := buildMemberMcpHandler(w, user)
	if err != nil {
		return nil, err
	}

	memberMcpHolder.Lock()
	if len(memberMcpHolder.handlers) >= memberMcpMaxCache {
		memberMcpHolder.handlers = map[string]*mcp.StreamableHTTPHandler{}
	}
	memberMcpHolder.handlers[key] = handler
	memberMcpHolder.Unlock()

	return handler, nil
}

// invalidateMemberMcpCache Key 变更/停用后作废该会员的缓存
func invalidateMemberMcpCache(userId uint) {
	if userId == 0 {
		return
	}
	needle := fmt.Sprintf("|%d|", userId)
	memberMcpHolder.Lock()
	for key := range memberMcpHolder.handlers {
		if strings.Contains(key, needle) {
			delete(memberMcpHolder.handlers, key)
		}
	}
	memberMcpHolder.Unlock()
}

// buildMemberMcpHandler 组装会员工具集
func buildMemberMcpHandler(w *Website, user *model.User) (*mcp.StreamableHTTPHandler, error) {
	cfg := server.DefaultConfig()
	cfg.ServerName = "AnQiCMS-Member"
	cfg.Instructions = w.TplTr("MemberMcpInstructions")

	srv, err := server.New(cfg)
	if err != nil {
		return nil, err
	}
	userId := user.Id
	tools := memberMcpTools(w, userId)
	for _, td := range tools {
		if err := srv.AddTool(td.Tool, td.Handler); err != nil {
			return nil, err
		}
	}

	// StreamableHTTPHandler 的 session 绑定在 handler 上，返回前已按 key 缓存
	h := srv.StreamableHTTPHandler()
	httpHandler, ok := h.(*mcp.StreamableHTTPHandler)
	if !ok {
		return nil, errors.New(w.TplTr("MemberMcpErrHandlerType"))
	}
	return httpHandler, nil
}

// ─────────────────────────────────────────── 工具定义

func memberMcpTools(w *Website, userId uint) []server.ToolDef {
	text := func(v any) (*mcp.CallToolResult, error) {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return mcpError(w.TplTr("MemberMcpErrMarshal") + err.Error())
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
		}, nil
	}

	// args 转 map（CallToolRequest 的入参是 json.RawMessage）
	parse := func(req *mcp.CallToolRequest) map[string]any {
		out := map[string]any{}
		if req != nil && len(req.Params.Arguments) > 0 {
			_ = json.Unmarshal(req.Params.Arguments, &out)
		}
		return out
	}

	defs := make([]server.ToolDef, 0, 11)

	// 1. 账号资料
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "member_profile",
			Title:       w.TplTr("MemberMcpProfileTitle"),
			Description: w.TplTr("MemberMcpProfileDesc"),
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			info, err := w.GetUserInfoById(userId)
			if err != nil {
				return mcpError(err.Error())
			}
			group, _ := w.GetUserGroupInfo(info.GroupId)
			out := map[string]any{
				"id":           info.Id,
				"user_name":    info.UserName,
				"real_name":    info.RealName,
				"email":        info.Email,
				"phone":        info.Phone,
				"balance":      info.Balance,
				"balance_text": fmt.Sprintf("%.2f", float64(info.Balance)/100),
				"points":       info.Points,
				"invite_code":  info.InviteCode,
				"created_time": info.CreatedTime,
			}
			if group != nil {
				out["group"] = group.Title
			}
			if auth := w.GetUserAuthByUserId(userId); auth != nil {
				out["verify_status"] = auth.Status
				out["verify_type"] = auth.Type
			} else {
				out["verify_status"] = -2
			}
			return text(out)
		},
	})

	// 2. 我的内容列表
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "content_list",
			Title:       w.TplTr("MemberMcpContentListTitle"),
			Description: w.TplTr("MemberMcpContentListDesc"),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPageFrom1")},
					"page_size": map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPageSizeMax")},
				},
			},
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			page := intArg(args, "page", 1)
			size := intArg(args, "page_size", 20)
			if size > 50 {
				size = 50
			}
			items, total := w.GetUserPublished(userId, page, size)
			list := make([]map[string]any, 0, len(items))
			for _, item := range items {
				row := map[string]any{
					"id":           item.Id,
					"title":        item.Title,
					"draft":        item.Draft,
					"status":       item.Status,
					"created_time": item.CreatedTime,
					"views":        item.Views,
					"link":         item.Link,
				}
				list = append(list, row)
			}
			return text(map[string]any{"total": total, "page": page, "page_size": size, "list": list})
		},
	})

	// 3. 内容详情
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "content_get",
			Title:       w.TplTr("MemberMcpContentGetTitle"),
			Description: w.TplTr("MemberMcpContentGetDesc"),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamId")},
				},
				"required": []string{"id"},
			},
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			id := int64Arg(args, "id")
			if archive, err := w.GetArchiveById(id); err == nil && archive != nil {
				if archive.UserId != userId {
					return mcpError(w.TplTr("NoPermissionToOperateThisContent"))
				}
				return text(memberArchivePayload(w, archive))
			}
			if draft, err := w.GetArchiveDraftById(id); err == nil && draft != nil {
				if draft.UserId != userId || draft.Status == config.ContentStatusDelete {
					return mcpError(w.TplTr("NoPermissionToOperateThisContent"))
				}
				return text(memberArchivePayload(w, &draft.Archive))
			}
			return mcpError(w.TplTr("DocumentDoesNotExist"))
		},
	})

	// 4. 发布内容
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "content_create",
			Title:       w.TplTr("MemberMcpContentCreateTitle"),
			Description: w.TplTr("MemberMcpContentCreateDesc"),
			InputSchema: memberContentSchema(w, false),
			Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(false)},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			saveReq := memberBuildSaveRequest(w, userId, 0, args, false)
			if saveReq.Title == "" {
				return mcpError(w.TplTr("EnterTheTitle"))
			}
			if saveReq.CategoryId == 0 {
				return mcpError(w.TplTr("PleaseSelectCategory"))
			}
			archive, err := w.SaveArchive(saveReq)
			if err != nil {
				return mcpError(err.Error())
			}
			archive.Link = w.GetUrl("archive", archive, 0)
			return text(map[string]any{
				"id":    archive.Id,
				"title": archive.Title,
				"draft": saveReq.Draft,
				"link":  archive.Link,
			})
		},
	})

	// 5. 修改内容
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "content_update",
			Title:       w.TplTr("MemberMcpContentUpdateTitle"),
			Description: w.TplTr("MemberMcpContentUpdateDesc"),
			InputSchema: memberContentSchema(w, true),
			Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(false), IdempotentHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			id := int64Arg(args, "id")
			if id <= 0 {
				return mcpError(w.TplTr("DocumentDoesNotExist"))
			}
			if owner, exist := w.GetArchiveOwner(id); !exist || owner != userId {
				return mcpError(w.TplTr("NoPermissionToOperateThisContent"))
			}
			saveReq := memberBuildSaveRequest(w, userId, id, args, true)
			archive, err := w.SaveArchive(saveReq)
			if err != nil {
				return mcpError(err.Error())
			}
			archive.Link = w.GetUrl("archive", archive, 0)
			return text(map[string]any{
				"id":    archive.Id,
				"title": archive.Title,
				"link":  archive.Link,
			})
		},
	})

	// 6. 删除内容
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "content_delete",
			Title:       w.TplTr("MemberMcpContentDeleteTitle"),
			Description: w.TplTr("MemberMcpContentDeleteDesc"),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamId")},
				},
				"required": []string{"id"},
			},
			Annotations: &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			id := int64Arg(args, "id")
			owner, exist := w.GetArchiveOwner(id)
			if !exist {
				return mcpError(w.TplTr("DocumentDoesNotExist"))
			}
			if owner != userId {
				return mcpError(w.TplTr("NoPermissionToOperateThisContent"))
			}
			var err error
			if archive, getErr := w.GetArchiveById(id); getErr == nil && archive != nil {
				err = w.DeleteArchive(archive)
			} else if draft, getErr := w.GetArchiveDraftById(id); getErr == nil && draft != nil {
				err = w.DeleteArchiveDraft(draft)
			}
			if err != nil {
				return mcpError(err.Error())
			}
			return text(map[string]any{"id": id, "deleted": true})
		},
	})

	// 7. 评论列表
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "comment_list",
			Title:       w.TplTr("MemberMcpCommentListTitle"),
			Description: w.TplTr("MemberMcpCommentListDesc"),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPage")},
					"page_size": map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPageSize")},
					"status":    map[string]any{"type": "string", "enum": []string{"all", "pending", "approved", "hidden"}, "description": w.TplTr("MemberMcpParamStatusAll")},
				},
			},
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			page := intArg(args, "page", 1)
			size := intArg(args, "page_size", 20)
			if size > 50 {
				size = 50
			}
			status := -1
			switch strArg(args, "status") {
			case "pending":
				status = 0
			case "approved":
				status = 1
			case "hidden":
				status = 3
			}
			items, total := w.GetUserArchiveComments(userId, status, page, size)
			list := make([]map[string]any, 0, len(items))
			for _, item := range items {
				list = append(list, map[string]any{
					"id":            item.Id,
					"user_name":     item.UserName,
					"content":       item.Content,
					"status":        item.Status,
					"archive_id":    item.ArchiveId,
					"archive_title": item.ArchiveTitle,
					"created_time":  item.CreatedTime,
				})
			}
			return text(map[string]any{"total": total, "page": page, "page_size": size, "list": list})
		},
	})

	// 8. 回复评论
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "comment_reply",
			Title:       w.TplTr("MemberMcpCommentReplyTitle"),
			Description: w.TplTr("MemberMcpCommentReplyDesc"),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":      map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamCommentId")},
					"content": map[string]any{"type": "string", "description": w.TplTr("MemberMcpParamReplyContent")},
				},
				"required": []string{"id", "content"},
			},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			id := int64Arg(args, "id")
			content := strings.TrimSpace(strArg(args, "content"))
			if content == "" {
				return mcpError(w.TplTr("PleaseEnterTheContent"))
			}
			comment, err := w.GetUserArchiveComment(userId, uint(id))
			if err != nil {
				return mcpError(err.Error())
			}
			userInfo, _ := w.GetUserInfoById(userId)
			name := ""
			if userInfo != nil {
				name = userInfo.UserName
			}
			reply := &model.Comment{
				ArchiveId: comment.ArchiveId,
				UserId:    userId,
				UserName:  name,
				Content:   content,
				ParentId:  comment.Id,
				ToUid:     comment.UserId,
				Status:    1,
				Ip:        "mcp",
			}
			if err := w.DB.Create(reply).Error; err != nil {
				return mcpError(err.Error())
			}
			reply.UpdateCommentCount(w.DB)
			return text(map[string]any{"id": reply.Id, "parent_id": comment.Id})
		},
	})

	// 9. 我的留言
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "guestbook_list",
			Title:       w.TplTr("MemberMcpGuestbookTitle"),
			Description: w.TplTr("MemberMcpGuestbookDesc"),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPage")},
					"page_size": map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPageSize")},
				},
			},
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			page := intArg(args, "page", 1)
			size := intArg(args, "page_size", 20)
			if size > 50 {
				size = 50
			}
			items, total := w.GetUserGuestbooks(userId, page, size)
			list := make([]map[string]any, 0, len(items))
			for _, item := range items {
				list = append(list, map[string]any{
					"id":           item.Id,
					"content":      item.Content,
					"contact":      item.Contact,
					"status":       item.Status,
					"created_time": item.CreatedTime,
				})
			}
			return text(map[string]any{"total": total, "page": page, "page_size": size, "list": list})
		},
	})

	// 10. 我的订单
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "order_list",
			Title:       w.TplTr("MemberMcpOrderTitle"),
			Description: w.TplTr("MemberMcpOrderDesc"),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"page":      map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPage")},
					"page_size": map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPageSize")},
					"status":    map[string]any{"type": "string", "description": w.TplTr("MemberMcpParamOrderStatus")},
				},
			},
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := parse(req)
			page := intArg(args, "page", 1)
			size := intArg(args, "page_size", 20)
			if size > 50 {
				size = 50
			}
			orders, total := w.GetOrderList(func(tx *gorm.DB) *gorm.DB {
				return tx.Where("`user_id` = ?", userId)
			}, strArg(args, "status"), page, size)
			list := make([]map[string]any, 0, len(orders))
			for _, item := range orders {
				list = append(list, map[string]any{
					"order_id":     item.OrderId,
					"title":        memberOrderTitle(item),
					"amount":       item.Amount,
					"status":       item.Status,
					"created_time": item.CreatedTime,
				})
			}
			return text(map[string]any{"total": total, "page": page, "page_size": size, "list": list})
		},
	})

	// 11. 资金概览
	defs = append(defs, server.ToolDef{
		Tool: &mcp.Tool{
			Name:        "wallet_info",
			Title:       w.TplTr("MemberMcpWalletTitle"),
			Description: w.TplTr("MemberMcpWalletDesc"),
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
		},
		Handler: func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			info, err := w.GetUserInfoById(userId)
			if err != nil {
				return mcpError(err.Error())
			}
			invite := w.GetMemberInviteInfo(userId)
			withdraws, _ := w.GetUserWithdraws(userId, 1, 5)
			finances, _ := w.GetFinanceLogs(userId, 1, 5)
			wdList := make([]map[string]any, 0, len(withdraws))
			for _, item := range withdraws {
				wdList = append(wdList, map[string]any{
					"id":           item.Id,
					"amount":       item.Amount,
					"status":       item.Status,
					"created_time": item.CreatedTime,
				})
			}
			finList := make([]map[string]any, 0, len(finances))
			for _, item := range finances {
				finList = append(finList, map[string]any{
					"id":           item.Id,
					"direction":    item.Direction,
					"amount":       item.Amount,
					"action":       item.Action,
					"remark":       item.Remark,
					"created_time": item.CreatedTime,
				})
			}
			return text(map[string]any{
				"balance":         info.Balance,
				"balance_text":    fmt.Sprintf("%.2f", float64(info.Balance)/100),
				"points":          info.Points,
				"commission_wait": invite.CommissionWait,
				"commission_paid": invite.CommissionPaid,
				"withdraws":       wdList,
				"recent_finances": finList,
			})
		},
	})

	return defs
}

// ─────────────────────────────────────────── 辅助

func mcpError(msg string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "错误：" + msg}},
		IsError: true,
	}, nil
}

func boolPtr(v bool) *bool { return &v }

func strArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok2 := v.(string); ok2 {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func intArg(args map[string]any, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return int(i)
			}
		}
	}
	return def
}

func int64Arg(args map[string]any, key string) int64 {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case json.Number:
			if i, err := n.Int64(); err == nil {
				return i
			}
		}
	}
	return 0
}

func strSliceArg(args map[string]any, key string) []string {
	out := []string{}
	if v, ok := args[key]; ok {
		if arr, ok2 := v.([]any); ok2 {
			for _, item := range arr {
				if s, ok3 := item.(string); ok3 && strings.TrimSpace(s) != "" {
					out = append(out, strings.TrimSpace(s))
				}
			}
		}
		if arr, ok2 := v.([]string); ok2 {
			out = arr
		}
	}
	return out
}

// memberContentSchema 发布/修改内容的入参 schema
func memberContentSchema(w *Website, withId bool) map[string]any {
	props := map[string]any{
		"title":       map[string]any{"type": "string", "description": w.TplTr("MemberMcpParamTitle")},
		"category_id": map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamCategory")},
		"module_id":   map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamModule")},
		"content":     map[string]any{"type": "string", "description": w.TplTr("MemberMcpParamContent")},
		"description": map[string]any{"type": "string", "description": w.TplTr("MemberMcpParamDescription")},
		"keywords":    map[string]any{"type": "string", "description": w.TplTr("MemberMcpParamKeywords")},
		"url_token":   map[string]any{"type": "string", "description": w.TplTr("MemberMcpParamUrlToken")},
		"images":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": w.TplTr("MemberMcpParamImages")},
		"tags":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": w.TplTr("MemberMcpParamTags")},
		"draft":       map[string]any{"type": "boolean", "description": w.TplTr("MemberMcpParamDraft")},
		"price":       map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamPrice")},
		"stock":       map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamStock")},
	}
	schema := map[string]any{"type": "object", "properties": props}
	if withId {
		props["id"] = map[string]any{"type": "integer", "description": w.TplTr("MemberMcpParamId")}
		schema["required"] = []string{"id"}
	} else {
		schema["required"] = []string{"title", "category_id"}
	}
	return schema
}

// memberBuildSaveRequest 把工具入参归一成 SaveArchive 的请求体
func memberBuildSaveRequest(w *Website, userId uint, id int64, args map[string]any, partial bool) *request.Archive {
	req := &request.Archive{
		Id:     id,
		UserId: userId,
	}
	req.Partial = partial
	req.UpdateAll = !partial

	if v, ok := args["title"]; ok {
		if s, ok2 := v.(string); ok2 {
			req.Title = strings.TrimSpace(s)
		}
	}
	if v, ok := args["category_id"]; ok {
		req.CategoryId = uint(intArg(args, "category_id", 0))
		_ = v
	}
	if v, ok := args["module_id"]; ok {
		req.ModuleId = uint(intArg(args, "module_id", 0))
		_ = v
	}
	if req.ModuleId == 0 {
		req.ModuleId = 1
	}
	if v, ok := args["content"]; ok {
		if s, ok2 := v.(string); ok2 {
			req.Content = s
		}
	}
	if v, ok := args["description"]; ok {
		if s, ok2 := v.(string); ok2 {
			req.Description = s
		}
	}
	if v, ok := args["keywords"]; ok {
		if s, ok2 := v.(string); ok2 {
			req.Keywords = s
		}
	}
	if v, ok := args["url_token"]; ok {
		if s, ok2 := v.(string); ok2 {
			req.UrlToken = s
		}
	}
	if _, ok := args["images"]; ok {
		req.Images = strSliceArg(args, "images")
	}
	if _, ok := args["tags"]; ok {
		req.Tags = strSliceArg(args, "tags")
	}
	if _, ok := args["price"]; ok {
		req.Price = int64Arg(args, "price")
	}
	if _, ok := args["stock"]; ok {
		req.Stock = int64Arg(args, "stock")
	}

	// 是否进审核：站点未开放直接发布 → 草稿；用户组免审 → 直接发布
	draft := w.Safe.APIPublish != 1
	if group, err := w.GetUserGroupInfo(w.mustUserGroupId(userId)); err == nil && group != nil && group.Setting.ContentNoVerify {
		draft = false
	}
	if v, ok := args["draft"]; ok {
		if b, ok2 := v.(bool); ok2 {
			draft = b
		}
	}
	if id > 0 {
		// 修改时保持原有状态，不因为一次 PATCH 把已发布内容打回待审核
		draft = false
	}
	req.Draft = draft
	req.Extra = map[string]interface{}{}

	return req
}

// mustUserGroupId 取会员用户组 ID（失败返回 0，交给调用方兜底）
func (w *Website) mustUserGroupId(userId uint) uint {
	var groupId uint
	_ = w.DB.Model(&model.User{}).Where("`id` = ?", userId).Pluck("group_id", &groupId).Error
	return groupId
}

// memberArchivePayload 内容详情输出
func memberArchivePayload(w *Website, archive *model.Archive) map[string]any {
	out := map[string]any{
		"id":           archive.Id,
		"title":        archive.Title,
		"keywords":     archive.Keywords,
		"description":  archive.Description,
		"category_id":  archive.CategoryId,
		"module_id":    archive.ModuleId,
		"url_token":    archive.UrlToken,
		"views":        archive.Views,
		"created_time": archive.CreatedTime,
		"link":         w.GetUrl("archive", archive, 0),
	}
	if data, err := w.GetArchiveDataById(archive.Id); err == nil && data != nil {
		out["content"] = data.Content
	}
	if flags := w.GetArchiveFlags(archive.Id); flags != "" {
		out["flags"] = strings.Split(flags, ",")
	}
	if tags := w.GetTagsByItemId(archive.Id); len(tags) > 0 {
		names := make([]string, 0, len(tags))
		for _, tag := range tags {
			if tag != nil && tag.Title != "" {
				names = append(names, tag.Title)
			}
		}
		out["tags"] = names
	}
	if len(archive.Images) > 0 {
		out["images"] = archive.Images
	}

	return out
}

// memberMcpLog 便于排查（工具名 + 会员）
func memberMcpLog(userId uint, tool string, err error) {
	if err != nil {
		slog.Warn("member mcp tool failed", "userId", userId, "tool", tool, "error", err)
	}
}

// memberOrderTitle 订单展示标题：取第一条明细的商品标题，缺失时退回备注
func memberOrderTitle(order *model.Order) string {
	if order == nil {
		return ""
	}
	for _, detail := range order.Details {
		if detail != nil && detail.Goods != nil && detail.Goods.Title != "" {
			return detail.Goods.Title
		}
	}
	return order.Remark
}
