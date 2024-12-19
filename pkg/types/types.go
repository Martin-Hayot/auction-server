package types

import (
	"time"
)

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Auctions struct {
	ID               string    `json:"id"`
	Mileage          int       `json:"mileage"`
	State            string    `json:"state"`
	CirculationDate  time.Time `json:"circulationDate"`
	FuelType         string    `json:"fuelType"`
	Power            int       `json:"power"`
	Transmission     string    `json:"transmission"`
	CarBody          string    `json:"carBody"`
	GearBox          string    `json:"gearBox"`
	Color            string    `json:"color"`
	Doors            int       `json:"doors"`
	Seats            int       `json:"seats"`
	StartDate        time.Time `json:"startDate"`
	EndDate          time.Time `json:"endDate"`
	StartPrice       int       `json:"startPrice"`
	MaxPrice         int       `json:"maxPrice"`
	ReservePrice     int       `json:"reservePrice"`
	CurrentBid       int       `json:"currentBid"`
	BidIncrement     int       `json:"bidIncrement"`
	CurrentBidderID  *string   `json:"currentBidderId,omitempty"`
	BiddersCount     int       `json:"biddersCount"`
	WinnerID         *string   `json:"winnerId,omitempty"`
	OnlyForMerchants bool      `json:"onlyForMerchants"`
	Status           string    `json:"status"`
	CarID            string    `json:"carId"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}
