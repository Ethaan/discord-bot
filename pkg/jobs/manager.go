package jobs

import (
	"context"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/logger"
)

type Job interface {
	Run(ctx context.Context)
	Name() string
}

type Manager struct {
	jobs   []Job
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewManager(session *discordgo.Session, tibiaAPIURL string) *Manager {
	return &Manager{
		jobs: []Job{
			NewPowergamesHistoricalWorker(session, tibiaAPIURL),
		},
	}
}

func (m *Manager) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	for _, job := range m.jobs {
		m.wg.Add(1)
		j := job
		go func() {
			defer m.wg.Done()
			logger.Info("Starting scheduled job: %s", j.Name())
			j.Run(ctx)
			logger.Info("Scheduled job stopped: %s", j.Name())
		}()
	}

	logger.Success("Started %d scheduled jobs", len(m.jobs))
}

func (m *Manager) Stop() {
	if m.cancel != nil {
		logger.Info("Stopping scheduled jobs...")
		m.cancel()
		m.wg.Wait()
		logger.Success("All scheduled jobs stopped")
	}
}
