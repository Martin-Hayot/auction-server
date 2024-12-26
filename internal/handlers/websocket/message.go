package websocket

import (
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
	// Process data for bid message
	type BidMessage struct {
		AuctionID string `json:"auction_id"`
		Amount    int    `json:"amount"`
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

	log.Debugf("Client %s placed a bid of %v on auction %s", client.ID, bidMsg.Amount, bidMsg.AuctionID)

	if bidMsg.Amount <= auction.CurrentBid {
		client.Send <- []byte(errors.New(errors.ErrBidTooLow, "Bid amount must be higher than current price").ToJSON())
		log.Warn("Invalid bid amount")
		return
	}

	// Update auction with new bid
	auction.CurrentBid = bidMsg.Amount
	auction.CurrentBidderID = &client.ID
	auction.BiddersCount++

	auction, err = h.db.UpdateAuctionById(auction)
	if err != nil {
		log.Error("Error updating auction: ", err)
		return
	}

	// create new bid in database
	bid := types.Bid{
		AuctionID: auction.ID,
		UserID:    client.ID,
		Price:     bidMsg.Amount,
	}

	bid, err = h.db.CreateBid(bid)
	if err != nil {
		log.Error("Error creating bid: ", err)
		return
	}

	// add bid to auction list from client
	client.Auctions = append(client.Auctions, bid.AuctionID)

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
