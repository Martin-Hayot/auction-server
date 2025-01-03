package main

import (
	"net/http"
	"time"

	"github.com/Martin-Hayot/auction-server/configs"
	"github.com/Martin-Hayot/auction-server/internal/database"
	"github.com/Martin-Hayot/auction-server/internal/handlers/websocket"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
	db database.Service
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Every(1*time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Define the model for the Bubble Tea application
type model struct {
	table    table.Model
	quitting bool
}

// Update the Init method in the model struct
func (m model) Init() tea.Cmd {
	return tick()
}

func newTable() model {
	columns := []table.Column{
		{Title: "AUCTION ID", Width: 20},
		{Title: "HIGHEST BIDDER", Width: 20},
		{Title: "WINNER ID", Width: 20},
		{Title: "TIME LEFT", Width: 20},
	}

	auctions, err := db.GetCurrentAuctions()
	if err != nil {
		log.Error("Error getting auctions: ", err)
		// Return empty model on error
		return model{
			table: table.New(
				table.WithColumns(columns),
				table.WithRows([]table.Row{}),
			),
		}
	}

	rows := make([]table.Row, 0)
	for _, auction := range auctions {
		// Safe handling of nullable fields
		currentBidder := "-"
		if auction.CurrentBidderID != nil {
			currentBidder = *auction.CurrentBidderID
		}

		winner := "-"
		if auction.WinnerID != nil {
			winner = *auction.WinnerID
		}

		timeLeft := time.Until(auction.EndDate)
		timeLeftStr := timeLeft.String()

		if timeLeft < 0 {
			timeLeftStr = "Ended"
		}

		row := []string{
			auction.ID,
			currentBidder,
			winner,
			timeLeftStr,
		}
		rows = append(rows, row)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{table: t}
}

func updateTableRows(t table.Model) table.Model {
	auctions, err := db.GetCurrentAuctions()
	if err != nil {
		log.Error("Error getting auctions: ", err)
		return t
	}

	rows := make([]table.Row, 0)
	for _, auction := range auctions {
		currentBidder := "-"
		if auction.CurrentBidderID != nil {
			currentBidder = *auction.CurrentBidderID
		}

		winner := "-"
		if auction.WinnerID != nil {
			winner = *auction.WinnerID
		}

		timeLeft := time.Until(auction.EndDate)
		timeLeftStr := timeLeft.String()

		if timeLeft < 0 {
			timeLeftStr = "Ended"
		}

		row := []string{
			auction.ID,
			currentBidder,
			winner,
			timeLeftStr,
		}
		rows = append(rows, row)
	}

	t.SetRows(rows)
	return t
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tickMsg:
		m.table = updateTableRows(m.table)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	}

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)

}

// Render the view based on the current state of the model
func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	return baseStyle.Render(m.table.View()) + "\n" + helpStyle.Render("• q: exit\n")

}

func main() {
	// Load configurations
	cfg, err := configs.LoadConfig()
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	if cfg.Server.Env == "dev" {
		// dev specific configurations
	}

	port := cfg.Server.Port
	if port == "" {
		port = "8080" // Default port if not specified
	}

	// Setup logger
	if cfg.Server.LogLevel == "" {
		cfg.Server.LogLevel = "debug" // Default log level if not specified
	}
	logLevel, err := log.ParseLevel(cfg.Server.LogLevel)
	if err != nil {
		log.Error("Invalid log level: ", err)
	}
	log.SetLevel(logLevel)

	f, err := tea.LogToFile("debug.log", "debug")
	if err != nil {
		log.Fatal("fatal:", err)
	}
	defer f.Close()

	// Redirect logs to file
	log.SetOutput(f)

	// Initialize database service
	db = database.New(cfg)
	defer db.Close()

	// Initialize WebSocket handler
	auctionHandler := websocket.NewAuctionWebSocketHandler(db)

	// Start periodic check for auctions
	auctionHandler.StartPeriodicCheck()

	// Setup routes
	http.HandleFunc("/ws/auction", auctionHandler.HandleAuctions)

	// Start server in a goroutine
	log.Infof("Server started on port %s", port)
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	m := newTable()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running Bubble Tea program: %v", err)
	}

}
