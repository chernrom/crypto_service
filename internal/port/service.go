package port

import (
	"context"
	"crypto_service/internal/entities"
)

type ServiceProvider interface {
	ActualizeCoins(ctx context.Context) error
	GetAggregatedCoins(ctx context.Context, titles []string, aggregationType string) ([]*entities.Coin, error)
	GetCoins(ctx context.Context, titles []string) ([]*entities.Coin, error)
}
