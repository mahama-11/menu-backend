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
		{ID: "menu.asset.read", Category: "asset", Name: "menu.asset.read", Description: "Read assets"},
		{ID: "menu.job.create", Category: "job", Name: "menu.job.create", Description: "Create AI jobs"},
		{ID: "menu.job.read", Category: "job", Name: "menu.job.read", Description: "Read AI jobs"},
		{ID: "menu.job.manage", Category: "job", Name: "menu.job.manage", Description: "Manage AI jobs and outputs"},
		{ID: "menu.template.manage", Category: "template", Name: "menu.template.manage", Description: "Manage templates"},
		{ID: "menu.template.read", Category: "template", Name: "menu.template.read", Description: "Read templates"},
		{ID: "menu.share.read", Category: "share", Name: "menu.share.read", Description: "Read share posts"},
		{ID: "menu.share.manage", Category: "share", Name: "menu.share.manage", Description: "Manage share posts"},
		{ID: "menu.audit.read", Category: "audit", Name: "menu.audit.read", Description: "Read audit history"},
		{ID: "menu.analytics.read", Category: "analytics", Name: "menu.analytics.read", Description: "Read analytics"},
		{ID: "menu.referral.read", Category: "growth", Name: "menu.referral.read", Description: "Read referral and commission data"},
		{ID: "menu.referral.manage", Category: "growth", Name: "menu.referral.manage", Description: "Manage referral codes"},
		{ID: "menu.channel.read", Category: "growth", Name: "menu.channel.read", Description: "Read channel binding, commission, and settlement data"},
		{ID: "menu.channel.manage", Category: "growth", Name: "menu.channel.manage", Description: "Manage channel integration operations"},
		{ID: "menu.commercial.read", Category: "commercial", Name: "menu.commercial.read", Description: "Read menu commercial offerings and balances"},
		{ID: "menu.commercial.manage", Category: "commercial", Name: "menu.commercial.manage", Description: "Assign menu packages and simulate commercial fulfillment"},
	}
	mapping := map[string][]string{
		"menu.workspace_admin": {"menu.access", "menu.asset.upload", "menu.asset.read", "menu.job.create", "menu.job.read", "menu.job.manage", "menu.template.manage", "menu.template.read", "menu.share.read", "menu.share.manage", "menu.audit.read", "menu.analytics.read", "menu.referral.read", "menu.referral.manage", "menu.channel.read", "menu.channel.manage", "menu.commercial.read", "menu.commercial.manage"},
		"menu.editor":          {"menu.access", "menu.asset.upload", "menu.asset.read", "menu.job.create", "menu.job.read", "menu.template.read", "menu.share.read", "menu.share.manage", "menu.audit.read", "menu.referral.read", "menu.referral.manage", "menu.channel.read", "menu.commercial.read"},
		"menu.viewer":          {"menu.access", "menu.asset.read", "menu.job.read", "menu.template.read", "menu.share.read", "menu.audit.read", "menu.analytics.read", "menu.referral.read", "menu.channel.read", "menu.commercial.read"},
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
