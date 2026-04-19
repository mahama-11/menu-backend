package models

import "time"

// SharePost is the product-owned publishing object for a studio asset.
// It stays separate from StudioAsset so future public distribution,
// attribution, and engagement semantics do not leak into core asset truth.
type SharePost struct {
	ID             string     `gorm:"type:varchar(64);primaryKey" json:"id"`
	OrganizationID string     `gorm:"type:varchar(64);index:idx_share_post_org_user_created,priority:1;not null" json:"organization_id"`
	UserID         string     `gorm:"type:varchar(64);index:idx_share_post_org_user_created,priority:2;not null" json:"user_id"`
	AssetID        string     `gorm:"type:varchar(64);index;not null" json:"asset_id"`
	JobID          string     `gorm:"type:varchar(64);index" json:"job_id"`
	VariantID      string     `gorm:"type:varchar(64);index" json:"variant_id"`
	Title          string     `gorm:"type:varchar(255)" json:"title"`
	Caption        string     `gorm:"type:text" json:"caption"`
	Visibility     string     `gorm:"type:varchar(32);index;not null" json:"visibility"`
	Status         string     `gorm:"type:varchar(32);index;not null" json:"status"`
	ShareToken     string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"share_token"`
	ShareURL       string     `gorm:"type:text" json:"share_url"`
	ViewCount      int64      `gorm:"not null;default:0" json:"view_count"`
	LikeCount      int64      `gorm:"not null;default:0" json:"like_count"`
	FavoriteCount  int64      `gorm:"not null;default:0" json:"favorite_count"`
	Metadata       string     `gorm:"type:text" json:"metadata"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`
	CreatedAt      time.Time  `gorm:"index:idx_share_post_org_user_created,priority:3,sort:desc;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}
