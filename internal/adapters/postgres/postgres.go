package postgres

import (
	"context"
	"crypto_service/internal/entities"
	"strings"
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
	rowsInLine := 3
	values := make([]string,0)
	args := make([]any,0,len(coins)*rowsInLine)

	for i, coin := range coins{
		args = append(args, coin.Title,coin.Cost,coin.ActualAt)
		values = append(values, "($1, $2, $3)")
		strings.Join(values,", ")
	}

	

	query := `
		INSERT INTO crypto.coins (title, cost, actual_at)
		VALUES ($1, $2, $3)
	`
	store, err := s.pool.Exec(ctx, query, ...args)

	return nil
}
