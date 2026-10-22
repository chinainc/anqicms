package provider

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"kandaoni.com/anqicms/model"
)

// 积分与签到
//
// 积分为本模块新增能力（AnQiCMS 原有 Balance 是余额、TotalReward 是分销收益，都不是积分）。
// 积分变动一律走 AddPoints，保证「改余额 + 写流水」在同一事务内。

const (
	// SignBasePoints 签到基础积分
	SignBasePoints = 5
	// SignExtraMax 连续签到额外积分上限
	SignExtraMax = 10
	// PointLogLimit 流水单页上限
	PointLogLimit = 50
)

// AddPoints 增减积分，points<0 表示扣减（不足则失败）
func (w *Website) AddPoints(userId uint, points int64, typ int, remark string) (int64, error) {
	if userId == 0 {
		return 0, errors.New(w.Tr("PleaseLogin"))
	}
	if points == 0 {
		var current int64
		w.DB.Model(&model.User{}).Where("`id` = ?", userId).Pluck("points", &current)
		return current, nil
	}

	var after int64
	err := w.DB.Transaction(func(tx *gorm.DB) error {
		var res *gorm.DB
		if points < 0 {
			// 扣减必须带余额条件，避免并发把积分扣成负数
			res = tx.Model(&model.User{}).Where("`id` = ? and `points` >= ?", userId, -points).
				UpdateColumn("points", gorm.Expr("`points` + ?", points))
		} else {
			res = tx.Model(&model.User{}).Where("`id` = ?", userId).
				UpdateColumn("points", gorm.Expr("`points` + ?", points))
		}
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New(w.Tr("InsufficientPoints"))
		}
		if err := tx.Model(&model.User{}).Where("`id` = ?", userId).Pluck("points", &after).Error; err != nil {
			return err
		}
		return tx.Create(&model.PointLog{
			UserId:      userId,
			Points:      points,
			AfterPoints: after,
			Type:        typ,
			Remark:      remark,
		}).Error
	})

	return after, err
}

// GetPointLogs 积分流水
func (w *Website) GetPointLogs(userId uint, page, pageSize int) ([]*model.PointLog, int64) {
	var logs []*model.PointLog
	var total int64
	if userId == 0 {
		return logs, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > PointLogLimit {
		pageSize = PointLogLimit
	}
	tx := w.DB.Model(&model.PointLog{}).Where("`user_id` = ?", userId)
	tx.Count(&total)
	tx.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs)

	return logs, total
}

// Sign 按给定日期签到（便于测试），重复签到返回错误
func (w *Website) Sign(userId uint, day time.Time) (*model.UserSign, error) {
	if userId == 0 {
		return nil, errors.New(w.Tr("PleaseLogin"))
	}
	today := day.Format("2006-01-02")
	// 昨天是否签到，决定连续天数
	var lastSign model.UserSign
	days := 1
	if err := w.DB.Where("`user_id` = ? and `sign_date` = ?", userId, day.AddDate(0, 0, -1).Format("2006-01-02")).Take(&lastSign).Error; err == nil {
		days = lastSign.Days + 1
	}

	points := int64(SignBasePoints)
	extra := days - 1
	if extra > SignExtraMax {
		extra = SignExtraMax
	}
	if extra > 0 {
		points += int64(extra)
	}

	sign := &model.UserSign{
		UserId:   userId,
		SignDate: today,
		Days:     days,
		Points:   points,
	}
	if err := w.DB.Create(sign).Error; err != nil {
		// 唯一索引冲突即今日已签到
		return nil, errors.New(w.Tr("AlreadyCheckedInToday"))
	}

	// 备注词条含 %d 占位（见 locales/*/default.yml 的 SignRewardRemark）；
	// 键名走变量，避免 go vet 把 "SignRewardRemark" 当成没有占位符的格式串报错。
	signRewardKey := "SignRewardRemark"
	if _, err := w.AddPoints(userId, points, model.PointTypeSign, w.TplTr(signRewardKey, days)); err != nil {
		// 积分发放失败不应留下签到记录
		w.DB.Delete(sign)
		return nil, err
	}

	return sign, nil
}

// SignToday 使用当前时间签到
func (w *Website) SignToday(userId uint) (*model.UserSign, error) {
	return w.Sign(userId, time.Now())
}

// SignInfo 签到概况：今日是否已签、连续天数、本月签到次数、当前积分
func (w *Website) SignInfo(userId uint) (checkedToday bool, days int, monthCount int64, points int64) {
	if userId == 0 {
		return
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	var sign model.UserSign
	if err := w.DB.Where("`user_id` = ? and `sign_date` = ?", userId, today).Take(&sign).Error; err == nil {
		checkedToday = true
		days = sign.Days
	} else {
		var lastSign model.UserSign
		if err := w.DB.Where("`user_id` = ?", userId).Order("id desc").Take(&lastSign).Error; err == nil {
			// 昨天签过，连续天数继续累计
			if lastSign.SignDate == now.AddDate(0, 0, -1).Format("2006-01-02") {
				days = lastSign.Days
			}
		}
	}
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	w.DB.Model(&model.UserSign{}).Where("`user_id` = ? and `sign_date` >= ?", userId, monthStart).Count(&monthCount)
	w.DB.Model(&model.User{}).Where("`id` = ?", userId).Pluck("points", &points)

	return
}
