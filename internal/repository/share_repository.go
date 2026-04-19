package repository

import (
	"fmt"
	"time"

	"menu-service/internal/models"

	"gorm.io/gorm"
)

type ShareRepository struct {
	db *gorm.DB
}

func NewShareRepository(db *gorm.DB) *ShareRepository {
	return &ShareRepository{db: db}
}

func (r *ShareRepository) CreatePost(item *models.SharePost) error {
	if item.ID == "" {
		item.ID = buildShareID("share")
	}
	return r.db.Create(item).Error
}

func (r *ShareRepository) SavePost(item *models.SharePost) error {
	return r.db.Save(item).Error
}

func (r *ShareRepository) ListPosts(orgID, userID, status string, limit int) ([]models.SharePost, error) {
	var items []models.SharePost
	q := r.db.Where("organization_id = ?", orgID).Order("created_at desc")
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ShareRepository) FindPostByID(orgID, postID string) (*models.SharePost, error) {
	var item models.SharePost
	if err := r.db.Where("organization_id = ? AND id = ?", orgID, postID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShareRepository) FindPostsByAssetIDs(orgID string, assetIDs []string) ([]models.SharePost, error) {
	if len(assetIDs) == 0 {
		return []models.SharePost{}, nil
	}
	var items []models.SharePost
	if err := r.db.Where("organization_id = ? AND asset_id IN ?", orgID, assetIDs).Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func buildShareID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
