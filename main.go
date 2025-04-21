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
	"strconv"
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
	Nickname  string           `json:"nickname"`
	PlayerID  string           `json:"player_id"`
	Games     map[string]Games `json:"games"`
	Avatar    string           `json:"avatar"`
	FaceitURL string           `json:"faceit_url"`
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

		client := &http.Client{}

		if err := acknowledgeInteraction(dg, i); err != nil {
			return
		}

		steamURL := strings.ToLower(i.ApplicationCommandData().Options[0].StringValue())

		if !strings.Contains(steamURL, "https://steamcommunity.com/profiles/") && !strings.Contains(steamURL, "https://steamcommunity.com/id/") {
			return // respond with wrong syntax
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
		url := fmt.Sprintf("https://open.faceit.com/data/v4/players?game_player_id=%v&game=cs2", steam64ID)

		// Make a GET request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}

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
		_ = json.Unmarshal(body, &player)

		// Extract relevant data
		faceitEloCS2 := player.Games["cs2"].FaceitElo
		faceitName := player.Games["cs2"].GamePlayerName
		faceitRegion := player.Games["cs2"].Region
		faceitSkillCS2 := player.Games["cs2"].SkillLevel
		faceitAvatar := player.Avatar
		faceitURL := strings.Replace(player.FaceitURL, "{lang}", "en", 1) // Replace {lang} with 'en'
		faceitEloCSGO := player.Games["csgo"].FaceitElo
		faceitSkillCSGO := player.Games["csgo"].SkillLevel

		embedColor := findEmbedColor(faceitSkillCS2)

		var embed *discordgo.MessageEmbed

		// FACEIT logic
		if faceitName == "" && faceitURL == "" {
			embed = &discordgo.MessageEmbed{
				Title: "Player has no FACEIT account. He is probably cheating!!!!!",
				Color: 0xff0000,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "\u200B",
						Value:  "\u200B",
						Inline: false,
					},
				},
			}
		} else {
			embed = &discordgo.MessageEmbed{
				Title: "Player Information",
				Color: embedColor,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "\u200B",
						Value:  "**FACEIT CS2 Stats:**",
						Inline: false,
					},
					{
						Name:   "Player Name",
						Value:  faceitName,
						Inline: true,
					},
					{
						Name:   "Faceit Elo",
						Value:  fmt.Sprintf("%d", faceitEloCS2),
						Inline: true,
					},
					{
						Name:   "Skill Level",
						Value:  strconv.Itoa(faceitSkillCS2),
						Inline: true,
					},
					{
						Name:   "Region",
						Value:  faceitRegion,
						Inline: true,
					},
					{
						Name:   "\u200B",
						Value:  "**FACEIT CS:GO Stats:**",
						Inline: false,
					},
					{
						Name:   "Faceit Elo",
						Value:  fmt.Sprintf("%d", faceitEloCSGO),
						Inline: true,
					},
					{
						Name:   "Skill Level",
						Value:  strconv.Itoa(faceitSkillCSGO),
						Inline: true,
					},
				},
				Thumbnail: &discordgo.MessageEmbedThumbnail{
					URL: faceitAvatar,
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Data retrieved from Steam and FACEIT API",
				},
			}
		}

		// Always add Steam info
		last2, total := steamStats(steam64ID)
		if total != 0 {
			steamFields := []*discordgo.MessageEmbedField{
				{
					Name:   "\u200B",
					Value:  "**CS2 Stats:**",
					Inline: false,
				},
				{
					Name:   "Total Playtime",
					Value:  fmt.Sprintf("%d hours", total/60),
					Inline: true,
				},
				{
					Name:   "Last two weeks",
					Value:  fmt.Sprintf("%d hours", last2/60),
					Inline: true,
				},
			}
			embed.Fields = append(embed.Fields, steamFields...)
		} else {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "**Private Steam Profile**",
				Value:  "\u200B",
				Inline: false,
			})
		}

		// Make the thumbnail clickable by using the image URL
		embed.URL = faceitURL

		// Send the embed response
		if err := sendEmbedResponse(s, i, embed); err != nil {
			log.Printf("Error sending detailed response: %v", err)
		}
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

func acknowledgeInteraction(dg *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := dg.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
	}
	return err
}

func sendEmbedResponse(s *discordgo.Session, i *discordgo.InteractionCreate, embed *discordgo.MessageEmbed) error {
	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("Error sending follow-up embed message: %v", err)
	}
	return err
}

// Helper function to interpolate between two colors
func interpolateColor(color1, color2 int, t float64) int {
	r1 := (color1 >> 16) & 0xFF
	g1 := (color1 >> 8) & 0xFF
	b1 := color1 & 0xFF

	r2 := (color2 >> 16) & 0xFF
	g2 := (color2 >> 8) & 0xFF
	b2 := color2 & 0xFF

	r := int(float64(r1)*(1-t) + float64(r2)*t)
	g := int(float64(g1)*(1-t) + float64(g2)*t)
	b := int(float64(b1)*(1-t) + float64(b2)*t)

	return (r << 16) | (g << 8) | b
}

// Function to determine the color based on skill level as an int
func findEmbedColor(skillLevel int) int {
	// Default color (blue)
	defaultColor := 0x3498db

	if skillLevel >= 1 && skillLevel <= 5 {
		// Green to yellow (1 to 5)
		green := 0x00FF00
		yellow := 0xFFFF00
		t := float64(skillLevel-1) / 4.0
		return interpolateColor(green, yellow, t)
	} else if skillLevel > 5 && skillLevel <= 10 {
		// Yellow to red (6 to 10)
		yellow := 0xFFFF00
		red := 0xFF0000
		t := float64(skillLevel-5) / 5.0
		return interpolateColor(yellow, red, t)
	}

	// Return default color as a fallback
	return defaultColor
}

type Game struct {
	AppID           int `json:"appid"`
	Playtime2Weeks  int `json:"playtime_2weeks"`
	PlaytimeForever int `json:"playtime_forever"`
}

type Response struct {
	Response struct {
		Games []Game `json:"games"`
	} `json:"response"`
}

func steamStats(steamID string) (last2weeks, totalplaytime int) {

	urlSteamStats := fmt.Sprintf("http://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=%v&steamid=%v&format=json", os.Getenv("STEAM_API"), steamID)

	resp, err := http.Get(urlSteamStats)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return 0, 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return 0, 0
	}
	var data Response
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Printf("Error parsing JSON: %v", err)
		return 0, 0
	}

	for _, game := range data.Response.Games {
		if game.AppID == 730 {
			return game.Playtime2Weeks, game.PlaytimeForever
		}
	}
	return 0, 0
}
