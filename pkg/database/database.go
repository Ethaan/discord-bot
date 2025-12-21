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
		&GuildConfig{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Success("Database migrations completed")
	return nil
}

func InitializeGuildConfig(guildID, parentCategoryID string, session interface{}) error {
	if guildID == "" {
		return fmt.Errorf("guild ID cannot be empty")
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var config GuildConfig
	result := tx.Where("guild_id = ?", guildID).First(&config)

	isNewConfig := false
	if result.Error == gorm.ErrRecordNotFound {
		logger.Info("Guild config not found for guild %s, creating default config...", guildID)

		config = GuildConfig{
			GuildID:         guildID,
			ListsCategoryID: parentCategoryID,
		}

		if err := tx.Create(&config).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create guild config: %w", err)
		}

		logger.Success("Created default guild config for guild %s", guildID)
		isNewConfig = true
	} else if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to check guild config: %w", result.Error)
	} else {
		logger.Info("Guild config already exists for guild %s", guildID)
	}

	updateResult := tx.Model(&List{}).
		Where("guild_id = ? AND notify_everyone IS NULL", guildID).
		Update("notify_everyone", false)

	if updateResult.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update existing lists: %w", updateResult.Error)
	}

	if updateResult.RowsAffected > 0 {
		logger.Info("Updated %d existing lists with default NotifyEveryone value", updateResult.RowsAffected)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	if isNewConfig && parentCategoryID != "" && session != nil {
		logger.Info("Attempting to move existing list channels to category %s...", parentCategoryID)

		var lists []List
		if err := DB.Where("guild_id = ?", guildID).Find(&lists).Error; err != nil {
			logger.Warn("Failed to fetch lists for channel migration: %v", err)
			return nil
		}

		if len(lists) > 0 {
			logger.Info("Found %d existing list channels. To move them to the category:", len(lists))
			logger.Info("  Option 1: Manually drag and drop them in Discord")
			logger.Info("  Option 2: The bot will handle this on next restart with session available")
		}
	}

	return nil
}
