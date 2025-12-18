# Discord Bot for Tibia

A production-ready Discord bot built with Go that acts as middleware between Discord and Tibia APIs. The bot provides real-time character notifications and slash commands for managing hunting lists.

## Features

- ✅ **Slash Commands** - `/ping`, `/create-list`, `/close-list`, `/add`
- ✅ **Premium Alerts** - Automated worker that monitors character premium status
- ✅ **PostgreSQL Database** - Persistent storage with automatic migrations
- ✅ **API Agnostic** - Works with any Tibia API following the OpenAPI spec
- ✅ **Production Ready** - Deployed on Fly.io with Neon PostgreSQL

## Bot Install Link

Add this bot to your Discord server (requires Manage Server permission):

https://discord.com/oauth2/authorize?client_id=1451202384226156752&permissions=268684304&integration_type=0&scope=applications.commands+bot

---

## Deploy to Fly.io

This bot is designed to run on Fly.io with **Neon PostgreSQL** as the database. Neon provides a generous free tier with serverless Postgres that scales automatically.

### Prerequisites

1. ✅ [Fly.io account](https://fly.io/signup) (free tier available)
2. ✅ [flyctl CLI installed](https://fly.io/docs/hands-on/install-flyctl/)
3. ✅ [Neon account](https://neon.tech) for PostgreSQL database (free tier: 0.5 GB storage)
4. ✅ Discord Bot Token from [Discord Developer Portal](https://discord.com/developers/applications)
5. ✅ Discord Guild ID (enable Developer Mode → Right-click server → Copy Server ID)

### Setup Instructions

#### 1. Create Neon PostgreSQL Database (Recommended)

**Why Neon?**
- Free tier with 0.5 GB storage
- Serverless - scales automatically
- Built-in connection pooling
- Perfect for Discord bots

**Steps:**
1. Go to [Neon Console](https://console.neon.tech/)
2. Create a new project (choose a region close to your Fly.io region)
3. Copy the **connection string** (format: `postgresql://user:pass@host.neon.tech/dbname?sslmode=require`)

#### 2. Deploy to Fly.io

```bash
# Navigate to project directory
cd discord-bot

# Login to Fly.io
fly auth login

# Launch the app (creates the app on Fly.io)
fly launch --no-deploy

# Set required secrets
fly secrets set DISCORD_BOT_TOKEN="your-discord-bot-token"
fly secrets set DISCORD_GUILD_ID="your-discord-guild-id"
fly secrets set DATABASE_URL="postgresql://user:pass@host.neon.tech/dbname?sslmode=require"

# Deploy the bot
fly deploy
```

#### 3. Verify Deployment

```bash
# Check if the bot is running
fly status

# View logs (should show successful startup)
fly logs
```

**What success looks like:**
```
✓ Database connected successfully
✓ Database migrations completed
✓ Discord bot logged in as: YourBotName
✓ Registered guild command: ping
✓ Registered guild command: create-list
✓ Registered guild command: close-list
✓ Registered guild command: add
✓ Discord bot is now running
```

**Test in Discord:**
- Go to your Discord server
- Type `/` in any channel
- You should see the bot's slash commands appear
- Try `/ping` to verify the bot responds

### Environment Variables

**Required Secrets** (set via `fly secrets set`):

- `DISCORD_BOT_TOKEN` - Your Discord bot token from the Developer Portal
- `DISCORD_GUILD_ID` - Your Discord server ID (Right-click server → Copy Server ID)
  - **Note:** This can be empty (`""`) for global commands, but guild-specific commands appear instantly
- `DATABASE_URL` - Neon PostgreSQL connection string (includes username, password, host, and database name)

**Pre-configured in fly.toml:**
- `TIBIA_API_URL` - Points to the Miracle74 API at `https://miracle74-api.fly.dev`
- `DB_SSLMODE` - Set to `require` for production database connections

---

## Architecture

```
Discord Server
    ↓
Discord Bot (Fly.io)
    ↓
    ├─→ Neon PostgreSQL (Database)
    └─→ Miracle74 API (Character Data)
```

---

## Troubleshooting

### Bot not responding to commands
- Verify bot is online in Discord (green status)
- Check logs: `fly logs`
- Ensure `DISCORD_GUILD_ID` is set correctly
- Verify bot has correct permissions in your server

### Database connection errors
- Check `DATABASE_URL` is correct
- Ensure Neon database is active (not suspended)
- Verify SSL mode is set to `require`

### Commands not appearing
- If `DISCORD_GUILD_ID` is empty, global commands take up to 1 hour
- If set, commands appear instantly in that specific server
- Try kicking and re-inviting the bot

---

## Local Development

### Requirements

- Go 1.24+
- Docker (for local PostgreSQL) OR Neon account
- `mise` (optional)

### Run Locally

**Option 1: Local PostgreSQL with Docker**
```bash
git clone https://github.com/ethaan/discord-bot.git
cd discord-bot

# Copy environment variables
cp .env.example .env
# Edit .env with your values

# Start PostgreSQL
docker-compose up -d

# Run the bot
go run cmd/bot/main.go
```

**Option 2: Use Neon for local development**
```bash
# Copy environment variables
cp .env.example .env

# Edit .env and set DATABASE_URL to your Neon connection string
# Comment out individual DB_* variables

# Run the bot
go run cmd/bot/main.go
```

**Using mise:**
```bash
mise run docker-up  # If using local PostgreSQL
mise run dev
```

---

## API Agnostic Design

This bot is designed to work with **any Tibia API** that follows the expected contract. The current production deployment uses the [Miracle74 API](https://miracle74-api.fly.dev).

**API Specification:**
See the [OpenAPI specification](https://github.com/Ethaan/miracle74-api/blob/main/openapi.yaml) for the expected API contract (`GET /characters/:name` endpoint).

Configure your API endpoint via the `TIBIA_API_URL` environment variable. As long as your API follows the OpenAPI spec, the bot will work seamlessly with it.

---

## Available Commands

- `/ping` - Test if the bot is responding
- `/create-list <name>` - Create a new hunting list
- `/close-list <id>` - Close an active hunting list
- `/add <character-name>` - Add a character to the current list

---

## Tech Stack

- **Language:** Go 1.24
- **Discord Library:** [discordgo](https://github.com/bwmarrin/discordgo)
- **Database:** PostgreSQL (via [GORM](https://gorm.io/))
- **Hosting:** Fly.io (256MB RAM, shared CPU)
- **Database Hosting:** Neon (Serverless PostgreSQL)
- **API:** Miracle74 API (Tibia character data)