package services

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/database"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/repositories"
)

type GuildConfigService struct {
	configRepo *repositories.GuildConfigRepository
	listRepo   *repositories.ListRepository
}

func NewGuildConfigService() *GuildConfigService {
	return &GuildConfigService{
		configRepo: repositories.NewGuildConfigRepository(),
		listRepo:   repositories.NewListRepository(),
	}
}

func (s *GuildConfigService) MigrateExistingChannels(guildID string, session *discordgo.Session) error {
	config, err := s.configRepo.FindByGuildID(guildID)
	if err != nil {
		return fmt.Errorf("failed to get guild config: %w", err)
	}

	if config.ListsCategoryID == "" {
		logger.Info("No parent category configured, skipping channel migration")
		return nil
	}

	category, err := session.Channel(config.ListsCategoryID)
	if err != nil {
		logger.Warn("Failed to verify category %s: %v", config.ListsCategoryID, err)
		return fmt.Errorf("category not found or inaccessible: %w", err)
	}

	channelTypeName := "unknown"
	switch category.Type {
	case discordgo.ChannelTypeGuildText:
		channelTypeName = "text channel"
	case discordgo.ChannelTypeGuildVoice:
		channelTypeName = "voice channel"
	case discordgo.ChannelTypeGuildCategory:
		channelTypeName = "category"
	case discordgo.ChannelTypeGuildNews:
		channelTypeName = "announcement channel"
	case discordgo.ChannelTypeGuildForum:
		channelTypeName = "forum channel"
	default:
		channelTypeName = fmt.Sprintf("type %d", category.Type)
	}

	logger.Info("Found channel: Name='%s', ID='%s', Type='%s' (%d)",
		category.Name, category.ID, channelTypeName, category.Type)

	if category.Type != discordgo.ChannelTypeGuildCategory {
		return fmt.Errorf("channel '%s' (ID: %s) is a %s, not a category",
			category.Name, config.ListsCategoryID, channelTypeName)
	}

	logger.Info("Verified category '%s' - proceeding with migration", category.Name)

	lists, err := s.listRepo.FindByType("")
	if err != nil {
		return fmt.Errorf("failed to fetch lists: %w", err)
	}

	guildLists := make([]database.List, 0)
	for _, list := range lists {
		if list.GuildID == guildID {
			guildLists = append(guildLists, list)
		}
	}

	if len(guildLists) == 0 {
		logger.Info("No existing lists found to migrate")
		return nil
	}

	logger.Info("Found %d list channels to migrate to category '%s'", len(guildLists), category.Name)

	successCount := 0
	failCount := 0

	for _, list := range guildLists {
		channel, err := session.Channel(list.ChannelID)
		if err != nil {
			logger.Warn("Failed to get channel %s: %v", list.ChannelID, err)
			failCount++
			continue
		}

		if channel.ParentID == config.ListsCategoryID {
			logger.Info("Channel #%s already in correct category, skipping", channel.Name)
			successCount++
			continue
		}

		_, err = session.ChannelEditComplex(list.ChannelID, &discordgo.ChannelEdit{
			ParentID: config.ListsCategoryID,
		})

		if err != nil {
			logger.Warn("Failed to move channel #%s to category: %v", channel.Name, err)
			failCount++
		} else {
			logger.Success("Moved channel #%s to category '%s'", channel.Name, category.Name)
			successCount++
		}
	}

	logger.Info("Channel migration complete: %d successful, %d failed", successCount, failCount)

	if failCount > 0 {
		logger.Info("Failed channels can be manually moved by dragging them in Discord")
	}

	return nil
}
