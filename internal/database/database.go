package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/Martin-Hayot/auction-server/configs"
	"github.com/Martin-Hayot/auction-server/pkg/types"
	"github.com/charmbracelet/log"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// USER METHODS
	GetUserByEmail(email string) (types.User, error)

	// AUCTION METHODS
	GetAuctionById(auctionID string) (types.Auctions, error)
}

type service struct {
	db *sql.DB
}

var dbInstance *service

func Get() service {
	if dbInstance == nil {
		log.Error("Database not initialized")
	}

	return *dbInstance
}

func New(cfg *configs.Config) Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	dbConfig := cfg.Database
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name,
		dbConfig.SSLMode,
	)
	db, err := sql.Open("pgx", connStr)

	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal("Error connecting with gorm: ", err)
	}

	if err != nil {
		log.Fatal("Error with migration: ", err)
	}

	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Info("Disconnected from database")
	return s.db.Close()
}

func (s *service) GetUserByEmail(email string) (types.User, error) {
	var user types.User
	err := s.db.QueryRow(`SELECT id, name, email, role FROM public."User" WHERE email = $1`, email).Scan(&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		return types.User{}, fmt.Errorf("error getting user by email: %w", err)
	}
	return user, nil
}

func (s *service) GetAuctionById(auctionID string) (types.Auctions, error) {
	var auction types.Auctions
	query := `
        SELECT 
            "id", 
            "mileage", 
            "state", 
            "circulationDate", 
            "fuelType", 
            "power", 
            "transmission", 
            "carBody", 
            "gearBox", 
            "color", 
            "doors", 
            "seats", 
            "startDate", 
            "endDate", 
            "startPrice", 
            "maxPrice", 
            "reservePrice", 
            "currentBid", 
            "bidIncrement", 
            "currentBidderId", 
            "biddersCount", 
            "winnerId", 
            "onlyForMerchants", 
            "status", 
            "carId", 
            "createdAt", 
            "updatedAt" 
        FROM public."Auctions" 
        WHERE "id" = $1
    `
	err := s.db.QueryRow(query, auctionID).Scan(
		&auction.ID,
		&auction.Mileage,
		&auction.State,
		&auction.CirculationDate,
		&auction.FuelType,
		&auction.Power,
		&auction.Transmission,
		&auction.CarBody,
		&auction.GearBox,
		&auction.Color,
		&auction.Doors,
		&auction.Seats,
		&auction.StartDate,
		&auction.EndDate,
		&auction.StartPrice,
		&auction.MaxPrice,
		&auction.ReservePrice,
		&auction.CurrentBid,
		&auction.BidIncrement,
		&auction.CurrentBidderID,
		&auction.BiddersCount,
		&auction.WinnerID,
		&auction.OnlyForMerchants,
		&auction.Status,
		&auction.CarID,
		&auction.CreatedAt,
		&auction.UpdatedAt,
	)

	if err != nil {
		return types.Auctions{}, fmt.Errorf("error getting auction by id: %w", err)
	}
	return auction, nil
}
