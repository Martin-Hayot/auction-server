package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestWebSocket(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(auctionHandler))
	defer server.Close()

	// Create a WebSocket URL
	url := "ws" + server.URL[len("http"):]

	// Connect to the WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer ws.Close()

	// Test sending a message
	message := []byte("test message")
	err = ws.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Test receiving a message
	_, receivedMessage, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read message: %v", err)
	}

	if string(receivedMessage) != string(message) {
		t.Fatalf("Expected message %s, but got %s", message, receivedMessage)
	}
}
