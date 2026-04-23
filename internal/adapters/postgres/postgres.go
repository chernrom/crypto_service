package postgres

import (
	"context"
	"crypto_service/internal/entities"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

const (
	min = "MIN"
	max = "MAX"
	avg = "AVG"
)

type PostgresStorage struct {
	pool     *pgxpool.Pool
	once     sync.Once
	cancelFn context.CancelFunc
}

type CoinRowDTO struct {
	Title    string
	Cost     float64
	ActualAt time.Time
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

	rows, err := s.pool.Query(ctx, "SELECT DISTINCT ON (title) title, cost, actual_at FROM crypto.coins WHERE title = ANY($1) ORDER BY title, actual_at DESC;", titles)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "query titles error: %v", err)
	}
	defer rows.Close()

	dtoList, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[CoinRowDTO])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows error")
	}

	var coins []*entities.Coin
	for _, dto := range dtoList {
		entity, err := entities.NewCoin(dto.Title, dto.Cost, dto.ActualAt)
		if err != nil {
			return nil, err
		}

		coins = append(coins, entity)
	}

	return coins, nil
}

func (s *PostgresStorage) GetAggregatedCoins(ctx context.Context, titles []string, aggregationType string) ([]*entities.Coin, error) {
	if len(titles) == 0 {
		return []*entities.Coin{}, nil
	}

	if aggregationType == "" {
		return []*entities.Coin{}, nil
	}

	validAggs := map[string]string{
		"avg": "AVG",
		"max": "MAX",
		"min": "MIN",
	}

	sqlFunc, ok := validAggs[aggregationType]
	if !ok {
		return nil, errors.New("invalid aggregation type")
	}

	query := fmt.Sprintf(`SELECT title, %s(cost)::float as cost FROM crypto.coins WHERE title = ANY($1) GROUP BY title`, sqlFunc)

	rows, err := s.pool.Query(ctx, query, titles)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "query titles error: %v", err)
	}
	defer rows.Close()

	dtoList, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[CoinRowDTO])
	if err != nil {
		return nil, errors.Wrap(err, "collect rows error")
	}

	var coins []*entities.Coin
	for _, dto := range dtoList {
		entity, err := entities.NewCoin(dto.Title, dto.Cost, time.Now())
		if err != nil {
			return nil, err
		}

		coins = append(coins, entity)
	}

	return coins, nil
}

func (s *PostgresStorage) GetCoinsWithAggregation(ctx context.Context, titles []string, aggregationType string) ([]*entities.Coin, error) {
	var aggFunc string
	switch strings.ToUpper(aggregationType) {
	case min, max, avg:
		aggFunc = strings.ToUpper(aggregationType)
	default:
		err := errors.Wrapf(entities.ErrInvalidParam, "invalid aggregation type: %v", aggFunc)
		return nil, err
	}

	query := `SELECT c.title, ` + aggFunc + `(c.cost) AS cost, CURRENT_DATE AS actual_at
	FROM crypto.coins c
	WHERE c.title = ANY($1)
	AND DATE(c.actual_at) = CURRENT_DATE
	GROUP BY c.title
	ORDER BY c.title`
	params := []any{
		titles,
	}

	rows, err := s.pool.Query(ctx, query, params...)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "query titles error: %v", err)
	}
	defer rows.Close()

	coins := make([]*entities.Coin, 0)

	for rows.Next() {
		var (
			title    string
			cost     float64
			actualAt time.Time
		)
		if err := rows.Scan(&title, &cost, &actualAt); err != nil {
			return nil, errors.Wrapf(entities.ErrInternal, "scan error: %v", err)
		}

		coin, err := entities.NewCoin(title, cost, actualAt)
		if err != nil {
			return nil, errors.Wrap(err, "new coin error")
		}
		coins = append(coins, coin)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "rows err error: %v", err)
	}

	if len(coins) == 0 {
		return nil, errors.Wrap(entities.ErrNotFound, "required titles not found")
	}

	return coins, nil
}
