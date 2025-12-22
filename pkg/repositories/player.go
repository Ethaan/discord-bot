package repositories

import (
	"github.com/ethaan/discord-api/pkg/database"
)

type PlayerRepository struct{}

func NewPlayerRepository() *PlayerRepository {
	return &PlayerRepository{}
}

func (r *PlayerRepository) FindOrCreate(name string, level int, vocation, country string) (*database.Player, error) {
	var player database.Player

	result := database.DB.Where("name = ?", name).First(&player)
	if result.Error == nil {
		if player.Level != level || player.Vocation != vocation || player.Country != country {
			player.Level = level
			player.Vocation = vocation
			player.Country = country
			if err := database.DB.Save(&player).Error; err != nil {
				return nil, err
			}
		}
		return &player, nil
	}

	player = database.Player{
		Name:     name,
		Level:    level,
		Vocation: vocation,
		Country:  country,
	}

	if err := database.DB.Create(&player).Error; err != nil {
		return nil, err
	}

	return &player, nil
}

func (r *PlayerRepository) FindByName(name string) (*database.Player, error) {
	var player database.Player
	if err := database.DB.Where("name = ?", name).First(&player).Error; err != nil {
		return nil, err
	}
	return &player, nil
}
