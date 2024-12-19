package main

import (
	"net/http"

	"github.com/Martin-Hayot/auction-server/configs"
	"github.com/Martin-Hayot/auction-server/internal/database"
	"github.com/Martin-Hayot/auction-server/internal/handlers/websocket"
	"github.com/charmbracelet/log"
)

var db database.Service

func main() {
	// Load configurations
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	port := cfg.Server.Port
	if port == "" {
		port = "8080" // Default port if not specified
	}

	// Initialize database service
	db = database.New(cfg)
	defer db.Close()

	auctionHandler := websocket.NewAuctionWebSocketHandler(db)

	// Setup routes
	http.HandleFunc("/ws/auction", auctionHandler.HandleAuctionWebSocket)

	log.Infof("Server started on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
