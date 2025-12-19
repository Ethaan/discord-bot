package workers

import (
	"context"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/logger"
)

type Worker interface {
	Run(ctx context.Context)
	Name() string
}

type Manager struct {
	workers []Worker
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func NewManager(session *discordgo.Session, tibiaAPIURL string) *Manager {
	return &Manager{
		workers: []Worker{
			NewPremiumWorker(session, tibiaAPIURL),
			NewResidenceWorker(session, tibiaAPIURL),
		},
	}
}

func (m *Manager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	for _, worker := range m.workers {
		m.wg.Add(1)
		w := worker
		go func() {
			defer m.wg.Done()
			logger.Info("Starting worker: %s", w.Name())
			w.Run(ctx)
			logger.Info("Worker stopped: %s", w.Name())
		}()
	}

	logger.Success("Started %d workers", len(m.workers))
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		logger.Info("Stopping workers...")
		m.cancel()
		m.wg.Wait()
		logger.Success("All workers stopped")
	}
}
