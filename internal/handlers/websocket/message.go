package websocket

import (
	"context"
	"encoding/json"

	"github.com/Martin-Hayot/auction-server/pkg/errors"
	"github.com/Martin-Hayot/auction-server/pkg/types"
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
		log.Debug("Client requested an update")
	default:
		log.Printf("Unknown message type: %s", msg.Type)
		client.Send <- []byte(errors.New(errors.ErrUnknownMessageType, "Unknown message type").ToJSON())
	}
}

// Handlers for specific message types
func (h *AuctionHandler) handleBidMessage(client *Client, data string) {
	type BidMessage struct {
		AuctionID string `json:"auction_id"`
		Amount    int    `json:"amount"`
	}
	var bidMsg BidMessage

	err := json.Unmarshal([]byte(data), &bidMsg)
	if err != nil {
		client.Send <- []byte(errors.New(errors.ErrBadMessageFormat, "Invalid bid message").ToJSON())
		return
	}

	ctx := context.Background()
	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		log.Error("Error starting transaction: ", err)
		client.Send <- []byte(errors.New(errors.ErrInternalServer, "Internal server error").ToJSON())
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	auction, err := h.db.GetAuctionByIdTx(ctx, tx, bidMsg.AuctionID)
	if err != nil {
		log.Error("Error retrieving auction: ", err)
		client.Send <- []byte(errors.New(errors.ErrInternalServer, "Internal server error").ToJSON())
		return
	}

	if bidMsg.Amount <= auction.CurrentBid {
		client.Send <- []byte(errors.New(errors.ErrBidTooLow, "Bid amount must be higher than current price").ToJSON())
		return
	}

	// Update auction
	auction.CurrentBid = bidMsg.Amount
	auction.CurrentBidderID = &client.ID
	auction.BiddersCount++
	auction, err = h.db.UpdateAuctionByIdTx(ctx, tx, auction)
	if err != nil {
		log.Error("Error updating auction: ", err)
		return
	}

	// Create new bid
	bid := types.Bid{
		AuctionID: auction.ID,
		UserID:    client.ID,
		Price:     bidMsg.Amount,
	}
	_, err = h.db.CreateBidTx(ctx, tx, bid)
	if err != nil {
		log.Error("Error creating bid: ", err)
		return
	}

	// Broadcast bid to all clients
	rawMessage, err := json.Marshal(&Message{Type: "bid", Data: data})
	if err != nil {
		log.Error("Error marshalling bid message: ", err)
		return
	}
	h.Broadcast(rawMessage)
}

func (h *AuctionHandler) handleAuctionEnd(auctionID string) {
	// Process auction end
	log.Debugf("Auction %s has ended", auctionID)

	// designate winner
	auction, err := h.db.GetAuctionById(auctionID)

	if err != nil {
		log.Error("Error retrieving auction: ", err)
		return
	}

	if auction.CurrentBid < auction.ReservePrice {
		log.Debug("Auction did not meet reserve price")
		auction.Status = "reserve_not_met"
	} else {
		log.Debug("Auction met reserve price")
		auction.Status = "sold"
		auction.WinnerID = auction.CurrentBidderID

		// Update auction in database
		_, err = h.db.UpdateAuctionById(auction)
		if err != nil {
			log.Error("Error updating auction: ", err)
			return
		}
	}

	h.Broadcast([]byte(`{"type": "auction_end", "data": "Auction has ended"}`))
}
