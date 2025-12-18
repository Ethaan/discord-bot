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
	repo         *repositories.ListRepository
	itemRepo     *repositories.ListItemRepository
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
