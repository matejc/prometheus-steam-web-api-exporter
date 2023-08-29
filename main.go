package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"prometheus-steam-web-api-exporter/collectors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func main() {
	var (
		steamAPIKey     = flag.String("steam-api-key", "", "API key to use for requests to the Steam Web API.")
		steamIDs        = flag.String("steam-ids", "", "Comma-separated list of SteamIDs whose playtime should be scraped.")
		steamCollectors = flag.String("collectors", "playtime,price", "Comma-separated list of Steam collectors.")
		steamCollectorsSlice []string
	)
	flag.Parse()

	steamAPIKeyEnv := os.Getenv("STEAM_API_KEY")
	if *steamAPIKey == "" && steamAPIKeyEnv != "" {
		*steamAPIKey = steamAPIKeyEnv
	}

	steamIDsEnv := os.Getenv("STEAM_IDS")
	if *steamIDs == "" && steamIDsEnv != "" {
		*steamIDs = steamIDsEnv
	}

	if *steamAPIKey == "" {
		fmt.Println("Steam Web API key not provided.")
		os.Exit(1)
	}

	if *steamIDs == "" {
		fmt.Println("SteamIDs not provided.")
		os.Exit(1)
	}

	if *steamCollectors == "" {
		fmt.Println("Steam collectors are empty!")
		os.Exit(1)
	}

	for _, steamCollector := range strings.Split(*steamCollectors, ",") {
		steamCollectorsSlice = append(steamCollectorsSlice, strings.TrimSpace(steamCollector))
	}

	steamData := collectors.NewSteamData(*steamAPIKey, *steamIDs)
	registry := prometheus.NewRegistry()

	if stringInSlice("playtime", steamCollectorsSlice) {
		fmt.Println("Registering playtime collector ...")
		playtimeCollector := collectors.NewPlaytimeCollector(steamData)
		registry.MustRegister(playtimeCollector)
	}
	if stringInSlice("price", steamCollectorsSlice) {
		fmt.Println("Registering price collector ...")
		priceCollector := collectors.NewPriceCollector(steamData)
		registry.MustRegister(priceCollector)
	}
	if stringInSlice("achievements", steamCollectorsSlice) {
		fmt.Println("Registering achievements collector ...")
		achievementsCollector := collectors.NewAchievementsCollector(steamData)
		registry.MustRegister(achievementsCollector)
	}

	// Register the process and Go metrics.
	// registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	// registry.MustRegister(prometheus.NewGoCollector())

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	err := http.ListenAndServe(":6630", nil)
	if err != nil {
		fmt.Println("Failed to start server:", err)
		os.Exit(1)
	}
}
