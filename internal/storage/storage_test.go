package storage

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"menu-service/internal/config"
)

func TestValidateAutoMigratePolicy(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.DatabaseConfig
		ginMode string
		wantErr bool
	}{
		{
			name: "disabled is allowed",
			cfg: config.DatabaseConfig{
				Driver:             "postgres",
				AutoMigrateEnabled: false,
			},
			ginMode: "release",
			wantErr: false,
		},
		{
			name: "sqlite is allowed",
			cfg: config.DatabaseConfig{
				Driver:             "sqlite",
				AutoMigrateEnabled: true,
			},
			ginMode: "release",
			wantErr: false,
		},
		{
			name: "debug postgres is allowed",
			cfg: config.DatabaseConfig{
				Driver:             "postgres",
				AutoMigrateEnabled: true,
			},
			ginMode: "debug",
			wantErr: false,
		},
		{
			name: "release postgres blocked by default",
			cfg: config.DatabaseConfig{
				Driver:             "postgres",
				AutoMigrateEnabled: true,
			},
			ginMode: "release",
			wantErr: true,
		},
		{
			name: "release postgres allowed with override",
			cfg: config.DatabaseConfig{
				Driver:              "postgres",
				AutoMigrateEnabled:  true,
				AllowStartupMigrate: true,
			},
			ginMode: "release",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAutoMigratePolicy(tt.cfg, tt.ginMode)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuditLogIDColumnMigrationSQL(t *testing.T) {
	sql, ok := auditLogIDColumnMigrationSQL("postgres", "menu_audit_logs", "bigint", "int8")
	if !ok {
		t.Fatalf("expected postgres bigint audit log id to require migration")
	}
	want := `ALTER TABLE "menu_audit_logs" ALTER COLUMN "id" TYPE varchar(64) USING "id"::text`
	if sql != want {
		t.Fatalf("unexpected migration SQL:\n got: %s\nwant: %s", sql, want)
	}

	if _, ok := auditLogIDColumnMigrationSQL("postgres", "menu_audit_logs", "character varying", "varchar"); ok {
		t.Fatalf("varchar audit log id should not require migration")
	}
	if _, ok := auditLogIDColumnMigrationSQL("sqlite", "menu_audit_logs", "integer", "integer"); ok {
		t.Fatalf("sqlite should not run postgres audit id migration")
	}
}

func TestRunSchemaBootstrapCreatesCommercialTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		NamingStrategy: menuNamingStrategy{NamingStrategy: schema.NamingStrategy{TablePrefix: "menu_"}},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := RunSchemaBootstrap(db, "menu_"); err != nil {
		t.Fatalf("RunSchemaBootstrap() error = %v", err)
	}

	for _, table := range []string{
		"menu_commercial_orders",
		"menu_commercial_payments",
		"menu_commercial_fulfillments",
	} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %s to exist after bootstrap", table)
		}
	}
}
