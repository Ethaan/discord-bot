package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ethaan/discord-api/pkg/config"
	"github.com/ethaan/discord-api/pkg/database"
	"github.com/ethaan/discord-api/pkg/discord"
	"github.com/ethaan/discord-api/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	if cfg.DiscordToken == "" || cfg.DiscordToken == "your_bot_token_here" {
		logger.Error("DISCORD_BOT_TOKEN is not set. Please set it in .env file")
		os.Exit(1)
	}

	if err := database.Connect(cfg.Database.DSN()); err != nil {
		logger.Error("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.AutoMigrate(); err != nil {
		logger.Error("Failed to run database migrations: %v", err)
		os.Exit(1)
	}

	if err := database.InitializeGuildConfig(cfg.DiscordGuildID, cfg.ParentCategoryID, nil); err != nil {
		logger.Error("Failed to initialize guild config: %v", err)
		os.Exit(1)
	}

	bot, err := discord.New(cfg.DiscordToken, cfg.DiscordGuildID, cfg.TibiaAPIURL)
	if err != nil {
		logger.Error("Failed to create Discord bot: %v", err)
		os.Exit(1)
	}

	bot.RegisterCommand(discord.PingCommand())
	bot.RegisterCommand(discord.CreateListCommand())
	bot.RegisterCommand(discord.CloseListCommand())
	bot.RegisterCommand(discord.AddCommand())
	bot.RegisterCommand(discord.AddByGuildCommand())
	bot.RegisterCommand(discord.ListCommand())
	bot.RegisterCommand(discord.RemoveCommand())
	bot.RegisterCommand(discord.EnableEveryoneCommand())
	bot.RegisterCommand(discord.DisableEveryoneCommand())
	bot.RegisterCommand(discord.ScannCommand())

	if err := bot.Start(); err != nil {
		logger.Error("Failed to start Discord bot: %v", err)
		os.Exit(1)
	}

	// Attempt to migrate existing channels to the configured category
	if err := bot.MigrateListChannels(); err != nil {
		logger.Warn("Failed to migrate list channels: %v", err)
		logger.Info("You can manually move channels by dragging them in Discord")
	}

	logger.Info("Bot is running. Press CTRL-C to exit")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	logger.Info("Shutting down bot...")
	if err := bot.Stop(); err != nil {
		logger.Error("Error stopping bot: %v", err)
	}
}
