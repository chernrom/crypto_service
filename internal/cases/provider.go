package cases

import (
	"context"

	"crypto_service/internal/entities"
)

//go:generate mockgen -source=provider.go -destination=mocks/provider_mock.go -package=mocks
type CryptoProvider interface {
	GetActualCoins(ctx context.Context, titles []string) ([]*entities.Coin, error)
}
