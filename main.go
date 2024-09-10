package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// struct for API responses
type ResolveVanityURLResponse struct {
	Response struct {
		SteamID string `json:"steamid"`
		Success int    `json:"success"`
	} `json:"response"`
}

type PlayerInfo struct {
	Nickname string           `json:"nickname"`
	PlayerID string           `json:"player_id"`
	Games    map[string]Games `json:"games"`
}
type Games struct {
	FaceitElo      int    `json:"faceit_elo"`
	GamePlayerID   string `json:"game_player_id"`
	GamePlayerName string `json:"game_player_name"`
	Region         string `json:"region"`
	SkillLevel     int    `json:"skill_level"`
}

type APIResponse struct {
	Items []interface{} `json:"items"`
}

// Variables to manage command-line flags.
var GuildIDflag = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
var RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")

// Discord session
var dg *discordgo.Session

// Discord commands definition
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "faceit",
		Description: "it just like faceitfinder",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "steam_url",
				Description: "the full steam url of the person you want to look up",
				Required:    true,
			},
		},
	},
}

var commandHandlers = map[string]func(dg *discordgo.Session, i *discordgo.InteractionCreate){
	"faceit": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		steamURL := strings.ToLower(i.ApplicationCommandData().Options[0].StringValue())
		log.Print("called faceit")

		if !strings.Contains(steamURL, "https://steamcommunity.com/profiles/") && !strings.Contains(steamURL, "https://steamcommunity.com/id/") {
			return // repond with wrong syntax
		}

		steamSplit := strings.Split(steamURL, "/")

		steam64ID := steamSplit[4]

		if steamSplit[3] == "id" {

			steamAPIKEY := os.Getenv("STEAM_API")
			steamAPIUrl := fmt.Sprintf("https://api.steampowered.com/ISteamUser/ResolveVanityURL/v1/?key=%v&vanityurl=%v", steamAPIKEY, steamSplit[4])

			req, err := http.NewRequest("GET", steamAPIUrl, nil)
			if err != nil {
				return // error
			}

			client := &http.Client{}
			resp, err := client.Do(req)

			if err != nil {
				log.Printf("Error making request: %v. Retrying...", err)

			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Error reading response: %v. Retrying...", err)
			}

			var resolveVanityURLResponse ResolveVanityURLResponse
			err = json.Unmarshal(body, &resolveVanityURLResponse)
			if err != nil {
				fmt.Printf("Received JSON: %s\n", body)
				log.Printf("Error parsing JSON: %v. Retrying...", err)
			}

			steam64ID = resolveVanityURLResponse.Response.SteamID
		}
		FACEITAPI := os.Getenv("FACEIT_API")
		// Create an HTTP client
		client := &http.Client{}
		url := "https://open.faceit.com/data/v4/players"
		url = url + FACEITAPI
		// Get the nickname query parameter from the request
		nicknameQuery := steam64ID

		// Append the nickname query parameter to the URL
		//reqURL := "https://open.faceit.com/data/v4/players=" + nicknameQuery
		reqURL := url + nicknameQuery

		// Make a GET request
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}
		log.Println(req)

		// Add headers if needed (e.g., authentication)
		req.Header.Add("Authorization", "Bearer "+FACEITAPI)

		// Send the request
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error sending request:", err)
			return
		}
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading response:", err)
			return
		}

		// Parse JSON response
		var player PlayerInfo
		if err := json.Unmarshal(body, &player); err != nil {
			log.Println("Failed to unmarshal JSON:", err)
			return
		}
		log.Println(player)
		/*
			playerFormat := PlayerInfo{
				Nickname: player.Nickname,
				PlayerID: player.PlayerID,
				Games: map[string]Games{
					"cs2": {
						FaceitElo:      player.Games["cs2"].FaceitElo,
						GamePlayerID:   player.Games["cs2"].GamePlayerID,
						GamePlayerName: player.Games["cs2"].GamePlayerName,
						Region:         player.Games["cs2"].Region,
						SkillLevel:     player.Games["cs2"].SkillLevel,
					},
				},
			}

			responsePlayer, err := json.Marshal(playerFormat)
			if err != nil {
				http.Error(w, "Failed to marshal JSON response", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(responsePlayer)
			if err != nil {
				log.Println("Failed to write JSON response:", err)
				return
			}

			// Print the player's nickname
			fmt.Println("Player's nickname:", player.Nickname) */
	},
}

func init() {
	flag.Parse()
	envErr := godotenv.Load()
	if envErr != nil {
		log.Fatalf("Error loading .env file: %v", envErr)
	}

	BotToken := os.Getenv("DISCORD_API")

	var err error
	dg, err = discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	err := dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, *GuildIDflag, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer dg.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	if *RemoveCommands {
		log.Println("Removing commands...")

		registeredCommands, err := dg.ApplicationCommands(dg.State.User.ID, *GuildIDflag)
		if err != nil {
			log.Fatalf("Could not fetch registered commands: %v", err)
		}

		for _, v := range registeredCommands {
			err := dg.ApplicationCommandDelete(dg.State.User.ID, *GuildIDflag, v.ID)
			if err != nil {
				log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
			}
		}
	}

}
