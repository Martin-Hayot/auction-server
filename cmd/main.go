package main

import (
	"net/http"

	"github.com/Martin-Hayot/auction-server/configs"
	"github.com/Martin-Hayot/auction-server/internal/database"
	"github.com/Martin-Hayot/auction-server/internal/handlers/websocket"
	"github.com/charmbracelet/log"
)

func main() {
	// Load configurations
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	if cfg.Server.Env == "dev" {
		// dev specific configurations
	}

	port := cfg.Server.Port
	if port == "" {
		port = "8080" // Default port if not specified
	}

	// Setup logger
	if cfg.Server.LogLevel == "" {
		cfg.Server.LogLevel = "debug" // Default log level if not specified
	}
	logLevel, err := log.ParseLevel(cfg.Server.LogLevel)
	if err != nil {
		log.Error("Invalid log level: ", err)
	}
	log.SetLevel(logLevel)

	// Initialize database service
	db := database.New(cfg)
	defer db.Close()

	// Initialize WebSocket handler
	auctionHandler := websocket.NewAuctionWebSocketHandler(db)

	// Start periodic check for auctions
	auctionHandler.StartPeriodicCheck()

	// Setup routes
	http.HandleFunc("/ws/auction", auctionHandler.HandleAuctions)

	// Start server
	log.Infof("Server started on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
