package cases

import (
	"context"
	"crypto_service/internal/entities"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	coins          []*entities.Coin
	setCoinsCalled bool
	setCoins       []*entities.Coin
	err            error
}

type fakeService struct {
	getCoinsCalled bool
	coins          []*entities.Coin
	err            error
}

func (f *fakeService) GetCoins(
	ctx context.Context,
	titles []string,
) ([]*entities.Coin, error) {
	f.getCoinsCalled = true
	return f.coins, f.err
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
	f.setCoinsCalled = true
	f.setCoins = coins
	return nil
}

func (f *fakeCache) Invalidate(ctx context.Context) error {
	return nil
}

func (f *fakeService) GetAggregatedCoins(
	ctx context.Context,
	titles []string,
	aggregate entities.Aggregate,
) ([]*entities.Coin, error) {
	return nil, nil
}

func (f *fakeService) ActualizeCoins(ctx context.Context) error {
	return nil
}

func TestCachedService_GetCoins(t *testing.T) {
	btc, err := entities.NewCoin("btc", 0.1231231, time.Now())
	require.NoError(t, err)

	tests := []struct {
		name               string
		cacheCoins         []*entities.Coin
		serviceCoins       []*entities.Coin
		wantCoins          []*entities.Coin
		wantSetCoins       []*entities.Coin
		wantServiceCalled  bool
		wantSetCoinsCalled bool
	}{
		{
			name:               "cache hit",
			cacheCoins:         []*entities.Coin{btc},
			wantCoins:          []*entities.Coin{btc},
			wantSetCoins:       nil,
			wantServiceCalled:  false,
			wantSetCoinsCalled: false,
		},
		{
			name:               "cache miss",
			cacheCoins:         nil,
			serviceCoins:       []*entities.Coin{btc},
			wantCoins:          []*entities.Coin{btc},
			wantSetCoins:       []*entities.Coin{btc},
			wantServiceCalled:  true,
			wantSetCoinsCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cache := &fakeCache{
				coins: tc.cacheCoins,
			}

			service := &fakeService{
				coins: tc.serviceCoins,
			}

			cachedService, err := NewCachedService(service, cache)
			require.NoError(t, err)

			coins, err := cachedService.GetCoins(context.Background(), []string{"btc"})
			require.NoError(t, err)

			require.Equal(t, tc.wantCoins, coins)
			require.Equal(t, tc.wantServiceCalled, service.getCoinsCalled)
			require.Equal(t, tc.wantSetCoinsCalled, cache.setCoinsCalled)
			require.Equal(t, tc.wantSetCoins, cache.setCoins)

		})
	}
}
