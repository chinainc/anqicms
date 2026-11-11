package request

// ShopRequest 店铺资料（含装修配置）
type ShopRequest struct {
	Name       string `json:"name"`
	Logo       string `json:"logo"`
	Banner     string `json:"banner"`
	Intro      string `json:"intro"`
	Notice     string `json:"notice"`
	Contact    string `json:"contact"`
	ThemeColor string `json:"theme_color"`
	Decoration string `json:"decoration"` // 店铺装修 JSON
}

// DepositRequest 保证金操作
type DepositRequest struct {
	Amount int64 `json:"amount"` // 单位：分
}

// CommerceCartRequest 会员商城购物车操作
// （上游 3.8.0 的 request.CartRequest 字段集不同，改名并存）
type CommerceCartRequest struct {
	ArchiveId int64 `json:"archive_id"`
	CartId    uint  `json:"cart_id"`
	Quantity  int   `json:"quantity"`
}
