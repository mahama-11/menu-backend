package repository

import (
	"errors"
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

func (r *ShareRepository) FindPostByToken(token string) (*models.SharePost, error) {
	var item models.SharePost
	if err := r.db.Where("share_token = ?", token).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShareRepository) ListPublicPosts(limit int, sort string) ([]models.SharePost, error) {
	var items []models.SharePost
	q := r.db.Where("visibility = ? AND status = ?", "public", "published")

	switch sort {
	case "popular":
		q = q.Order("like_count desc").Order("favorite_count desc").Order("view_count desc").Order("published_at desc").Order("created_at desc")
	default:
		q = q.Order("published_at desc").Order("created_at desc")
	}

	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ShareRepository) ListFavoritePosts(orgID, userID string, limit int) ([]models.SharePost, error) {
	var favorites []models.SharePostFavorite
	q := r.db.
		Model(&models.SharePostFavorite{}).
		Where("organization_id = ? AND user_id = ?", orgID, userID).
		Order("created_at desc")

	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&favorites).Error; err != nil {
		return nil, err
	}
	if len(favorites) == 0 {
		return []models.SharePost{}, nil
	}

	postIDs := make([]string, 0, len(favorites))
	for _, favorite := range favorites {
		postIDs = append(postIDs, favorite.SharePostID)
	}

	var items []models.SharePost
	if err := r.db.Where("organization_id = ? AND id IN ?", orgID, postIDs).Find(&items).Error; err != nil {
		return nil, err
	}

	itemsByID := make(map[string]models.SharePost, len(items))
	for _, item := range items {
		itemsByID[item.ID] = item
	}

	ordered := make([]models.SharePost, 0, len(favorites))
	for _, favorite := range favorites {
		if item, ok := itemsByID[favorite.SharePostID]; ok {
			ordered = append(ordered, item)
		}
	}
	return ordered, nil
}

func (r *ShareRepository) IncrementViewCount(postID string) error {
	return r.db.Model(&models.SharePost{}).Where("id = ?", postID).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

func (r *ShareRepository) GetEngagementState(orgID, userID, postID string) (viewerLiked bool, viewerFavorited bool, err error) {
	if userID == "" {
		return false, false, nil
	}

	var like models.SharePostLike
	if err = r.db.Where("organization_id = ? AND share_post_id = ? AND user_id = ?", orgID, postID, userID).First(&like).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, err
		}
		err = nil
	} else {
		viewerLiked = true
	}

	var favorite models.SharePostFavorite
	if err = r.db.Where("organization_id = ? AND share_post_id = ? AND user_id = ?", orgID, postID, userID).First(&favorite).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, err
		}
		err = nil
	} else {
		viewerFavorited = true
	}

	return viewerLiked, viewerFavorited, nil
}

func (r *ShareRepository) SetLike(orgID, userID, postID string, active bool) (bool, error) {
	resultActive := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var item models.SharePostLike
		err := tx.Where("organization_id = ? AND share_post_id = ? AND user_id = ?", orgID, postID, userID).First(&item).Error
		exists := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		switch {
		case active && !exists:
			item = models.SharePostLike{
				ID:             buildShareID("share_like"),
				OrganizationID: orgID,
				SharePostID:    postID,
				UserID:         userID,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.SharePost{}).Where("organization_id = ? AND id = ?", orgID, postID).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
				return err
			}
			resultActive = true
		case !active && exists:
			if err := tx.Delete(&item).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.SharePost{}).Where("organization_id = ? AND id = ?", orgID, postID).UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error; err != nil {
				return err
			}
			resultActive = false
		default:
			resultActive = active && exists
		}

		return nil
	})
	return resultActive, err
}

func (r *ShareRepository) SetFavorite(orgID, userID, postID string, active bool) (bool, error) {
	resultActive := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var item models.SharePostFavorite
		err := tx.Where("organization_id = ? AND share_post_id = ? AND user_id = ?", orgID, postID, userID).First(&item).Error
		exists := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		switch {
		case active && !exists:
			item = models.SharePostFavorite{
				ID:             buildShareID("share_fav"),
				OrganizationID: orgID,
				SharePostID:    postID,
				UserID:         userID,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.SharePost{}).Where("organization_id = ? AND id = ?", orgID, postID).UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1")).Error; err != nil {
				return err
			}
			resultActive = true
		case !active && exists:
			if err := tx.Delete(&item).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.SharePost{}).Where("organization_id = ? AND id = ?", orgID, postID).UpdateColumn("favorite_count", gorm.Expr("CASE WHEN favorite_count > 0 THEN favorite_count - 1 ELSE 0 END")).Error; err != nil {
				return err
			}
			resultActive = false
		default:
			resultActive = active && exists
		}

		return nil
	})
	return resultActive, err
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
