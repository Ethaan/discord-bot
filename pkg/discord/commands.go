package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/logger"
)

type CommandHandler func(s *discordgo.Session, i *discordgo.InteractionCreate) error
type AutocompleteHandler func(s *discordgo.Session, i *discordgo.InteractionCreate) ([]*discordgo.ApplicationCommandOptionChoice, error)

type Command struct {
	Name                string
	Description         string
	Options             []*discordgo.ApplicationCommandOption
	Handler             CommandHandler
	AutocompleteHandler AutocompleteHandler
}

func (b *Bot) RegisterCommand(cmd *Command) {
	b.commands = append(b.commands, cmd)
}

func (b *Bot) registerCommands() error {
	logger.Info("Registering %d commands...", len(b.commands))

	for _, cmd := range b.commands {
		appCmd := &discordgo.ApplicationCommand{
			Name:        cmd.Name,
			Description: cmd.Description,
			Options:     cmd.Options,
		}

		var err error
		if b.guildID != "" {
			_, err = b.session.ApplicationCommandCreate(b.session.State.User.ID, b.guildID, appCmd)
			if err == nil {
				logger.Debug("Registered guild command: %s", cmd.Name)
			}
		} else {
			_, err = b.session.ApplicationCommandCreate(b.session.State.User.ID, "", appCmd)
			if err == nil {
				logger.Debug("Registered global command: %s", cmd.Name)
			}
		}

		if err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.Name, err)
		}
	}

	return nil
}

func (b *Bot) removeCommands() error {
	logger.Info("Removing registered commands...")

	var commands []*discordgo.ApplicationCommand
	var err error

	if b.guildID != "" {
		commands, err = b.session.ApplicationCommands(b.session.State.User.ID, b.guildID)
	} else {
		commands, err = b.session.ApplicationCommands(b.session.State.User.ID, "")
	}

	if err != nil {
		return fmt.Errorf("failed to fetch commands: %w", err)
	}

	for _, cmd := range commands {
		var deleteErr error
		if b.guildID != "" {
			deleteErr = b.session.ApplicationCommandDelete(b.session.State.User.ID, b.guildID, cmd.ID)
		} else {
			deleteErr = b.session.ApplicationCommandDelete(b.session.State.User.ID, "", cmd.ID)
		}

		if deleteErr != nil {
			logger.Warn("Failed to delete command %s: %v", cmd.Name, deleteErr)
		}
	}

	return nil
}
