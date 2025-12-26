package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/database"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/repositories"
	"github.com/ethaan/discord-api/pkg/tibia"
)

type PremiumWorker struct {
	session      *discordgo.Session
	listRepo     *repositories.ListRepository
	itemRepo     *repositories.ListItemRepository
	tibiaClient  *tibia.Client
	pollInterval time.Duration
}

func NewPremiumWorker(session *discordgo.Session, tibiaAPIURL string) *PremiumWorker {
	return &PremiumWorker{
		session:      session,
		listRepo:     repositories.NewListRepository(),
		itemRepo:     repositories.NewListItemRepository(),
		tibiaClient:  tibia.NewClient(tibiaAPIURL),
		pollInterval: 1 * time.Hour,
	}
}

func (w *PremiumWorker) Name() string {
	return "premium-alerts"
}

func (w *PremiumWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.checkPremiumStatus()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkPremiumStatus()
		}
	}
}

func (w *PremiumWorker) checkPremiumStatus() {
	lists, err := w.listRepo.FindByType("premium-alerts")
	if err != nil {
		logger.Worker("premium-alerts", "Error fetching lists: %v", err)
		return
	}

	logger.Worker("premium-alerts", "Checking %d lists", len(lists))

	for _, list := range lists {
		items, err := w.itemRepo.FindByListID(list.ID)
		if err != nil {
			logger.Worker("premium-alerts", "Error fetching items for list %d: %v", list.ID, err)
			continue
		}

		for _, item := range items {
			w.checkCharacter(&list, &item)
			time.Sleep(1 * time.Second)
		}
	}
}

func (w *PremiumWorker) checkCharacter(list *database.List, item *database.ListItem) {
	character, err := w.tibiaClient.GetCharacter(item.Name)
	if err != nil {
		logger.Worker("premium-alerts", "Error fetching character %s: %v", item.Name, err)
		return
	}

	isPremium := character.IsPremium

	var metadata map[string]interface{}
	if err := json.Unmarshal(item.Metadata, &metadata); err != nil {
		metadata = make(map[string]interface{})
	}

	lastStatus, hasStatus := metadata["premium_status"].(bool)

	if !hasStatus {
		metadata["premium_status"] = isPremium
		w.updateMetadata(item, metadata)
		logger.Worker("premium-alerts", "Initial status for %s: premium=%v", item.Name, isPremium)
		return
	}

	if lastStatus != isPremium {
		logger.Worker("premium-alerts", "Status changed for %s: %v -> %v", item.Name, lastStatus, isPremium)
		w.sendNotification(list, item, isPremium)
		metadata["premium_status"] = isPremium
		w.updateMetadata(item, metadata)
	}
}

func (w *PremiumWorker) updateMetadata(item *database.ListItem, metadata map[string]interface{}) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		logger.Error("Error encoding metadata: %v", err)
		return
	}

	item.Metadata = metadataJSON
	if err := w.itemRepo.Update(item); err != nil {
		logger.Error("Error updating item: %v", err)
	}
}

func (w *PremiumWorker) sendNotification(list *database.List, item *database.ListItem, isPremium bool) {
	var color int
	var status string
	var emoji string

	if isPremium {
		color = 0x00FF00
		status = "Premium Account"
		emoji = "✅"
	} else {
		color = 0xFF0000
		status = "Free Account"
		emoji = "🔴"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Premium Status Changed", emoji),
		Description: fmt.Sprintf("**%s** is now a **%s**", item.Name, status),
		Color:       color,
		Timestamp:   fmt.Sprintf("%d", time.Now().Unix()),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Premium Alert",
		},
	}

	content := ""
	if list.NotifyEveryone {
		content = "@everyone"
	}

	_, err := w.session.ChannelMessageSendComplex(list.ChannelID, &discordgo.MessageSend{
		Content: content,
		Embed:   embed,
	})

	if err != nil {
		logger.Error("Error sending notification: %v", err)
	}
}
