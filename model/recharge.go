package model

// Recharge 账户充值申请。
//
// 本站没有独立的在线充值通道，会员在前台提交充值申请（金额 + 支付方式 + 备注），
// 管理员在后台通过「修改余额」入账后，系统自动把对应的申请标记为已到账。
// 这里单独建表而不是复用 finance，避免这笔「申请」污染会员的资金流水。
type Recharge struct {
	Id          uint   `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime;index:idx_created_time"`
	UpdatedTime int64  `json:"updated_time" gorm:"column:updated_time;type:bigint(20);autoUpdateTime;index:idx_updated_time"`
	UserId      uint   `json:"user_id" gorm:"column:user_id;type:int(10) unsigned not null;default:0;index:idx_recharge_user_id"`
	Amount      int64  `json:"amount" gorm:"column:amount;type:bigint(20) not null;default:0"` // 充值金额，单位：分
	PayWay      string `json:"pay_way" gorm:"column:pay_way;type:varchar(20) not null;default:''"`
	Remark      string `json:"remark" gorm:"column:remark;type:varchar(250) not null;default:''"`
	Status      int    `json:"status" gorm:"column:status;type:tinyint(1) not null;default:0"` // 0 待确认 1 已到账 -1 已关闭
	ConfirmTime int64  `json:"confirm_time" gorm:"column:confirm_time;type:bigint(20) not null;default:0"`
	UserName    string `json:"user_name" gorm:"-"`
}
