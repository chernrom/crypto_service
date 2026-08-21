package cases

import (
	"context"
	"crypto_service/internal/entities"
)

type fakeCache struct {
	coins []*entities.Coin
	err   error
}

func (f *fakeCache) GetCoins(
	ctx context.Context,
	key string,
) ([]*entities.Coin, error) {
	return f.coins, f.err
}

func (f *fakeCache) SetCoins(
	ctx context.Context,
	key string,
	coins []*entities.Coin,
) error {
	return nil
}

func (f *fakeCache) Invalidate(ctx context.Context) error {
	return nil
}
