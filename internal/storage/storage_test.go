package storage

import (
	"testing"

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
