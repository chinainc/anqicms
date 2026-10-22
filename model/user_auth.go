package model

// UserAuth 会员实名 / 企业认证申请。
//
// 一个用户只保留一条记录：重新提交时覆盖原记录并把状态重置为待审核。
// 审核通过后会把姓名（企业为法人姓名）回写到 users.real_name，
// 提现打款等依赖实名的地方因此不需要额外改动。
type UserAuth struct {
	Id          uint  `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime int64 `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime;index:idx_created_time"`
	UpdatedTime int64 `json:"updated_time" gorm:"column:updated_time;type:bigint(20);autoUpdateTime;index:idx_updated_time"`
	UserId      uint  `json:"user_id" gorm:"column:user_id;type:int(10) unsigned not null;default:0;uniqueIndex:idx_auth_user_id"`
	Type        int   `json:"type" gorm:"column:type;type:tinyint(1) not null;default:1"` // 1 个人实名 2 企业认证

	// 个人
	RealName    string `json:"real_name" gorm:"column:real_name;type:varchar(50) not null;default:''"`
	IdCard      string `json:"id_card" gorm:"column:id_card;type:varchar(32) not null;default:''"`
	IdCardFront string `json:"id_card_front" gorm:"column:id_card_front;type:varchar(255) not null;default:''"`
	IdCardBack  string `json:"id_card_back" gorm:"column:id_card_back;type:varchar(255) not null;default:''"`
	IdCardHand  string `json:"id_card_hand" gorm:"column:id_card_hand;type:varchar(255) not null;default:''"`

	// 企业
	CompanyName    string `json:"company_name" gorm:"column:company_name;type:varchar(120) not null;default:''"`
	CompanyNo      string `json:"company_no" gorm:"column:company_no;type:varchar(64) not null;default:''"`
	CompanyLegal   string `json:"company_legal" gorm:"column:company_legal;type:varchar(50) not null;default:''"`
	CompanyLicence string `json:"company_licence" gorm:"column:company_licence;type:varchar(255) not null;default:''"`
	Phone          string `json:"phone" gorm:"column:phone;type:varchar(20) not null;default:''"`

	Status       int    `json:"status" gorm:"column:status;type:tinyint(1) not null;default:0;index:idx_auth_status"` // 0 待审核 1 已通过 -1 已拒绝
	Remark       string `json:"remark" gorm:"column:remark;type:varchar(250) not null;default:''"`
	AuditTime    int64  `json:"audit_time" gorm:"column:audit_time;type:bigint(20) not null;default:0"`
	AuditAdminId uint   `json:"audit_admin_id" gorm:"column:audit_admin_id;type:int(10) unsigned not null;default:0"`

	UserName string `json:"user_name" gorm:"-"`
}
