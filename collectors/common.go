package collectors

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Jleagle/steam-go/steamapi"
)

type SteamCache struct {
	profile     steamapi.PlayerSummary
	games       steamapi.OwnedGames
	timestamp   time.Time
}

type SteamData struct {
	steamAPIKey string
	steamIDs    []int64
	client      steamapi.Client
	_cache       map[int64]SteamCache
	get         func(int64)(SteamCache, error)
}

var (
  steamData   SteamData
)

func GetSteamData(steamID int64) (SteamCache, error) {
	if data, ok := steamData._cache[steamID]; ok && time.Since(data.timestamp) < 24*time.Hour {
		fmt.Printf("[common] Cache still valid for %d minutes, SteamID: %d\n", int(math.Round(24*time.Hour.Minutes() - time.Since(data.timestamp).Minutes())), steamID)
		return data, nil
	}

	// Retrieve the user's profile summary
	profile, err := steamData.client.GetPlayer(steamID)
	if err != nil {
		fmt.Printf("Error retrieving player summary for SteamID %d: %s\n", steamID, err)
		return SteamCache{}, err
	}

	// Retrieve the list of owned games for the specified user
	games, err := steamData.client.GetOwnedGames(steamID)
	if err != nil {
		fmt.Printf("Error retrieving owned games for SteamID %d: %s\n", steamID, err)
		return SteamCache{}, err
	}

	steamData._cache = make(map[int64]SteamCache)
	steamData._cache[steamID] = SteamCache{ profile: profile, games: games, timestamp: time.Now() }
	return steamData._cache[steamID], nil
}

func NewSteamData(steamAPIKey string, steamIDs string) SteamData {
	steamData.steamAPIKey = steamAPIKey

	// Split the comma-separated SteamIDs into a slice
	steamIDSlice := strings.Split(steamIDs, ",")
	for _, steamID := range steamIDSlice {
		// Parse the SteamID string to an int64
		steamIDInt, err := strconv.ParseInt(strings.TrimSpace(steamID), 10, 64)
		if err != nil {
			fmt.Printf("Error parsing SteamID %s: %s\n", steamID, err)
			continue
		}
		steamData.steamIDs = append(steamData.steamIDs, steamIDInt)
	}

	// Create a new client with your Steam API key
	steamData.client = *steamapi.NewClient()
	steamData.client.SetKey(steamData.steamAPIKey)

	steamData.get = GetSteamData

	return steamData
}
