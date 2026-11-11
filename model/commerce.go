package model

// 会员商城相关模型：购物车、积分、签到、店铺、保证金
// 建表方式与项目其它模型一致：GORM AutoMigrate（provider/dao.go 的 models 清单）

// CommerceCart 会员商城购物车条目，同一用户同一内容只保留一行，数量累加。
// 注意：上游 3.8.0 引入了自己的 model.Cart（支持 SKU/token，表名 cart），
// 本地商城购物车使用独立模型与表名，避免冲突。
type CommerceCart struct {
	Id          uint  `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime int64 `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime;index:idx_created_time"`
	UpdatedTime int64 `json:"updated_time" gorm:"column:updated_time;type:bigint(20);autoUpdateTime;index:idx_updated_time"`
	UserId      uint  `json:"user_id" gorm:"column:user_id;type:int(10) unsigned not null;default:0;uniqueIndex:idx_user_archive,priority:1"`
	ArchiveId   int64 `json:"archive_id" gorm:"column:archive_id;type:bigint(20) not null;default:0;uniqueIndex:idx_user_archive,priority:2"`
	Quantity    int   `json:"quantity" gorm:"column:quantity;type:int(10) not null;default:1"`

	// 以下字段不入库，读取时填充
	Archive *Archive `json:"archive" gorm:"-"`
	Amount  int64    `json:"amount" gorm:"-"`
	Thumb   string   `json:"thumb" gorm:"-"`
	Link    string   `json:"link" gorm:"-"`
}

// TableName 独立表名，避免与上游 model.Cart 的 cart 表混用
func (CommerceCart) TableName() string {
	return "commerce_cart"
}

// PointLog 积分流水
type PointLog struct {
	Id          uint   `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime;index:idx_created_time"`
	UpdatedTime int64  `json:"updated_time" gorm:"column:updated_time;type:bigint(20);autoUpdateTime"`
	UserId      uint   `json:"user_id" gorm:"column:user_id;type:int(10) unsigned not null;default:0;index:idx_point_user_id"`
	Points      int64  `json:"points" gorm:"column:points;type:bigint(20) not null;default:0"`             // 正数增加，负数扣减
	AfterPoints int64  `json:"after_points" gorm:"column:after_points;type:bigint(20) not null;default:0"` // 变动后余额
	Type        int    `json:"type" gorm:"column:type;type:tinyint(1) not null;default:1"`                 // 1签到 2消费 3管理调整 4兑换
	Remark      string `json:"remark" gorm:"column:remark;type:varchar(250) not null;default:''"`
}

// UserSign 签到记录，一个用户一天一条
type UserSign struct {
	Id          uint   `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime;index:idx_created_time"`
	UserId      uint   `json:"user_id" gorm:"column:user_id;type:int(10) unsigned not null;default:0;uniqueIndex:idx_sign_user_date,priority:1"`
	SignDate    string `json:"sign_date" gorm:"column:sign_date;type:varchar(10) not null;default:'';uniqueIndex:idx_sign_user_date,priority:2"`
	Days        int    `json:"days" gorm:"column:days;type:int(10) not null;default:1"` // 连续签到天数
	Points      int64  `json:"points" gorm:"column:points;type:bigint(20) not null;default:0"`
}

// Shop 店铺（每个会员最多一个），seller 身份即 shop 的 OwnerUserId
type Shop struct {
	Id           uint   `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime  int64  `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime;index:idx_created_time"`
	UpdatedTime  int64  `json:"updated_time" gorm:"column:updated_time;type:bigint(20);autoUpdateTime;index:idx_updated_time"`
	UserId       uint   `json:"user_id" gorm:"column:user_id;type:int(10) unsigned not null;default:0;uniqueIndex:idx_shop_user_id"`
	Name         string `json:"name" gorm:"column:name;type:varchar(60) not null;default:''"`
	Logo         string `json:"logo" gorm:"column:logo;type:varchar(255) not null;default:''"`
	Banner       string `json:"banner" gorm:"column:banner;type:varchar(255) not null;default:''"`
	Intro        string `json:"intro" gorm:"column:intro;type:varchar(1000) not null;default:''"`
	Notice       string `json:"notice" gorm:"column:notice;type:varchar(500) not null;default:''"`
	Contact      string `json:"contact" gorm:"column:contact;type:varchar(100) not null;default:''"`
	ThemeColor   string `json:"theme_color" gorm:"column:theme_color;type:varchar(20) not null;default:''"`
	Decoration   string `json:"decoration" gorm:"column:decoration;type:text"`                    // 店铺装修 JSON
	Status       int    `json:"status" gorm:"column:status;type:tinyint(1) not null;default:0"`   // 0 待审核 1 正常 2 已关闭
	Deposit      int64  `json:"deposit" gorm:"column:deposit;type:bigint(20) not null;default:0"` // 保证金余额（分）
	DepositTotal int64  `json:"deposit_total" gorm:"column:deposit_total;type:bigint(20) not null;default:0"`
	ProductCount int64  `json:"product_count" gorm:"column:product_count;type:bigint(20) not null;default:0"`
	OrderCount   int64  `json:"order_count" gorm:"column:order_count;type:bigint(20) not null;default:0"`
	SaleAmount   int64  `json:"sale_amount" gorm:"column:sale_amount;type:bigint(20) not null;default:0"`

	// 下面两个字段不入库：库里存相对路径（如 uploads/2026/10/x.png），
	// 展示时由 provider.StorageFullUrl 按当前存储方式补成可访问地址。
	LogoUrl   string `json:"logo_url" gorm:"-"`
	BannerUrl string `json:"banner_url" gorm:"-"`
}

// ShopDepositLog 保证金流水
type ShopDepositLog struct {
	Id          uint   `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime;index:idx_created_time"`
	UpdatedTime int64  `json:"updated_time" gorm:"column:updated_time;type:bigint(20);autoUpdateTime"`
	ShopId      uint   `json:"shop_id" gorm:"column:shop_id;type:int(10) unsigned not null;default:0;index:idx_deposit_shop_id"`
	UserId      uint   `json:"user_id" gorm:"column:user_id;type:int(10) unsigned not null;default:0;index:idx_deposit_user_id"`
	Amount      int64  `json:"amount" gorm:"column:amount;type:bigint(20) not null;default:0"`             // 正数缴纳，负数退还/扣划
	AfterAmount int64  `json:"after_amount" gorm:"column:after_amount;type:bigint(20) not null;default:0"` // 变动后保证金
	Type        int    `json:"type" gorm:"column:type;type:tinyint(1) not null;default:1"`                 // 1 缴纳 2 退还 3 扣划
	Remark      string `json:"remark" gorm:"column:remark;type:varchar(250) not null;default:''"`
}

// 流水类型
const (
	PointTypeSign   = 1
	PointTypeUse    = 2
	PointTypeAdmin  = 3
	PointTypeRedeem = 4

	DepositTypePay    = 1
	DepositTypeRefund = 2
	DepositTypeDeduct = 3

	ShopStatusPending = 0
	ShopStatusNormal  = 1
	ShopStatusClosed  = 2
)
