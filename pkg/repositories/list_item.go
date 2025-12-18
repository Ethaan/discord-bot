package repositories

import (
	"github.com/ethaan/discord-api/pkg/database"
	"gorm.io/gorm"
)

type ListItemRepository struct {
	db *gorm.DB
}

func NewListItemRepository() *ListItemRepository {
	return &ListItemRepository{
		db: database.DB,
	}
}

func (r *ListItemRepository) Create(item *database.ListItem) error {
	return r.db.Create(item).Error
}

func (r *ListItemRepository) FindByListID(listID uint) ([]database.ListItem, error) {
	var items []database.ListItem
	err := r.db.Where("list_id = ?", listID).Find(&items).Error
	return items, err
}

func (r *ListItemRepository) FindByName(listID uint, name string) (*database.ListItem, error) {
	var item database.ListItem
	err := r.db.Where("list_id = ? AND name = ?", listID, name).First(&item).Error
	return &item, err
}

func (r *ListItemRepository) Update(item *database.ListItem) error {
	return r.db.Save(item).Error
}

func (r *ListItemRepository) Delete(item *database.ListItem) error {
	return r.db.Delete(item).Error
}
