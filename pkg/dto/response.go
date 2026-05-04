package dto

import "time"

type CoinDTO struct {
	Title    string    `json:"title"`
	Cost     float64   `json:"cost"`
	ActualAt time.Time `json:"actual_at,omitempty"`
}

type CoinsDTO struct {
	Coins []CoinDTO `json:"coins"`
}

type ErrorDTO struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}
