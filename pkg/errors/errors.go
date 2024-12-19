package errors

import "fmt"

type AppError struct {
	Code    int    // HTTP status code or custom error code
	Message string // User-facing message
	Err     error  // Underlying error (optional)
}

const (
	ErrInvalidToken     = 1001
	ErrAuctionNotFound  = 1002
	ErrBidTooLow        = 1003
	ErrAuctionClosed    = 1004
	ErrWebSocketUpgrade = 1005

	ErrInternalServer = 500
)

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Wrapping utility
func Wrap(err error, message string) *AppError {
	return &AppError{Message: message, Err: err}
}

// Error creation utility
func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}
