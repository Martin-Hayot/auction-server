package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwe"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"golang.org/x/crypto/hkdf"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // To personalize the origin check
}

func generateEncryptionKey() ([]byte, error) {
	authSecret := os.Getenv("AUTH_SECRET")
	if authSecret == "" {
		return nil, fmt.Errorf("AUTH_SECRET not set")
	}

	salt := "authjs.session-token"
	info := fmt.Sprintf("Auth.js Generated Encryption Key (%s)", salt)

	// HKDF with SHA-256
	hash := sha256.New
	kdf := hkdf.New(hash, []byte(authSecret), []byte(salt), []byte(info))

	// Change to 32 bytes (256 bits) for AES-256
	key := make([]byte, 32)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	return key, nil
}

func jweToJwt(encryptedToken string) (string, error) {
	key, err := generateEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("key generation failed: %w", err)
	}

	log.Debug("Attempting JWE decryption", "keyLength", len(key))

	// Decrypt JWE using DIRECT key encryption and A256GCM content encryption
	decrypted, err := jwe.Decrypt([]byte(encryptedToken),
		jwe.WithKey(jwa.DIRECT(), key))
	if err != nil {
		return "", fmt.Errorf("JWE decryption failed: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(decrypted, &payload); err != nil {
		return "", fmt.Errorf("failed to parse payload: %w", err)
	}

	token := jwt.New()
	for k, v := range payload {
		token.Set(k, v)
	}

	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), []byte(os.Getenv("AUTH_SECRET"))))
	if err != nil {
		return "", fmt.Errorf("JWT signing failed: %w", err)
	}

	return string(signed), nil
}

func validateTokenFromCookie(r *http.Request) (bool, error) {
	cookie, err := r.Cookie("authjs.session-token")
	if err != nil {
		return false, fmt.Errorf("no session cookie: %w", err)
	}

	// Convert JWE to JWT
	jwtString, err := jweToJwt(cookie.Value)
	if err != nil {
		log.Error("Failed to convert JWE to JWT", "error", err)
		return false, err
	}

	// Verify JWT
	token, err := jwt.Parse([]byte(jwtString),
		jwt.WithKey(jwa.HS256(), []byte(os.Getenv("AUTH_SECRET"))),
		jwt.WithValidate(true))
	if err != nil {
		return false, fmt.Errorf("invalid JWT: %w", err)
	}

	// Check expiration
	if exp, ok := token.Expiration(); ok && exp.Before(time.Now()) {
		return false, fmt.Errorf("token expired")
	}

	return true, nil
}

// Auction WebSocket handler
func auctionHandler(w http.ResponseWriter, r *http.Request) {
	isValid, err := validateTokenFromCookie(r)
	if !isValid || err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
