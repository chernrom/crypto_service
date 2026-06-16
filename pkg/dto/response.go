package dto

import "time"

// CoinDTO represents coin entity
// swagger:model CoinDTO
type CoinDTO struct {
	// coin title
	// required: true
	// example: btc
	Title string `json:"title"`

	// coin cost in USD
	// required: true
	// example: 63097.42
	Cost float64 `json:"cost"`

	// time when coin cost was actualized
	// required: false
	// example: 2026-06-09T12:30:00Z
	ActualAt time.Time `json:"actual_at,omitempty"`
}

// CoinsDTO  represents array of coins
// swagger:model CoinsDTO
type CoinsDTO struct {
	// array of coins
	// required: true
	// example: [{"title":"btc","cost":63097.42,"actual_at":"2026-06-09T12:30:00Z"}]
	Coins []CoinDTO `json:"coins"`
}

// ErrorDTO represents error response
// swagger:model ErrorDTO
type ErrorDTO struct {
	// error message
	// required: true
	// example: titles field is empty
	Message string `json:"message"`

	// HTTP status code
	// required: true
	// example: 500
	StatusCode int `json:"status_code"`
}
