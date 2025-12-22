package repositories

import (
	"time"

	"github.com/ethaan/discord-api/pkg/database"
)

type ScanResult struct {
	CharacterName   string
	AdjacentCount   int
	ConfidenceLevel string
}

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

func (r *OnlineSessionRepository) ScanCharacter(characterName string, adjacentWindowSeconds int, maxResults int) ([]ScanResult, error) {
	query := `
		WITH target_sessions AS (
			SELECT s.login_at, s.logout_at
			FROM online_sessions s
			JOIN players p ON s.player_id = p.id
			WHERE p.name = ?
		),
		candidate_characters AS (
			SELECT DISTINCT p2.id, p2.name
			FROM players p2
			WHERE p2.name != ?
			  -- Never online at same time (all pairs must not overlap)
			  AND NOT EXISTS (
				SELECT 1 FROM online_sessions s2
				WHERE s2.player_id = p2.id
				  AND EXISTS (
					SELECT 1 FROM target_sessions ts
					WHERE s2.login_at < COALESCE(ts.logout_at, NOW())
					  AND COALESCE(s2.logout_at, NOW()) > ts.login_at
				  )
			  )
		)
		SELECT
			c.name as character_name,
			COUNT(*) as adjacent_count
		FROM candidate_characters c
		JOIN online_sessions s2 ON s2.player_id = c.id
		CROSS JOIN target_sessions ts
		WHERE
			-- Adjacent transitions (within configured seconds)
			(ABS(EXTRACT(EPOCH FROM (COALESCE(s2.logout_at, NOW()) - ts.login_at))) < ?
			 OR ABS(EXTRACT(EPOCH FROM (COALESCE(ts.logout_at, NOW()) - s2.login_at))) < ?)
		GROUP BY c.id, c.name
		HAVING COUNT(*) >= 1
		ORDER BY adjacent_count DESC
		LIMIT ?
	`

	var results []struct {
		CharacterName string
		AdjacentCount int
	}

	err := database.DB.Raw(query, characterName, characterName, adjacentWindowSeconds, adjacentWindowSeconds, maxResults).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	scanResults := make([]ScanResult, len(results))
	for i, result := range results {
		scanResults[i] = ScanResult{
			CharacterName:   result.CharacterName,
			AdjacentCount:   result.AdjacentCount,
			ConfidenceLevel: "",
		}
	}

	return scanResults, nil
}
