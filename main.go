package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"prometheus-steam-web-api-exporter/collectors"
)

func main() {
	var (
		steamAPIKey = flag.String("steam-api-key", "", "API key to use for requests to the Steam Web API.")
		steamIDs    = flag.String("steam-ids", "", "Comma-separated list of SteamIDs whose playtime should be scraped.")
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

	playtimeCollector := collectors.NewPlaytimeCollector(*steamAPIKey, *steamIDs)

	// Create a new Prometheus registry and register the collector
	registry := prometheus.NewRegistry()
	registry.MustRegister(playtimeCollector)

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	err := http.ListenAndServe(":6630", nil)
	if err != nil {
		fmt.Println("Failed to start server:", err)
		os.Exit(1)
	}
}