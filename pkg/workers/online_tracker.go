package workers

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/repositories"
	"github.com/ethaan/discord-api/pkg/tibia"
)

type OnlineTrackerWorker struct {
	session         *discordgo.Session
	playerRepo      *repositories.PlayerRepository
	sessionRepo     *repositories.OnlineSessionRepository
	tibiaClient     *tibia.Client
	pollInterval    time.Duration
	lastOnlinePlayers map[string]uint
}

func NewOnlineTrackerWorker(session *discordgo.Session, tibiaAPIURL string) *OnlineTrackerWorker {
	return &OnlineTrackerWorker{
		session:          session,
		playerRepo:       repositories.NewPlayerRepository(),
		sessionRepo:      repositories.NewOnlineSessionRepository(),
		tibiaClient:      tibia.NewClient(tibiaAPIURL),
		pollInterval:     10 * time.Second,
		lastOnlinePlayers: make(map[string]uint),
	}
}

func (w *OnlineTrackerWorker) Name() string {
	return "online-tracker"
}

func (w *OnlineTrackerWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.trackOnlinePlayers()

	for {
		select {
		case <-ctx.Done():
			w.closeAllActiveSessions()
			return
		case <-ticker.C:
			w.trackOnlinePlayers()
		}
	}
}

func (w *OnlineTrackerWorker) trackOnlinePlayers() {
	response, err := w.tibiaClient.GetWhosOnline()
	if err != nil {
		logger.Worker("online-tracker", "Error fetching whos online: %v", err)
		return
	}

	now := time.Now()
	currentOnline := make(map[string]uint)

	for _, player := range response.Players {
		dbPlayer, err := w.playerRepo.FindOrCreate(player.Name, player.Level, player.Vocation, player.Country)
		if err != nil {
			logger.Worker("online-tracker", "Error creating/updating player %s: %v", player.Name, err)
			continue
		}

		currentOnline[player.Name] = dbPlayer.ID

		if _, wasOnline := w.lastOnlinePlayers[player.Name]; !wasOnline {
			activeSession, err := w.sessionRepo.FindActiveSession(dbPlayer.ID)
			if err != nil || activeSession == nil {
				if err := w.sessionRepo.CreateSession(dbPlayer.ID, now); err != nil {
					logger.Worker("online-tracker", "Error creating session for %s: %v", player.Name, err)
				}
			}
		}
	}

	for name, playerID := range w.lastOnlinePlayers {
		if _, isOnline := currentOnline[name]; !isOnline {
			activeSession, err := w.sessionRepo.FindActiveSession(playerID)
			if err == nil && activeSession != nil {
				if err := w.sessionRepo.CloseSession(activeSession.ID, now); err != nil {
					logger.Worker("online-tracker", "Error closing session for %s: %v", name, err)
				}
			}
		}
	}

	w.lastOnlinePlayers = currentOnline
	logger.Worker("online-tracker", "Tracked %d online players", len(currentOnline))
}

func (w *OnlineTrackerWorker) closeAllActiveSessions() {
	now := time.Now()
	for name, playerID := range w.lastOnlinePlayers {
		activeSession, err := w.sessionRepo.FindActiveSession(playerID)
		if err == nil && activeSession != nil {
			if err := w.sessionRepo.CloseSession(activeSession.ID, now); err != nil {
				logger.Worker("online-tracker", "Error closing session for %s on shutdown: %v", name, err)
			}
		}
	}
}
