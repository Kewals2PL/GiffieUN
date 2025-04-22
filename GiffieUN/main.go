package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

var (
	BotToken     string
	TenorAPIKey  string
	GuildID      = "1303639782890541149"
	TenorBaseURL = "https://tenor.googleapis.com/v2/search"
)

func main() {
	err := godotenv.Load("config.env")
	if err != nil {
		log.Fatalf("Błąd podczas ładowania pliku config.env: %v", err)
	}

	BotToken = os.Getenv("BOT_TOKEN")
	TenorAPIKey = os.Getenv("TENOR_API_KEY")

	if BotToken == "" || TenorAPIKey == "" {
		log.Fatal("Brak BOT_TOKEN lub TENOR_API_KEY w pliku config.env")
	}

	dg, err := discordgo.New("Bot " + BotToken)
	if err != nil {
		log.Fatalf("Błąd tworzenia sesji: %v", err)
	}

	dg.AddHandler(handleInteraction)

	err = dg.Open()
	if err != nil {
		log.Fatalf("Błąd przy uruchamianiu sesji Discord: %v", err)
	}
	defer dg.Close()

	// Usuń poprzednie komendy i zarejestruj nową lokalnie
	removeAllCommands(dg)
	registerCommand(dg)

	fmt.Println("Giffie działa! Naciśnij CTRL+C, aby zakończyć.")
	select {}
}

func removeAllCommands(s *discordgo.Session) {
	cmds, err := s.ApplicationCommands(s.State.User.ID, GuildID)
	if err != nil {
		log.Printf("Błąd pobierania komend: %v", err)
		return
	}

	for _, cmd := range cmds {
		err := s.ApplicationCommandDelete(s.State.User.ID, GuildID, cmd.ID)
		if err != nil {
			log.Printf("Błąd usuwania komendy '%s': %v", cmd.Name, err)
		}
	}
}

func registerCommand(s *discordgo.Session) {
	_, err := s.ApplicationCommandCreate(s.State.User.ID, GuildID, &discordgo.ApplicationCommand{
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
	if i.ApplicationCommandData().Name != "givemeagif" {
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
	params.Add("key", TenorAPIKey)
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
		return "", fmt.Errorf("brak wyników")
	}
	return result.Results[0].MediaFormats.Gif.URL, nil
}
