package models

import "time"

type MenuRole struct {
	ID          string    `gorm:"primaryKey;type:varchar(64)" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `gorm:"default:false" json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MenuPermission struct {
	ID          string    `gorm:"primaryKey;type:varchar(64)" json:"id"`
	Category    string    `json:"category"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type MenuRolePermission struct {
	RoleID       string `gorm:"primaryKey;type:varchar(64)" json:"role_id"`
	PermissionID string `gorm:"primaryKey;type:varchar(64)" json:"permission_id"`
}

type MenuSubjectRole struct {
	ID        string    `gorm:"primaryKey;type:varchar(64)" json:"id"`
	UserID    string    `gorm:"index;not null" json:"user_id"`
	OrgID     string    `gorm:"index;not null" json:"org_id"`
	RoleID    string    `gorm:"index;not null" json:"role_id"`
	ScopeType string    `gorm:"index;default:'organization'" json:"scope_type"`
	ScopeID   string    `gorm:"index" json:"scope_id"`
	Source    string    `gorm:"default:'local'" json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
