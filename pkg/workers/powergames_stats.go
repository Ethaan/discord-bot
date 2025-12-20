package workers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/database"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/repositories"
	"github.com/ethaan/discord-api/pkg/tibia"
)

type PowergamesStatsWorker struct {
	session      *discordgo.Session
	listRepo     *repositories.ListRepository
	itemRepo     *repositories.ListItemRepository
	tibiaClient  *tibia.Client
	pollInterval time.Duration
}

func NewPowergamesStatsWorker(session *discordgo.Session, tibiaAPIURL string) *PowergamesStatsWorker {
	return &PowergamesStatsWorker{
		session:      session,
		listRepo:     repositories.NewListRepository(),
		itemRepo:     repositories.NewListItemRepository(),
		tibiaClient:  tibia.NewClient(tibiaAPIURL),
		pollInterval: 1 * time.Minute,
	}
}

func (w *PowergamesStatsWorker) Name() string {
	return "powergames-stats"
}

func (w *PowergamesStatsWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.updateAllStats()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.updateAllStats()
		}
	}
}

func (w *PowergamesStatsWorker) updateAllStats() {
	lists, err := w.listRepo.FindByType("powergames-stats")
	if err != nil {
		logger.Worker("powergames-stats", "Error fetching lists: %v", err)
		return
	}

	logger.Worker("powergames-stats", "Updating %d channels", len(lists))

	for _, list := range lists {
		if err := w.updateChannelStats(&list); err != nil {
			logger.Worker("powergames-stats", "Error updating channel %s: %v", list.ChannelID, err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (w *PowergamesStatsWorker) updateChannelStats(list *database.List) error {
	items, err := w.itemRepo.FindByListID(list.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch list items: %w", err)
	}

	if len(items) == 0 {
		logger.Worker("powergames-stats", "Skipping channel %s (empty list)", list.ChannelID)
		return nil
	}

	powergamers, err := w.tibiaClient.GetPowergamers("today", "", false)
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

	embed := w.buildStatsEmbed(powergamers, len(items))

	messages, err := w.session.ChannelMessages(list.ChannelID, 1, "", "", "")
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	botID := w.session.State.User.ID
	if len(messages) > 0 && messages[0].Author.ID == botID {
		_, err = w.session.ChannelMessageEditEmbed(list.ChannelID, messages[0].ID, embed)
		if err != nil {
			return fmt.Errorf("failed to update message: %w", err)
		}
		logger.Worker("powergames-stats", "Updated stats in channel %s", list.ChannelID)
	} else {
		_, err = w.session.ChannelMessageSendEmbed(list.ChannelID, embed)
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
		logger.Worker("powergames-stats", "Posted new stats in channel %s", list.ChannelID)
	}

	return nil
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

func (w *PowergamesStatsWorker) buildStatsEmbed(powergamers []tibia.Powergamer, listItemCount int) *discordgo.MessageEmbed {
	var description strings.Builder

	if len(powergamers) == 0 {
		if listItemCount > 0 {
			description.WriteString(fmt.Sprintf(
				"📊 No powergamers found.\n\nNone of the %d characters in your list were in today's powergamer rankings.",
				listItemCount,
			))
		} else {
			description.WriteString("📊 No powergamers found for today.")
		}
	} else {
		maxNameLen := 0
		const maxAllowedNameLen = 16

		for _, pg := range powergamers {
			if len(pg.Name) > maxNameLen {
				maxNameLen = len(pg.Name)
			}
		}
		if maxNameLen > maxAllowedNameLen {
			maxNameLen = maxAllowedNameLen
		}

		description.WriteString("```text\n")
		description.WriteString("Voc Lvl Name")
		description.WriteString(strings.Repeat(" ", maxNameLen-4))
		description.WriteString(" EXP+\n")

		for _, pg := range powergamers {
			name := pg.Name
			if len(name) > maxNameLen {
				name = name[:maxNameLen-1] + "…"
			}

			description.WriteString(fmt.Sprintf(
				"%-2s %-3d %-*s %s\n",
				tibia.VocationEmoji(pg.Vocation),
				pg.Level,
				maxNameLen,
				name,
				formatTibiaNumber(pg.Today),
			))
		}

		description.WriteString("```")
	}

	footer := fmt.Sprintf("All Vocations • Showing top %d of %d", len(powergamers), len(powergamers))
	if len(powergamers) > 25 {
		footer = fmt.Sprintf("All Vocations • Showing top 25 of %d", len(powergamers))
	}

	return &discordgo.MessageEmbed{
		Title:       "📊 Powergamer Statistics - Today",
		Description: description.String(),
		Color:       0xFFD700,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: footer,
		},
	}
}
