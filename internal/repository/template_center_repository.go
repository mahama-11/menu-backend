package repository

import (
	"fmt"
	"strings"
	"time"

	"menu-service/internal/models"

	"gorm.io/gorm"
)

type TemplateCenterRepository struct {
	db *gorm.DB
}

type TemplateCatalogListFilter struct {
	Cuisine  string
	DishType string
	Platform string
	Mood     string
	Query    string
	Status   string
}

func NewTemplateCenterRepository(db *gorm.DB) *TemplateCenterRepository {
	return &TemplateCenterRepository{db: db}
}

func (r *TemplateCenterRepository) CreateCatalog(item *models.TemplateCatalog) error {
	if item.ID == "" {
		item.ID = buildTemplateID("catalog")
	}
	return r.db.Create(item).Error
}

func (r *TemplateCenterRepository) SaveCatalog(item *models.TemplateCatalog) error {
	return r.db.Save(item).Error
}

func (r *TemplateCenterRepository) ListCatalogs(filter TemplateCatalogListFilter) ([]models.TemplateCatalog, error) {
	var items []models.TemplateCatalog
	q := r.db.Model(&models.TemplateCatalog{}).Order("sort_order asc, recommend_score desc, created_at desc")
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Cuisine != "" {
		q = q.Where("cuisine = ?", filter.Cuisine)
	}
	if filter.DishType != "" {
		q = q.Where("dish_type = ?", filter.DishType)
	}
	if filter.Platform != "" {
		q = q.Where("platforms_json LIKE ?", "%\""+filter.Platform+"\"%")
	}
	if filter.Mood != "" {
		q = q.Where("moods_json LIKE ?", "%\""+filter.Mood+"\"%")
	}
	if filter.Query != "" {
		like := "%" + strings.TrimSpace(filter.Query) + "%"
		q = q.Where("name LIKE ? OR description LIKE ? OR tags_json LIKE ?", like, like, like)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TemplateCenterRepository) FindCatalogByID(templateID string) (*models.TemplateCatalog, error) {
	var item models.TemplateCatalog
	if err := r.db.Where("id = ?", templateID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TemplateCenterRepository) CreateCatalogVersion(item *models.TemplateCatalogVersion) error {
	if item.ID == "" {
		item.ID = buildTemplateID("tplver")
	}
	return r.db.Create(item).Error
}

func (r *TemplateCenterRepository) SaveCatalogVersion(item *models.TemplateCatalogVersion) error {
	return r.db.Save(item).Error
}

func (r *TemplateCenterRepository) ReplaceCatalogExamples(versionID string, items []models.TemplateCatalogExample) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_version_id = ?", versionID).Delete(&models.TemplateCatalogExample{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for i := range items {
			if items[i].ID == "" {
				items[i].ID = buildTemplateID("tplex")
			}
			items[i].TemplateVersionID = versionID
		}
		return tx.Create(&items).Error
	})
}

func (r *TemplateCenterRepository) ListCatalogExamples(versionID string) ([]models.TemplateCatalogExample, error) {
	var items []models.TemplateCatalogExample
	if err := r.db.Where("template_version_id = ?", versionID).Order("sort_order asc, created_at asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TemplateCenterRepository) FindCatalogVersionByID(versionID string) (*models.TemplateCatalogVersion, error) {
	var item models.TemplateCatalogVersion
	if err := r.db.Where("id = ?", versionID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TemplateCenterRepository) ListCatalogVersions(templateID string) ([]models.TemplateCatalogVersion, error) {
	var items []models.TemplateCatalogVersion
	if err := r.db.Where("template_catalog_id = ?", templateID).Order("version_no desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TemplateCenterRepository) CreateFavorite(item *models.TemplateFavorite) error {
	if item.ID == "" {
		item.ID = buildTemplateID("tplfav")
	}
	return r.db.Create(item).Error
}

func (r *TemplateCenterRepository) FindFavorite(templateID, userID, orgID string) (*models.TemplateFavorite, error) {
	var item models.TemplateFavorite
	if err := r.db.Where("template_catalog_id = ? AND user_id = ? AND organization_id = ?", templateID, userID, orgID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TemplateCenterRepository) DeleteFavorite(templateID, userID, orgID string) error {
	return r.db.Where("template_catalog_id = ? AND user_id = ? AND organization_id = ?", templateID, userID, orgID).Delete(&models.TemplateFavorite{}).Error
}

func (r *TemplateCenterRepository) ListFavorites(userID, orgID string) ([]models.TemplateFavorite, error) {
	var items []models.TemplateFavorite
	if err := r.db.Where("user_id = ? AND organization_id = ?", userID, orgID).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TemplateCenterRepository) CreateUsageEvent(item *models.TemplateUsageEvent) error {
	if item.ID == "" {
		item.ID = buildTemplateID("tplevt")
	}
	return r.db.Create(item).Error
}

func buildTemplateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
