package database

import (
	"fmt"
	"log"
	"time"

	"github.com/ethaan/discord-api/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(dsn string) error {
	var err error

	gLogger := gormLogger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		gormLogger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  gormLogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gLogger,
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	logger.Success("Database connected successfully")

	return nil
}

func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func AutoMigrate() error {
	logger.Info("Running database migrations...")

	err := DB.AutoMigrate(
		&List{},
		&ListItem{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Success("Database migrations completed")
	return nil
}
