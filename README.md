# Discord Bot for Tibia

A Discord bot built with Go that acts as middleware between Discord and various Tibia APIs. The bot runs as a persistent process and uses Discord's API (via discordgo) to provide real-time notifications and commands.

## API Agnostic Design

This bot is designed to work with **any Tibia API** that follows the expected contract. Configure your API endpoint via the `TIBIA_API_URL` environment variable.

**API Specification:**
See the [OpenAPI specification](https://github.com/Ethaan/miracle74-api/blob/main/openapi.yaml) for the expected API contract (`GET /characters/:name` endpoint).

As long as your API follows this spec, the bot will work seamlessly with it.

# Bot Install Link

Below its the permissions int for the Bot to function.

https://discord.com/oauth2/authorize?client_id=1451202384226156752&permissions=268684304&integration_type=0&scope=applications.commands+bot