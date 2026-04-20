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
	pool     *pgxpool.Pool
	once     sync.Once
	cancelFn context.CancelFunc
}

func NewPostgresStorage(connString string) (*PostgresStorage, error) {
	if connString == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "empty connect field")
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		cancel()
		return nil, errors.Wrapf(entities.ErrInternal, "pool creation error: %v", err)
	}

	return &PostgresStorage{
		pool:     pool,
		cancelFn: cancel,
	}, nil
}

func (s *PostgresStorage) Close() {
	s.once.Do(func() {
		s.cancelFn()
	})
}

func (s *PostgresStorage) GetAllTitles(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, "SELECT DISTINCT title FROM crypto.coins;")
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
	query := `INSERT INTO crypto.coins (title, cost, actual_at) VALUES ($1, $2, $3)`
	batch := &pgx.Batch{}

	for _, coin := range coins {
		batch.Queue(query, coin.Title(), coin.Cost(), coin.ActualAt())
	}

	batchRes := s.pool.SendBatch(ctx, batch)
	for range coins {
		if _, err := batchRes.Exec(); err != nil {
			return errors.Wrapf(entities.ErrInternal, "batch error: %v", err)
		}
	}

	if err := batchRes.Close(); err != nil {
		return errors.Wrapf(entities.ErrInternal, "batch close error: %v", err)
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
