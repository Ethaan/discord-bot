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

type ResidenceWorker struct {
	session      *discordgo.Session
	listRepo     *repositories.ListRepository
	itemRepo     *repositories.ListItemRepository
	tibiaClient  *tibia.Client
	pollInterval time.Duration
}

func NewResidenceWorker(session *discordgo.Session, tibiaAPIURL string) *ResidenceWorker {
	return &ResidenceWorker{
		session:      session,
		listRepo:     repositories.NewListRepository(),
		itemRepo:     repositories.NewListItemRepository(),
		tibiaClient:  tibia.NewClient(tibiaAPIURL),
		pollInterval: 1 * time.Minute,
	}
}

func (w *ResidenceWorker) Name() string {
	return "residence-change"
}

func (w *ResidenceWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.checkResidenceStatus()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.checkResidenceStatus()
		}
	}
}

func (w *ResidenceWorker) checkResidenceStatus() {
	lists, err := w.listRepo.FindByType("residence-change")
	if err != nil {
		logger.Worker("residence-change", "Error fetching lists: %v", err)
		return
	}

	logger.Worker("residence-change", "Checking %d lists", len(lists))

	for _, list := range lists {
		items, err := w.itemRepo.FindByListID(list.ID)
		if err != nil {
			logger.Worker("residence-change", "Error fetching items for list %d: %v", list.ID, err)
			continue
		}

		for _, item := range items {
			w.checkCharacter(&list, &item)
			time.Sleep(1 * time.Second)
		}
	}
}

func (w *ResidenceWorker) checkCharacter(list *database.List, item *database.ListItem) {
	character, err := w.tibiaClient.GetCharacter(item.Name)
	if err != nil {
		logger.Worker("residence-change", "Error fetching character %s: %v", item.Name, err)
		return
	}

	currentResidence := character.Residence

	var metadata map[string]interface{}
	if err := json.Unmarshal(item.Metadata, &metadata); err != nil {
		metadata = make(map[string]interface{})
	}

	lastResidence, hasResidence := metadata["residence"].(string)

	if !hasResidence {
		metadata["residence"] = currentResidence
		w.updateMetadata(item, metadata)
		logger.Worker("residence-change", "Initial residence for %s: %s", item.Name, currentResidence)
		return
	}

	if lastResidence != currentResidence {
		logger.Worker("residence-change", "Residence changed for %s: %s -> %s", item.Name, lastResidence, currentResidence)
		w.sendNotification(list, item, lastResidence, currentResidence)
		metadata["residence"] = currentResidence
		w.updateMetadata(item, metadata)
	}
}

func (w *ResidenceWorker) updateMetadata(item *database.ListItem, metadata map[string]interface{}) {
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

func (w *ResidenceWorker) sendNotification(list *database.List, item *database.ListItem, oldResidence, newResidence string) {
	message := fmt.Sprintf("**%s** changed residence: %s → %s", item.Name, oldResidence, newResidence)

	_, err := w.session.ChannelMessageSend(list.ChannelID, message)
	if err != nil {
		logger.Error("Error sending notification: %v", err)
	}
}
