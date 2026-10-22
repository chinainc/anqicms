package provider

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
)

// 会员中心的四块业务：
//  1. 评论管理：我发布的文档收到的评论（审核 / 删除 / 回复）
//  2. 留言管理：我提交的留言（查看 / 删除）
//  3. 分销邀请：邀请链接、二维码、团队与佣金概览
//  4. 资金：余额提现申请、账户充值申请
//
// 这一层只做数据与校验，HTTP 相关处理在 controller/member_center.go。

// ---------------------------------------------------------------- 评论管理

// MemberCommentItem 评论 + 所属文档信息（会员端展示用）
type MemberCommentItem struct {
	*model.Comment
	ArchiveTitle  string
	ArchiveUrl    string
	ParentContent string
}

// memberCommentQuery 我发布的文档收到的评论。
// status < 0 表示不过滤状态。
func (w *Website) memberCommentQuery(userId uint, status int) *gorm.DB {
	tx := w.DB.Table("comments AS c").
		Joins("INNER JOIN archives AS a ON a.id = c.archive_id").
		Where("a.user_id = ?", userId)
	if status >= 0 {
		tx = tx.Where("c.status = ?", status)
	}
	return tx
}

// GetUserArchiveComments 分页获取我发布的文档收到的评论
func (w *Website) GetUserArchiveComments(userId uint, status int, page, pageSize int) ([]*MemberCommentItem, int64) {
	if userId == 0 {
		return nil, 0
	}
	var total int64
	w.memberCommentQuery(userId, status).Count(&total)
	if total == 0 {
		return nil, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 12
	}

	var rows []*memberCommentRow
	w.memberCommentQuery(userId, status).
		Select("c.*, a.title AS archive_title").
		Order("c.id desc").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&rows)

	list := make([]*MemberCommentItem, 0, len(rows))
	if len(rows) == 0 {
		return list, total
	}

	// 文档链接：批量取一次
	archiveIds := make([]int64, 0, len(rows))
	parentIds := make([]uint, 0, len(rows))
	for _, row := range rows {
		archiveIds = append(archiveIds, row.ArchiveId)
		if row.ParentId > 0 {
			parentIds = append(parentIds, row.ParentId)
		}
	}
	urls := w.memberArchiveUrls(archiveIds)
	parents := w.memberCommentContents(parentIds)

	for _, row := range rows {
		item := &MemberCommentItem{
			Comment:      &row.Comment,
			ArchiveTitle: row.ArchiveTitle,
			ArchiveUrl:   urls[row.ArchiveId],
		}
		if row.ParentId > 0 {
			item.ParentContent = parents[row.ParentId]
		}
		list = append(list, item)
	}

	return list, total
}

type memberCommentRow struct {
	model.Comment
	ArchiveTitle string `gorm:"column:archive_title"`
}

// GetUserCommentStat 待审核 / 全部评论数
func (w *Website) GetUserCommentStat(userId uint) (pending int64, total int64) {
	if userId == 0 {
		return 0, 0
	}
	w.memberCommentQuery(userId, -1).Count(&total)
	w.memberCommentQuery(userId, 0).Count(&pending)
	return
}

// GetUserArchiveComment 取评论并校验它属于我发布的文档
func (w *Website) GetUserArchiveComment(userId uint, commentId uint) (*model.Comment, error) {
	if userId == 0 || commentId == 0 {
		return nil, errors.New(w.TplTr("DocumentDoesNotExist"))
	}
	var comment model.Comment
	if err := w.DB.Where("`id` = ?", commentId).Take(&comment).Error; err != nil {
		return nil, errors.New(w.TplTr("CommentDoesNotExist"))
	}
	var owner uint
	if err := w.DB.Model(&model.Archive{}).Where("`id` = ?", comment.ArchiveId).Pluck("user_id", &owner).Error; err != nil || owner != userId {
		return nil, errors.New(w.TplTr("NoPermissionToOperateThisContent"))
	}
	return &comment, nil
}

// memberArchiveUrls 批量取文档链接
func (w *Website) memberArchiveUrls(ids []int64) map[int64]string {
	urls := map[int64]string{}
	ids = uniqueInt64(ids)
	if len(ids) == 0 {
		return urls
	}
	var archives []*model.Archive
	w.DB.Where("`id` IN ?", ids).Find(&archives)
	for _, archive := range archives {
		urls[archive.Id] = w.GetUrl("archive", archive, 0)
	}
	return urls
}

// memberCommentContents 批量取评论内容（回复时展示被回复的内容）
func (w *Website) memberCommentContents(ids []uint) map[uint]string {
	contents := map[uint]string{}
	ids = uniqueUint(ids)
	if len(ids) == 0 {
		return contents
	}
	var comments []*model.Comment
	w.DB.Where("`id` IN ?", ids).Find(&comments)
	for _, comment := range comments {
		contents[comment.Id] = comment.Content
	}
	return contents
}

func uniqueInt64(items []int64) []int64 {
	seen := make(map[int64]struct{}, len(items))
	out := make([]int64, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func uniqueUint(items []uint) []uint {
	seen := make(map[uint]struct{}, len(items))
	out := make([]uint, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// ---------------------------------------------------------------- 留言管理

// GetUserGuestbooks 我提交的留言
func (w *Website) GetUserGuestbooks(userId uint, page, pageSize int) ([]*model.Guestbook, int64) {
	if userId == 0 {
		return nil, 0
	}
	var total int64
	tx := w.DB.Model(&model.Guestbook{}).Where("`user_id` = ?", userId)
	tx.Count(&total)
	if total == 0 {
		return nil, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 12
	}
	var list []*model.Guestbook
	w.DB.Where("`user_id` = ?", userId).Order("id desc").
		Limit(pageSize).Offset((page - 1) * pageSize).Find(&list)
	return list, total
}

// DeleteUserGuestbook 删除自己的留言
func (w *Website) DeleteUserGuestbook(userId uint, id uint) error {
	if userId == 0 || id == 0 {
		return errors.New(w.TplTr("GuestbookDoesNotExist"))
	}
	res := w.DB.Where("`id` = ? AND `user_id` = ?", id, userId).Delete(&model.Guestbook{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New(w.TplTr("GuestbookDoesNotExist"))
	}
	return nil
}

// ---------------------------------------------------------------- 分销邀请

// MemberInviteInfo 邀请页展示的数据
type MemberInviteInfo struct {
	InviteCode     string
	InviteLink     string
	QrcodeUrl      string
	TeamCount      int64 // 直接邀请
	TeamTotal      int64 // 含二级
	CommissionWait int64 // 可提现佣金（分）
	CommissionPaid int64 // 已提现佣金（分）
	TotalReward    int64 // 累计收益（分）
}

// EnsureInviteCode 老用户可能没有邀请码，按需生成
func (w *Website) EnsureInviteCode(userId uint) string {
	var code string
	w.DB.Model(&model.User{}).Where("`id` = ?", userId).Pluck("invite_code", &code)
	if code != "" {
		return code
	}
	for i := 0; i < 5; i++ {
		code = strings.ToLower(config.GenerateRandString(8))
		var exist int64
		w.DB.Model(&model.User{}).Where("`invite_code` = ?", code).Count(&exist)
		if exist > 0 {
			continue
		}
		if err := w.DB.Model(&model.User{}).Where("`id` = ?", userId).UpdateColumn("invite_code", code).Error; err == nil {
			return code
		}
	}
	return ""
}

// GetMemberInviteInfo 邀请链接、二维码与团队/佣金概览
func (w *Website) GetMemberInviteInfo(userId uint) *MemberInviteInfo {
	info := &MemberInviteInfo{InviteCode: w.EnsureInviteCode(userId)}
	baseUrl := strings.TrimRight(w.System.BaseUrl, "/")
	if baseUrl == "" {
		baseUrl = strings.TrimRight(w.GetUrl("/", nil, 0), "/")
	}
	if info.InviteCode != "" {
		info.InviteLink = baseUrl + "/register?invite=" + info.InviteCode
		info.QrcodeUrl = w.GetUrl("/member/invite-qrcode", nil, 0) + "?t=" + fmt.Sprintf("%d", time.Now().Unix())
	}

	// 一级邀请
	var level1 []uint
	w.DB.Model(&model.User{}).Where("`parent_id` = ?", userId).Pluck("id", &level1)
	info.TeamCount = int64(len(level1))
	info.TeamTotal = info.TeamCount
	if len(level1) > 0 {
		var level2 int64
		w.DB.Model(&model.User{}).Where("`parent_id` IN ?", level1).Count(&level2)
		info.TeamTotal += level2
	}

	// 只统计「推广佣金」：commissions 表里同时有卖家的销售收益，
	// 这里按订单的分享人过滤，避免把销售收入算成推广佣金。
	var wait, paid model.Commission
	w.DB.Model(&model.Commission{}).
		Joins("INNER JOIN orders ON orders.order_id = commissions.order_id").
		Where("commissions.user_id = ? AND (orders.share_user_id = ? OR orders.share_parent_user_id = ?) AND commissions.status = ?", userId, userId, userId, config.CommissionStatusWait).
		Select("SUM(commissions.amount) AS amount").Take(&wait)
	w.DB.Model(&model.Commission{}).
		Joins("INNER JOIN orders ON orders.order_id = commissions.order_id").
		Where("commissions.user_id = ? AND (orders.share_user_id = ? OR orders.share_parent_user_id = ?) AND commissions.status = ?", userId, userId, userId, config.CommissionStatusPaid).
		Select("SUM(commissions.amount) AS amount").Take(&paid)
	info.CommissionWait = wait.Amount
	info.CommissionPaid = paid.Amount
	_ = w.DB.Model(&model.User{}).Where("`id` = ?", userId).Pluck("total_reward", &info.TotalReward).Error

	return info
}

// MemberInviteMember 邀请记录的展示项
type MemberInviteMember struct {
	Id          uint
	UserName    string
	AvatarURL   string
	CreatedTime int64
	Level       int
	OrderCount  int64
	Commission  int64
}

// GetMemberInviteMembers 我邀请的成员（含二级），level 标记层级
func (w *Website) GetMemberInviteMembers(userId uint, limit int) []*MemberInviteMember {
	if userId == 0 {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	var level1 []*model.User
	w.DB.Where("`parent_id` = ?", userId).Order("id desc").Limit(limit).Find(&level1)
	list := make([]*MemberInviteMember, 0, len(level1))
	seen := map[uint]struct{}{}
	for _, user := range level1 {
		seen[user.Id] = struct{}{}
		list = append(list, &MemberInviteMember{
			Id:          user.Id,
			UserName:    user.UserName,
			AvatarURL:   user.AvatarURL,
			CreatedTime: user.CreatedTime,
			Level:       1,
		})
	}
	// 二级成员
	if len(level1) > 0 {
		ids := make([]uint, 0, len(level1))
		for _, user := range level1 {
			ids = append(ids, user.Id)
		}
		var level2 []*model.User
		w.DB.Where("`parent_id` IN ?", ids).Order("id desc").Limit(limit).Find(&level2)
		for _, user := range level2 {
			if _, ok := seen[user.Id]; ok {
				continue
			}
			list = append(list, &MemberInviteMember{
				Id:          user.Id,
				UserName:    user.UserName,
				AvatarURL:   user.AvatarURL,
				CreatedTime: user.CreatedTime,
				Level:       2,
			})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Id > list[j].Id })

	if len(list) > limit {
		list = list[:limit]
	}

	// 每个成员给我带来的佣金（按订单的分享人归集）
	if len(list) > 0 {
		ids := make([]uint, 0, len(list))
		for _, item := range list {
			ids = append(ids, item.Id)
		}
		type row struct {
			ShareUserId uint
			Amount      int64
		}
		var rows []row
		w.DB.Model(&model.Commission{}).
			Select("orders.share_user_id AS share_user_id, SUM(commissions.amount) AS amount").
			Joins("INNER JOIN orders ON orders.order_id = commissions.order_id").
			Where("commissions.user_id = ? AND orders.share_user_id IN ?", userId, ids).
			Group("orders.share_user_id").Scan(&rows)
		for _, r := range rows {
			for _, item := range list {
				if item.Id == r.ShareUserId {
					item.Commission = r.Amount
				}
			}
		}
	}

	return list
}

// ---------------------------------------------------------------- 余额提现

// WithdrawWayBalance 余额提现方式（1 是原站的微信提现，这里从 2 开始）
const (
	WithdrawWayAlipay = 2
	WithdrawWayBank   = 3
	WithdrawWayWechat = 1
)

// ApplyBalanceWithdraw 从账户余额申请提现。
//
// 余额在申请时就扣减（避免重复申请把余额花两次），同时写一条状态为 0 的提现流水；
// 后台审核通过时只把这条流水标记为已完成，不再重复记账。
func (w *Website) ApplyBalanceWithdraw(userId uint, amount int64, way int, account, realName, remark string) (*model.UserWithdraw, error) {
	if userId == 0 {
		return nil, errors.New(w.TplTr("PleaseLogin"))
	}
	if amount <= 0 {
		return nil, errors.New(w.TplTr("PleaseEnterTheAmount"))
	}
	account = strings.TrimSpace(account)
	realName = strings.TrimSpace(realName)
	if account == "" || realName == "" {
		return nil, errors.New(w.TplTr("PleaseFillInTheWithdrawalAccount"))
	}
	if way != WithdrawWayAlipay && way != WithdrawWayBank && way != WithdrawWayWechat {
		way = WithdrawWayAlipay
	}

	var withdraw *model.UserWithdraw
	err := w.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.User{}).Where("`id` = ? AND `balance` >= ?", userId, amount).
			UpdateColumn("balance", gorm.Expr("`balance` - ?", amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New(w.TplTr("InsufficientBalance"))
		}
		var balance int64
		tx.Model(&model.User{}).Where("`id` = ?", userId).Pluck("balance", &balance)

		item := &model.UserWithdraw{
			UserId:      userId,
			Amount:      amount,
			WithdrawWay: way,
			Status:      config.WithdrawStatusWaiting,
			Remark:      buildWithdrawRemark(way, realName, account, remark),
		}
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		finance := &model.Finance{
			UserId:      userId,
			Direction:   config.FinanceOutput,
			Amount:      amount,
			AfterAmount: balance,
			Action:      config.FinanceActionWithdraw,
			OrderId:     fmt.Sprintf("W%d", item.Id),
			Status:      0,
			Remark:      w.TplTr("WithdrawApplying"),
		}
		if err := tx.Create(finance).Error; err != nil {
			return err
		}
		withdraw = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return withdraw, nil
}

// buildWithdrawRemark 提现备注：方式|姓名|账号|用户备注（后台可以直接看到收款信息）
func buildWithdrawRemark(way int, realName, account, remark string) string {
	wayName := "支付宝"
	if way == WithdrawWayBank {
		wayName = "银行卡"
	} else if way == WithdrawWayWechat {
		wayName = "微信"
	}
	parts := []string{wayName, realName, account}
	if remark != "" {
		parts = append(parts, remark)
	}
	out := strings.Join(parts, "|")
	if len(out) > 240 {
		out = out[:240]
	}
	return out
}

// GetUserWithdraws 我的提现记录
func (w *Website) GetUserWithdraws(userId uint, page, pageSize int) ([]*model.UserWithdraw, int64) {
	if userId == 0 {
		return nil, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 12
	}
	var total int64
	w.DB.Model(&model.UserWithdraw{}).Where("`user_id` = ?", userId).Count(&total)
	var list []*model.UserWithdraw
	w.DB.Where("`user_id` = ?", userId).Order("id desc").
		Limit(pageSize).Offset((page - 1) * pageSize).Find(&list)
	return list, total
}

// ---------------------------------------------------------------- 账户充值

// ApplyRecharge 提交充值申请（管理员在后台增加余额后自动标记为已到账）
func (w *Website) ApplyRecharge(userId uint, amount int64, payWay, remark string) (*model.Recharge, error) {
	if userId == 0 {
		return nil, errors.New(w.TplTr("PleaseLogin"))
	}
	if amount <= 0 {
		return nil, errors.New(w.TplTr("PleaseEnterTheAmount"))
	}
	if amount > 100000000 {
		return nil, errors.New(w.TplTr("RechargeAmountTooLarge"))
	}
	if len(remark) > 200 {
		remark = remark[:200]
	}
	payWay = strings.TrimSpace(payWay)
	if payWay == "" {
		payWay = "支付宝"
	}
	item := &model.Recharge{
		UserId: userId,
		Amount: amount,
		PayWay: payWay,
		Remark: remark,
		Status: 0,
	}
	if err := w.DB.Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// GetUserRecharges 我的充值记录
func (w *Website) GetUserRecharges(userId uint, page, pageSize int) ([]*model.Recharge, int64) {
	if userId == 0 {
		return nil, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 12
	}
	var total int64
	w.DB.Model(&model.Recharge{}).Where("`user_id` = ?", userId).Count(&total)
	var list []*model.Recharge
	w.DB.Where("`user_id` = ?", userId).Order("id desc").
		Limit(pageSize).Offset((page - 1) * pageSize).Find(&list)
	return list, total
}

// settlePendingRecharges 管理员给用户加余额后，按金额从早到晚把充值申请标记为已到账
func (w *Website) settlePendingRecharges(userId uint, amount int64) {
	if userId == 0 || amount <= 0 {
		return
	}
	var pendings []*model.Recharge
	w.DB.Where("`user_id` = ? AND `status` = ?", userId, 0).Order("id asc").Find(&pendings)
	var used int64
	now := time.Now().Unix()
	for _, item := range pendings {
		if used+item.Amount > amount {
			break
		}
		used += item.Amount
		w.DB.Model(&model.Recharge{}).Where("`id` = ?", item.Id).Updates(map[string]interface{}{
			"status":       1,
			"confirm_time": now,
		})
	}
}
