package collectors

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	gamesPlayed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "steam_playtime",
		Help: "Amount of time played on Steam by the specified user.",
	}, []string{"steam_profile_name", "steam_id", "name", "app_id"})
)

type PlaytimeCollector struct {
	steamData SteamData
}

func NewPlaytimeCollector(steamData SteamData) *PlaytimeCollector {
	return &PlaytimeCollector{
		steamData: steamData,
	}
}

func (c *PlaytimeCollector) Describe(descs chan<- *prometheus.Desc) {
	gamesPlayed.Describe(descs)
}

func (c *PlaytimeCollector) Collect(metrics chan<- prometheus.Metric) {
	for _, steamID := range c.steamData.steamIDs {
		// Retrieve the user's profile summary
		cache, err := c.steamData.get(steamID)
		if err != nil {
			fmt.Printf("Error retrieving data for SteamID %d: %s\n", steamID, err)
			continue
		}

		// Set the playtime for each game as a Prometheus gauge
		for _, game := range cache.games.Games {
			gamesPlayed.With(prometheus.Labels{
				"steam_profile_name": cache.profile.PersonaName,
				"steam_id": fmt.Sprintf("%d", steamID),
				"name": game.Name,
				"app_id": fmt.Sprintf("%d", game.AppID),
			}).Set(float64(game.PlaytimeForever))
		}
	}

	gamesPlayed.Collect(metrics)
}
