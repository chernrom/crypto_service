package port

import (
	"context"

	"crypto_service/internal/entities"
)

type ServiceProvider interface {
	ActualizeCoins(ctx context.Context) error
	GetAggregatedCoins(ctx context.Context, titles []string, aggregate entities.Aggregate) ([]*entities.Coin, error)
	GetCoins(ctx context.Context, titles []string) ([]*entities.Coin, error)
}
