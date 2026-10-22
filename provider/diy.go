package provider

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"

	"kandaoni.com/anqicms/model"
)

// 可视化装修。
//
// 设计：一个页面 = 若干「区块」(module)，区块类型固定（banner/notice/...），
// 每个区块的字段由 Go 侧 schema 描述（DiyModuleTypes），后台页面据此自动生成表单，
// 前台由 RenderDiyPage 统一渲染。数据只有一张表 diy_pages，Modules 存 JSON。

// DiyModule 一个区块
type DiyModule struct {
	Id       string                 `json:"id"`
	Type     string                 `json:"type"`
	Settings map[string]interface{} `json:"settings"`
}

// DiyOption 下拉选项
type DiyOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// DiyField 区块字段描述（后台据此渲染表单）
type DiyField struct {
	Name        string      `json:"name"`
	Label       string      `json:"label"`
	Type        string      `json:"type"` // text / textarea / number / color / image / richtext / select / list
	Placeholder string      `json:"placeholder,omitempty"`
	Tip         string      `json:"tip,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Options     []DiyOption `json:"options,omitempty"`
	Fields      []DiyField  `json:"fields,omitempty"` // type=list 时的子字段
}

// DiyModuleType 区块类型定义
type DiyModuleType struct {
	Type   string     `json:"type"`
	Title  string     `json:"title"`
	Icon   string     `json:"icon"`
	Desc   string     `json:"desc"`
	Fields []DiyField `json:"fields"`
}

// DiyModuleTypes 全部可用区块（顺序即后台模块库顺序）
func DiyModuleTypes() []DiyModuleType {
	return []DiyModuleType{
		{
			Type:  "banner",
			Title: "轮播图",
			Icon:  "🖼",
			Desc:  "顶部大图轮播，可配多张",
			Fields: []DiyField{
				{
					Name:  "items",
					Label: "图片",
					Type:  "list",
					Tip:   "建议 1920×600，第一张为首屏图",
					Fields: []DiyField{
						{Name: "image", Label: "图片", Type: "image"},
						{Name: "link", Label: "跳转链接", Type: "text", Placeholder: "留空则不跳转"},
						{Name: "title", Label: "标题", Type: "text", Placeholder: "可留空"},
					},
				},
				{Name: "height", Label: "高度(px)", Type: "number", Default: 420},
			},
		},
		{
			Type:  "notice",
			Title: "公告条",
			Icon:  "📢",
			Desc:  "一行提示文字，可带链接",
			Fields: []DiyField{
				{Name: "text", Label: "内容", Type: "text", Placeholder: "例如：双十一全场 8 折，满 999 减 100"},
				{Name: "link", Label: "链接", Type: "text", Placeholder: "可留空"},
				{Name: "color", Label: "文字颜色", Type: "color", Default: "#0068AB"},
			},
		},
		{
			Type:  "richtext",
			Title: "富文本",
			Icon:  "📝",
			Desc:  "自由排版，支持 HTML",
			Fields: []DiyField{
				{Name: "title", Label: "区块标题", Type: "text", Placeholder: "可留空"},
				{Name: "html", Label: "内容", Type: "richtext", Tip: "支持 HTML 标签"},
			},
		},
		{
			Type:  "image",
			Title: "单图",
			Icon:  "🏞",
			Desc:  "一张通栏或居中的图片",
			Fields: []DiyField{
				{Name: "image", Label: "图片", Type: "image"},
				{Name: "link", Label: "跳转链接", Type: "text", Placeholder: "可留空"},
				{Name: "align", Label: "对齐", Type: "select", Default: "center", Options: []DiyOption{
					{Label: "居中", Value: "center"}, {Label: "通栏", Value: "full"},
				}},
			},
		},
		{
			Type:  "nav",
			Title: "图文导航",
			Icon:  "🧭",
			Desc:  "图标 + 标题 + 链接的宫格",
			Fields: []DiyField{
				{Name: "title", Label: "区块标题", Type: "text", Placeholder: "可留空"},
				{
					Name:  "items",
					Label: "导航项",
					Type:  "list",
					Fields: []DiyField{
						{Name: "icon", Label: "图标图片", Type: "image"},
						{Name: "title", Label: "标题", Type: "text"},
						{Name: "link", Label: "链接", Type: "text"},
					},
				},
			},
		},
		{
			Type:  "products",
			Title: "产品列表",
			Icon:  "📦",
			Desc:  "按栏目取产品，卡片网格",
			Fields: []DiyField{
				{Name: "title", Label: "区块标题", Type: "text", Default: "热门产品"},
				{Name: "category_id", Label: "栏目 ID", Type: "number", Tip: "留空表示全部产品"},
				{Name: "limit", Label: "数量", Type: "number", Default: 8},
			},
		},
		{
			Type:  "articles",
			Title: "文章列表",
			Icon:  "📰",
			Desc:  "按栏目取文章，标题 + 摘要",
			Fields: []DiyField{
				{Name: "title", Label: "区块标题", Type: "text", Default: "新闻资讯"},
				{Name: "category_id", Label: "栏目 ID", Type: "number", Tip: "留空表示全部文章"},
				{Name: "limit", Label: "数量", Type: "number", Default: 6},
			},
		},
		{
			Type:  "divider",
			Title: "分隔/留白",
			Icon:  "➖",
			Desc:  "控制区块之间的间距",
			Fields: []DiyField{
				{Name: "height", Label: "高度(px)", Type: "number", Default: 40},
			},
		},
	}
}

// DiyPageKeys 可装修的页面
func DiyPageKeys() []DiyOption {
	return []DiyOption{
		{Label: "网站首页", Value: "home"},
	}
}

// GetDiyPage 取页面配置，不存在返回 nil
func (w *Website) GetDiyPage(key string) *model.DiyPage {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	var page model.DiyPage
	if err := w.DB.Where("`key` = ?", key).Take(&page).Error; err != nil {
		return nil
	}
	return &page
}

// GetDiyModules 解析出区块列表
func (w *Website) GetDiyModules(key string) []DiyModule {
	page := w.GetDiyPage(key)
	if page == nil || strings.TrimSpace(page.Modules) == "" {
		return nil
	}
	var list []DiyModule
	if err := json.Unmarshal([]byte(page.Modules), &list); err != nil {
		return nil
	}
	return list
}

// SaveDiyPage 保存页面区块
func (w *Website) SaveDiyPage(key, title string, modules []DiyModule) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%s", w.TplTr("InvalidOperation"))
	}
	// 兜底：只接受已知页面 key，避免后台被写入任意键
	valid := false
	for _, item := range DiyPageKeys() {
		if item.Value == key {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("%s", w.TplTr("InvalidOperation"))
	}
	if modules == nil {
		modules = []DiyModule{}
	}
	// 补齐 id，保证前端排序稳定
	for i := range modules {
		if modules[i].Id == "" {
			modules[i].Id = fmt.Sprintf("%s-%d-%d", modules[i].Type, i, len(modules))
		}
		if modules[i].Settings == nil {
			modules[i].Settings = map[string]interface{}{}
		}
	}
	raw, err := json.Marshal(modules)
	if err != nil {
		return err
	}
	page := w.GetDiyPage(key)
	if page == nil {
		page = &model.DiyPage{Key: key, Title: title, Modules: string(raw), Status: 1}
		if title == "" {
			page.Title = key
		}
		return w.DB.Create(page).Error
	}
	return w.DB.Model(&model.DiyPage{}).Where("`id` = ?", page.Id).Updates(map[string]interface{}{
		"title":   title,
		"modules": string(raw),
		"status":  1,
	}).Error
}

// ResetDiyPage 清空页面装修（前台回退到模板默认内容）
func (w *Website) ResetDiyPage(key string) error {
	page := w.GetDiyPage(key)
	if page == nil {
		return nil
	}
	return w.DB.Model(&model.DiyPage{}).Where("`id` = ?", page.Id).UpdateColumn("modules", "[]").Error
}

// RenderDiyPage 把页面区块渲染成 HTML；没有配置返回空串（模板据此回退默认内容）
func (w *Website) RenderDiyPage(key string) string {
	modules := w.GetDiyModules(key)
	if len(modules) == 0 {
		return ""
	}
	var b strings.Builder
	for _, item := range modules {
		block := w.renderDiyModule(item)
		if block != "" {
			b.WriteString(block)
		}
	}
	return b.String()
}

// ─────────────────────────────────────────── 渲染

func (w *Website) renderDiyModule(item DiyModule) string {
	s := item.Settings
	switch item.Type {
	case "banner":
		return w.renderDiyBanner(s)
	case "notice":
		return w.renderDiyNotice(s)
	case "richtext":
		return w.renderDiyRichText(s)
	case "image":
		return w.renderDiyImage(s)
	case "nav":
		return w.renderDiyNav(s)
	case "products":
		return w.renderDiyArchiveList(s, 2, true)
	case "articles":
		return w.renderDiyArchiveList(s, 1, false)
	case "divider":
		height := diyInt(s, "height", 40)
		return fmt.Sprintf(`<div class="diy-divider" style="height:%dpx"></div>`, height)
	}
	return ""
}

func (w *Website) renderDiyBanner(s map[string]interface{}) string {
	items := diyList(s, "items")
	if len(items) == 0 {
		return ""
	}
	height := diyInt(s, "height", 420)
	var b strings.Builder
	b.WriteString(`<div class="diy-block diy-banner"><div class="container"><div class="diy-banner__track" style="height:` + strconv.Itoa(height) + `px">`)
	for _, row := range items {
		src := w.StorageFullUrl(diyStr(row, "image"))
		if src == "" {
			continue
		}
		link := diyStr(row, "link")
		title := diyStr(row, "title")
		alt := html.EscapeString(title)
		if link != "" {
			b.WriteString(`<a class="diy-banner__item" href="` + html.EscapeString(link) + `" target="_blank"><img src="` + html.EscapeString(src) + `" alt="` + alt + `"></a>`)
		} else {
			b.WriteString(`<div class="diy-banner__item"><img src="` + html.EscapeString(src) + `" alt="` + alt + `"></div>`)
		}
	}
	b.WriteString(`</div></div></div>`)
	return b.String()
}

func (w *Website) renderDiyNotice(s map[string]interface{}) string {
	text := diyStr(s, "text")
	if text == "" {
		return ""
	}
	color := diyStr(s, "color")
	if color == "" {
		color = "#0068AB"
	}
	link := diyStr(s, "link")
	inner := `<span class="diy-notice__icon">📢</span><span class="diy-notice__text">` + html.EscapeString(text) + `</span>`
	if link != "" {
		inner = `<a href="` + html.EscapeString(link) + `">` + inner + `</a>`
	}
	return `<div class="diy-block diy-notice" style="color:` + html.EscapeString(color) + `"><div class="container">` + inner + `</div></div>`
}

func (w *Website) renderDiyRichText(s map[string]interface{}) string {
	body := diyStr(s, "html")
	if body == "" {
		return ""
	}
	title := diyStr(s, "title")
	var b strings.Builder
	b.WriteString(`<div class="diy-block diy-richtext"><div class="container">`)
	if title != "" {
		b.WriteString(`<div class="diy-block__head"><h2>` + html.EscapeString(title) + `</h2></div>`)
	}
	b.WriteString(`<div class="diy-richtext__body">` + body + `</div></div></div>`)
	return b.String()
}

func (w *Website) renderDiyImage(s map[string]interface{}) string {
	src := w.StorageFullUrl(diyStr(s, "image"))
	if src == "" {
		return ""
	}
	link := diyStr(s, "link")
	align := diyStr(s, "align")
	cls := "diy-image diy-image--center"
	if align == "full" {
		cls = "diy-image diy-image--full"
	}
	img := `<img src="` + html.EscapeString(src) + `" alt="">`
	if link != "" {
		img = `<a href="` + html.EscapeString(link) + `" target="_blank">` + img + `</a>`
	}
	if align == "full" {
		return `<div class="diy-block ` + cls + `">` + img + `</div>`
	}
	return `<div class="diy-block ` + cls + `"><div class="container">` + img + `</div></div>`
}

func (w *Website) renderDiyNav(s map[string]interface{}) string {
	items := diyList(s, "items")
	if len(items) == 0 {
		return ""
	}
	title := diyStr(s, "title")
	var b strings.Builder
	b.WriteString(`<div class="diy-block diy-nav"><div class="container">`)
	if title != "" {
		b.WriteString(`<div class="diy-block__head"><h2>` + html.EscapeString(title) + `</h2></div>`)
	}
	b.WriteString(`<div class="diy-nav__grid">`)
	for _, row := range items {
		name := diyStr(row, "title")
		if name == "" {
			continue
		}
		link := diyStr(row, "link")
		if link == "" {
			link = "javascript:;"
		}
		icon := w.StorageFullUrl(diyStr(row, "icon"))
		b.WriteString(`<a class="diy-nav__item" href="` + html.EscapeString(link) + `">`)
		if icon != "" {
			b.WriteString(`<span class="diy-nav__icon"><img src="` + html.EscapeString(icon) + `" alt=""></span>`)
		}
		b.WriteString(`<span class="diy-nav__title">` + html.EscapeString(name) + `</span></a>`)
	}
	b.WriteString(`</div></div></div>`)
	return b.String()
}

// renderDiyArchiveList 产品/文章卡片列表
func (w *Website) renderDiyArchiveList(s map[string]interface{}, defaultModule uint, withPrice bool) string {
	limit := diyInt(s, "limit", 8)
	if limit <= 0 || limit > 48 {
		limit = 8
	}
	categoryId := diyInt(s, "category_id", 0)

	// archives 表只有已发布内容（草稿在 archive_drafts），没有 status 字段
	tx := w.DB.Model(&model.Archive{}).Where("`module_id` = ?", defaultModule)
	if categoryId > 0 {
		tx = tx.Where("`category_id` = ?", categoryId)
	}
	var list []*model.Archive
	tx.Order("id desc").Limit(limit).Find(&list)
	if len(list) == 0 {
		return ""
	}

	title := diyStr(s, "title")
	var b strings.Builder
	b.WriteString(`<div class="diy-block diy-archives"><div class="container">`)
	if title != "" {
		b.WriteString(`<div class="diy-block__head"><h2>` + html.EscapeString(title) + `</h2></div>`)
	}
	b.WriteString(`<div class="diy-archives__grid">`)
	for _, item := range list {
		item.GetThumb(w.PluginStorage.StorageUrl, w.Content.DefaultThumb)
		link := w.GetUrl("archive", item, 0)
		thumb := w.StorageFullUrl(item.Thumb)
		b.WriteString(`<a class="diy-card" href="` + html.EscapeString(link) + `">`)
		if thumb != "" {
			b.WriteString(`<span class="diy-card__thumb"><img src="` + html.EscapeString(thumb) + `" alt="` + html.EscapeString(item.Title) + `" loading="lazy"></span>`)
		}
		b.WriteString(`<span class="diy-card__body"><em class="diy-card__title">` + html.EscapeString(item.Title) + `</em>`)
		if withPrice && item.Price > 0 {
			b.WriteString(`<em class="diy-card__price">¥` + fmt.Sprintf("%.2f", float64(item.Price)/100) + `</em>`)
		} else if item.Description != "" {
			b.WriteString(`<em class="diy-card__desc">` + html.EscapeString(diyTruncate(item.Description, 56)) + `</em>`)
		}
		b.WriteString(`</span></a>`)
	}
	b.WriteString(`</div></div></div>`)
	return b.String()
}

// ─────────────────────────────────────────── 取值工具

func diyStr(s map[string]interface{}, key string) string {
	if s == nil {
		return ""
	}
	v, ok := s[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "1"
		}
		return ""
	}
	return ""
}

func diyInt(s map[string]interface{}, key string, def int) int {
	raw := diyStr(s, key)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(strings.Split(raw, ".")[0])
	if err != nil {
		return def
	}
	return n
}

func diyList(s map[string]interface{}, key string) []map[string]interface{} {
	if s == nil {
		return nil
	}
	v, ok := s[key]
	if !ok {
		return nil
	}
	rows, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		if m, ok2 := row.(map[string]interface{}); ok2 {
			out = append(out, m)
		}
	}
	return out
}

func diyTruncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
