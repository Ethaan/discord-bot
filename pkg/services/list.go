package services

import (
	"encoding/json"
	"fmt"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/database"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/repositories"
)

type ListService struct {
	repo     *repositories.ListRepository
	itemRepo *repositories.ListItemRepository
}

func NewListService() *ListService {
	return &ListService{
		repo:     repositories.NewListRepository(),
		itemRepo: repositories.NewListItemRepository(),
	}
}

type CreateListInput struct {
	Name        string
	Description string
	Type        string
	GuildID     string
	Session     *discordgo.Session
}

func (s *ListService) CreateList(input CreateListInput) (*database.List, error) {
	uniqueID, err := gonanoid.New(6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique ID: %w", err)
	}

	channelName := strings.ToLower(strings.ReplaceAll(input.Name, " ", "-"))
	channelName = fmt.Sprintf("%s-%s", channelName, uniqueID)

	logger.Info("Creating channel '%s' with type '%s'", channelName, input.Type)

	channel, err := input.Session.GuildChannelCreate(input.GuildID, channelName, discordgo.ChannelTypeGuildText)
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	_, err = input.Session.ChannelEditComplex(channel.ID, &discordgo.ChannelEdit{
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				ID:   input.GuildID, // @everyone role has same ID as guild
				Type: discordgo.PermissionOverwriteTypeRole,
				Allow: discordgo.PermissionViewChannel |
					discordgo.PermissionReadMessageHistory,
			},
		},
	})

	if err != nil {
		logger.Warn("Failed to set channel permissions: %v", err)
	}

	topic := input.Description
	if topic == "" {
		topic = fmt.Sprintf("📋 %s monitoring list", input.Type)
	}

	_, err = input.Session.ChannelEdit(channel.ID, &discordgo.ChannelEdit{
		Topic: topic,
	})

	if err != nil {
		logger.Warn("Failed to set channel topic: %v", err)
	}

	list := &database.List{
		ChannelID:   channel.ID,
		Name:        input.Name,
		Description: input.Description,
		Type:        input.Type,
		GuildID:     input.GuildID,
	}

	if err := s.repo.Create(list); err != nil {
		return nil, fmt.Errorf("failed to save list to database: %w", err)
	}

	logger.Success("Created list #%s (Channel: %s, DB ID: %d)", channelName, channel.ID, list.ID)

	return list, nil
}

type CloseListInput struct {
	ChannelID string
	Session   *discordgo.Session
}

func (s *ListService) CloseList(input CloseListInput) error {
	list, err := s.repo.FindByChannelID(input.ChannelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("this channel is not a monitoring list")
		}
		return fmt.Errorf("failed to find list: %w", err)
	}

	logger.Info("Closing list '%s' (Channel: %s, DB ID: %d)", list.Name, list.ChannelID, list.ID)

	if err := s.repo.Delete(list); err != nil {
		return fmt.Errorf("failed to delete list from database: %w", err)
	}

	_, err = input.Session.ChannelDelete(input.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to delete channel: %w", err)
	}

	logger.Success("Closed list '%s'", list.Name)

	return nil
}

func (s *ListService) GetListByChannelID(channelID string) (*database.List, error) {
	list, err := s.repo.FindByChannelID(channelID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("channel is not a monitoring list")
		}
		return nil, fmt.Errorf("failed to find list: %w", err)
	}
	return list, nil
}

func (s *ListService) UpdateList(list *database.List) error {
	if err := s.repo.Update(list); err != nil {
		return fmt.Errorf("failed to update list: %w", err)
	}
	logger.Info("Updated list '%s' (ID: %d)", list.Name, list.ID)
	return nil
}

type AddItemInput struct {
	ListID   uint
	Name     string
	Metadata map[string]interface{}
}

func (s *ListService) AddItem(input AddItemInput) (*database.ListItem, error) {
	existing, err := s.itemRepo.FindByName(input.ListID, input.Name)
	if err == nil && existing.ID > 0 {
		return nil, fmt.Errorf("item '%s' already exists in this list", input.Name)
	}

	metadataJSON := []byte("{}")
	if input.Metadata != nil && len(input.Metadata) > 0 {
		var jsonErr error
		metadataJSON, jsonErr = json.Marshal(input.Metadata)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to encode metadata: %w", jsonErr)
		}
	}

	list, err := s.repo.FindByID(input.ListID)
	if err != nil {
		return nil, fmt.Errorf("list not found: %w", err)
	}

	item := &database.ListItem{
		ListID:    input.ListID,
		ChannelID: list.ChannelID,
		Name:      input.Name,
		Metadata:  metadataJSON,
	}

	if err := s.itemRepo.Create(item); err != nil {
		return nil, fmt.Errorf("failed to create list item: %w", err)
	}

	logger.Success("Added '%s' to list (ID: %d)", input.Name, input.ListID)

	return item, nil
}

type ListItemWithMetadata struct {
	ID        uint
	ListID    uint
	Name      string
	Metadata  map[string]interface{}
	CreatedAt string
}

func (s *ListService) GetListItems(listID uint) ([]ListItemWithMetadata, error) {
	items, err := s.itemRepo.FindByListID(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch items: %w", err)
	}

	result := make([]ListItemWithMetadata, len(items))
	for i, item := range items {
		var metadata map[string]interface{}
		if err := json.Unmarshal(item.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}

		result[i] = ListItemWithMetadata{
			ID:        item.ID,
			ListID:    item.ListID,
			Name:      item.Name,
			Metadata:  metadata,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return result, nil
}

type RemoveItemInput struct {
	ListID uint
	Name   string
}

func (s *ListService) RemoveItem(input RemoveItemInput) error {
	item, err := s.itemRepo.FindByName(input.ListID, input.Name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("item '%s' not found in this list", input.Name)
		}
		return fmt.Errorf("failed to find item: %w", err)
	}

	if err := s.itemRepo.Delete(item); err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	logger.Success("Removed '%s' from list (ID: %d)", input.Name, input.ListID)

	return nil
}

type BatchAddResult struct {
	Added      int
	Duplicates int
	Failed     int
	Total      int
}

func (s *ListService) BatchAddItems(listID uint, names []string) (*BatchAddResult, error) {
	result := &BatchAddResult{
		Total: len(names),
	}

	if len(names) == 0 {
		return result, nil
	}

	list, err := s.repo.FindByID(listID)
	if err != nil {
		return nil, fmt.Errorf("list not found: %w", err)
	}

	existingItems, err := s.itemRepo.FindByListID(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing items: %w", err)
	}

	existingNames := make(map[string]bool)
	for _, item := range existingItems {
		existingNames[item.Name] = true
	}

	itemsToAdd := make([]database.ListItem, 0)
	metadataJSON := []byte("{}")

	for _, name := range names {
		if existingNames[name] {
			result.Duplicates++
			continue
		}

		itemsToAdd = append(itemsToAdd, database.ListItem{
			ListID:    listID,
			ChannelID: list.ChannelID,
			Name:      name,
			Metadata:  metadataJSON,
		})

		existingNames[name] = true
	}

	chunkSize := 50
	for i := 0; i < len(itemsToAdd); i += chunkSize {
		end := i + chunkSize
		if end > len(itemsToAdd) {
			end = len(itemsToAdd)
		}

		chunk := itemsToAdd[i:end]
		if err := s.itemRepo.BulkCreate(chunk); err != nil {
			logger.Error("Failed to bulk create chunk: %v", err)
			result.Failed += len(chunk)
		} else {
			result.Added += len(chunk)
		}
	}

	logger.Success("Batch added %d items to list (ID: %d), %d duplicates, %d failed",
		result.Added, listID, result.Duplicates, result.Failed)

	return result, nil
}
