package database

import (
	_ "embed"
	"fmt"
	"log"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"turnin/internal/models"
)

//go:embed seed.sql
var seedSQL string

// Open uses Supabase/PostgreSQL when DATABASE_URL is provided in production,
// and falls back to zero-config SQLite in local development and tests.
func Open(sqlitePath string) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Warn),
		TranslateError: true,
	}

	if pgURL := os.Getenv("DATABASE_URL"); pgURL != "" {
		log.Println("[database] driver=postgres (DATABASE_URL detected)")
		return gorm.Open(postgres.New(postgres.Config{
			DSN:                  pgURL,
			PreferSimpleProtocol: true, // Required for Supabase Transaction Pooler (port 6543)
		}), gormCfg)
	}

	log.Printf("[database] driver=sqlite path=%s", sqlitePath)
	dsn := sqlitePath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_time_format=sqlite"
	return gorm.Open(sqlite.Open(dsn), gormCfg)
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{}, &models.Course{}, &models.Assignment{}, &models.Submission{})
}

func Seed(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(seedSQL); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("apply seed: %w", err)
	}
	return tx.Commit()
}
