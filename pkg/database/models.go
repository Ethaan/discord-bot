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
