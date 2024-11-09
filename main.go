package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // to personalize the origin check
}

func validateTokenFromCookie(r *http.Request) (bool, error) {
	// Extract the token from the cookie
	cookie, err := r.Cookie("authjs.session-token")
	if err != nil {
		return false, err
	}

	tokenString := cookie.Value
	log.Infof("Token: %s", tokenString)

	authSecret := os.Getenv("AUTH_SECRET")
	// Validate the token
	token, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.HS256(), []byte(authSecret)), jwt.WithValidate(true))

	log.Infof("Token: %s", token)

	if err != nil {
		return false, err
	}

	// Check if token is expired
	if exp, ok := token.Expiration(); ok && exp.Before(time.Now()) {
		return false, nil
	}
	return true, nil
}

func auctionHandler(w http.ResponseWriter, r *http.Request) {

	isValid, err := validateTokenFromCookie(r)
	if !isValid || err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	// Extract token from query parameters
	token := r.URL.Query().Get("token")
	if token == "" {
		log.Error("Token not provided")
		http.Error(w, "Token not provided", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Error connecting to websocket: ", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte)}
	go client.readMessages()
	go client.writeMessages()
}

func (c *Client) readMessages() {
	defer c.conn.Close()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Error("Error reading messages: ", err)
			break
		}
		// Auction Logic

		ip := c.conn.LocalAddr()
		log.Infof("Received message from %s: %s", ip, message)

		// Unmarshal message
		type Message struct {
			Type string `json:"type"`
		}

		var validTypes = map[string]bool{
			"join":   true,
			"bid":    true,
			"update": true,
		}

		var msg Message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Error("Error unmarshaling message: ", err)
			break
		}

		log.Infof("Message type: %s", msg.Type)

		// Validate message type
		if !validTypes[msg.Type] {
			log.Errorf("Invalid message type: %s", msg.Type)
			break
		}

		if msg.Type == "join" {
			// Add client to list

		}

		if msg.Type == "bid" {
			// Check bid
		}

		if msg.Type == "update" {
			// Update auction
		}

		// Further processing based on message type

		// validate message

		// check if message is valid
		// check if message is a bid
		// check if bid is higher than current bid
		// if bid is higher, update current bid
		// send message to all clients
		// if message is not a bid, ignore
		// if message is invalid, send error message

		c.send <- message
	}
}

func (c *Client) writeMessages() {
	defer c.conn.Close()
	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Error("Error sending message: ", err)
			break
		}
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Error("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	http.HandleFunc("/ws/auction", auctionHandler)
	log.Infof("Server started on port %s", port)
	http.ListenAndServe(":"+port, nil)
}
