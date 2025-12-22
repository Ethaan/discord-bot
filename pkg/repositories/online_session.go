package repositories

import (
	"time"

	"github.com/ethaan/discord-api/pkg/database"
)

type OnlineSessionRepository struct{}

func NewOnlineSessionRepository() *OnlineSessionRepository {
	return &OnlineSessionRepository{}
}

func (r *OnlineSessionRepository) FindActiveSession(playerID uint) (*database.OnlineSession, error) {
	var session database.OnlineSession
	err := database.DB.Where("player_id = ? AND logout_at IS NULL", playerID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *OnlineSessionRepository) CreateSession(playerID uint, loginAt time.Time) error {
	session := database.OnlineSession{
		PlayerID: playerID,
		LoginAt:  loginAt,
	}
	return database.DB.Create(&session).Error
}

func (r *OnlineSessionRepository) CloseSession(sessionID uint, logoutAt time.Time) error {
	return database.DB.Model(&database.OnlineSession{}).
		Where("id = ?", sessionID).
		Update("logout_at", logoutAt).Error
}

func (r *OnlineSessionRepository) FindByPlayerID(playerID uint) ([]database.OnlineSession, error) {
	var sessions []database.OnlineSession
	err := database.DB.Where("player_id = ?", playerID).
		Order("login_at DESC").
		Find(&sessions).Error
	return sessions, err
}
