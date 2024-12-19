package websocket

import (
	"net/http"
	"sync"

	"github.com/Martin-Hayot/auction-server/internal/auth"
	"github.com/Martin-Hayot/auction-server/internal/database"
	"github.com/Martin-Hayot/auction-server/pkg/types"
	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

type AuctionHandler struct {
	db database.Service // Injected dependency
}

func NewAuctionWebSocketHandler(db database.Service) *AuctionHandler {
	return &AuctionHandler{db: db}
}

var (
	connectedClients = make(map[*Client]bool) // Track all connected clients
	clientLock       = sync.Mutex{}           // Prevent race conditions
	upgrader         = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// AuctionHandler upgrades the HTTP request to a WebSocket connection.
func (h *AuctionHandler) handleAuctions(w http.ResponseWriter, r *http.Request, user types.User) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Infof("Failed to upgrade connection: %v", err)
		http.Error(w, "Failed to establish connection", http.StatusInternalServerError)
		return
	}

	// Initialize a new client
	client := &Client{
		ID:          user.ID,
		Email:       user.Email,
		Conn:        conn,
		Send:        make(chan []byte),
		RateLimiter: rate.NewLimiter(1, 3),
	}

	clientLock.Lock()
	connectedClients[client] = true
	clientLock.Unlock()

	// Start handling the client
	go client.ReadMessages(h.HandleMessage)
	go client.WriteMessages()
}

// handleAuctionWebSocket integrates authentication and WebSocket handling.
func (h *AuctionHandler) HandleAuctionWebSocket(w http.ResponseWriter, r *http.Request) {
	// Validate the token from the cookie
	token, err := auth.ValidateTokenFromCookie(r)
	if err != nil || token == nil {
		log.Error("Invalid token: ", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var email string
	err = token.Get("email", &email)
	if err != nil {
		log.Error("Error retrieving email from token claims")
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
	h.handleAuctions(w, r, user)
}

// Broadcast sends a message to all connected clients.
func Broadcast(message []byte) {
	clientLock.Lock()
	defer clientLock.Unlock()

	for client := range connectedClients {
		select {
		case client.Send <- message:
		default:
			// Remove disconnected clients
			delete(connectedClients, client)
			client.Disconnect()
		}
	}
}
