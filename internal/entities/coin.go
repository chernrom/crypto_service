package entities

import (
	"log/slog"
	"time"

	"github.com/pkg/errors"
)

type Coin struct {
	title    string
	cost     float64
	actualAt time.Time
}

func NewCoin(title string, cost float64, actualAt time.Time) (*Coin, error) {
	if title == "" {
		slog.Error("new coin failed", "error", ErrInvalidParam, "reason", "empty title")
		return nil, errors.Wrap(ErrInvalidParam, "empty title")
	}

	if cost < 0 {
		slog.Error("new coin failed", "error", ErrInvalidParam, "reason", "negative cost", "cost", cost)
		return nil, errors.Wrap(ErrInvalidParam, "negative cost")
	}

	if actualAt.IsZero() {
		slog.Error("new coin failed", "error", ErrInvalidParam, "reason", "actualAt is zero")
		return nil, errors.Wrap(ErrInvalidParam, "actualAt is zero")
	}

	return &Coin{
		title:    title,
		cost:     cost,
		actualAt: actualAt,
	}, nil
}

func (c *Coin) Title() string {
	return c.title
}

func (c *Coin) Cost() float64 {
	return c.cost
}

func (c *Coin) ActualAt() time.Time {
	return c.actualAt
}
