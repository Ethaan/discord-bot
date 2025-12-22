package database

import (
	"time"

	"gorm.io/datatypes"
)

type List struct {
	ID             uint   `gorm:"primaryKey"`
	ChannelID      string `gorm:"uniqueIndex;not null"`
	Name           string `gorm:"not null"`
	Description    string `gorm:""`
	Type           string `gorm:"not null"`
	GuildID        string `gorm:"not null"`
	NotifyEveryone bool   `gorm:"default:false"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (List) TableName() string {
	return "lists"
}

type ListItem struct {
	ID        uint            `gorm:"primaryKey"`
	ListID    uint            `gorm:"not null;index"`
	ChannelID string          `gorm:"not null;index"`
	Name      string          `gorm:"not null"`
	Metadata  datatypes.JSON  `gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time
	UpdatedAt time.Time
	List      List            `gorm:"foreignKey:ListID;constraint:OnDelete:CASCADE"`
}

func (ListItem) TableName() string {
	return "list_items"
}

type GuildConfig struct {
	ID              uint   `gorm:"primaryKey"`
	GuildID         string `gorm:"uniqueIndex;not null"`
	ListsCategoryID string `gorm:""`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (GuildConfig) TableName() string {
	return "guild_configs"
}

type Player struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"uniqueIndex;not null"`
	Level     int    `gorm:""`
	Vocation  string `gorm:""`
	Country   string `gorm:""`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Player) TableName() string {
	return "players"
}

type OnlineSession struct {
	ID        uint       `gorm:"primaryKey"`
	PlayerID  uint       `gorm:"index:idx_player_time,idx_time_range;not null"`
	LoginAt   time.Time  `gorm:"index:idx_player_time,idx_time_range;not null"`
	LogoutAt  *time.Time `gorm:"index:idx_time_range"`
	CreatedAt time.Time
}

func (OnlineSession) TableName() string {
	return "online_sessions"
}
