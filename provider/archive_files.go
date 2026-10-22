package provider

import (
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
)

// 本文件负责：删除文档（以及删除模型）时，把正文/图集/自定义字段里引用的
// uploads 文件一起清掉。
//
// 设计要点：
//  1. 只处理 uploads/ 下的本站文件，外链、data: 一律忽略；
//  2. 删文件前必须确认没有别的地方还在引用它（其它文档正文、图集、栏目、导航、
//     友链、标签、地区、店铺、站点设置、模块副表的自定义字段），
//     否则会把「多篇文档共用的一张图」或站点 logo 误删；
//  3. 附件库里有记录的，走 DeleteAttachment：文件 + 缩略图 + 附件记录一起删，
//     避免媒体库里留下死链。
const archiveUploadRefLimit = 500

// uploadPathRe 从任意文本（HTML 正文、JSON、纯路径）里捞出 uploads/ 开头的路径。
// 终止符覆盖引号、尖括号、空白、括号、逗号、分号、竖线、反斜杠、? # 等常见分隔符，
// 避免把后面的 HTML/JSON 一起吞进来。
var uploadPathRe = regexp.MustCompile(`(?i)uploads/[^\s"'<>()\[\]{}\\,;|*?#]+`)

// archiveUploadSafeName 模块表名/字段名做一次白名单校验后再拼进 SQL。
var archiveUploadSafeName = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// archiveUploadOtherRefs 其它可能引用 uploads 文件的表（表名, 列名）。
// 都是固定常量，直接拼 SQL；不存在这种表时由 HasTable 兜底跳过。
var archiveUploadOtherRefs = []struct {
	table   string
	columns []string
}{
	{"settings", []string{"value"}},
	{"categories", []string{"images", "logo", "content"}},
	{"navs", []string{"logo"}},
	{"links", []string{"logo"}},
	{"tags", []string{"logo"}},
	{"places", []string{"images", "logo", "content"}},
	{"shops", []string{"logo"}},
}

// NormalizeUploadRef 把各种写法的图片地址统一成 `uploads/202309/15/a.png` 形式。
// 不是本站 uploads 资源（外链、data:、其它相对路径）时返回空串。
func NormalizeUploadRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	lower := strings.ToLower(ref)
	if strings.HasPrefix(lower, "data:") {
		return ""
	}
	// 允许绝对地址（本站域名 / 存储域名）、协议相对地址、带前导斜杠或纯相对路径，
	// 统一从 "uploads/" 处截断。非本站域名截出来的路径在本地/附件库里查不到，
	// 后续 uploadFileExists 会自然跳过。
	idx := strings.Index(lower, "uploads/")
	if idx < 0 {
		return ""
	}
	ref = ref[idx:]
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	if unescaped, err := url.PathUnescape(ref); err == nil {
		ref = unescaped
	}
	ref = strings.Trim(ref, "\"'.,;:")
	if ref == "" || strings.Contains(ref, "..") {
		return ""
	}

	return ref
}

// ExtractUploadRefs 从一段文本里提取全部 uploads 相对路径（去重、保序）。
func ExtractUploadRefs(text string) []string {
	if text == "" || !strings.Contains(strings.ToLower(text), "uploads/") {
		return nil
	}
	matches := uploadPathRe.FindAllString(text, -1)
	refs := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		ref := NormalizeUploadRef(m)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}

	return refs
}

// collectArchiveUploadRefs 收集一批文档引用的所有 uploads 文件。
//
// 必须在数据库记录被删除之前调用：正文、图集都还在库里。
func (w *Website) collectArchiveUploadRefs(ids []int64, moduleId uint, extra ...[]string) []string {
	if w.DB == nil || len(ids) == 0 {
		return nil
	}
	refs := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	appendText := func(text string) {
		for _, ref := range ExtractUploadRefs(text) {
			if _, ok := seen[ref]; ok {
				continue
			}
			seen[ref] = struct{}{}
			refs = append(refs, ref)
		}
	}

	for _, list := range extra {
		for _, v := range list {
			appendText(v)
		}
	}

	// 正文
	var contents []string
	w.DB.Model(&model.ArchiveData{}).Unscoped().Where("`id` IN (?)", ids).Pluck("content", &contents)
	for _, v := range contents {
		appendText(v)
	}
	// 图集（正式表 + 草稿表，任一存在即收集）
	var images []string
	w.DB.Model(&model.Archive{}).Unscoped().Where("`id` IN (?)", ids).Pluck("images", &images)
	w.DB.Model(&model.ArchiveDraft{}).Unscoped().Where("`id` IN (?)", ids).Pluck("images", &images)
	for _, v := range images {
		appendText(v)
	}

	// 模块副表的自定义字段（图片/文件/编辑器/图集/timeline 等都以文本落库）
	if module := w.GetModuleFromCache(moduleId); module != nil {
		table := module.TableName
		if table != "" && archiveUploadSafeName.MatchString(table) && w.DB.Migrator().HasTable(table) {
			rows := make([]map[string]any, 0, len(ids))
			w.DB.Table(table).Where("`id` IN (?)", ids).Find(&rows)
			for _, row := range rows {
				for _, v := range row {
					switch val := v.(type) {
					case string:
						appendText(val)
					case []byte:
						appendText(string(val))
					}
				}
			}
		}
	}

	return refs
}

// CleanupArchiveUploads 删除一批 uploads 文件（含缩略图与附件库记录）。
//
// 只要还能在别处找到引用就保留；超过 archiveUploadRefLimit 的部分直接保留，
// 宁可留下垃圾文件也不误删（删模型这种批量操作不该把请求拖成几分钟）。
func (w *Website) CleanupArchiveUploads(refs []string) int {
	if w.DB == nil || len(refs) == 0 {
		return 0
	}
	deleted := 0
	checked := 0
	seen := make(map[string]struct{}, len(refs))
	for _, raw := range refs {
		ref := NormalizeUploadRef(raw)
		if ref == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		if checked >= archiveUploadRefLimit {
			log.Printf("清理文档附件已达到单次上限(%d)，剩余文件保留：%s", archiveUploadRefLimit, ref)
			break
		}
		checked++
		if !w.uploadFileExists(ref) {
			continue
		}
		if w.uploadReferencedElsewhere(ref) {
			continue
		}
		w.deleteUploadRef(ref)
		deleted++
	}

	return deleted
}

// uploadFileExists 判断文件是否真的存在（本地文件或附件库记录），
// 避免正文里「看起来像路径」的字符串被当成文件去删。
// 云存储且未保留本地副本时只认附件库记录：远程 HEAD 太慢，也没有必要。
func (w *Website) uploadFileExists(ref string) bool {
	if attach, err := w.GetAttachmentByFileLocation(ref); err == nil && attach != nil {
		return true
	}
	if w.PluginStorage.StorageType == config.StorageTypeLocal || w.PluginStorage.KeepLocal {
		if _, err := os.Stat(w.PublicPath + strings.TrimPrefix(ref, "/")); err == nil {
			return true
		}
	}

	return false
}

// uploadReferencedElsewhere 判断文件是否还被其它文档/栏目/设置引用。
func (w *Website) uploadReferencedElsewhere(ref string) bool {
	like := "%" + ref + "%"
	// 正文与图集：文档可能很多，但这几条是判断共享素材的关键
	if w.tableHasLike("archive_data", []string{"content"}, like) {
		return true
	}
	if w.tableHasLike("archives", []string{"images"}, like) {
		return true
	}
	if w.tableHasLike("archive_drafts", []string{"images"}, like) {
		return true
	}
	// 站点设置（logo/banner/DIY/导航/友链等大量配置以 JSON 存在这里）
	for _, item := range archiveUploadOtherRefs {
		if w.tableHasLike(item.table, item.columns, like) {
			return true
		}
	}
	// 模块副表的自定义字段
	for _, module := range w.GetCacheModules() {
		table := module.TableName
		if table == "" || !archiveUploadSafeName.MatchString(table) {
			continue
		}
		columns := make([]string, 0, 4)
		for _, field := range module.Fields {
			switch field.Type {
			case config.CustomFieldTypeImage, config.CustomFieldTypeFile, config.CustomFieldTypeImages,
				config.CustomFieldTypeEditor, config.CustomFieldTypeTexts, config.CustomFieldTypeTimeline:
				if archiveUploadSafeName.MatchString(field.FieldName) {
					columns = append(columns, field.FieldName)
				}
			}
		}
		if len(columns) == 0 {
			continue
		}
		if w.tableHasLike(table, columns, like) {
			return true
		}
	}

	return false
}

// tableHasLike 判断表里是否存在某几列之一匹配 like 的行。
func (w *Website) tableHasLike(table string, columns []string, like string) bool {
	if w.DB == nil || len(columns) == 0 || !archiveUploadSafeName.MatchString(table) {
		return false
	}
	if !w.DB.Migrator().HasTable(table) {
		return false
	}
	conds := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		if !archiveUploadSafeName.MatchString(column) {
			continue
		}
		conds = append(conds, "`"+column+"` LIKE ?")
		args = append(args, like)
	}
	if len(conds) == 0 {
		return false
	}
	row := w.DB.Raw("SELECT 1 FROM `"+table+"` WHERE "+strings.Join(conds, " OR ")+" LIMIT 1", args...).Row()
	if row == nil {
		return false
	}
	var one int
	if err := row.Scan(&one); err != nil {
		return false
	}

	return true
}

// deleteUploadRef 删除文件本体；有附件记录时连记录和缩略图一起删。
func (w *Website) deleteUploadRef(ref string) {
	ref = strings.TrimPrefix(ref, "/")
	if !strings.HasPrefix(ref, "uploads/") {
		return
	}
	if attach, err := w.GetAttachmentByFileLocation(ref); err == nil && attach != nil {
		if err := w.DeleteAttachment(attach); err != nil {
			log.Println("删除文档附件失败：", ref, err)
		}
		return
	}
	if err := w.DeleteFile(ref); err != nil {
		log.Println("删除文档附件文件失败：", ref, err)
	}
	paths, name := filepath.Split(ref)
	if name != "" {
		_ = w.DeleteFile(paths + "thumb_" + name)
	}
}
