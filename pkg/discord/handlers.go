package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/services"
)

var validListTypes = []string{
	"premium-alerts",
}

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
				Content: "❌ This channel is not a monitoring list. Use this command in a list channel.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	if list.Type != "premium-alerts" {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Use `/add-exp-lock` for %s lists", list.Type),
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
			Content: fmt.Sprintf("✅ Added **%s** to premium-alerts monitoring", name),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
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
				Content: "❌ This channel is not a monitoring list. Use this command in a list channel.",
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
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📋 This list is empty. Use `/add` to add characters.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	}

	var content string
	content = fmt.Sprintf("📋 **%s** (%s)\n\n", list.Name, list.Type)

	for _, item := range items {
		switch list.Type {
		case "premium-alerts":
			status := "⏳ PENDING"
			if isPremium, ok := item.Metadata["premium_status"].(bool); ok {
				if isPremium {
					status = "✅ PREMIUM"
				} else {
					status = "🔴 FREE ACCOUNT"
				}
			}
			content += fmt.Sprintf("**%s**: %s\n", item.Name, status)
		default:
			content += fmt.Sprintf("• **%s**\n", item.Name)
		}
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
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
				Content: "❌ This channel is not a monitoring list. Use this command in a list channel.",
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
