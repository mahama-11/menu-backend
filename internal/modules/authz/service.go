package authz

import (
	"fmt"
	"sort"

	"menu-service/internal/platform"
	"menu-service/internal/repository"
)

type Service struct {
	repo     *repository.AuthzRepository
	platform *platform.Client
}

type AccessContext struct {
	UserID              string   `json:"user_id"`
	OrgID               string   `json:"org_id"`
	PlatformOrgRole     string   `json:"platform_org_role"`
	PlatformPermissions []string `json:"platform_permissions"`
	MenuRoles           []string `json:"menu_roles"`
	MenuPermissions     []string `json:"menu_permissions"`
	Entitlements        []string `json:"entitlements,omitempty"`
}

func NewService(repo *repository.AuthzRepository, platformClient *platform.Client) *Service {
	return &Service{repo: repo, platform: platformClient}
}

func (s *Service) Bootstrap() error {
	return s.repo.SeedDefaults()
}

func (s *Service) Resolve(userID, orgID string) (*AccessContext, error) {
	platformCtx, err := s.platform.GetAccessContext(userID, orgID)
	if err != nil {
		return nil, err
	}
	roles := defaultMenuRoles(platformCtx.OrgRole)
	localRoles, err := s.repo.ListSubjectRoles(userID, orgID)
	if err != nil {
		return nil, err
	}
	for _, role := range localRoles {
		roles = append(roles, role.RoleID)
	}
	roles = uniqueStrings(roles)
	permissions, err := s.repo.ListRolePermissionIDs(roles)
	if err != nil {
		return nil, err
	}
	return &AccessContext{
		UserID:              userID,
		OrgID:               orgID,
		PlatformOrgRole:     platformCtx.OrgRole,
		PlatformPermissions: platformCtx.Permissions,
		MenuRoles:           roles,
		MenuPermissions:     permissions,
	}, nil
}

func (s *Service) EnsurePermission(userID, orgID, permission string) (*AccessContext, error) {
	ctx, err := s.Resolve(userID, orgID)
	if err != nil {
		return nil, err
	}
	for _, item := range ctx.MenuPermissions {
		if item == permission {
			return ctx, nil
		}
	}
	return nil, fmt.Errorf("missing menu permission: %s", permission)
}

func defaultMenuRoles(platformOrgRole string) []string {
	switch platformOrgRole {
	case "owner", "admin":
		return []string{"menu.workspace_admin"}
	case "viewer":
		return []string{"menu.viewer"}
	default:
		return []string{"menu.editor"}
	}
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
