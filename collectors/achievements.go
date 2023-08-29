package collectors

import (
	"fmt"
	"math"
	"time"

	"github.com/Jleagle/unmarshal-go"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	gamesAchievements = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "steam_achievements_unlocktime",
		Help: "Amount of time played on Steam by the specified user.",
	}, []string{"steam_profile_name", "steam_id", "game_name", "app_id", "name", "apiname", "achieved", "description"})
)

type Achievement struct {
	apiname     string
	name        string
	achieved    string
	unlocktime  float64
	description string
}

type AchievementsCache struct {
	achievements []Achievement
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
	gamesAchievements.Describe(descs)
}

func boolToString(b unmarshal.Bool) string {
	if b {
		return "1"
	}
	return "0"
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
					for _, playerAchievement := range existingAchievements.achievements {
						gamesAchievements.With(prometheus.Labels{
							"steam_profile_name": cache.profile.PersonaName,
							"steam_id": fmt.Sprintf("%d", steamID),
							"game_name": game.Name,
							"app_id": fmt.Sprintf("%d", game.AppID),
							"name": playerAchievement.name,
							"apiname": playerAchievement.apiname,
							"achieved": playerAchievement.achieved,
							"description": playerAchievement.description,
						}).Set(playerAchievement.unlocktime)
					}
					continue
				}
			}

			playerAchievements, err := c.steamData.client.GetPlayerAchievements(uint64(steamID), uint32(game.AppID))
			if err != nil {
				fmt.Printf("WARN: Error retrieving app details for app ID %d: %s\n", game.AppID, err)
				c.cache[cacheKey] = AchievementsCache{achievements: []Achievement{}, timestamp: time.Now(), interval: time.Hour}
				continue
			}

			if playerAchievements.Success {
				var achievements []Achievement
				for _, playerAchievement := range playerAchievements.Achievements {
					achievements = append(achievements, Achievement{
						apiname: playerAchievement.APIName,
						name: playerAchievement.Name,
						unlocktime: float64(playerAchievement.UnlockTime),
						achieved: boolToString(playerAchievement.Achieved),
						description: playerAchievement.Description,
					})
					gamesAchievements.With(prometheus.Labels{
						"steam_profile_name": cache.profile.PersonaName,
						"steam_id": fmt.Sprintf("%d", steamID),
						"game_name": game.Name,
						"app_id": fmt.Sprintf("%d", game.AppID),
						"name": playerAchievement.Name,
						"apiname": playerAchievement.APIName,
						"achieved": boolToString(playerAchievement.Achieved),
						"description": playerAchievement.Description,
					}).Set(float64(playerAchievement.UnlockTime))
				}
				c.cache[cacheKey] = AchievementsCache{achievements: achievements, timestamp: time.Now(), interval: 24*time.Hour}
			} else {
				fmt.Printf("WARN: Couldn't find achievements information for %d: %s\n", game.AppID, playerAchievements.Error)
				c.cache[cacheKey] = AchievementsCache{achievements: []Achievement{}, timestamp: time.Now(), interval: 24*time.Hour}
			}
		}
	}

	gamesAchievements.Collect(metrics)
}
