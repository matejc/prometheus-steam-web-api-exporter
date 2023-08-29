package collectors

import (
	"fmt"
	"math"
	"time"

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
    steamData    SteamData
    CacheInitial map[string]cachedPriceInitial
    CacheFinal   map[string]cachedPriceFinal
}

type cachedPriceInitial struct {
    priceInitial float64
    timestamp    time.Time
    interval     time.Duration
}

type cachedPriceFinal struct {
    priceFinal   float64
    timestamp    time.Time
    interval     time.Duration
}

func (c *PriceCollector) Describe(ch chan<- *prometheus.Desc) {
    gamePricesInitial.Describe(ch)
    gamePricesFinal.Describe(ch)
}

func NewPriceCollector(steamData SteamData) *PriceCollector {
    return &PriceCollector{
	steamData:    steamData,
        CacheInitial: make(map[string]cachedPriceInitial),
        CacheFinal:   make(map[string]cachedPriceFinal),
    }
}

func (c *PriceCollector) Collect(metrics chan<- prometheus.Metric) {
	for _, steamID := range c.steamData.steamIDs {
		// Retrieve the user's data summary
		cache, err := c.steamData.get(steamID)
		if err != nil {
			fmt.Printf("Error retrieving data for SteamID %d: %s\n", steamID, err)
			continue
		}

		// Set the price for each game as a Prometheus gauge
		for _, game := range cache.games.Games {
			// Check if the price is in the cache
			cacheKey := fmt.Sprintf("%d-%d", steamID, game.AppID)
			existingPriceInitial, found := c.CacheInitial[cacheKey]
			existingPriceFinal, found := c.CacheFinal[cacheKey]

			// If the price is in the cache and less than a day old, use the cached value
			if time.Since(existingPriceInitial.timestamp) < existingPriceInitial.interval && time.Since(existingPriceFinal.timestamp) < existingPriceFinal.interval {
				fmt.Printf("[price] Cache still valid for %d minutes, cacheKey: %s\n", int(math.Round(existingPriceFinal.interval.Minutes() - time.Since(existingPriceFinal.timestamp).Minutes())), cacheKey)
				if found {
					gamePricesInitial.With(prometheus.Labels{"steam_profile_name": cache.profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamID), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(existingPriceInitial.priceInitial)
					gamePricesFinal.With(prometheus.Labels{"steam_profile_name": cache.profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamID), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(existingPriceFinal.priceFinal)
					continue
				}
			}

			gameDetails, err := c.steamData.client.GetAppDetails(uint(game.AppID), "ProductCCEU", "LanguageEnglish", []string{"price_overview"})
			if err != nil {
				fmt.Printf("WARN: Error retrieving app details for app ID %d: %s\n", game.AppID, err)
				c.CacheInitial[cacheKey] = cachedPriceInitial{priceInitial: 0, timestamp: time.Now(), interval: time.Hour}
				c.CacheFinal[cacheKey] = cachedPriceFinal{priceFinal: 0, timestamp: time.Now(), interval: time.Hour}
				continue
			}

			if gameDetails.Data != nil && gameDetails.Data.PriceOverview != nil && gameDetails.Data.PriceOverview.Initial > 0 {
				priceInitial := float64(gameDetails.Data.PriceOverview.Initial)

				gamePricesInitial.With(prometheus.Labels{"steam_profile_name": cache.profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamID), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(priceInitial)
				c.CacheInitial[cacheKey] = cachedPriceInitial{priceInitial: priceInitial, timestamp: time.Now(), interval: 24*time.Hour}
			} else {
				fmt.Printf("INFO: Couldn't find initial price information in the response or values is 0. Setting the initial price for app ID %d to zero.\n", game.AppID)
				gamePricesInitial.With(prometheus.Labels{"steam_profile_name": cache.profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamID), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(0)
				c.CacheInitial[cacheKey] = cachedPriceInitial{priceInitial: 0, timestamp: time.Now(), interval: 24*time.Hour}
			}

			if gameDetails.Data != nil && gameDetails.Data.PriceOverview != nil && gameDetails.Data.PriceOverview.Final > 0 {
				priceFinal := float64(gameDetails.Data.PriceOverview.Final)

				gamePricesFinal.With(prometheus.Labels{"steam_profile_name": cache.profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamID), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(priceFinal)
				c.CacheFinal[cacheKey] = cachedPriceFinal{priceFinal: priceFinal, timestamp: time.Now(), interval: 24*time.Hour}
			} else {
				fmt.Printf("INFO: Couldn't find final price information in the response or values is 0. Setting the final price for app ID %d to zero.\n", game.AppID)
				gamePricesFinal.With(prometheus.Labels{"steam_profile_name": cache.profile.PersonaName, "steam_id": fmt.Sprintf("%d", steamID), "name": game.Name, "app_id": fmt.Sprintf("%d", game.AppID)}).Set(0)
				c.CacheFinal[cacheKey] = cachedPriceFinal{priceFinal: 0, timestamp: time.Now(), interval: 24*time.Hour}
			}
		}
	}

	gamePricesInitial.Collect(metrics)
	gamePricesFinal.Collect(metrics)
}
