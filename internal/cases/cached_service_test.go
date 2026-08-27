package cases

import (
	"context"
	"crypto_service/internal/entities"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	coins            []*entities.Coin
	setCoinsCalled   bool
	setCoins         []*entities.Coin
	err              error
	setErr           error
	invalidateCalled bool
	invalidateErr    error
}

type fakeService struct {
	getCoinsCalled       bool
	coins                []*entities.Coin
	err                  error
	actualizeCoinsCalled bool
	actualizeErr         error
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
	return f.setErr
}

func (f *fakeCache) Invalidate(ctx context.Context) error {
	f.invalidateCalled = true
	return f.invalidateErr
}

func (f *fakeService) GetAggregatedCoins(
	ctx context.Context,
	titles []string,
	aggregate entities.Aggregate,
) ([]*entities.Coin, error) {
	return nil, nil
}

func (f *fakeService) ActualizeCoins(ctx context.Context) error {
	f.actualizeCoinsCalled = true
	return f.actualizeErr
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
		cacheErr           error
		setErr             error
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
		{
			name:               "cache err",
			cacheCoins:         nil,
			serviceCoins:       []*entities.Coin{btc},
			wantCoins:          []*entities.Coin{btc},
			wantSetCoins:       []*entities.Coin{btc},
			wantServiceCalled:  true,
			wantSetCoinsCalled: true,
			cacheErr:           errors.New("cache unavailable"),
		},
		{
			name:               "set cache err",
			cacheCoins:         nil,
			serviceCoins:       []*entities.Coin{btc},
			wantCoins:          []*entities.Coin{btc},
			wantSetCoins:       []*entities.Coin{btc},
			wantServiceCalled:  true,
			wantSetCoinsCalled: true,
			cacheErr:           nil,
			setErr:             errors.New("cache is broken"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cache := &fakeCache{
				coins:  tc.cacheCoins,
				err:    tc.cacheErr,
				setErr: tc.setErr,
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
