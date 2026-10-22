package provider

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"kandaoni.com/anqicms/model"
)

// 会员实名 / 企业认证：提交、查询、后台审核。

const (
	UserAuthStatusPending  = 0
	UserAuthStatusPassed   = 1
	UserAuthStatusRejected = -1
	// UserAuthStatusAll 列表筛选用：不过滤状态（-1 是「已驳回」，不能拿来表示全部）
	UserAuthStatusAll = -2

	UserAuthTypePerson   = 1
	UserAuthTypeCompany  = 2
	UserAuthRemarkMaxLen = 200
)

// UserAuthRequest 会员提交认证的入参
type UserAuthRequest struct {
	Type           int
	RealName       string
	IdCard         string
	IdCardFront    string
	IdCardBack     string
	IdCardHand     string
	CompanyName    string
	CompanyNo      string
	CompanyLegal   string
	CompanyLicence string
	Phone          string
}

// GetUserAuthByUserId 取用户的认证记录，没有返回 nil
func (w *Website) GetUserAuthByUserId(userId uint) *model.UserAuth {
	if userId == 0 {
		return nil
	}
	var item model.UserAuth
	if err := w.DB.Where("`user_id` = ?", userId).Take(&item).Error; err != nil {
		return nil
	}
	return &item
}

// GetUserAuthById 按 ID 取认证记录（附带会员用户名）
func (w *Website) GetUserAuthById(id uint) (*model.UserAuth, error) {
	var item model.UserAuth
	if err := w.DB.Where("`id` = ?", id).Take(&item).Error; err != nil {
		return nil, errors.New(w.TplTr("UserAuthDoesNotExist"))
	}
	if user, err := w.GetUserInfoById(item.UserId); err == nil && user != nil {
		item.UserName = user.UserName
	}
	return &item, nil
}

// ApplyUserAuth 会员提交（或重新提交）认证申请。
// 已通过的记录不允许直接覆盖，必须由管理员先撤销，避免用户改掉审核通过的资料。
func (w *Website) ApplyUserAuth(userId uint, req *UserAuthRequest) (*model.UserAuth, error) {
	if userId == 0 {
		return nil, errors.New(w.TplTr("PleaseLogin"))
	}
	if req == nil {
		return nil, errors.New(w.TplTr("PleaseFillInTheCompleteCertificationInformation"))
	}

	authType := req.Type
	if authType != UserAuthTypeCompany {
		authType = UserAuthTypePerson
	}

	item := &model.UserAuth{
		UserId:    userId,
		Type:      authType,
		Phone:     strings.TrimSpace(req.Phone),
		Status:    UserAuthStatusPending,
		Remark:    "",
		AuditTime: 0,
	}
	item.RealName = strings.TrimSpace(req.RealName)
	item.IdCard = strings.ToUpper(strings.TrimSpace(req.IdCard))
	item.IdCardFront = strings.TrimSpace(req.IdCardFront)
	item.IdCardBack = strings.TrimSpace(req.IdCardBack)
	item.IdCardHand = strings.TrimSpace(req.IdCardHand)

	if authType == UserAuthTypeCompany {
		item.CompanyName = strings.TrimSpace(req.CompanyName)
		item.CompanyNo = strings.ToUpper(strings.TrimSpace(req.CompanyNo))
		item.CompanyLegal = strings.TrimSpace(req.CompanyLegal)
		item.CompanyLicence = strings.TrimSpace(req.CompanyLicence)
		if item.CompanyName == "" || item.CompanyLicence == "" || item.CompanyLegal == "" {
			return nil, errors.New(w.TplTr("PleaseFillInTheCompleteCertificationInformation"))
		}
	} else {
		if item.RealName == "" || item.IdCard == "" || item.IdCardFront == "" || item.IdCardBack == "" {
			return nil, errors.New(w.TplTr("PleaseFillInTheCompleteCertificationInformation"))
		}
		if len(item.IdCard) != 18 {
			return nil, errors.New(w.TplTr("IdCardNumberFormatIsIncorrect"))
		}
	}

	err := w.DB.Transaction(func(tx *gorm.DB) error {
		var exist model.UserAuth
		err := tx.Where("`user_id` = ?", userId).Take(&exist).Error
		if err == nil {
			if exist.Status == UserAuthStatusPassed {
				return errors.New(w.TplTr("CertificationHasBeenApproved"))
			}
			item.Id = exist.Id
			item.CreatedTime = exist.CreatedTime
			return tx.Model(&model.UserAuth{}).Where("`id` = ?", exist.Id).
				Select("type", "real_name", "id_card", "id_card_front", "id_card_back", "id_card_hand",
					"company_name", "company_no", "company_legal", "company_licence", "phone",
					"status", "remark", "audit_time", "audit_admin_id").
				Updates(item).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(item).Error
	})
	if err != nil {
		return nil, err
	}

	return w.GetUserAuthByUserId(userId), nil
}

// GetUserAuthList 后台列表：按状态/关键词筛选
func (w *Website) GetUserAuthList(status int, keyword string, page, pageSize int) ([]*model.UserAuth, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	keyword = strings.TrimSpace(keyword)
	filter := func(db *gorm.DB) *gorm.DB {
		if status != UserAuthStatusAll {
			db = db.Where("`status` = ?", status)
		}
		if keyword != "" {
			like := "%" + keyword + "%"
			db = db.Where("`real_name` LIKE ? OR `company_name` LIKE ? OR `id_card` LIKE ? OR `company_no` LIKE ?", like, like, like, like)
		}
		return db
	}

	var total int64
	w.DB.Model(&model.UserAuth{}).Scopes(filter).Count(&total)
	var list []*model.UserAuth
	if total > 0 {
		w.DB.Model(&model.UserAuth{}).Scopes(filter).
			Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&list)
	}

	// 附上会员用户名
	if len(list) > 0 {
		ids := make([]uint, 0, len(list))
		for _, item := range list {
			ids = append(ids, item.UserId)
		}
		users := w.GetUsersInfoByIds(ids)
		names := make(map[uint]string, len(users))
		for _, user := range users {
			names[user.Id] = user.UserName
		}
		for _, item := range list {
			item.UserName = names[item.UserId]
		}
	}

	return list, total
}

// GetUserAuthStat 各状态数量
func (w *Website) GetUserAuthStat() (pending, passed, rejected int64) {
	w.DB.Model(&model.UserAuth{}).Where("`status` = ?", UserAuthStatusPending).Count(&pending)
	w.DB.Model(&model.UserAuth{}).Where("`status` = ?", UserAuthStatusPassed).Count(&passed)
	w.DB.Model(&model.UserAuth{}).Where("`status` = ?", UserAuthStatusRejected).Count(&rejected)
	return
}

// AuditUserAuth 后台审核：通过 / 拒绝
func (w *Website) AuditUserAuth(id uint, status int, remark string, adminId uint) (*model.UserAuth, error) {
	item, err := w.GetUserAuthById(id)
	if err != nil {
		return nil, err
	}
	if status != UserAuthStatusPassed && status != UserAuthStatusRejected {
		return nil, errors.New(w.TplTr("InvalidOperation"))
	}
	if status == UserAuthStatusRejected && strings.TrimSpace(remark) == "" {
		return nil, errors.New(w.TplTr("PleaseFillInTheReasonForRejection"))
	}
	remark = strings.TrimSpace(remark)
	if len([]rune(remark)) > UserAuthRemarkMaxLen {
		remark = string([]rune(remark)[:UserAuthRemarkMaxLen])
	}

	err = w.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.UserAuth{}).Where("`id` = ?", item.Id).Updates(map[string]interface{}{
			"status":         status,
			"remark":         remark,
			"audit_time":     time.Now().Unix(),
			"audit_admin_id": adminId,
		}).Error; err != nil {
			return err
		}
		if status == UserAuthStatusPassed {
			// 实名结果回写到会员资料（提现打款会用到 real_name）
			realName := item.RealName
			if item.Type == UserAuthTypeCompany && item.CompanyLegal != "" {
				realName = item.CompanyLegal
			}
			if realName != "" {
				if err := tx.Model(&model.User{}).Where("`id` = ?", item.UserId).
					UpdateColumn("real_name", realName).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return w.GetUserAuthById(item.Id)
}
