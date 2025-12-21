package repositories

import (
	"github.com/ethaan/discord-api/pkg/database"
	"gorm.io/gorm"
)

type GuildConfigRepository struct {
	db *gorm.DB
}

func NewGuildConfigRepository() *GuildConfigRepository {
	return &GuildConfigRepository{
		db: database.DB,
	}
}

func (r *GuildConfigRepository) Create(config *database.GuildConfig) error {
	return r.db.Create(config).Error
}

func (r *GuildConfigRepository) FindByGuildID(guildID string) (*database.GuildConfig, error) {
	var config database.GuildConfig
	err := r.db.Where("guild_id = ?", guildID).First(&config).Error
	return &config, err
}

func (r *GuildConfigRepository) Update(config *database.GuildConfig) error {
	return r.db.Save(config).Error
}

func (r *GuildConfigRepository) Upsert(config *database.GuildConfig) error {
	return r.db.Where("guild_id = ?", config.GuildID).
		Assign(config).
		FirstOrCreate(config).Error
}
