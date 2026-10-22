package provider

import (
	"errors"

	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/request"
)

// 购物车
//
// 说明：AnQiCMS 原本没有购物车，下单是「直接 POST /api/order/create」。
// 这里补一层购物车，结算时把购物车条目转成 CreateOrder 需要的 details，
// 复用既有下单逻辑（库存、卖家抽成、分销、余额支付等都不重复实现）。

// CartItemLimit 单个商品最大数量
const CartItemLimit = 999

// AddCart 加入购物车，同一商品累加数量
func (w *Website) AddCart(userId uint, archiveId int64, quantity int) (*model.Cart, error) {
	if userId == 0 {
		return nil, errors.New(w.Tr("PleaseLogin"))
	}
	if quantity <= 0 {
		quantity = 1
	}
	archive, err := w.GetArchiveById(archiveId)
	if err != nil {
		return nil, errors.New(w.Tr("TheContentDoesNotExist"))
	}
	if archive.UserId == userId {
		return nil, errors.New(w.Tr("YouCannotBuyYourOwnGoods"))
	}
	if archive.Stock > 0 && int64(quantity) > archive.Stock {
		return nil, errors.New(w.Tr("InsufficientStock"))
	}

	var cart model.Cart
	w.DB.Where("`user_id` = ? and `archive_id` = ?", userId, archiveId).Take(&cart)
	if cart.Id > 0 {
		cart.Quantity += quantity
		if cart.Quantity > CartItemLimit {
			cart.Quantity = CartItemLimit
		}
		w.DB.Model(&cart).UpdateColumn("quantity", cart.Quantity)
	} else {
		cart = model.Cart{
			UserId:    userId,
			ArchiveId: archiveId,
			Quantity:  quantity,
		}
		if err = w.DB.Create(&cart).Error; err != nil {
			return nil, err
		}
	}

	return &cart, nil
}

// UpdateCartQuantity 修改数量，quantity<=0 表示移除
func (w *Website) UpdateCartQuantity(userId, cartId uint, quantity int) error {
	var cart model.Cart
	if err := w.DB.Where("`id` = ? and `user_id` = ?", cartId, userId).Take(&cart).Error; err != nil {
		return errors.New(w.Tr("TheContentDoesNotExist"))
	}
	if quantity <= 0 {
		return w.DB.Delete(&cart).Error
	}
	if quantity > CartItemLimit {
		quantity = CartItemLimit
	}
	return w.DB.Model(&cart).UpdateColumn("quantity", quantity).Error
}

// DeleteCart 移除购物车条目
func (w *Website) DeleteCart(userId, cartId uint) error {
	return w.DB.Where("`id` = ? and `user_id` = ?", cartId, userId).Delete(&model.Cart{}).Error
}

// ClearCart 清空购物车
func (w *Website) ClearCart(userId uint) error {
	return w.DB.Where("`user_id` = ?", userId).Delete(&model.Cart{}).Error
}

// GetCartList 购物车列表，同时算出每条小计与合计（单位：分）
func (w *Website) GetCartList(userId uint) ([]*model.Cart, int64) {
	var carts []*model.Cart
	if userId == 0 {
		return carts, 0
	}
	w.DB.Where("`user_id` = ?", userId).Order("id desc").Find(&carts)
	if len(carts) == 0 {
		return carts, 0
	}

	archiveIds := make([]int64, 0, len(carts))
	for _, item := range carts {
		archiveIds = append(archiveIds, item.ArchiveId)
	}
	var archives []*model.Archive
	w.DB.Where("`id` IN(?)", archiveIds).Find(&archives)
	archiveMap := make(map[int64]*model.Archive, len(archives))
	for _, item := range archives {
		item.Link = w.GetUrl(PatternArchive, item, 0)
		item.Thumb = item.GetThumb(w.PluginStorage.StorageUrl, w.GetDefaultThumb(int(item.Id)))
		archiveMap[item.Id] = item
	}

	var total int64
	list := make([]*model.Cart, 0, len(carts))
	for _, item := range carts {
		archive, ok := archiveMap[item.ArchiveId]
		if !ok {
			// 商品已被删除，直接清掉这条购物车记录
			w.DB.Delete(item)
			continue
		}
		item.Archive = archive
		item.Thumb = archive.Thumb
		item.Link = archive.Link
		item.Amount = archive.Price * int64(item.Quantity)
		total += item.Amount
		list = append(list, item)
	}

	return list, total
}

// CountCart 购物车条目数
func (w *Website) CountCart(userId uint) int64 {
	if userId == 0 {
		return 0
	}
	var total int64
	w.DB.Model(&model.Cart{}).Where("`user_id` = ?", userId).Count(&total)
	return total
}

// CheckoutCart 购物车结算，req 只需带地址与备注
func (w *Website) CheckoutCart(userId uint, req *request.OrderRequest) (*model.Order, error) {
	carts, total := w.GetCartList(userId)
	if len(carts) == 0 {
		return nil, errors.New(w.Tr("CartIsEmpty"))
	}
	if total <= 0 {
		return nil, errors.New(w.Tr("TheOrderAmountIsIncorrect"))
	}

	details := make([]request.OrderDetail, 0, len(carts))
	for _, item := range carts {
		details = append(details, request.OrderDetail{
			GoodsId:     item.ArchiveId,
			Price:       item.Archive.Price,
			OriginPrice: item.Archive.Price,
			Amount:      item.Amount,
			RealAmount:  item.Amount,
			Quantity:    item.Quantity,
			UserId:      userId,
		})
	}
	req.Details = details
	req.UserId = userId

	order, err := w.CreateOrder(userId, req)
	if err != nil {
		return nil, err
	}
	// 下单成功后清空购物车
	w.ClearCart(userId)

	return order, nil
}
