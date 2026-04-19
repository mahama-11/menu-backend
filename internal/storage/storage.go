package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/pkg/logger"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func InitDB(cfg config.DatabaseConfig, ginMode string) (*gorm.DB, error) {
	db, err := ConnectDB(cfg)
	if err != nil {
		return nil, err
	}
	if cfg.AutoMigrateEnabled {
		if err := validateAutoMigratePolicy(cfg, ginMode); err != nil {
			return nil, err
		}
		if err := RunSchemaBootstrap(db, cfg.TablePrefix); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func ConnectDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
	newLogger := gormlogger.New(
		log.New(os.Stdout, "", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	var (
		db  *gorm.DB
		err error
	)

	switch cfg.Driver {
	case "sqlite":
		if mkdirErr := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); mkdirErr != nil {
			return nil, fmt.Errorf("create sqlite dir: %w", mkdirErr)
		}
		db, err = gorm.Open(sqlite.Open(cfg.SQLitePath), &gorm.Config{
			Logger:         newLogger,
			NamingStrategy: menuNamingStrategy{NamingStrategy: schema.NamingStrategy{TablePrefix: cfg.TablePrefix}},
		})
	default:
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host,
			cfg.Port,
			cfg.User,
			cfg.Password,
			cfg.DBName,
			cfg.SSLMode,
		)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger:         newLogger,
			NamingStrategy: menuNamingStrategy{NamingStrategy: schema.NamingStrategy{TablePrefix: cfg.TablePrefix}},
		})
	}
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func RunSchemaBootstrap(db *gorm.DB, tablePrefix string) error {
	if err := preAutoMigrate(db, tablePrefix); err != nil {
		return fmt.Errorf("pre auto migrate: %w", err)
	}
	if err := autoMigrate(db); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}

func validateAutoMigratePolicy(cfg config.DatabaseConfig, ginMode string) error {
	if !cfg.AutoMigrateEnabled {
		return nil
	}
	if strings.EqualFold(cfg.Driver, "sqlite") {
		return nil
	}
	if strings.EqualFold(ginMode, "debug") {
		return nil
	}
	if cfg.AllowStartupMigrate {
		return nil
	}
	return fmt.Errorf("startup auto migrate blocked for driver=%s gin_mode=%s: use explicit versioned migrations or set database.allow_startup_migrate_in_non_dev=true for break-glass only", cfg.Driver, ginMode)
}

func InitRedis(cfg config.RedisConfig) (*redis.Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return client, nil
}

func autoMigrate(db *gorm.DB) error {
	modelsToMigrate := []any{
		&models.AuditLog{},
		&models.MenuRole{},
		&models.MenuPermission{},
		&models.MenuRolePermission{},
		&models.MenuSubjectRole{},
		&models.UserPreference{},
		&models.Activity{},
		&models.StudioAsset{},
		&models.StylePreset{},
		&models.GenerationJob{},
		&models.GenerationVariant{},
		&models.StudioChargeIntent{},
		&models.SharePost{},
	}

	for _, model := range modelsToMigrate {
		if err := db.AutoMigrate(model); err != nil {
			logger.With("model", fmt.Sprintf("%T", model)).Error("database.auto_migrate.failed", "error", err)
			return err
		}
		logger.With("model", fmt.Sprintf("%T", model)).Info("database.auto_migrate.completed")
	}
	return nil
}

func preAutoMigrate(db *gorm.DB, tablePrefix string) error {
	migrator := db.Migrator()
	if tablePrefix != "" {
		renames := [][2]string{
			{"user_preferences", tablePrefix + "user_preferences"},
			{"activities", tablePrefix + "activities"},
			{"audit_logs", tablePrefix + "audit_logs"},
			{"studio_assets", tablePrefix + "studio_assets"},
			{"style_presets", tablePrefix + "style_presets"},
			{"generation_jobs", tablePrefix + "generation_jobs"},
			{"generation_variants", tablePrefix + "generation_variants"},
			{"studio_charge_intents", tablePrefix + "studio_charge_intents"},
			{"share_posts", tablePrefix + "share_posts"},
		}
		for _, item := range renames {
			oldName := item[0]
			newName := item[1]
			if !migrator.HasTable(oldName) || migrator.HasTable(newName) {
				continue
			}
			if err := migrator.RenameTable(oldName, newName); err != nil {
				return fmt.Errorf("rename table %s -> %s: %w", oldName, newName, err)
			}
			logger.With("from", oldName, "to", newName).Info("database.pre_auto_migrate.renamed_table")
		}
	}

	auditTable := tablePrefix + "audit_logs"
	if auditTable == "" {
		auditTable = "audit_logs"
	}
	if migrator.HasTable(auditTable) && !migrator.HasColumn(auditTable, "target_type") {
		if err := db.Exec(fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN target_type text`, auditTable)).Error; err != nil {
			return err
		}
		if err := db.Exec(fmt.Sprintf(`UPDATE "%s" SET target_type = 'legacy_audit' WHERE target_type IS NULL OR target_type = ''`, auditTable)).Error; err != nil {
			return err
		}
		if err := db.Exec(fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN target_type SET NOT NULL`, auditTable)).Error; err != nil {
			return err
		}
		logger.With("table", auditTable, "column", "target_type").Info("database.pre_auto_migrate.backfilled_non_null_column")
	}
	return nil
}

type menuNamingStrategy struct {
	schema.NamingStrategy
}

func (s menuNamingStrategy) TableName(str string) string {
	base := s.NamingStrategy.TableName(str)
	if s.TablePrefix == "" {
		return base
	}
	base = strings.TrimPrefix(base, s.TablePrefix)
	base = strings.TrimPrefix(base, "menu_")
	return s.TablePrefix + base
}
