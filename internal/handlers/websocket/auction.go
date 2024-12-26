package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/Martin-Hayot/auction-server/internal/auth"
	"github.com/Martin-Hayot/auction-server/internal/database"
	"github.com/Martin-Hayot/auction-server/pkg/types"
	"github.com/charmbracelet/log"
	"github.com/go-co-op/gocron"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

// AuctionHandler handles WebSocket connections for the auction system.
type AuctionHandler struct {
	db               database.Service
	connectedClients sync.Map   // Thread-safe map of connected clients.
	clientLock       sync.Mutex // Mutex to synchronize access to connectedClients.
	CurrentAuctions  []types.Auctions
}

func (h *AuctionHandler) StartPeriodicCheck() {
	scheduler := gocron.NewScheduler(time.UTC)
	scheduler.Every(1).Minute().Do(h.CheckAuctionsStatus)
	scheduler.StartAsync()
}

func (h *AuctionHandler) CheckAuctionsStatus() {
	var err error
	h.CurrentAuctions, err = h.db.GetCurrentAuctions()
	if err != nil {
		log.Error("Error getting current auctions: ", err)
		return
	}

	log.Debugf("%v active auction", len(h.CurrentAuctions))

	for _, auction := range h.CurrentAuctions {

		// Check if the auction has ended
		if time.Now().After(auction.EndDate) {
			log.Debugf("Auction %s ended", auction.ID)
			h.handleAuctionEnd(auction.ID)
			continue
		}
		time.AfterFunc(time.Until(auction.EndDate), func() {
			log.Debugf("Auction %s ended", auction.ID)
			h.handleAuctionEnd(auction.ID)
		})
	}
}

// NewAuctionWebSocketHandler creates a new instance of AuctionHandler.
func NewAuctionWebSocketHandler(db database.Service) *AuctionHandler {
	return &AuctionHandler{
		db:               db,
		connectedClients: sync.Map{},
	}
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// upgradeToWebSocket upgrades the HTTP request to a WebSocket connection and initializes a new client.
// It adds the client to the list of connected clients and starts handling the client's messages.
func (h *AuctionHandler) upgradeToWebSocket(w http.ResponseWriter, r *http.Request, user types.User) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debugf("Failed to upgrade connection: %v", err)
		http.Error(w, "Failed to establish connection", http.StatusInternalServerError)
		return
	}

	// Initialize a new client
	client := &Client{
		ID:    user.ID,
		Email: user.Email,
		// Auctions:    ,
		Conn:        conn,
		Send:        make(chan []byte),
		RateLimiter: rate.NewLimiter(1, 3),
	}

	// Add the client to the list of connected clients
	h.clientLock.Lock()
	h.connectedClients.Store(client, true)
	h.clientLock.Unlock()

	// Start handling the client
	go client.ReadMessages(h.HandleMessage)
	go client.WriteMessages()
}

// HandleAuctions handles incoming HTTP requests for the auction WebSocket.
// It validates the user's token, retrieves the user from the database, and upgrades the connection to a WebSocket.
func (h *AuctionHandler) HandleAuctions(w http.ResponseWriter, r *http.Request) {
	// Validate the token from the cookie
	token, err := auth.ValidateTokenFromCookie(r)
	if err != nil || token == nil {
		log.Debug("Invalid token: ", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var email string
	err = token.Get("email", &email)
	if err != nil {
		log.Error("Error retrieving email from token claims", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the user exists
	user, err := h.db.GetUserByEmail(email)
	if err != nil {
		log.Error("User not found: ", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Pass to WebSocket handler
	h.upgradeToWebSocket(w, r, user)
}

// Broadcast sends a message to all connected clients.
// It iterates over the connected clients and attempts to send the message to each client.
func (h *AuctionHandler) Broadcast(message []byte) {
	h.connectedClients.Range(func(key, value any) bool {
		client := key.(*Client)

		// Check if the client is closed
		client.mu.Lock()
		if client.closed {
			client.mu.Unlock()
			h.connectedClients.Delete(client) // Remove disconnected clients
			return true
		}
		client.mu.Unlock()

		// Try to send the message
		select {
		case client.Send <- message:
			// Message sent successfully
		default:
			client.Disconnect(h) // Disconnect the client on failure
		}
		return true // Continue iteration
	})
}
