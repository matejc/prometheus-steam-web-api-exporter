package collectors

import (
    "fmt"
    "os"
    "time"
    "strconv"
    "strings"

    "github.com/Jleagle/steam-go/steamapi"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    gamePrices = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "steam_game_price",
        Help: "Price of a game on Steam.",
    }, []string{"steam_profile_name", "steam_id", "name", "app_id"})
)

type PriceCollector struct {
    SteamAPIKey string
    SteamIDs    string
    Cache       map[string]cachedPrice
}

type cachedPrice struct {
    price      float64
    timestamp  time.Time
}

func (c *PriceCollector) Describe(ch chan<- *prometheus.Desc) {
    gamePrices.Describe(ch)
}

func NewPriceCollector(steamAPIKey, steamIDs string) *PriceCollector {
    return &PriceCollector{
        SteamAPIKey: steamAPIKey,
        SteamIDs:    steamIDs,
        Cache:       make(map[string]cachedPrice),
    }
}

func (c *PriceCollector) Collect(metrics chan<- prometheus.Metric) {
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

		// Set the price for each game as a Prometheus gauge
		for _, game := range games.Games {
			// Check if the price is in the cache
			cacheKey := fmt.Sprintf("%s-%d", steamID, game.AppID)
			existingPrice, found := c.Cache[cacheKey]

			// If the price is in the cache and less than a day old, use the cached value
			if found && time.Since(existingPrice.timestamp) < 24*time.Hour {
				gamePrices.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(existingPrice.price)
				continue
			} else {
				gameDetails, err := client.GetAppDetails(uint(game.AppID), "ProductCCEU", "LanguageEnglish", []string{"price_overview"})
				if err != nil {
					fmt.Printf("Error retrieving app details for app ID %d: %s\n", game.AppID, err)
					c.Cache[cacheKey] = cachedPrice{price: 0, timestamp: time.Now()}
					continue
				}

				if gameDetails.Data != nil && gameDetails.Data.PriceOverview != nil && gameDetails.Data.PriceOverview.Initial > 0 {
					price := float64(gameDetails.Data.PriceOverview.Initial)

					gamePrices.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(price)
					c.Cache[cacheKey] = cachedPrice{price: price, timestamp: time.Now()}
				} else {
                    fmt.Printf("Couldn't find pricing information in the response. Setting the price for app ID %d to zero.\n", game.AppID, err)
					c.Cache[cacheKey] = cachedPrice{price: 0, timestamp: time.Now()}
				}
			}
		}
		gamePrices.Collect(metrics)
	}
}

func RegisterPriceCollector() {
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

    collector := NewPriceCollector(steamAPIKey, steamIDs)
    prometheus.MustRegister(collector)
}