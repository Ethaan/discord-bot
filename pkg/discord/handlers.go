package discord

import (
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/services"
	"github.com/ethaan/discord-api/pkg/tibia"
)

var validListTypes = []string{
	"premium-alerts",
	"residence-change",
	"powergames-stats",
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

	if list.Type != "premium-alerts" && list.Type != "residence-change" && list.Type != "powergames-stats" {
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

	if list.Type != "premium-alerts" && list.Type != "residence-change" && list.Type != "powergames-stats" {
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

	// Build response message
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

	if len(items) == 0 && list.Type != "powergames-stats" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📋 This list is empty. Use `/add` to add characters.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	var description string

	// Special handling for powergames-stats - no items needed
	if list.Type == "powergames-stats" {
		description = "📊 Use `/stats` to view powergamer statistics.\n\n" +
			"Available filters:\n" +
			"• **days**: Choose time period (today, last2days, etc.)\n" +
			"• **vocation**: Filter by vocation (sorcerers, druids, paladins, knights)"
	} else {
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
			default:
				description += fmt.Sprintf("• **%s**\n", item.Name)
			}
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📋 %s", list.Name),
		Description: description,
		Color:       0x5865F2, // Discord Blurple
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

func StatsCommand() *Command {
	return &Command{
		Name:        "stats",
		Description: "Show powergamer statistics",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "days",
				Description: "Time period (default: today)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Today", Value: "today"},
					{Name: "Last Day", Value: "lastday"},
					{Name: "Last 2 Days", Value: "last2days"},
					{Name: "Last 3 Days", Value: "last3days"},
					{Name: "Last 4 Days", Value: "last4days"},
					{Name: "Last 5 Days", Value: "last5days"},
					{Name: "Last 6 Days", Value: "last6days"},
					{Name: "Last 7 Days", Value: "last7days"},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "vocation",
				Description: "Filter by vocation (default: all)",
				Required:    false,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "All Vocations", Value: ""},
					{Name: "No Vocation", Value: "0"},
					{Name: "Sorcerers", Value: "1"},
					{Name: "Druids", Value: "2"},
					{Name: "Paladins", Value: "3"},
					{Name: "Knights", Value: "4"},
				},
			},
		},
		Handler: handleStats,
	}
}

func handleStats(s *discordgo.Session, i *discordgo.InteractionCreate) error {
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

	if list.Type != "powergames-stats" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ This command is only available in powergames-stats lists",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	// Defer the response since API call might take time
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return err
	}

	// Extract optional parameters
	options := i.ApplicationCommandData().Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	days := "today"
	if daysOpt, ok := optionMap["days"]; ok {
		days = daysOpt.StringValue()
	}

	vocation := ""
	if vocationOpt, ok := optionMap["vocation"]; ok {
		vocation = vocationOpt.StringValue()
	}

	// Fetch powergamers data
	tibiaAPIURL := os.Getenv("TIBIA_API_URL")
	if tibiaAPIURL == "" {
		tibiaAPIURL = "http://localhost:8080"
	}

	tibiaClient := tibia.NewClient(tibiaAPIURL)
	powergamers, err := tibiaClient.GetPowergamers(days, vocation, true)
	if err != nil {
		content := fmt.Sprintf("❌ Failed to fetch powergamer statistics: %v", err)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return err
	}

	// Filter by list items if any characters are added
	items, err := listService.GetListItems(list.ID)
	if err == nil && len(items) > 0 {
		// Create a map of character names in the list for fast lookup
		listNames := make(map[string]bool)
		for _, item := range items {
			listNames[strings.ToLower(item.Name)] = true
		}

		// Filter powergamers to only include characters in the list
		filtered := make([]tibia.Powergamer, 0)
		for _, pg := range powergamers {
			if listNames[strings.ToLower(pg.Name)] {
				filtered = append(filtered, pg)
			}
		}
		powergamers = filtered
	}

	if len(powergamers) == 0 {
		content := "📊 No powergamers found for the selected filters."
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return err
	}

	// Build embed with statistics
	var description strings.Builder
	limit := 25
	if len(powergamers) < limit {
		limit = len(powergamers)
	}

	for i := 0; i < limit; i++ {
		pg := powergamers[i]
		description.WriteString(fmt.Sprintf("**%d. %s** (%s)\n", i+1, pg.Name, pg.Vocation))
		description.WriteString(fmt.Sprintf("   Level: %d | Exp Gain: %d | Level Gain: %d\n\n", pg.Level, pg.ExperienceGain, pg.LevelGain))
	}

	// Map vocation code to name for title
	vocationName := "All Vocations"
	switch vocation {
	case "0":
		vocationName = "No Vocation"
	case "1":
		vocationName = "Sorcerers"
	case "2":
		vocationName = "Druids"
	case "3":
		vocationName = "Paladins"
	case "4":
		vocationName = "Knights"
	}

	// Map days to readable name
	daysName := days
	switch days {
	case "today":
		daysName = "Today"
	case "lastday":
		daysName = "Last Day"
	case "last2days":
		daysName = "Last 2 Days"
	case "last3days":
		daysName = "Last 3 Days"
	case "last4days":
		daysName = "Last 4 Days"
	case "last5days":
		daysName = "Last 5 Days"
	case "last6days":
		daysName = "Last 6 Days"
	case "last7days":
		daysName = "Last 7 Days"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📊 Powergamer Statistics - %s", daysName),
		Description: description.String(),
		Color:       0xFFD700, // Gold color
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Vocation: %s • Showing top %d of %d", vocationName, limit, len(powergamers)),
		},
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})

	return err
}
