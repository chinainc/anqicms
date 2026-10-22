package provider

import (
	"errors"
	"regexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"kandaoni.com/anqicms/config"
	"kandaoni.com/anqicms/model"
	"kandaoni.com/anqicms/request"
)

func (w *Website) GetModules() ([]model.Module, error) {
	var modules []model.Module
	err := w.DB.Order("id asc").Find(&modules).Error
	if err != nil {
		return nil, err
	}
	return modules, nil
}

func (w *Website) GetModuleById(id uint) (*model.Module, error) {
	var module model.Module
	db := w.DB
	err := db.Where("`id` = ?", id).First(&module).Error
	if err != nil {
		return nil, err
	}

	return &module, nil
}

func (w *Website) GetModuleByTableName(tableName string) (*model.Module, error) {
	var module model.Module
	db := w.DB
	err := db.Where("`table_name` = ?", tableName).First(&module).Error
	if err != nil {
		return nil, err
	}

	return &module, nil
}

func (w *Website) GetModuleByUrlToken(urlToken string) (*model.Module, error) {
	var module model.Module
	db := w.DB
	err := db.Where("`url_token` = ?", urlToken).First(&module).Error
	if err != nil {
		return nil, err
	}

	return &module, nil
}

func (w *Website) SaveModule(req *request.ModuleRequest) (module *model.Module, err error) {
	if req.Id > 0 {
		module, err = w.GetModuleById(req.Id)
		if err != nil {
			// 表示不存在，则新建一个
			module = &model.Module{
				Status: 1,
			}
			module.Id = req.Id
		}
	} else {
		module = &model.Module{
			Status: 1,
		}
	}
	oldTableName := ""
	if req.UpdateAll || req.TableName != "" {
		// 检查tableName
		exists, err := w.GetModuleByTableName(req.TableName)
		if err == nil && exists.Id != req.Id {
			return nil, errors.New(w.Tr("ModelTableNameAlreadyExists"))
		}

		oldTableName = module.TableName
		module.TableName = req.TableName

		if oldTableName != module.TableName {
			// 表示是新表
			if w.DB.Migrator().HasTable(module.TableName) {
				return nil, errors.New(w.Tr("ModelTableNameAlreadyExists"))
			}
		}
	}

	if req.UpdateAll || req.UrlToken != "" {
		// 检查tableName
		exists, err := w.GetModuleByUrlToken(req.UrlToken)
		if err == nil && exists.Id != req.Id {
			return nil, errors.New(w.Tr("ModelUrlAliasAlreadyExists"))
		}
	}

	if req.UpdateAll || req.Fields != nil {
		// 检查fields
		for i := range req.Fields {
			// 不允许使用已存在的字段
			archiveFields, err := getColumns(w.DB, &model.Archive{})
			if err == nil {
				for _, val := range archiveFields {
					if val == req.Fields[i].FieldName {
						return nil, errors.New(req.Fields[i].FieldName + w.Tr("FieldAlreadyExists"))
					}
				}
			}
			match, err := regexp.MatchString(`^[a-z][0-9a-z_]+$`, req.Fields[i].FieldName)
			if err != nil || !match {
				return nil, errors.New(req.Fields[i].FieldName + w.Tr("IncorrectNaming"))
			}
		}
		module.Fields = req.Fields
	}
	if req.UpdateAll || req.CategoryFields != nil {
		// 检查 categoryFields
		// 检查fields
		for i := range req.CategoryFields {
			// 不允许使用已存在的字段
			categoryFields, err := getColumns(w.DB, &model.Category{})
			if err == nil {
				for _, val := range categoryFields {
					if val == req.CategoryFields[i].FieldName {
						return nil, errors.New(req.CategoryFields[i].FieldName + w.Tr("FieldAlreadyExists"))
					}
				}
			}
			match, err := regexp.MatchString(`^[a-z][0-9a-z_]+$`, req.CategoryFields[i].FieldName)
			if err != nil || !match {
				return nil, errors.New(req.CategoryFields[i].FieldName + w.Tr("IncorrectNaming"))
			}
		}
		module.CategoryFields = req.CategoryFields
	}

	if req.UpdateAll || req.Title != "" {
		module.Title = req.Title
	}
	if req.UpdateAll || req.Name != "" {
		module.Name = req.Name
	}
	if req.UpdateAll || req.UrlToken != "" {
		module.UrlToken = req.UrlToken
	}
	if req.UpdateAll || req.Status > 0 {
		module.Status = req.Status
	}
	if req.UpdateAll || req.TitleName != "" {
		module.Keywords = req.Keywords
	}
	if req.UpdateAll || req.Description != "" {
		module.Description = req.Description
	}

	err = w.DB.Save(module).Error
	if err != nil {
		return
	}
	// sync table
	if oldTableName != "" && oldTableName != module.TableName {
		if w.DB.Migrator().HasTable(oldTableName) {
			w.DB.Migrator().RenameTable(oldTableName, module.TableName)
		}
	}
	module.Database = w.Mysql.Database
	tplPath := w.GetTemplateDir() + "/" + module.TableName
	module.Migrate(w.DB, tplPath, true)

	w.DeleteCacheModules()

	return
}

func (w *Website) DeleteModuleField(moduleId uint, fieldName string) error {
	module, err := w.GetModuleById(moduleId)
	if err != nil {
		return err
	}

	if !w.DB.Migrator().HasTable(module.TableName) {
		return nil
	}

	for i, val := range module.Fields {
		if val.FieldName == fieldName {
			if module.HasColumn(w.DB, val.FieldName) {
				w.DB.Exec("ALTER TABLE ? DROP COLUMN ?", clause.Table{Name: module.TableName}, clause.Column{Name: val.FieldName})
			}

			module.Fields = append(module.Fields[:i], module.Fields[i+1:]...)
			break
		}
	}
	// 回写
	err = w.DB.Save(module).Error
	return err
}

func (w *Website) DeleteModule(module *model.Module) error {
	if module == nil || w.DB == nil {
		return errors.New("module not found")
	}
	var firstErr error
	var files []string
	// 单次批量，避免大表一次性 load 出所有 id
	const batchSize = 1000
	for i := 0; i < 100000; i++ {
		ids := make([]int64, 0, batchSize)
		err := w.DB.Model(&model.Archive{}).Unscoped().Where("`module_id` = ?", module.Id).Order("id").Limit(batchSize).Pluck("id", &ids).Error
		if err != nil {
			firstErr = err
			break
		}
		if len(ids) == 0 {
			break
		}
		// 正文里引用的图片/附件必须在删除数据库记录之前收集
		files = append(files, w.collectArchiveUploadRefs(ids, module.Id)...)
		if err = w.deleteArchiveRelationsBatch(ids, module.TableName); err != nil {
			firstErr = err
			break
		}
	}
	// 草稿箱 / 待审核 / 回收站里的内容同样属于该模型，必须一起删
	// （历史实现只删 archives，导致删掉模型后回收站里还能看到这些文档和正文）
	for i := 0; i < 100000; i++ {
		ids := make([]int64, 0, batchSize)
		err := w.DB.Model(&model.ArchiveDraft{}).Unscoped().Where("`module_id` = ?", module.Id).Order("id").Limit(batchSize).Pluck("id", &ids).Error
		if err != nil {
			firstErr = err
			break
		}
		if len(ids) == 0 {
			break
		}
		files = append(files, w.collectArchiveUploadRefs(ids, module.Id)...)
		if err = w.deleteArchiveRelationsBatch(ids, module.TableName); err != nil {
			firstErr = err
			break
		}
	}
	// 删除模型表
	if w.DB.Migrator().HasTable(module.TableName) {
		if err := w.DB.Migrator().DropTable(module.TableName); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	// 模型下的栏目一起删掉，否则会留下指向已删除模型的孤儿栏目
	if err := w.DB.Where("`module_id` = ?", module.Id).Delete(&model.Category{}).Error; err != nil && firstErr == nil {
		firstErr = err
	}
	// 删除 module
	if err := w.DB.Delete(module).Error; err != nil && firstErr == nil {
		firstErr = err
	}
	// 清理正文引用的 uploads 文件（其它文档/栏目/设置仍在用的会保留）
	w.CleanupArchiveUploads(files)

	return firstErr
}

// deleteArchiveRelationsBatch 按 id 批量删除文档的正文、模块副表、分类、属性、标签、草稿行与正式行。
func (w *Website) deleteArchiveRelationsBatch(ids []int64, moduleTable string) error {
	if len(ids) == 0 {
		return nil
	}
	var firstErr error
	saveErr := func(tx *gorm.DB) {
		if tx.Error != nil && firstErr == nil {
			firstErr = tx.Error
		}
	}
	saveErr(w.DB.Unscoped().Where("`id` IN (?)", ids).Delete(&model.ArchiveData{}))
	if moduleTable != "" && archiveUploadSafeName.MatchString(moduleTable) && w.DB.Migrator().HasTable(moduleTable) {
		saveErr(w.DB.Exec("DELETE FROM ? WHERE `id` IN (?)", clause.Table{Name: moduleTable}, ids))
	}
	saveErr(w.DB.Unscoped().Where("`archive_id` IN (?)", ids).Delete(&model.ArchiveCategory{}))
	saveErr(w.DB.Unscoped().Where("`archive_id` IN (?)", ids).Delete(&model.ArchiveFlag{}))
	saveErr(w.DB.Unscoped().Where("`item_id` IN (?)", ids).Delete(&model.TagData{}))
	saveErr(w.DB.Unscoped().Where("`id` IN (?)", ids).Delete(&model.ArchiveDraft{}))
	saveErr(w.DB.Unscoped().Where("`id` IN (?)", ids).Delete(&model.Archive{}))

	return firstErr
}

func (w *Website) DeleteCacheModules() {
	w.Cache.Delete("modules")
}

func (w *Website) GetCacheModules() []model.Module {
	if w.DB == nil {
		return nil
	}
	var modules []model.Module

	err := w.Cache.Get("modules", &modules)
	if err == nil && len(modules) > 0 {
		return modules
	}

	err = w.DB.Model(model.Module{}).Where("`status` = ?", config.ContentStatusOK).Find(&modules).Error
	if err != nil {
		return nil
	}
	if len(modules) > 0 {
		_ = w.Cache.Set("modules", modules, 0)
	}

	return modules
}

func (w *Website) GetModuleFromCache(moduleId uint) *model.Module {
	modules := w.GetCacheModules()
	for i := range modules {
		if modules[i].Id == moduleId {
			return &modules[i]
		}
	}

	return nil
}

func (w *Website) GetModuleFromCacheByToken(urlToken string) *model.Module {
	modules := w.GetCacheModules()
	for i := range modules {
		if modules[i].UrlToken == urlToken {
			return &modules[i]
		}
	}

	return nil
}
