package storage

import (
	"testing"
	"gorm.io/gorm/schema"
)

func TestMenuNamingStrategy_UsesMenuPrefix(t *testing.T) {
	ns := menuNamingStrategy{NamingStrategy: schema.NamingStrategy{TablePrefix: "menu_"}}

	cases := map[string]string{
		ns.TableName("AuditLog"):           "menu_audit_logs",
		ns.TableName("MenuRole"):           "menu_roles",
		ns.TableName("MenuPermission"):     "menu_permissions",
		ns.TableName("MenuRolePermission"): "menu_role_permissions",
		ns.TableName("MenuSubjectRole"):    "menu_subject_roles",
		ns.TableName("UserPreference"):     "menu_user_preferences",
		ns.TableName("Activity"):           "menu_activities",
	}

	for got, want := range cases {
		if got != want {
			t.Fatalf("unexpected table name, want=%s got=%s", want, got)
		}
		if len(got) < len("menu_") || got[:5] != "menu_" {
			t.Fatalf("table name must keep menu_ prefix, got=%s", got)
		}
	}
}

func TestMenuNamingStrategy_ModelNameMappingStaysStable(t *testing.T) {
	ns := menuNamingStrategy{NamingStrategy: schema.NamingStrategy{TablePrefix: "menu_"}}
	cases := map[string]string{
		"AuditLog":           "menu_audit_logs",
		"MenuRole":           "menu_roles",
		"MenuPermission":     "menu_permissions",
		"MenuRolePermission": "menu_role_permissions",
		"MenuSubjectRole":    "menu_subject_roles",
		"UserPreference":     "menu_user_preferences",
		"Activity":           "menu_activities",
	}

	for modelName, want := range cases {
		got := ns.TableName(modelName)
		if got != want {
			t.Fatalf("table mapping changed, model=%s want=%s got=%s", modelName, want, got)
		}
	}
}
