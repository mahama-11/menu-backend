package repository

import (
	"errors"
	"fmt"
	"time"

	"menu-service/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetPreference(userID, orgID string) (*models.UserPreference, error) {
	var item models.UserPreference
	if err := r.db.Where("user_id = ? AND organization_id = ?", userID, orgID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *UserRepository) UpsertPreference(userID, orgID, language string) (*models.UserPreference, error) {
	item, lookupErr := r.GetPreference(userID, orgID)
	if lookupErr == nil {
		item.LanguagePreference = language
		item.UpdatedAt = time.Now()
		if saveErr := r.db.Save(item).Error; saveErr != nil {
			return nil, saveErr
		}
		return item, nil
	}
	if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return nil, lookupErr
	}
	now := time.Now()
	item = &models.UserPreference{
		ID:                 buildMenuID("pref"),
		UserID:             userID,
		OrganizationID:     orgID,
		LanguagePreference: language,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if createErr := r.db.Create(item).Error; createErr != nil {
		return nil, createErr
	}
	return item, nil
}

func (r *UserRepository) ListActivities(userID, orgID string, limit, offset int) ([]models.Activity, int64, error) {
	var items []models.Activity
	var total int64
	base := r.db.Model(&models.Activity{}).Where("user_id = ? AND organization_id = ?", userID, orgID)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query := base.Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *UserRepository) CreateActivity(item *models.Activity) error {
	if item.ID == "" {
		item.ID = buildMenuID("act")
	}
	return r.db.Create(item).Error
}

func buildMenuID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
