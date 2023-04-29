package collectors

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Jleagle/steam-go/steamapi"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	gamesPlayed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "steam_playtime",
		Help: "Amount of time played on Steam by the specified user.",
	}, []string{"steam_profile_name", "steam_id", "name", "app_id"})
)

type PlaytimeCollector struct {
	SteamAPIKey string
	SteamIDs    string
}

func NewPlaytimeCollector(steamAPIKey, steamIDs string) *PlaytimeCollector {
	return &PlaytimeCollector{
		SteamAPIKey: steamAPIKey,
		SteamIDs:    steamIDs,
	}
}

func (c *PlaytimeCollector) Describe(descs chan<- *prometheus.Desc) {
	gamesPlayed.Describe(descs)
}

func (c *PlaytimeCollector) Collect(metrics chan<- prometheus.Metric) {
	// Create a new client with your Steam API key
	client := steamapi.NewClient()
	client.SetKey(c.SteamAPIKey)

	// Split the comma-separated SteamIDs into a slice
	steamIDSlice := strings.Split(c.SteamIDs, ",")

	for _, steamID := range steamIDSlice {
		// Parse the SteamID string to an int64
		steamIDInt, err := strconv.ParseInt(strings.TrimSpace(steamID), 10, 64)
		if err != nil {
			fmt.Printf("Error parsing SteamID %s: %s", steamID, err)
			continue
		}

		// Retrieve the user's profile summary
		profile, err := client.GetPlayer(steamIDInt)
		if err != nil {
			fmt.Printf("Error retrieving player summary for SteamID %d: %s", steamIDInt, err)
			continue
		}

		// Retrieve the list of owned games for the specified user
		games, err := client.GetOwnedGames(steamIDInt)
		if err != nil {
			fmt.Printf("Error retrieving owned games for SteamID %d: %s", steamIDInt, err)
			continue
		}

		// Set the playtime for each game as a Prometheus gauge
		for _, game := range games.Games {
			gamesPlayed.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(float64(game.PlaytimeForever))
		}
	}

	gamesPlayed.Collect(metrics)
}

func RegisterPlaytimeCollector() {
	steamAPIKey := os.Getenv("STEAM_API_KEY")
	if steamAPIKey == "" {
		fmt.Println("Steam Web API key not provided.")
		os.Exit(1)
	}

	steamIDs := os.Getenv("STEAM_IDS")
	if steamIDs == "" {
		fmt.Println("SteamIDs not provided.")
		os.Exit(1)
	}

	collector := NewPlaytimeCollector(steamAPIKey, steamIDs)
	prometheus.MustRegister(collector)
}