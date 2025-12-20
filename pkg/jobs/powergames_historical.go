package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/ascii"
	"github.com/ethaan/discord-api/pkg/database"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/repositories"
	"github.com/ethaan/discord-api/pkg/tibia"
	"github.com/go-co-op/gocron/v2"
)

type PowergamesHistoricalWorker struct {
	session     *discordgo.Session
	listRepo    *repositories.ListRepository
	itemRepo    *repositories.ListItemRepository
	tibiaClient *tibia.Client
	scheduler   gocron.Scheduler
}

func NewPowergamesHistoricalWorker(session *discordgo.Session, tibiaAPIURL string) *PowergamesHistoricalWorker {
	return &PowergamesHistoricalWorker{
		session:     session,
		listRepo:    repositories.NewListRepository(),
		itemRepo:    repositories.NewListItemRepository(),
		tibiaClient: tibia.NewClient(tibiaAPIURL),
	}
}

func (w *PowergamesHistoricalWorker) Name() string {
	return "powergames-historical"
}

func (w *PowergamesHistoricalWorker) Run(ctx context.Context) {
	brazilLocation := time.FixedZone("BRT", -3*60*60)

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(brazilLocation))
	if err != nil {
		logger.Error("Failed to create scheduler: %v", err)
		return
	}
	w.scheduler = scheduler

	_, err = scheduler.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 5, 0))),
		gocron.NewTask(func() {
			logger.Worker("powergames-historical", "Running scheduled job at %s BRT", time.Now().In(brazilLocation).Format("15:04:05"))
			w.postHistoricalStats()
		}),
	)

	if err != nil {
		logger.Error("Failed to schedule job: %v", err)
		return
	}

	scheduler.Start()
	logger.Worker("powergames-historical", "Scheduler started - will run daily at 00:05 AM BRT")

	<-ctx.Done()

	if err := scheduler.Shutdown(); err != nil {
		logger.Error("Error shutting down scheduler: %v", err)
	}
}

func (w *PowergamesHistoricalWorker) postHistoricalStats() {
	lists, err := w.listRepo.FindByType("powergamer-stats-historical")
	if err != nil {
		logger.Worker("powergames-historical", "Error fetching lists: %v", err)
		return
	}

	logger.Worker("powergames-historical", "Posting historical stats to %d channels", len(lists))

	for _, list := range lists {
		if err := w.postChannelStats(&list); err != nil {
			logger.Worker("powergames-historical", "Error posting to channel %s: %v", list.ChannelID, err)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (w *PowergamesHistoricalWorker) postChannelStats(list *database.List) error {
	items, err := w.itemRepo.FindByListID(list.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch list items: %w", err)
	}

	if len(items) == 0 {
		logger.Worker("powergames-historical", "Skipping channel %s (empty list)", list.ChannelID)
		return nil
	}

	powergamers, err := w.tibiaClient.GetPowergamers("lastday", "", false)
	if err != nil {
		return fmt.Errorf("failed to fetch powergamers: %w", err)
	}

	listNames := make(map[string]bool)
	for _, item := range items {
		normalizedName := strings.TrimSpace(strings.ToLower(item.Name))
		listNames[normalizedName] = true
	}

	filtered := make([]tibia.Powergamer, 0)
	for _, pg := range powergamers {
		normalizedPgName := strings.TrimSpace(strings.ToLower(pg.Name))
		if listNames[normalizedPgName] && pg.Today > 0 {
			filtered = append(filtered, pg)
		}
	}
	powergamers = filtered

	embed := w.buildHistoricalStatsEmbed(powergamers, len(items))

	_, err = w.session.ChannelMessageSendEmbed(list.ChannelID, embed)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	logger.Worker("powergames-historical", "Posted historical stats to channel %s (%d powergamers)", list.ChannelID, len(powergamers))
	return nil
}

func (w *PowergamesHistoricalWorker) buildHistoricalStatsEmbed(powergamers []tibia.Powergamer, listItemCount int) *discordgo.MessageEmbed {
	var description strings.Builder

	if len(powergamers) == 0 {
		if listItemCount > 0 {
			description.WriteString(fmt.Sprintf(
				"📊 No powergamers found.\n\nNone of the %d characters in your list were in yesterday's powergamer rankings.",
				listItemCount,
			))
		} else {
			description.WriteString("📊 No powergamers found for yesterday.")
		}
	} else {
		description.WriteString("```text\n")
		description.WriteString(ascii.BuildTextTableForPowergamers(powergamers))
		description.WriteString("```")
	}

	footer := fmt.Sprintf("All Vocations • Showing top %d of %d", len(powergamers), len(powergamers))

	brazilLocation := time.FixedZone("BRT", -3*60*60)
	yesterday := time.Now().In(brazilLocation).Add(-24 * time.Hour)

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📊 Powergamer Statistics - %s", yesterday.Format("Jan 2, 2006")),
		Description: description.String(),
		Color:       0xFFD700,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: footer,
		},
	}
}

func formatTibiaNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		if n%1000 == 0 {
			return fmt.Sprintf("%dk", n/1000)
		}
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	} else {
		if n%1000000 == 0 {
			return fmt.Sprintf("%dkk", n/1000000)
		}
		return fmt.Sprintf("%.1fkk", float64(n)/1000000.0)
	}
}
