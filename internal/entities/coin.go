package entities

import (
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
		return nil, errors.Wrap(ErrInvalidParam, "empty name")
	}
	if cost < 0 {
		return nil, errors.Wrap(ErrInvalidParam, "negative cost")
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
