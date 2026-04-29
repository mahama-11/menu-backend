package migration

import (
	"fmt"
	"sort"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/internal/storage"

	"gorm.io/gorm"
)

type Step struct {
	Version int64
	Name    string
	Up      func(*gorm.DB) error
}

type Status struct {
	Version   int64
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

type record struct {
	Version   int64     `gorm:"primaryKey;autoIncrement:false"`
	Name      string    `gorm:"not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func Steps(cfg config.DatabaseConfig) []Step {
	steps := []Step{
		{
			Version: 202604170001,
			Name:    "baseline_schema_bootstrap",
			Up: func(db *gorm.DB) error {
				return storage.RunSchemaBootstrap(db, cfg.TablePrefix)
			},
		},
		{
			Version: 202604170002,
			Name:    "commercial_orders_bootstrap",
			Up: func(db *gorm.DB) error {
				return db.AutoMigrate(&models.CommercialOrder{}, &models.CommercialPayment{}, &models.CommercialFulfillment{})
			},
		},
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].Version < steps[j].Version })
	return steps
}

func Up(db *gorm.DB, cfg config.DatabaseConfig) error {
	if err := ensureMetadataTable(db, cfg.TablePrefix); err != nil {
		return err
	}
	applied, err := appliedVersions(db, cfg.TablePrefix)
	if err != nil {
		return err
	}
	for _, step := range Steps(cfg) {
		if _, ok := applied[step.Version]; ok {
			continue
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := step.Up(tx); err != nil {
				return err
			}
			return tx.Table(metadataTable(cfg.TablePrefix)).Create(&record{
				Version:   step.Version,
				Name:      step.Name,
				AppliedAt: time.Now(),
			}).Error
		}); err != nil {
			return fmt.Errorf("apply migration %d_%s: %w", step.Version, step.Name, err)
		}
	}
	return nil
}

func CurrentVersion(db *gorm.DB, cfg config.DatabaseConfig) (int64, error) {
	if err := ensureMetadataTable(db, cfg.TablePrefix); err != nil {
		return 0, err
	}
	var item record
	err := db.Table(metadataTable(cfg.TablePrefix)).Order("version desc").First(&item).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return item.Version, nil
}

func ListStatus(db *gorm.DB, cfg config.DatabaseConfig) ([]Status, error) {
	if err := ensureMetadataTable(db, cfg.TablePrefix); err != nil {
		return nil, err
	}
	var records []record
	if err := db.Table(metadataTable(cfg.TablePrefix)).Order("version asc").Find(&records).Error; err != nil {
		return nil, err
	}
	byVersion := make(map[int64]record, len(records))
	for _, item := range records {
		byVersion[item.Version] = item
	}
	out := make([]Status, 0, len(Steps(cfg)))
	for _, step := range Steps(cfg) {
		status := Status{Version: step.Version, Name: step.Name}
		if item, ok := byVersion[step.Version]; ok {
			status.Applied = true
			appliedAt := item.AppliedAt
			status.AppliedAt = &appliedAt
		}
		out = append(out, status)
	}
	return out, nil
}

func ensureMetadataTable(db *gorm.DB, tablePrefix string) error {
	return db.Table(metadataTable(tablePrefix)).AutoMigrate(&record{})
}

func appliedVersions(db *gorm.DB, tablePrefix string) (map[int64]struct{}, error) {
	var records []record
	if err := db.Table(metadataTable(tablePrefix)).Find(&records).Error; err != nil {
		return nil, err
	}
	out := make(map[int64]struct{}, len(records))
	for _, item := range records {
		out[item.Version] = struct{}{}
	}
	return out, nil
}

func metadataTable(tablePrefix string) string {
	return tablePrefix + "schema_migrations"
}
