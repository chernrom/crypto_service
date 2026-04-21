package postgres_test

import (
	"context"
	"crypto_service/internal/adapters/postgres"
	"crypto_service/internal/entities"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var (
	connStr = "postgres://postgres:postgres@localhost:5432/app?sslmode=disable"
)

func makeDb(t *testing.T) *postgres.PostgresStorage {
	t.Helper()

	storage, err := postgres.NewPostgresStorage(connStr)
	require.NoError(t, err)
	return storage
}

func flushStorage(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	query := `DELETE FROM crypto.coins;`
	_, err = pool.Exec(ctx, query)
	require.NoError(t, err)
}

func Test_GetAllTitles(t *testing.T) {
	t.Parallel()

	defer flushStorage(t)

	btc1, err := entities.NewCoin("btc", 0.13, time.Now())
	require.NoError(t, err)
	btc2, err := entities.NewCoin("btc", 0.566, time.Now().Add(1*time.Second))
	require.NoError(t, err)
	eth1, err := entities.NewCoin("eth", 0.22, time.Now())
	require.NoError(t, err)
	eth2, err := entities.NewCoin("eth", 0.321, time.Now().Add(1*time.Second))
	require.NoError(t, err)

	ctx := context.Background()
	db := makeDb(t)
	err = db.Store(ctx, []*entities.Coin{btc1, btc2, eth1, eth2})
	require.NoError(t, err)

	titles, err := db.GetAllTitles(ctx)
	require.NoError(t, err)

	require.Equal(t, 2, len(titles))
	require.ElementsMatch(t, []string{"btc", "eth"}, titles)

}

func Test_GetCoinsByTitles(t *testing.T) {
	t.Parallel()

	defer flushStorage(t)

	btc1, err := entities.NewCoin("btc", 0.13, time.Now())
	require.NoError(t, err)
	btc2, err := entities.NewCoin("btc", 0.566, time.Now().Add(1*time.Second))
	require.NoError(t, err)
	eth1, err := entities.NewCoin("eth", 0.22, time.Now())
	require.NoError(t, err)
	eth2, err := entities.NewCoin("eth", 0.321, time.Now().Add(1*time.Second))
	require.NoError(t, err)

	ctx := context.Background()
	db := makeDb(t)
	err = db.Store(ctx, []*entities.Coin{btc1, btc2, eth1, eth2})
	require.NoError(t, err)

	coins, err := db.GetCoinsByTitles(ctx, []string{btc1.Title(), btc2.Title(), eth1.Title(), eth2.Title()})
	require.NoError(t, err)

	var titles []string
	for _, coin := range coins {
		titles = append(titles, coin.Title())
	}

	require.Equal(t, 2, len(coins))
	require.ElementsMatch(t, []string{"btc", "eth"}, titles)

	var costs []float64
	for _, c := range coins {
		costs = append(costs, c.Cost())
	}
	require.ElementsMatch(t, []float64{0.566, 0.321}, costs)
}
