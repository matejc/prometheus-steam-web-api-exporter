package collectors

import (
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	gamesAchievementsAchieved = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "steam_achievements_achieved",
		Help: "Amount of achievements on Steam by the specified user.",
	}, []string{"steam_profile_name", "steam_id", "name", "app_id"})
	gamesAchievementsRemaining = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "steam_achievements_remaining",
		Help: "Amount of achievements on Steam by the specified user.",
	}, []string{"steam_profile_name", "steam_id", "name", "app_id"})
)

type AchievementsCache struct {
	achieved     float64
	remaining    float64
	timestamp    time.Time
	interval     time.Duration
}

type AchievementsCollector struct {
	steamData SteamData
	cache     map[string]AchievementsCache
}

func NewAchievementsCollector(steamData SteamData) *AchievementsCollector {
	return &AchievementsCollector{
		steamData: steamData,
		cache:     make(map[string]AchievementsCache),
	}
}

func (c *AchievementsCollector) Describe(descs chan<- *prometheus.Desc) {
	gamesAchievementsAchieved.Describe(descs)
	gamesAchievementsRemaining.Describe(descs)
}

func (c *AchievementsCollector) Collect(metrics chan<- prometheus.Metric) {
	for _, steamID := range c.steamData.steamIDs {
		// Retrieve the user's data summary
		cache, err := c.steamData.get(steamID)
		if err != nil {
			fmt.Printf("Error retrieving data for SteamID %d: %s\n", steamID, err)
			continue
		}

		// Set achievements for each game as a Prometheus gauge
		for _, game := range cache.games.Games {
			// Check if the price is in the cache
			cacheKey := fmt.Sprintf("%d-%d", steamID, game.AppID)
			existingAchievements, found := c.cache[cacheKey]

			// If the achievemnts are in the cache and less than a day old, use the cached value
			if time.Since(existingAchievements.timestamp) < existingAchievements.interval {
				fmt.Printf("[achievements] Cache still valid for %d minutes, cacheKey: %s\n", int(math.Round(existingAchievements.interval.Minutes() - time.Since(existingAchievements.timestamp).Minutes())), cacheKey)
				if found {
					gamesAchievementsAchieved.With(prometheus.Labels{
						"steam_profile_name": cache.profile.PersonaName,
						"steam_id": fmt.Sprintf("%d", steamID),
						"name": game.Name,
						"app_id": fmt.Sprintf("%d", game.AppID),
					}).Set(existingAchievements.achieved)
					gamesAchievementsRemaining.With(prometheus.Labels{
						"steam_profile_name": cache.profile.PersonaName,
						"steam_id": fmt.Sprintf("%d", steamID),
						"name": game.Name,
						"app_id": fmt.Sprintf("%d", game.AppID),
					}).Set(existingAchievements.remaining)
					continue
				}
			}

			playerAchievements, err := c.steamData.client.GetPlayerAchievements(uint64(steamID), uint32(game.AppID))
			if err != nil {
				fmt.Printf("WARN: Error retrieving app details for app ID %d: %s\n", game.AppID, err)
				c.cache[cacheKey] = AchievementsCache{achieved: 0, remaining: 0, timestamp: time.Now(), interval: time.Hour}
				continue
			}

			if playerAchievements.Success {
				var achieved float64 = 0
				var remaining float64 = 0
				for _, playerAchievement := range playerAchievements.Achievements {
					if bool(playerAchievement.Achieved) {
						achieved++;
					} else {
						remaining++;
					}
					gamesAchievementsAchieved.With(prometheus.Labels{
						"steam_profile_name": cache.profile.PersonaName,
						"steam_id": fmt.Sprintf("%d", steamID),
						"name": game.Name,
						"app_id": fmt.Sprintf("%d", game.AppID),
					}).Set(achieved)
					gamesAchievementsRemaining.With(prometheus.Labels{
						"steam_profile_name": cache.profile.PersonaName,
						"steam_id": fmt.Sprintf("%d", steamID),
						"name": game.Name,
						"app_id": fmt.Sprintf("%d", game.AppID),
					}).Set(remaining)
				}
				c.cache[cacheKey] = AchievementsCache{achieved: achieved, remaining: remaining, timestamp: time.Now(), interval: 24*time.Hour}
			} else {
				fmt.Printf("WARN: Couldn't find achievements information for %d: %s\n", game.AppID, playerAchievements.Error)
				c.cache[cacheKey] = AchievementsCache{achieved: 0, remaining: 0, timestamp: time.Now(), interval: 24*time.Hour}
			}
		}
	}

	gamesAchievementsAchieved.Collect(metrics)
	gamesAchievementsRemaining.Collect(metrics)
}
