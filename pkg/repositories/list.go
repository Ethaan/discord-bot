package repositories

import (
	"github.com/ethaan/discord-api/pkg/database"
	"gorm.io/gorm"
)

type ListRepository struct {
	db *gorm.DB
}

func NewListRepository() *ListRepository {
	return &ListRepository{
		db: database.DB,
	}
}

func (r *ListRepository) Create(list *database.List) error {
	return r.db.Create(list).Error
}

func (r *ListRepository) FindByChannelID(channelID string) (*database.List, error) {
	var list database.List
	err := r.db.Where("channel_id = ?", channelID).First(&list).Error
	return &list, err
}

func (r *ListRepository) FindByID(id uint) (*database.List, error) {
	var list database.List
	err := r.db.Where("id = ?", id).First(&list).Error
	return &list, err
}

func (r *ListRepository) FindByType(listType string) ([]database.List, error) {
	var lists []database.List
	err := r.db.Where("type = ?", listType).Find(&lists).Error
	return lists, err
}

func (r *ListRepository) Update(list *database.List) error {
	return r.db.Save(list).Error
}

func (r *ListRepository) Delete(list *database.List) error {
	return r.db.Delete(list).Error
}
