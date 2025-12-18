package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/ethaan/discord-api/pkg/logger"
	"github.com/ethaan/discord-api/pkg/workers"
)

type Bot struct {
	session       *discordgo.Session
	commands      []*Command
	guildID       string
	workerManager *workers.Manager
}

func New(token, guildID, tibiaAPIURL string) (*Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("discord bot token is required")
	}

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("failed to create discord session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	bot := &Bot{
		session:       session,
		commands:      make([]*Command, 0),
		guildID:       guildID,
		workerManager: workers.NewManager(session, tibiaAPIURL),
	}

	return bot, nil
}

func (b *Bot) Start() error {
	b.session.AddHandler(b.handleInteractionCreate)

	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Success("Discord bot logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open discord session: %w", err)
	}

	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	b.workerManager.Start()

	logger.Success("Discord bot is now running")
	return nil
}

func (b *Bot) Stop() error {
	b.workerManager.Stop()

	if err := b.removeCommands(); err != nil {
		logger.Error("Error removing commands: %v", err)
	}

	return b.session.Close()
}

func (b *Bot) handleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	for _, cmd := range b.commands {
		if cmd.Name == i.ApplicationCommandData().Name {
			if err := cmd.Handler(s, i); err != nil {
				logger.Error("Error handling command %s: %v", cmd.Name, err)

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("Error executing command: %v", err),
						Flags:   discordgo.MessageFlagsEphemeral,
					},
				})
			}
			return
		}
	}
}
