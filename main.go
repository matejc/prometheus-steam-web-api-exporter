package main

import (
    "flag"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "strings"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/Jleagle/steam-go/steamapi"
)

var (
    gamesPlayed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "steam_playtime",
        Help: "Amount of time played on Steam by the specified user.",
    }, []string{"steam_profile_name", "steam_id", "name", "app_id"})

    steamAPIKey string
    steamIDs    string
)

func init() {
    prometheus.MustRegister(gamesPlayed)
}

func main() {
    flag.StringVar(&steamAPIKey, "steam-api-key", "", "API key to use for requests to the Steam Web API.")
    flag.StringVar(&steamIDs, "steam-ids", "", "Comma-separated list of SteamIDs whose playtime should be scraped.")
    flag.Parse()

    steamAPIKeyEnv := os.Getenv("STEAM_API_KEY")
    if steamAPIKeyEnv != "" {
        steamAPIKey = steamAPIKeyEnv
    }

    steamIDsEnv := os.Getenv("STEAM_IDS")
    if steamIDsEnv != "" {
        steamIDs = steamIDsEnv
    }

    if steamAPIKey == "" {
        fmt.Println("Steam Web API key not provided.")
        os.Exit(1)
    }

    if steamIDs == "" {
        fmt.Println("SteamIDs not provided.")
        os.Exit(1)
    }

    http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        // Create a new client with your Steam API key
        client := steamapi.NewClient()
        client.SetKey(steamAPIKey)

        // Split the comma-separated SteamIDs into a slice
        steamIDSlice := strings.Split(steamIDs, ",")

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

        promhttp.Handler().ServeHTTP(w, r)
    })

    http.ListenAndServe(":6630", nil)
}