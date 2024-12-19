package websocket

import (
	"encoding/json"

	"github.com/charmbracelet/log"
)

type Message struct {
	Type string `json:"type"` // Type of the message (e.g., "bid", "update")
	Data string `json:"data"` // Payload of the message
}

// ParseMessage validates and parses incoming messages.
func ParseMessage(rawMessage []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(rawMessage, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// HandleMessage routes the message based on its type.
func (h *AuctionHandler) HandleMessage(client *Client, rawMessage []byte) {
	msg, err := ParseMessage(rawMessage)
	if err != nil {
		log.Infof("Invalid message from client %s: %v", client.ID, err)
		client.Send <- []byte(`{"type": "error", "message": "Invalid message format"}`)
		return
	}

	switch msg.Type {
	case "join":
		log.Info("Client joined the auction")
	case "bid":
		h.handleBidMessage(client, msg.Data)
	case "update":
		handleUpdateMessage(client, msg.Data)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		client.Send <- []byte(`{"type": "error", "message": "Unknown message type"}`)
	}
}

// Handlers for specific message types
func (h *AuctionHandler) handleBidMessage(client *Client, data string) {
	// Process data for bid message
	type BidMessage struct {
		AuctionID string  `json:"auction_id"`
		Amount    float64 `json:"amount"`
	}
	var bidMsg BidMessage

	err := json.Unmarshal([]byte(data), &bidMsg)
	if err != nil {
		log.Infof("Invalid bid message from client %s: %v", client.ID, err)
		client.Send <- []byte(`{"type": "error", "message": "Invalid bid message"}`)
		return
	}
	// Retrieve auction and check if bid is valid
	auction, err := h.db.GetAuctionById(bidMsg.AuctionID)
	if err != nil {
		log.Error("Error retrieving auction: ", err)
		return
	}

	log.Infof("Client %s placed a bid of $%.2f on auction %s", client.ID, bidMsg.Amount, bidMsg.AuctionID)

	if bidMsg.Amount <= float64(auction.CurrentBid) {
		client.Send <- []byte(`{"type": "error", "message": "Bid amount must be higher than current price"}`)
		return
	}
}

func handleUpdateMessage(client *Client, data string) {
	// Process update message
}
