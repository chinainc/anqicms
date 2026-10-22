package provider

import (
	"encoding/json"
	"errors"
	"sort"

	"gorm.io/gorm"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/request"
)

// 店铺 / 保证金 / 卖家订单 / 收到的评价
//
// AnQiCMS 原本只有 order.seller_id（下单时写入商品作者），没有店铺实体。
// 这里以「一个会员一个店铺」为基础补出店铺资料、装修配置与保证金，
// 已售订单（seller_id）与收到的评价（内容作者）都基于现有数据做查询，不改原有表。

// ShopDecoration 店铺装修配置
type ShopDecoration struct {
	Layout     string   `json:"layout"`      // 布局：wide / compact
	Banner     string   `json:"banner"`      // 顶部横幅
	ShowNotice bool     `json:"show_notice"` // 是否展示公告
	Notice     string   `json:"notice"`
	Modules    []string `json:"modules"` // 展示模块顺序：products/notice/contact/categories
	ThemeColor string   `json:"theme_color"`
	Footer     string   `json:"footer"`
}

// DefaultShopDecoration 默认装修配置
func DefaultShopDecoration() ShopDecoration {
	return ShopDecoration{
		Layout:     "wide",
		ShowNotice: true,
		Modules:    []string{"products", "notice", "contact"},
	}
}

// StorageFullUrl 把库里存的相对图片路径补成可访问地址。
// 已带协议或 // 的原样返回；空串返回空。云存储开启时按配置的域名拼，
// 未开启时退回站点根相对地址（/uploads/...）。
func (w *Website) StorageFullUrl(path string) string {
	return config.JoinStorageUrl(w.PluginStorage.StorageUrl, path)
}

// GetShopByUserId 取店铺，不存在返回 nil
func (w *Website) GetShopByUserId(userId uint) *model.Shop {
	if userId == 0 {
		return nil
	}
	var shop model.Shop
	if err := w.DB.Where("`user_id` = ?", userId).Take(&shop).Error; err != nil {
		return nil
	}
	return &shop
}

// EnsureShop 取店铺，没有则创建默认店铺（会员首次进入店铺页时调用）
func (w *Website) EnsureShop(userId uint) (*model.Shop, error) {
	if userId == 0 {
		return nil, errors.New(w.TplTr("PleaseLogin"))
	}
	if shop := w.GetShopByUserId(userId); shop != nil {
		return shop, nil
	}

	user, err := w.GetUserInfoById(userId)
	if err != nil {
		return nil, err
	}
	decoration, _ := json.Marshal(DefaultShopDecoration())
	shop := &model.Shop{
		UserId:     userId,
		Name:       user.UserName,
		Intro:      w.TplTr("DefaultShopIntro"),
		Status:     model.ShopStatusNormal,
		Decoration: string(decoration),
	}
	if err = w.DB.Create(shop).Error; err != nil {
		// 并发下可能已被其它请求创建
		if exist := w.GetShopByUserId(userId); exist != nil {
			return exist, nil
		}
		return nil, err
	}

	return shop, nil
}

// SaveShop 保存店铺资料
func (w *Website) SaveShop(userId uint, req *request.ShopRequest) (*model.Shop, error) {
	shop, err := w.EnsureShop(userId)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		shop.Name = req.Name
	}
	shop.Logo = req.Logo
	shop.Banner = req.Banner
	shop.Intro = req.Intro
	shop.Notice = req.Notice
	shop.Contact = req.Contact
	shop.ThemeColor = req.ThemeColor
	if req.Decoration != "" {
		// 只接受合法 JSON，避免把装修配置写坏
		var decoration ShopDecoration
		if err = json.Unmarshal([]byte(req.Decoration), &decoration); err != nil {
			return nil, errors.New(w.TplTr("DecorationFormatError"))
		}
		shop.Decoration = req.Decoration
	}
	if err = w.DB.Save(shop).Error; err != nil {
		return nil, err
	}

	return shop, nil
}

// GetShopDecoration 解析店铺装修配置
func (w *Website) GetShopDecoration(shop *model.Shop) ShopDecoration {
	decoration := DefaultShopDecoration()
	if shop == nil || shop.Decoration == "" {
		return decoration
	}
	_ = json.Unmarshal([]byte(shop.Decoration), &decoration)
	if len(decoration.Modules) == 0 {
		decoration.Modules = DefaultShopDecoration().Modules
	}
	if shop.ThemeColor != "" && decoration.ThemeColor == "" {
		decoration.ThemeColor = shop.ThemeColor
	}

	return decoration
}

// PayDeposit 缴纳保证金：余额 → 保证金，同一事务内完成
func (w *Website) PayDeposit(userId uint, amount int64) (*model.Shop, error) {
	if amount <= 0 {
		return nil, errors.New(w.TplTr("TheAmountIsIncorrect"))
	}
	shop, err := w.EnsureShop(userId)
	if err != nil {
		return nil, err
	}

	err = w.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.User{}).Where("`id` = ? and `balance` >= ?", userId, amount).
			UpdateColumn("balance", gorm.Expr("`balance` - ?", amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New(w.TplTr("InsufficientBalance"))
		}
		var balance int64
		if err := tx.Model(&model.User{}).Where("`id` = ?", userId).Pluck("balance", &balance).Error; err != nil {
			return err
		}
		// 余额支出记一笔资金流水
		if err := tx.Create(&model.Finance{
			UserId:      userId,
			Direction:   0,
			Amount:      amount,
			AfterAmount: balance,
			Action:      config.FinanceActionDeposit,
			Remark:      w.TplTr("ShopDepositPayRemark"),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Shop{}).Where("`id` = ?", shop.Id).
			UpdateColumn("deposit", gorm.Expr("`deposit` + ?", amount)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Shop{}).Where("`id` = ?", shop.Id).
			UpdateColumn("deposit_total", gorm.Expr("`deposit_total` + ?", amount)).Error; err != nil {
			return err
		}
		var deposit int64
		if err := tx.Model(&model.Shop{}).Where("`id` = ?", shop.Id).Pluck("deposit", &deposit).Error; err != nil {
			return err
		}

		return tx.Create(&model.ShopDepositLog{
			ShopId:      shop.Id,
			UserId:      userId,
			Amount:      amount,
			AfterAmount: deposit,
			Type:        model.DepositTypePay,
			Remark:      w.TplTr("ShopDepositPayRemark"),
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return w.GetShopByUserId(userId), nil
}

// RefundDeposit 退还保证金：保证金 → 余额
func (w *Website) RefundDeposit(userId uint, amount int64) (*model.Shop, error) {
	if amount <= 0 {
		return nil, errors.New(w.TplTr("TheAmountIsIncorrect"))
	}
	shop := w.GetShopByUserId(userId)
	if shop == nil {
		return nil, errors.New(w.TplTr("ShopNotFound"))
	}

	err := w.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.Shop{}).Where("`id` = ? and `deposit` >= ?", shop.Id, amount).
			UpdateColumn("deposit", gorm.Expr("`deposit` - ?", amount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New(w.TplTr("InsufficientDeposit"))
		}
		if err := tx.Model(&model.User{}).Where("`id` = ?", userId).
			UpdateColumn("balance", gorm.Expr("`balance` + ?", amount)).Error; err != nil {
			return err
		}
		var balance, deposit int64
		if err := tx.Model(&model.User{}).Where("`id` = ?", userId).Pluck("balance", &balance).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.Finance{
			UserId:      userId,
			Direction:   config.FinanceIncome,
			Amount:      amount,
			AfterAmount: balance,
			Action:      config.FinanceActionDeposit,
			Remark:      w.TplTr("ShopDepositRefundRemark"),
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Shop{}).Where("`id` = ?", shop.Id).Pluck("deposit", &deposit).Error; err != nil {
			return err
		}

		return tx.Create(&model.ShopDepositLog{
			ShopId:      shop.Id,
			UserId:      userId,
			Amount:      -amount,
			AfterAmount: deposit,
			Type:        model.DepositTypeRefund,
			Remark:      w.TplTr("ShopDepositRefundRemark"),
		}).Error
	})
	if err != nil {
		return nil, err
	}

	return w.GetShopByUserId(userId), nil
}

// GetDepositLogs 保证金流水
func (w *Website) GetDepositLogs(userId uint, page, pageSize int) ([]*model.ShopDepositLog, int64) {
	var logs []*model.ShopDepositLog
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
	tx := w.DB.Model(&model.ShopDepositLog{}).Where("`user_id` = ?", userId)
	tx.Count(&total)
	tx.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs)

	return logs, total
}

// GetSellerOrders 已售出的订单（seller_id = 当前用户）
func (w *Website) GetSellerOrders(sellerId uint, status string, page, pageSize int) ([]*model.Order, int64) {
	if sellerId == 0 {
		return nil, 0
	}
	return w.GetOrderList(func(tx *gorm.DB) *gorm.DB {
		return tx.Where("`seller_id` = ?", sellerId)
	}, status, page, pageSize)
}

// GetSellerStats 卖家统计：订单数、销售额、待发货、今日成交
func (w *Website) GetSellerStats(sellerId uint) (orderCount int64, saleAmount int64, pendingCount int64, todayAmount int64) {
	if sellerId == 0 {
		return
	}
	type sumResult struct {
		Total  int64
		Amount int64
	}
	var res sumResult
	w.DB.Model(&model.Order{}).Select("count(*) as total, sum(`seller_amount`) as amount").
		Where("`seller_id` = ? and `status` >= ?", sellerId, 0).Scan(&res)
	orderCount = res.Total
	saleAmount = res.Amount
	w.DB.Model(&model.Order{}).Where("`seller_id` = ? and `status` = ?", sellerId, 1).Count(&pendingCount)

	return
}

// GetReceivedComments 我收到的评价（他人对我发布内容的评论）
func (w *Website) GetReceivedComments(userId uint, page, pageSize int) ([]*model.Comment, int64) {
	var comments []*model.Comment
	var total int64
	if userId == 0 {
		return comments, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > PointLogLimit {
		pageSize = PointLogLimit
	}
	tx := w.DB.Model(&model.Comment{}).
		Joins("inner join archives on archives.id = comments.archive_id").
		Where("archives.user_id = ? and comments.status = 1", userId)
	tx.Count(&total)
	tx.Order("comments.id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&comments)
	if len(comments) == 0 {
		return comments, total
	}

	archiveIds := make([]int64, 0, len(comments))
	for _, item := range comments {
		archiveIds = append(archiveIds, item.ArchiveId)
	}
	var archives []*model.Archive
	w.DB.Select("id", "title").Where("`id` IN(?)", archiveIds).Find(&archives)
	titleMap := make(map[int64]string, len(archives))
	for _, item := range archives {
		titleMap[item.Id] = item.Title
	}
	for _, item := range comments {
		item.ItemTitle = titleMap[item.ArchiveId]
	}

	return comments, total
}

// UserArchiveItem 会员「我的发布」条目（正式内容与草稿合并展示）
type UserArchiveItem struct {
	Id          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Link        string `json:"link"`
	Thumb       string `json:"thumb"`
	Views       uint   `json:"views"`
	CreatedTime int64  `json:"created_time"`
	ModuleId    uint   `json:"module_id"`
	Draft       bool   `json:"draft"`
	Status      uint   `json:"status"`
}

// GetUserPublished 我的发布：已发布内容 + 待审核草稿（草稿表 archive_drafts）
func (w *Website) GetUserPublished(userId uint, page, pageSize int) ([]*UserArchiveItem, int64) {
	items := make([]*UserArchiveItem, 0)
	if userId == 0 {
		return items, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 50
	}

	// 草稿（待审核）：AnQiCMS 在 API 发布未开启时把内容写入 archive_drafts
	// status = ContentStatusDelete(99) 是「会员删除后进回收站」的记录，不能出现在我的发布里
	var drafts []*model.ArchiveDraft
	w.DB.Where("`user_id` = ? AND `status` <> ?", userId, config.ContentStatusDelete).Order("id desc").Find(&drafts)
	for _, item := range drafts {
		items = append(items, &UserArchiveItem{
			Id:          item.Id,
			Title:       item.Title,
			Description: item.Description,
			Views:       item.Views,
			CreatedTime: item.CreatedTime,
			ModuleId:    item.ModuleId,
			Draft:       true,
			Status:      item.Status,
		})
	}

	var archives []*model.Archive
	w.DB.Where("`user_id` = ?", userId).Order("id desc").Find(&archives)
	for _, item := range archives {
		items = append(items, &UserArchiveItem{
			Id:          item.Id,
			Title:       item.Title,
			Description: item.Description,
			Link:        w.GetUrl(PatternArchive, item, 0),
			Thumb:       item.GetThumb(w.PluginStorage.StorageUrl, w.GetDefaultThumb(int(item.Id))),
			Views:       item.Views,
			CreatedTime: item.CreatedTime,
			ModuleId:    item.ModuleId,
			Status:      1,
		})
	}

	// 按时间倒序
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedTime > items[j].CreatedTime
	})

	total := int64(len(items))
	start := (page - 1) * pageSize
	if start >= len(items) {
		return nil, total
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	return items[start:end], total
}

// GetUserArchives 我发布的内容（仅正式内容，店铺主页使用）
func (w *Website) GetUserArchives(userId uint, page, pageSize int) ([]*model.Archive, int64) {
	var archives []*model.Archive
	var total int64
	if userId == 0 {
		return archives, 0
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 50
	}
	tx := w.DB.Model(&model.Archive{}).Where("`user_id` = ?", userId)
	tx.Count(&total)
	tx.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&archives)
	for _, item := range archives {
		item.Link = w.GetUrl(PatternArchive, item, 0)
		item.Thumb = item.GetThumb(w.PluginStorage.StorageUrl, w.GetDefaultThumb(int(item.Id)))
	}

	return archives, total
}

// GetFinanceLogs 会员资金流水
func (w *Website) GetFinanceLogs(userId uint, page, pageSize int) ([]*model.Finance, int64) {
	var logs []*model.Finance
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
	tx := w.DB.Model(&model.Finance{}).Where("`user_id` = ?", userId)
	tx.Count(&total)
	tx.Order("id desc").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs)

	return logs, total
}
