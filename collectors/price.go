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
    gamePricesInitial = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "steam_game_price_initial",
        Help: "Initial price of a game on Steam.",
    }, []string{"steam_profile_name", "steam_id", "name", "app_id"})
	gamePricesFinal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "steam_game_price_final",
        Help: "Final price of a game on Steam.",
    }, []string{"steam_profile_name", "steam_id", "name", "app_id"})
)

type PriceCollector struct {
    SteamAPIKey  string
    SteamIDs     string
    CacheInitial map[string]cachedPriceInitial
    CacheFinal   map[string]cachedPriceFinal
}

type cachedPriceInitial struct {
    priceInitial float64
    timestamp    time.Time
}

type cachedPriceFinal struct {
    priceFinal   float64
    timestamp    time.Time
}

func (c *PriceCollector) Describe(ch chan<- *prometheus.Desc) {
    gamePricesInitial.Describe(ch)
    gamePricesFinal.Describe(ch)
}

func NewPriceCollector(steamAPIKey, steamIDs string) *PriceCollector {
    return &PriceCollector{
        SteamAPIKey:  steamAPIKey,
        SteamIDs:     steamIDs,
        CacheInitial: make(map[string]cachedPriceInitial),
        CacheFinal:   make(map[string]cachedPriceFinal),
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
			existingPriceInitial, found := c.CacheInitial[cacheKey]
			existingPriceFinal, found := c.CacheFinal[cacheKey]

			// If the price is in the cache and less than a day old, use the cached value
			if found && time.Since(existingPriceInitial.timestamp) < 24*time.Hour && time.Since(existingPriceFinal.timestamp) < 24*time.Hour {
				gamePricesInitial.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(existingPriceInitial.priceInitial)
				gamePricesFinal.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(existingPriceFinal.priceFinal)
				continue
			}

			gameDetails, err := client.GetAppDetails(uint(game.AppID), "ProductCCEU", "LanguageEnglish", []string{"price_overview"})
			if err != nil {
				fmt.Printf("WARN: Error retrieving app details for app ID %d: %s\n", game.AppID, err)
				c.CacheInitial[cacheKey] = cachedPriceInitial{priceInitial: 0, timestamp: time.Now()}
				c.CacheFinal[cacheKey] = cachedPriceFinal{priceFinal: 0, timestamp: time.Now()}
				gamePricesInitial.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(0)
				gamePricesFinal.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(0)
				continue
			}

			if gameDetails.Data != nil && gameDetails.Data.PriceOverview != nil && gameDetails.Data.PriceOverview.Initial > 0 {
				priceInitial := float64(gameDetails.Data.PriceOverview.Initial)

				gamePricesInitial.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(priceInitial)
				c.CacheInitial[cacheKey] = cachedPriceInitial{priceInitial: priceInitial, timestamp: time.Now()}
			} else {
				fmt.Printf("INFO: Couldn't find initial price information in the response or values is 0. Setting the initial price for app ID %d to zero.\n", game.AppID)
				c.CacheInitial[cacheKey] = cachedPriceInitial{priceInitial: 0, timestamp: time.Now()}
				gamePricesInitial.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(0)
			}

			if gameDetails.Data != nil && gameDetails.Data.PriceOverview != nil && gameDetails.Data.PriceOverview.Final > 0 {
				priceFinal := float64(gameDetails.Data.PriceOverview.Final)

				gamePricesFinal.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(priceFinal)
				c.CacheFinal[cacheKey] = cachedPriceFinal{priceFinal: priceFinal, timestamp: time.Now()}
			} else {
				fmt.Printf("INFO: Couldn't find final price information in the response or values is 0. Setting the final price for app ID %d to zero.\n", game.AppID)
				c.CacheFinal[cacheKey] = cachedPriceFinal{priceFinal: 0, timestamp: time.Now()}
				gamePricesFinal.With(prometheus.Labels{"steam_profile_name": profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamIDInt), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(0)
			}
		}
	}
	gamePricesInitial.Collect(metrics)
	gamePricesFinal.Collect(metrics)
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
