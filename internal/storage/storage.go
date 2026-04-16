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

func InitDB(cfg config.DatabaseConfig) (*gorm.DB, error) {
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

	if cfg.AutoMigrateEnabled {
		if err := preAutoMigrate(db, cfg.TablePrefix); err != nil {
			return nil, fmt.Errorf("pre auto migrate: %w", err)
		}
		if err := autoMigrate(db); err != nil {
			return nil, fmt.Errorf("auto migrate: %w", err)
		}
	}

	return db, nil
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
	if tablePrefix == "" {
		return nil
	}
	migrator := db.Migrator()
	renames := [][2]string{
		{"user_preferences", tablePrefix + "user_preferences"},
		{"activities", tablePrefix + "activities"},
		{"audit_logs", tablePrefix + "audit_logs"},
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
