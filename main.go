package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	BotToken    string `json:"BOT_TOKEN"`
	TenorAPIKey string `json:"TENOR_API_KEY"`
	GuildID     string `json:"DISCORD_GUILD_ID"`
}

var (
	cfg          Config
	TenorBaseURL = "https://tenor.googleapis.com/v2/search"
)

func main() {
	var err error
	cfg, err = loadConfig("config.json")
	if err != nil {
		log.Fatalf("Błąd wczytywania config.json: %v", err)
	}

	if cfg.BotToken == "" || cfg.TenorAPIKey == "" || cfg.GuildID == "" {
		log.Fatal("Brakuje BOT_TOKEN, TENOR_API_KEY lub DISCORD_GUILD_ID w pliku config.json")
	}

	dg, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		log.Fatalf("Błąd tworzenia sesji: %v", err)
	}

	dg.AddHandler(handleInteraction)

	err = dg.Open()
	if err != nil {
		log.Fatalf("Błąd przy uruchamianiu sesji Discord: %v", err)
	}
	defer dg.Close()

	removeAllCommands(dg)
	registerCommand(dg)

	fmt.Println("Giffie działa! Naciśnij CTRL+C, aby zakończyć.")
	select {}
}

func loadConfig(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	return config, err
}

func removeAllCommands(s *discordgo.Session) {
	cmds, err := s.ApplicationCommands(s.State.User.ID, cfg.GuildID)
	if err != nil {
		log.Printf("Błąd pobierania komend: %v", err)
		return
	}

	for _, cmd := range cmds {
		err := s.ApplicationCommandDelete(s.State.User.ID, cfg.GuildID, cmd.ID)
		if err != nil {
			log.Printf("Błąd usuwania komendy '%s': %v", cmd.Name, err)
		}
	}
}

func registerCommand(s *discordgo.Session) {
	_, err := s.ApplicationCommandCreate(s.State.User.ID, cfg.GuildID, &discordgo.ApplicationCommand{
		Name:        "searchgif",
		Description: "Wyszukaj GIF-a z Tenor",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "keyword",
				Description: "Słowo kluczowe do wyszukania GIF-a",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Błąd przy rejestracji komendy: %v", err)
	}
}

func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "searchgif" {
		return
	}

	keyword := i.ApplicationCommandData().Options[0].StringValue()
	log.Printf("Użytkownik %s wyszukuje GIF-a: %s", i.Member.User.Username, keyword)

	gifURL, err := fetchGIFfromTenor(keyword)
	if err != nil {
		gifURL = "Nie udało się znaleźć GIF-a. 😢"
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: gifURL,
		},
	})
}

func fetchGIFfromTenor(query string) (string, error) {
	params := url.Values{}
	params.Add("q", query)
	params.Add("key", cfg.TenorAPIKey)
	params.Add("limit", "1")
	params.Add("media_filter", "minimal")

	url := fmt.Sprintf("%s?%s", TenorBaseURL, params.Encode())
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			MediaFormats struct {
				Gif struct {
					URL string `json:"url"`
				} `json:"gif"`
			} `json:"media_formats"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Results) == 0 {
		return "", fmt.Errorf("brak rezultatów")
	}
	return result.Results[0].MediaFormats.Gif.URL, nil
}
