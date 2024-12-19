package websocket

import (
	"sync"

	"github.com/charmbracelet/log"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

type Client struct {
	ID          string
	Email       string
	Conn        *websocket.Conn
	Send        chan []byte   // Channel for outgoing messages
	RateLimiter *rate.Limiter // Rate limiter to prevent spamming
	closed      bool          // Flag to check if the connection is closed
	mu          sync.Mutex    // Mutex to protect the closed flag
}

// readMessages listens for incoming messages from the client.
func (c *Client) ReadMessages(handleMessage func(*Client, []byte)) {
	defer func() {
		c.Disconnect(nil) // Ensure cleanup
		log.Debugf("Connection closed for client %s", c.ID)
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Debugf("Error reading message from client %s: %v", c.ID, err)
			break
		}
		handleMessage(c, message)
	}
}

// writeMessages sends outgoing messages to the client.
func (c *Client) WriteMessages() {
	defer func() {
		c.Conn.Close()
	}()

	for message := range c.Send {
		c.mu.Lock()
		if c.closed {
			c.mu.Unlock()
			return
		}

		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		c.mu.Unlock()

		if err != nil {
			log.Debugf("Error sending message to client %s: %v", c.ID, err)
			return
		}
	}
}

// Disconnect cleans up client resources.
func (c *Client) Disconnect(handler *AuctionHandler) {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		close(c.Send)
	}
	c.mu.Unlock()

	if handler != nil {
		handler.connectedClients.Delete(c)
	}

	c.Conn.Close()
	log.Debugf("Client %s cleanup completed", c.ID) // Lower-level log here
}
