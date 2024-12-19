package websocket

import (
	"encoding/json"

	"github.com/Martin-Hayot/auction-server/pkg/errors"
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
	if !client.RateLimiter.Allow() {
		log.Warnf("Rate limit exceeded for client %s", client.ID)
		client.Send <- []byte(`{"type": "error", "message": "Rate limit exceeded"}`)
		return
	}

	msg, err := ParseMessage(rawMessage)
	if err != nil {
		log.Infof("Invalid message from client %s: %v", client.ID, err)
		client.Send <- []byte(errors.New(errors.ErrBadMessageFormat, "Invalid message format").ToJSON())
		return
	}

	switch msg.Type {
	case "join":
		log.Debug("Client joined the auction")
	case "bid":
		h.handleBidMessage(client, msg.Data)
	case "update":
		handleUpdateMessage(client, msg.Data)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		client.Send <- []byte(errors.New(errors.ErrUnknownMessageType, "Unknown message type").ToJSON())
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
		client.Send <- []byte(errors.New(errors.ErrBadMessageFormat, "Invalid bid message").ToJSON())
		return
	}
	// Retrieve auction and check if bid is valid
	auction, err := h.db.GetAuctionById(bidMsg.AuctionID)
	if err != nil {
		log.Error("Error retrieving auction: ", err)
		return
	}

	log.Debugf("Client %s placed a bid of $%.2f on auction %s", client.ID, bidMsg.Amount, bidMsg.AuctionID)

	if bidMsg.Amount <= float64(auction.CurrentBid) {
		client.Send <- []byte(errors.New(errors.ErrBidTooLow, "Bid amount must be higher than current price").ToJSON())
		log.Warn("Invalid bid amount")
		return
	}

	// Update auction with new bid

	// Broadcast bid to all clients
	rawMessage, err := json.Marshal(&Message{Type: "bid", Data: data})
	if err != nil {
		log.Error("Error marshalling bid message: ", err)
		return
	}
	h.Broadcast(rawMessage)
}

func handleUpdateMessage(client *Client, data string) {
	// Process update message
}
