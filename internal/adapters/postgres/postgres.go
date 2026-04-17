package postgres

import (
	"context"
	"crypto_service/internal/entities"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
	once sync.Once
}

func NewPostgresStorage(ctx context.Context, connString string) (*PostgresStorage, error) {
	if connString == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "empty connect field")
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, errors.Wrap(err, "pool creation error")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.Wrap(entities.ErrInternal, "pgxpool ping error")
	}

	return &PostgresStorage{
		pool: pool,
	}, nil
}

func (s *PostgresStorage) Close() {
	s.once.Do(
		func() {
			s.pool.Close()
		},
	)
}

func (s *PostgresStorage) GetAllTitles(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, "SELECT title FROM crypto.coins;")
	if err != nil {
		return nil, errors.Wrap(err, "query titles error")
	}
	defer rows.Close()

	titles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows error")
	}

	return titles, nil
}

func (s *PostgresStorage) Store(ctx context.Context, coins []*entities.Coin) error {
	if len(coins) == 0 {
		return nil
	}

	inputRows := [][]any{}
	for _, coin := range coins {
		inputRows = append(inputRows, []any{coin.Title, coin.Cost, coin.ActualAt})
	}

	copyFromResult, err := s.pool.CopyFrom(
		ctx,
		pgx.Identifier{"crypto", "coins"}, //вот тут вопрос crypto.coins или coins?
		[]string{"title", "cost", "actual_at"},
		pgx.CopyFromRows(inputRows),
	)
	if int(copyFromResult) != len(coins) {
		return errors.Errorf("expected to insert %d rows, got %d", len(coins), copyFromResult)
	}
	if err != nil {
		return errors.Wrap(err, "copy from error")
	}

	return nil
}

func (s *PostgresStorage) GetCoinsByTitles(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	if len(titles) == 0 {
		return []*entities.Coin{}, nil
	}

	rows, err := s.pool.Query(ctx, "SELECT title, cost, actual_at FROM crypto.coins WHERE title = ANY($1);", titles)
	if err != nil {
		return nil, errors.Wrap(err, "query titles error")
	}
	defer rows.Close()

	coinPointers, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[entities.Coin])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows error")
	}

	return coinPointers, nil
}

func (s *PostgresStorage) GetAggregatedCoins(ctx context.Context, titles []string, aggregationType string) ([]*entities.Coin, error) {
	if len(titles) == 0 {
		return nil, errors.New("titles is empty")
	}

	if aggregationType == "" {
		return nil, errors.New("aggregation type is empty")
	}

	return nil, nil

}
