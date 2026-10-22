package model

// DiyPage 可视化装修页面配置。
//
// 一条记录对应一个页面（key 区分，如 home=首页、shop=店铺页），
// Modules 存 JSON 数组，每项是一个区块：{id,type,settings{...}}。
// 前台由 provider.RenderDiyPage 统一渲染成 HTML，
// 后台 /system/diy.html 通过拖拽排序 + 表单编辑。
type DiyPage struct {
	Id          uint   `json:"id" gorm:"column:id;type:int(10) unsigned not null AUTO_INCREMENT;primaryKey"`
	CreatedTime int64  `json:"created_time" gorm:"column:created_time;type:bigint(20);autoCreateTime"`
	UpdatedTime int64  `json:"updated_time" gorm:"column:updated_time;type:bigint(20);autoUpdateTime"`
	Key         string `json:"key" gorm:"column:key;type:varchar(50) not null;default:'';uniqueIndex:idx_diy_key"`
	Title       string `json:"title" gorm:"column:title;type:varchar(100) not null;default:''"`
	Modules     string `json:"modules" gorm:"column:modules;type:longtext"`
	Status      int    `json:"status" gorm:"column:status;type:tinyint(1) not null;default:1"`
}
