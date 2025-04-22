# 🎮 FACEIT Finder Discord Bot

A Discord bot that allows users to fetch FACEIT statistics for a player using their **Steam profile URL**. The bot integrates with the **Steam API** and **FACEIT Open Data API** to collect relevant player information like Elo, skill level, region, and recent CS2 match results.

---

## ✨ Features

- 🔍 Resolves Steam vanity URLs to 64-bit IDs
- 🧠 Queries FACEIT player profiles using CS2 Steam IDs
- 📊 Shows CS2 and CS:GO Elo, skill level, and win/loss ratio
- 🏳️ Detects country, friend bans, and verification status
- 🎨 Sends rich embedded messages in Discord
- 🧪 Works in test guilds or globally
- 🔧 CLI flags for dev flexibility

---

## 🚀 Getting Started

### Prerequisites

- [Go](https://go.dev/) 1.18 or later
- A registered Discord bot token
- Valid **FACEIT API key** and **Steam API key**
- `.env` file containing:
  ```
  DISCORD_TOKEN=your_discord_bot_token
  STEAM_API=your_steam_api_key
  FACEIT_API=your_faceit_api_key
  ```

### Installation

```bash
git clone https://github.com/yourusername/faceit-finder-bot.git
cd faceit-finder-bot
go build -o faceit-bot
./faceit-bot
```

### CLI Flags

| Flag         | Description                                              |
|--------------|----------------------------------------------------------|
| `--guild`    | Register commands only in the given guild (for testing)  |
| `--rmcmd`    | Remove commands after shutdown (default: true)           |

Example:
```bash
./faceit-bot --guild 1234567890 --rmcmd=false
```

---

## ✅ Usage

In Discord, use the slash command:

```
/faceit steam_url: https://steamcommunity.com/id/exampleuser
```

The bot will respond with an embedded message containing FACEIT stats for the user if found.

---

## 🧠 Internals

- Discord slash commands powered by `github.com/bwmarrin/discordgo`
- Steam vanity URL resolution via Steam API endpoint `ISteamUser.ResolveVanityURL`
- FACEIT data fetched from FACEIT API endpoints:
  - `/data/v4/players`
  - `/data/v4/players/{id}/games/cs2/stats`
  - `/data/v4/players/{friend_id}/bans`

---

## 🛠️ Development

### Lint and run

```bash
go fmt ./...
go run .
```