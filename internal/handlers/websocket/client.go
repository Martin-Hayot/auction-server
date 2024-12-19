package websocket

import (
	"log"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

type Client struct {
	ID          string
	Email       string
	Conn        *websocket.Conn
	Send        chan []byte   // Channel for outgoing messages
	RateLimiter *rate.Limiter // Rate limiter to prevent spamming
}

// readMessages listens for incoming messages from the client.
func (c *Client) ReadMessages(handleMessage func(*Client, []byte)) {
	defer func() {
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from client %s: %v", c.ID, err)
			break
		}
		// Delegate processing of the message
		handleMessage(c, message)
	}
}

// writeMessages sends outgoing messages to the client.
func (c *Client) WriteMessages() {
	defer func() {
		c.Conn.Close()
	}()

	for message := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("Error sending message to client %s: %v", c.ID, err)
			break
		}
	}
}

// Close cleans up client resources.
func (c *Client) Disconnect() {
	clientLock.Lock()
	delete(connectedClients, c)
	clientLock.Unlock()
	close(c.Send)
	c.Conn.Close()
}
