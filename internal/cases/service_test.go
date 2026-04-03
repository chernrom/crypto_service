package cases_test

import (
	"context"
	"testing"
	"time"

	"crypto_service/internal/cases"
	"crypto_service/internal/cases/mocks"
	"crypto_service/internal/entities"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	errtest = errors.New("test error")
)

func TestNewService(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	mockStorage := mocks.NewMockStorage(ctrl)
	mockProvider := mocks.NewMockCryptoProvider(ctrl)

	tests := []struct {
		name     string
		provider cases.CryptoProvider
		storage  cases.Storage
		wantErr  bool
	}{
		{
			name:     "provider is nil",
			provider: nil,
			storage:  mockStorage,
			wantErr:  true,
		},
		{
			name:     "storage is nil",
			provider: mockProvider,
			storage:  nil,
			wantErr:  true,
		},
		{
			name:     "success",
			provider: mockProvider,
			storage:  mockStorage,
			wantErr:  false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, err := cases.NewService(tc.provider, tc.storage)
			if tc.wantErr {
				require.Nil(t, service)
				require.ErrorIs(t, err, entities.ErrInvalidParam)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, service)

		})
	}
}

func TestService_GetCoins(t *testing.T) {
	t.Parallel()
	type stages struct {
		stageStorageGetAllTitles        func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, err error)
		stageStorageGetAllTitlesErr     error
		stagesProviderGetActualCoins    func(ctx context.Context, t *testing.T, provider *mocks.MockCryptoProvider, titles []string, coins []*entities.Coin, err error)
		stagesProviderGetActualCoinsErr error
		stagesStorageStore              func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, coins []*entities.Coin, err error)
		stagesStorageStoreErr           error
		stageStorageGetLastCoins        func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, coins []*entities.Coin, err error)
		stageStorageGetLastCoinsErr     error
	}

	tests := []struct {
		name    string
		stages  stages
		wantErr bool
		resErr  error
	}{
		{
			name: "1",
			stages: stages{
				stageStorageGetAllTitles:    storageGetAllTitles,
				stageStorageGetAllTitlesErr: errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name: "2",
			stages: stages{
				stageStorageGetAllTitles:        storageGetAllTitles,
				stagesProviderGetActualCoins:    providerGetActualCoins,
				stagesProviderGetActualCoinsErr: errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name: "3",
			stages: stages{
				stageStorageGetAllTitles:     storageGetAllTitles,
				stagesProviderGetActualCoins: providerGetActualCoins,
				stagesStorageStore:           storageStore,
				stagesStorageStoreErr:        errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name: "4",
			stages: stages{
				stageStorageGetAllTitles:     storageGetAllTitles,
				stagesProviderGetActualCoins: providerGetActualCoins,
				stagesStorageStore:           storageStore,
				stageStorageGetLastCoins:     storageGetLastCoins,
			},
			wantErr: false,
		},
		{
			name: "5",
			stages: stages{
				stageStorageGetAllTitles:     storageGetAllTitles,
				stagesProviderGetActualCoins: providerGetActualCoins,
				stagesStorageStore:           storageStore,
				stageStorageGetLastCoins:     storageGetLastCoins,
				stageStorageGetLastCoinsErr:  errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			it.Cleanup(func() {
				ctrl.Finish()
			})

			mockStorage := mocks.NewMockStorage(ctrl)
			mockProvider := mocks.NewMockCryptoProvider(ctrl)

			service, err := cases.NewService(mockProvider, mockStorage)
			require.NoError(it, err)
			require.NotNil(it, service)

			ctx := context.Background()
			btcTitle := "BTC"
			ethTitle := "ETH"

			eth, err := entities.NewCoin(ethTitle, 100, time.Now())
			require.NoError(it, err)

			if tc.stages.stageStorageGetAllTitles != nil {
				tc.stages.stageStorageGetAllTitles(ctx, it, mockStorage, []string{btcTitle}, tc.stages.stageStorageGetAllTitlesErr)
			}

			if tc.stages.stagesProviderGetActualCoins != nil {
				tc.stages.stagesProviderGetActualCoins(ctx, it, mockProvider, []string{ethTitle}, []*entities.Coin{eth}, tc.stages.stagesProviderGetActualCoinsErr)
			}

			if tc.stages.stagesStorageStore != nil {
				tc.stages.stagesStorageStore(ctx, it, mockStorage, []*entities.Coin{eth}, tc.stages.stagesStorageStoreErr)
			}

			if tc.stages.stageStorageGetLastCoins != nil {
				tc.stages.stageStorageGetLastCoins(ctx, it, mockStorage, []string{btcTitle, ethTitle}, []*entities.Coin{eth}, tc.stages.stageStorageGetLastCoinsErr)
			}

			coins, err := service.GetCoins(ctx, []string{btcTitle, ethTitle})
			if tc.wantErr {
				require.Nil(it, coins)
				require.ErrorIs(it, err, tc.resErr)
				return
			}

			require.NoError(it, err)
			require.Equal(it, []*entities.Coin{eth}, coins)

		})
	}
}

func TestService_GetAggregatedCoins(t *testing.T) {
	t.Parallel()

	type stages struct {
		stageStorageGetAllTitles          func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, err error)
		stageStorageGetAllTitlesErr       error
		stagesProviderGetActualCoins      func(ctx context.Context, t *testing.T, provider *mocks.MockCryptoProvider, titles []string, coins []*entities.Coin, err error)
		stagesProviderGetActualCoinsErr   error
		stagesStorageStore                func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, coins []*entities.Coin, err error)
		stagesStorageStoreErr             error
		stageStorageGetAggregatedCoins    func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, aggregationType string, coins []*entities.Coin, err error)
		stageStorageGetAggregatedCoinsErr error
	}

	tests := []struct {
		name            string
		titles          []string
		existingTitles  []string
		providerTitles  []string
		aggregationType string
		stages          stages
		wantErr         bool
		resErr          error
	}{
		{
			name:            "1",
			titles:          []string{"btc", "eth"},
			existingTitles:  []string{"btc"},
			aggregationType: "SUM",
			stages:          stages{},
			wantErr:         true,
		},
		{
			name:            "2",
			titles:          []string{"btc", "eth"},
			existingTitles:  []string{"btc"},
			aggregationType: "AVG",
			stages: stages{
				stageStorageGetAllTitles:    storageGetAllTitles,
				stageStorageGetAllTitlesErr: errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name:            "3",
			titles:          []string{"btc", "eth"},
			existingTitles:  []string{"btc"},
			providerTitles:  []string{"eth"},
			aggregationType: "AVG",
			stages: stages{
				stageStorageGetAllTitles:        storageGetAllTitles,
				stagesProviderGetActualCoins:    providerGetActualCoins,
				stagesProviderGetActualCoinsErr: errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name:            "4",
			titles:          []string{"btc", "eth"},
			existingTitles:  []string{"btc"},
			providerTitles:  []string{"eth"},
			aggregationType: "AVG",
			stages: stages{
				stageStorageGetAllTitles:     storageGetAllTitles,
				stagesProviderGetActualCoins: providerGetActualCoins,
				stagesStorageStore:           storageStore,
				stagesStorageStoreErr:        errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name:            "5",
			titles:          []string{"btc", "eth"},
			existingTitles:  []string{"btc"},
			providerTitles:  []string{"eth"},
			aggregationType: "MAX",
			stages: stages{
				stageStorageGetAllTitles:          storageGetAllTitles,
				stagesProviderGetActualCoins:      providerGetActualCoins,
				stagesStorageStore:                storageStore,
				stageStorageGetAggregatedCoins:    storageGetAggregatedCoins,
				stageStorageGetAggregatedCoinsErr: errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name:            "6",
			titles:          []string{"btc", "eth"},
			existingTitles:  []string{"btc", "eth"},
			aggregationType: "MAX",
			stages: stages{
				stageStorageGetAllTitles:       storageGetAllTitles,
				stageStorageGetAggregatedCoins: storageGetAggregatedCoins,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			t.Cleanup(func() {
				ctrl.Finish()
			})

			mockProvider := mocks.NewMockCryptoProvider(ctrl)
			mockStorage := mocks.NewMockStorage(ctrl)

			service, err := cases.NewService(mockProvider, mockStorage)
			require.NoError(it, err)
			require.NotNil(it, service)

			ctx := context.Background()
			btcTitle := "btc"
			ethTitle := "eth"

			eth, err := entities.NewCoin(ethTitle, 100, time.Now())
			require.NoError(it, err)

			btc, err := entities.NewCoin(btcTitle, 200, time.Now())
			require.NoError(it, err)

			if tc.stages.stageStorageGetAllTitles != nil {
				tc.stages.stageStorageGetAllTitles(ctx, it, mockStorage, tc.existingTitles, tc.stages.stageStorageGetAllTitlesErr)
			}

			if tc.stages.stagesProviderGetActualCoins != nil {
				tc.stages.stagesProviderGetActualCoins(ctx, it, mockProvider, tc.providerTitles, []*entities.Coin{eth}, tc.stages.stagesProviderGetActualCoinsErr)
			}

			if tc.stages.stagesStorageStore != nil {
				tc.stages.stagesStorageStore(ctx, it, mockStorage, []*entities.Coin{eth}, tc.stages.stagesStorageStoreErr)
			}

			if tc.stages.stageStorageGetAggregatedCoins != nil {
				tc.stages.stageStorageGetAggregatedCoins(ctx, it, mockStorage, tc.titles, tc.aggregationType, []*entities.Coin{btc, eth}, tc.stages.stageStorageGetAggregatedCoinsErr)
			}

			coins, err := service.GetAggregatedCoins(ctx, tc.titles, tc.aggregationType)
			if tc.wantErr {
				require.Nil(it, coins)

				if tc.aggregationType == "SUM" {
					require.EqualError(it, err, "invalid aggregation type")
				} else {
					require.ErrorIs(it, err, tc.resErr)
				}

				return
			}

			require.NoError(it, err)
			require.Equal(it, []*entities.Coin{btc, eth}, coins)

		})
	}

}

func TestService_ActualizeCoins(t *testing.T) {
	t.Parallel()

	type stages struct {
		stageStorageGetAllTitles        func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, err error)
		stageStorageGetAllTitlesErr     error
		stagesStorageStore              func(ctx context.Context, t *testing.T, storage *mocks.MockStorage, coins []*entities.Coin, err error)
		stagesStorageStoreErr           error
		stagesProviderGetActualCoins    func(ctx context.Context, t *testing.T, provider *mocks.MockCryptoProvider, titles []string, coins []*entities.Coin, err error)
		stagesProviderGetActualCoinsErr error
	}

	tests := []struct {
		name              string
		titlesFromStorage []string
		stages            stages
		wantErr           bool
		resErr            error
	}{
		{
			name:              "1",
			titlesFromStorage: nil,
			stages: stages{
				stageStorageGetAllTitles:    storageGetAllTitles,
				stageStorageGetAllTitlesErr: errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name:              "2",
			titlesFromStorage: []string{},
			stages: stages{
				stageStorageGetAllTitles: storageGetAllTitles,
			},
			wantErr: false,
		},
		{
			name:              "3",
			titlesFromStorage: []string{"BTC"},
			stages: stages{
				stageStorageGetAllTitles:        storageGetAllTitles,
				stagesProviderGetActualCoins:    providerGetActualCoins,
				stagesProviderGetActualCoinsErr: errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name:              "4",
			titlesFromStorage: []string{"BTC"},
			stages: stages{
				stageStorageGetAllTitles:     storageGetAllTitles,
				stagesProviderGetActualCoins: providerGetActualCoins,
				stagesStorageStore:           storageStore,
				stagesStorageStoreErr:        errtest,
			},
			wantErr: true,
			resErr:  errtest,
		},
		{
			name:              "5",
			titlesFromStorage: []string{"BTC"},
			stages: stages{
				stageStorageGetAllTitles:     storageGetAllTitles,
				stagesProviderGetActualCoins: providerGetActualCoins,
				stagesStorageStore:           storageStore,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			ctrl := gomock.NewController(it)
			it.Cleanup(func() {
				ctrl.Finish()
			})

			mockStorage := mocks.NewMockStorage(ctrl)
			mockProvider := mocks.NewMockCryptoProvider(ctrl)

			service, err := cases.NewService(mockProvider, mockStorage)
			require.NoError(it, err)
			require.NotNil(it, service)

			ctx := context.Background()
			btcTitle := "BTC"

			btc, err := entities.NewCoin(btcTitle, 100, time.Now())
			require.NoError(it, err)

			if tc.stages.stageStorageGetAllTitles != nil {
				tc.stages.stageStorageGetAllTitles(ctx, it, mockStorage, tc.titlesFromStorage, tc.stages.stageStorageGetAllTitlesErr)
			}

			if tc.stages.stagesProviderGetActualCoins != nil {
				tc.stages.stagesProviderGetActualCoins(ctx, it, mockProvider, []string{btcTitle}, []*entities.Coin{btc}, tc.stages.stagesProviderGetActualCoinsErr)
			}

			if tc.stages.stagesStorageStore != nil {
				tc.stages.stagesStorageStore(ctx, it, mockStorage, []*entities.Coin{btc}, tc.stages.stagesStorageStoreErr)
			}

			err = service.ActualizeCoins(ctx)
			if tc.wantErr {
				require.Error(it, err)
				require.ErrorIs(it, err, tc.resErr)
				return
			}

			require.NoError(it, err)

		})
	}

}

func storageGetAllTitles(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, err error) {
	t.Helper()

	storage.EXPECT().GetAllTitles(ctx).Return(titles, err)
}

func storageStore(ctx context.Context, t *testing.T, storage *mocks.MockStorage, coins []*entities.Coin, err error) {
	t.Helper()

	storage.EXPECT().Store(ctx, coins).Return(err)
}

func storageGetLastCoins(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, coins []*entities.Coin, err error) {
	t.Helper()

	storage.EXPECT().GetLastCoins(ctx, titles).Return(coins, err)
}

func storageGetAggregatedCoins(ctx context.Context, t *testing.T, storage *mocks.MockStorage, titles []string, aggregationType string, coins []*entities.Coin, err error) {
	t.Helper()

	storage.EXPECT().GetAggregatedCoins(ctx, titles, aggregationType).Return(coins, err)
}

func providerGetActualCoins(ctx context.Context, t *testing.T, provider *mocks.MockCryptoProvider, titles []string, coins []*entities.Coin, err error) {
	t.Helper()

	provider.EXPECT().GetActualCoins(ctx, titles).Return(coins, err)
}
