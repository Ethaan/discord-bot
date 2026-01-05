# Discord Bot for Tibia

Discord bot for managing hunting lists and character notifications.

**Add to Discord:** https://discord.com/oauth2/authorize?client_id=1451202384226156752&permissions=268684304&integration_type=0&scope=applications.commands+bot

---

## Powered By

This project uses [Neon](https://neon.com) as its serverless Postgres database provider. Neon enables us to scale our database seamlessly with features like instant branching, autoscaling, and point-in-time recovery. Learn more about [Neon's serverless Postgres platform](https://neon.com/docs/introduction).

---

## Commands

- `/ping` - Test bot response
- `/create-list <name>` - Create hunting list
- `/close-list <id>` - Close list
- `/add <character>` - Add character to list
- `/list` - View all characters

---

## Development

```bash
git clone https://github.com/ethaan/discord-bot.git
cd discord-bot

cp .env.example .env
# Edit .env with your Discord token and Neon database credentials
# Get your DATABASE_URL from: https://console.neon.tech

docker-compose up -d
go run cmd/bot/main.go
```

Or with `mise`:
```bash
mise run docker-up
mise run dev
```

---

## Deploy

```bash
fly launch --no-deploy

fly secrets set DISCORD_BOT_TOKEN="..."
fly secrets set DISCORD_GUILD_ID="..."
fly secrets set DATABASE_URL="postgresql://..."  # Your Neon database connection string

fly deploy
```
