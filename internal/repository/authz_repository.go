package repository

import (
	"menu-service/internal/models"

	"gorm.io/gorm"
)

type AuthzRepository struct {
	db *gorm.DB
}

func NewAuthzRepository(db *gorm.DB) *AuthzRepository {
	return &AuthzRepository{db: db}
}

func (r *AuthzRepository) SeedDefaults() error {
	roles := []models.MenuRole{
		{ID: "menu.workspace_admin", Name: "menu.workspace_admin", Description: "Workspace admin", IsSystem: true},
		{ID: "menu.editor", Name: "menu.editor", Description: "Menu editor", IsSystem: true},
		{ID: "menu.viewer", Name: "menu.viewer", Description: "Read-only viewer", IsSystem: true},
	}
	permissions := []models.MenuPermission{
		{ID: "menu.access", Category: "access", Name: "menu.access", Description: "Access menu product"},
		{ID: "menu.asset.upload", Category: "asset", Name: "menu.asset.upload", Description: "Upload assets"},
		{ID: "menu.job.create", Category: "job", Name: "menu.job.create", Description: "Create AI jobs"},
		{ID: "menu.template.manage", Category: "template", Name: "menu.template.manage", Description: "Manage templates"},
		{ID: "menu.analytics.read", Category: "analytics", Name: "menu.analytics.read", Description: "Read analytics"},
		{ID: "menu.referral.read", Category: "growth", Name: "menu.referral.read", Description: "Read referral and commission data"},
		{ID: "menu.referral.manage", Category: "growth", Name: "menu.referral.manage", Description: "Manage referral codes"},
	}
	mapping := map[string][]string{
		"menu.workspace_admin": {"menu.access", "menu.asset.upload", "menu.job.create", "menu.template.manage", "menu.analytics.read", "menu.referral.read", "menu.referral.manage"},
		"menu.editor":          {"menu.access", "menu.asset.upload", "menu.job.create", "menu.referral.read", "menu.referral.manage"},
		"menu.viewer":          {"menu.access", "menu.analytics.read", "menu.referral.read"},
	}
	for _, role := range roles {
		if err := r.db.FirstOrCreate(&role, models.MenuRole{ID: role.ID}).Error; err != nil {
			return err
		}
	}
	for _, permission := range permissions {
		if err := r.db.FirstOrCreate(&permission, models.MenuPermission{ID: permission.ID}).Error; err != nil {
			return err
		}
	}
	for roleID, permissions := range mapping {
		for _, permissionID := range permissions {
			item := models.MenuRolePermission{RoleID: roleID, PermissionID: permissionID}
			if err := r.db.FirstOrCreate(&item, item).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *AuthzRepository) ListSubjectRoles(userID, orgID string) ([]models.MenuSubjectRole, error) {
	var items []models.MenuSubjectRole
	if err := r.db.Where("user_id = ? AND org_id = ?", userID, orgID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *AuthzRepository) ListRolePermissionIDs(roleIDs []string) ([]string, error) {
	var items []models.MenuRolePermission
	if len(roleIDs) == 0 {
		return []string{}, nil
	}
	if err := r.db.Where("role_id IN ?", roleIDs).Find(&items).Error; err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, exists := seen[item.PermissionID]; exists {
			continue
		}
		seen[item.PermissionID] = struct{}{}
		out = append(out, item.PermissionID)
	}
	return out, nil
}
