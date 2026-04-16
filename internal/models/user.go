package models

import "time"

type UserPreference struct {
	ID                 string    `gorm:"type:varchar(64);primaryKey"`
	UserID             string    `gorm:"type:varchar(64);index:idx_user_org_pref,unique;not null"`
	OrganizationID     string    `gorm:"type:varchar(64);index:idx_user_org_pref,unique;not null"`
	LanguagePreference string    `gorm:"type:varchar(16);not null;default:en"`
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}

type Activity struct {
	ID             string    `gorm:"type:varchar(64);primaryKey"`
	UserID         string    `gorm:"type:varchar(64);index:idx_activity_user_org_created,priority:1;not null"`
	OrganizationID string    `gorm:"type:varchar(64);index:idx_activity_user_org_created,priority:2;not null"`
	ActionType     string    `gorm:"type:varchar(64);not null"`
	ActionName     string    `gorm:"type:varchar(128);not null"`
	Status         string    `gorm:"type:varchar(32);not null;default:succeeded"`
	CreditsUsed    int64     `gorm:"not null;default:0"`
	ResultURL      string    `gorm:"type:text"`
	EventID        string    `gorm:"type:varchar(64)"`
	JobID          string    `gorm:"type:varchar(64)"`
	ErrorMessage   string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"index:idx_activity_user_org_created,priority:3,sort:desc;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}
