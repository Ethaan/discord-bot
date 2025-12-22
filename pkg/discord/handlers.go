package discord

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/database"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/repositories"
	"github.com/ethaan/discord-api/pkg/services"
	"github.com/ethaan/discord-api/pkg/tibia"
)

var validListTypes = []string{
	"premium-alerts",
	"residence-change",
	"powergames-stats",
	"powergamer-stats-historical",
	"scanner",
}

const errNotMonitoringList = "❌ This channel is not a monitoring list. Use this command in a list channel."

func PingCommand() *Command {
	return &Command{
		Name:        "ping",
		Description: "Responds with Pong!",
		Handler:     handlePing,
	}
}

func handlePing(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	logger.Debug("Ping command received")
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🏓 Pong!",
		},
	})
}

func CreateListCommand() *Command {
	choices := make([]*discordgo.ApplicationCommandOptionChoice, len(validListTypes))
	for i, listType := range validListTypes {
		choices[i] = &discordgo.ApplicationCommandOptionChoice{
			Name:  listType,
			Value: listType,
		}
	}

	return &Command{
		Name:        "create-list",
		Description: "Create a new monitoring list channel",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Name for the list (will become channel name)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "type",
				Description: "Type of monitoring list",
				Required:    true,
				Choices:     choices,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "description",
				Description: "Optional description for the list",
				Required:    false,
			},
		},
		Handler: handleCreateList,
	}
}

func handleCreateList(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	listName := optionMap["name"].StringValue()
	listType := optionMap["type"].StringValue()

	var description string
	if descOpt, ok := optionMap["description"]; ok {
		description = descOpt.StringValue()
	}

	logger.Info("Creating list: name=%s, type=%s", listName, listType)

	guildID := i.GuildID
	if guildID == "" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ This command must be used in a server",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	listService := services.NewListService()
	list, err := listService.CreateList(services.CreateListInput{
		Name:        listName,
		Description: description,
		Type:        listType,
		GuildID:     guildID,
		Session:     s,
	})

	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to create list: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Successfully created list channel <#%s> for **%s**!",
				list.ChannelID, listType),
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func CloseListCommand() *Command {
	return &Command{
		Name:        "close-list",
		Description: "Close and delete this monitoring list channel",
		Handler:     handleCloseList,
	}
}

func handleCloseList(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channelID := i.ChannelID

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Closing list and deleting channel...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		logger.Error("Error responding to interaction: %v", err)
		return err
	}

	listService := services.NewListService()
	err = listService.CloseList(services.CloseListInput{
		ChannelID: channelID,
		Session:   s,
	})

	if err != nil {
		logger.Error("Error closing list: %v", err)
	}

	return nil
}

func AddCommand() *Command {
	return &Command{
		Name:        "add",
		Description: "Add a character to this monitoring list",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Character name",
				Required:    true,
			},
		},
		Handler: handleAdd,
	}
}

func handleAdd(s *discordgo.Session, i *discordgo.InteractionCreate) error {

	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if list.Type != "premium-alerts" && list.Type != "residence-change" && list.Type != "powergames-stats" && list.Type != "powergamer-stats-historical" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Command not available for this list type %s", list.Type),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	name := optionMap["name"].StringValue()

	_, err = listService.AddItem(services.AddItemInput{
		ListID:   list.ID,
		Name:     name,
		Metadata: nil,
	})

	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to add item: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Added **%s** to %s monitoring", name, list.Type),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func AddByGuildCommand() *Command {
	return &Command{
		Name:        "add-by-guild",
		Description: "Add all members from a guild to this monitoring list",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "guild-id",
				Description: "Tibia guild ID",
				Required:    true,
			},
		},
		Handler: handleAddByGuild,
	}
}

func handleAddByGuild(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if list.Type != "premium-alerts" && list.Type != "residence-change" && list.Type != "powergames-stats" && list.Type != "powergamer-stats-historical" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Command not available for this list type %s", list.Type),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	guildID := int(optionMap["guild-id"].IntValue())

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return err
	}

	tibiaAPIURL := os.Getenv("TIBIA_API_URL")
	if tibiaAPIURL == "" {
		tibiaAPIURL = "http://localhost:8080"
	}

	tibiaClient := tibia.NewClient(tibiaAPIURL)
	guild, err := tibiaClient.GetGuildMembers(guildID)
	if err != nil {
		content := fmt.Sprintf("❌ Failed to fetch guild members: %v", err)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return err
	}

	names := make([]string, len(guild.Members))
	for idx, member := range guild.Members {
		names[idx] = member.Name
	}

	result, err := listService.BatchAddItems(list.ID, names)
	if err != nil {
		content := fmt.Sprintf("❌ Failed to add guild members: %v", err)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return err
	}

	content := fmt.Sprintf("✅ **Batch Add Complete**\n\n"+
		"📊 **Summary:**\n"+
		"• Total members: %d\n"+
		"• Added: %d\n"+
		"• Duplicates skipped: %d\n"+
		"• Failed: %d",
		result.Total, result.Added, result.Duplicates, result.Failed)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	return err
}

func RemoveCommand() *Command {
	return &Command{
		Name:        "remove",
		Description: "Remove a character from this monitoring list",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "name",
				Description:  "Character name",
				Required:     true,
				Autocomplete: true,
			},
		},
		Handler:             handleRemove,
		AutocompleteHandler: handleRemoveAutocomplete,
	}
}

func handleRemove(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	name := optionMap["name"].StringValue()

	err = listService.RemoveItem(services.RemoveItemInput{
		ListID: list.ID,
		Name:   name,
	})

	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to remove item: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Removed **%s** from the monitoring list", name),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleRemoveAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) ([]*discordgo.ApplicationCommandOptionChoice, error) {
	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return []*discordgo.ApplicationCommandOptionChoice{}, nil
	}

	items, err := listService.GetListItems(list.ID)
	if err != nil {
		return []*discordgo.ApplicationCommandOptionChoice{}, nil
	}

	options := i.ApplicationCommandData().Options
	var focusedValue string
	for _, opt := range options {
		if opt.Focused {
			focusedValue = strings.ToLower(opt.StringValue())
			break
		}
	}

	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0)
	for _, item := range items {
		if focusedValue == "" || strings.Contains(strings.ToLower(item.Name), focusedValue) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  item.Name,
				Value: item.Name,
			})

			if len(choices) >= 25 {
				break
			}
		}
	}

	return choices, nil
}

func ListCommand() *Command {
	return &Command{
		Name:        "list",
		Description: "Show all items in this monitoring list",
		Handler:     handleList,
	}
}

func handleList(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	items, err := listService.GetListItems(list.ID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to fetch list items: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if len(items) == 0 {
		var emptyMessage string
		if list.Type == "powergames-stats" || list.Type == "powergamer-stats-historical" {
			emptyMessage = "📋 This list is empty. Use `/add` to add characters to track.\n\n" +
				"📊 Stats will be posted automatically for tracked characters."
		} else {
			emptyMessage = "📋 This list is empty. Use `/add` to add characters."
		}

		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: emptyMessage,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	var description string

	for _, item := range items {
		switch list.Type {
		case "premium-alerts":
			status := "⏳ Pending"
			if isPremium, ok := item.Metadata["premium_status"].(bool); ok {
				if isPremium {
					status = "✅ Premium"
				} else {
					status = "🔴 Free"
				}
			}
			description += fmt.Sprintf("**%s**: %s\n", item.Name, status)
		case "residence-change":
			residence := "⏳ Pending"
			if currentResidence, ok := item.Metadata["residence"].(string); ok && currentResidence != "" {
				residence = currentResidence
			}
			description += fmt.Sprintf("**%s**: %s\n", item.Name, residence)
		case "powergames-stats", "powergamer-stats-historical":
			description += fmt.Sprintf("• **%s**\n", item.Name)
		default:
			description += fmt.Sprintf("• **%s**\n", item.Name)
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📋 %s", list.Name),
		Description: description,
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("List Type: %s • Total Items: %d", list.Type, len(items)),
		},
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func AddExpLockCommand() *Command {
	return &Command{
		Name:        "add-exp-lock",
		Description: "Add a character to this exp-lock monitoring list",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Character name",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "max_exp",
				Description: "Maximum experience threshold",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "period",
				Description: "Monitoring period",
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "weekly", Value: "weekly"},
					{Name: "bi-weekly", Value: "bi-weekly"},
					{Name: "monthly", Value: "monthly"},
				},
			},
		},
		Handler: handleAddExpLock,
	}
}

func handleAddExpLock(s *discordgo.Session, i *discordgo.InteractionCreate) error {

	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if list.Type != "exp-lock" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Use `/add` for %s lists", list.Type),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	name := optionMap["name"].StringValue()
	maxExp := optionMap["max_exp"].IntValue()
	period := optionMap["period"].StringValue()

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Would add **%s** to exp-lock monitoring (Max: %d, Period: %s)",
				name, maxExp, period),
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
}

func EnableEveryoneCommand() *Command {
	return &Command{
		Name:        "enable-everyone",
		Description: "Enable @everyone notifications for this list",
		Handler:     handleEnableEveryone,
	}
}

func handleEnableEveryone(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	list.NotifyEveryone = true
	err = listService.UpdateList(list)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to update list: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ @everyone notifications enabled for this list",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func DisableEveryoneCommand() *Command {
	return &Command{
		Name:        "disable-everyone",
		Description: "Disable @everyone notifications for this list",
		Handler:     handleDisableEveryone,
	}
}

func handleDisableEveryone(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	list.NotifyEveryone = false
	err = listService.UpdateList(list)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Failed to update list: %v", err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ @everyone notifications disabled for this list",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func ScanCommand() *Command {
	return &Command{
		Name:        "scan",
		Description: "Scan for characters related to a target character",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Character name to scan",
				Required:    true,
			},
		},
		Handler: handleScan,
	}
}

func handleScan(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	channelID := i.ChannelID

	listService := services.NewListService()
	list, err := listService.GetListByChannelID(channelID)
	if err != nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: errNotMonitoringList,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if list.Type != "scanner" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ This command can only be used in scanner list channels",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	characterName := optionMap["name"].StringValue()

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return err
	}

	startTime := time.Now()

	sessionRepo := repositories.NewOnlineSessionRepository()
	results, err := sessionRepo.ScanCharacter(
		characterName,
		ScanAdjacentWindowSeconds,
		ScanMaxResults,
	)

	if err != nil {
		content := fmt.Sprintf("❌ Failed to scan character: %v", err)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return err
	}

	playerRepo := repositories.NewPlayerRepository()
	player, err := playerRepo.FindByName(characterName)
	if err != nil || player == nil {
		content := fmt.Sprintf("❌ Character **%s** not found in database. The character may not have been tracked yet.", characterName)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return err
	}

	var veryHighConfidence, highConfidence, mediumConfidence, lowConfidence []repositories.ScanResult

	for _, result := range results {
		if result.AdjacentCount >= ScanVeryHighConfidenceThreshold {
			result.ConfidenceLevel = ConfidenceVeryHigh
			veryHighConfidence = append(veryHighConfidence, result)
		} else if result.AdjacentCount >= ScanHighConfidenceThreshold {
			result.ConfidenceLevel = ConfidenceHigh
			highConfidence = append(highConfidence, result)
		} else if result.AdjacentCount >= ScanMediumConfidenceThreshold {
			result.ConfidenceLevel = ConfidenceMedium
			mediumConfidence = append(mediumConfidence, result)
		} else if result.AdjacentCount >= ScanLowConfidenceThreshold {
			result.ConfidenceLevel = ConfidenceLow
			lowConfidence = append(lowConfidence, result)
		}
	}

	var totalCharacters int64
	err = database.DB.Model(&database.Player{}).Count(&totalCharacters).Error
	if err != nil {
		totalCharacters = 0
	}

	var responseContent strings.Builder
	responseContent.WriteString(fmt.Sprintf("🔍 **Scanning for characters related to %s...**\n\n", characterName))

	if len(results) == 0 {
		responseContent.WriteString("✅ No related characters found based on login/logout patterns.\n\n")
		responseContent.WriteString(fmt.Sprintf("📊 Analyzed %d characters in %.2fs", totalCharacters, time.Since(startTime).Seconds()))
	} else {
		if len(veryHighConfidence) > 0 {
			responseContent.WriteString(fmt.Sprintf("🔴 **Very High Confidence (%d+ adjacent sessions):**\n", ScanVeryHighConfidenceThreshold))
			for _, r := range veryHighConfidence {
				charLink := fmt.Sprintf("https://miracle74.com/?subtopic=characters&name=%s", r.CharacterName)
				responseContent.WriteString(fmt.Sprintf("  • **[%s](%s)** - %d adjacent transitions, never online together\n", r.CharacterName, charLink, r.AdjacentCount))
			}
			responseContent.WriteString("\n")
		}

		if len(highConfidence) > 0 {
			responseContent.WriteString(fmt.Sprintf("🟠 **High Confidence (%d-%d adjacent sessions):**\n", ScanHighConfidenceThreshold, ScanVeryHighConfidenceThreshold-1))
			for _, r := range highConfidence {
				charLink := fmt.Sprintf("https://miracle74.com/?subtopic=characters&name=%s", r.CharacterName)
				responseContent.WriteString(fmt.Sprintf("  • **[%s](%s)** - %d adjacent transitions, never online together\n", r.CharacterName, charLink, r.AdjacentCount))
			}
			responseContent.WriteString("\n")
		}

		if len(mediumConfidence) > 0 {
			responseContent.WriteString(fmt.Sprintf("🟡 **Medium Confidence (%d-%d adjacent sessions):**\n", ScanMediumConfidenceThreshold, ScanHighConfidenceThreshold-1))
			for _, r := range mediumConfidence {
				charLink := fmt.Sprintf("https://miracle74.com/?subtopic=characters&name=%s", r.CharacterName)
				responseContent.WriteString(fmt.Sprintf("  • **[%s](%s)** - %d adjacent transitions, never online together\n", r.CharacterName, charLink, r.AdjacentCount))
			}
			responseContent.WriteString("\n")
		}

		if len(lowConfidence) > 0 {
			responseContent.WriteString(fmt.Sprintf("⚪ **Low Confidence (%d-%d adjacent sessions):**\n", ScanLowConfidenceThreshold, ScanMediumConfidenceThreshold-1))
			for _, r := range lowConfidence {
				charLink := fmt.Sprintf("https://miracle74.com/?subtopic=characters&name=%s", r.CharacterName)
				responseContent.WriteString(fmt.Sprintf("  • **[%s](%s)** - %d adjacent transitions, never online together\n", r.CharacterName, charLink, r.AdjacentCount))
			}
			responseContent.WriteString("\n")
		}

		responseContent.WriteString(fmt.Sprintf("📊 Analyzed %d characters in %.2fs", totalCharacters, time.Since(startTime).Seconds()))
	}

	content := responseContent.String()
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})

	return err
}
