package postgres

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"crypto_service/internal/entities"
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
		slog.Error("new postgres storage failed", "error", entities.ErrInvalidParam, "reason", "empty conn string")
		return nil, errors.Wrap(entities.ErrInvalidParam, "empty connect field")
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		slog.Error("postgres pool creation failed", "error", err)
		cancel()
		return nil, errors.Wrapf(entities.ErrInternal, "pool creation error: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		slog.Error("postgres ping failed", "error", err)
		pool.Close()
		cancel()
		return nil, errors.Wrapf(entities.ErrInternal, "postgres ping error: %v", err)
	}

	return &PostgresStorage{
		pool:     pool,
		cancelFn: cancel,
	}, nil
}

func (s *PostgresStorage) Close() {
	if s == nil {
		return
	}

	s.once.Do(func() {
		if s.cancelFn != nil {
			s.cancelFn()
		}
		if s.pool != nil {
			s.pool.Close()
		}
	})
}

func (s *PostgresStorage) GetAllTitles(ctx context.Context) ([]string, error) {
	if s == nil || s.pool == nil {
		slog.Error("postgres storage is nil", "error", entities.ErrInternal)
		return nil, errors.Wrap(entities.ErrInternal, "postgres storage is nil")
	}

	rows, err := s.pool.Query(ctx, "SELECT DISTINCT title FROM crypto.coins;")
	if err != nil {
		slog.Error("query all titles failed", "error", err)
		return nil, errors.Wrap(err, "query titles error")
	}
	defer rows.Close()

	titles, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		slog.Error("collect title rows failed", "error", err)
		return nil, errors.Wrap(err, "collect rows error")
	}

	return titles, nil
}

func (s *PostgresStorage) Store(ctx context.Context, coins []*entities.Coin) error {
	if s == nil || s.pool == nil {
		slog.Error("postgres storage is nil", "error", entities.ErrInternal)
		return errors.Wrap(entities.ErrInternal, "postgres storage is nil")
	}

	if len(coins) == 0 {
		slog.Info("no coins to store")
		return nil
	}
	query := `INSERT INTO crypto.coins (title, cost, actual_at) VALUES ($1, $2, $3)`
	batch := &pgx.Batch{}

	for _, coin := range coins {
		if coin == nil {
			slog.Error("coin is nil", "error", entities.ErrInvalidParam)
			return errors.Wrap(entities.ErrInvalidParam, "coin is nil")
		}
		batch.Queue(query, coin.Title(), coin.Cost(), coin.ActualAt())
	}

	batchRes := s.pool.SendBatch(ctx, batch)
	if batchRes == nil {
		slog.Error("batch result is nil", "error", entities.ErrInternal)
		return errors.Wrap(entities.ErrInternal, "batch result is nil")
	}

	defer func() {
		if err := batchRes.Close(); err != nil {
			slog.Error("batch close failed", "error", err)
		}
	}()

	for range coins {
		if _, err := batchRes.Exec(); err != nil {
			slog.Error("batch exec failed", "error", err)
			return errors.Wrapf(entities.ErrInternal, "batch error: %v", err)
		}
	}

	return nil
}

func (s *PostgresStorage) GetCoinsByTitles(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	if s == nil || s.pool == nil {
		slog.Error("postgres storage is nil", "error", entities.ErrInternal)
		return nil, errors.Wrap(entities.ErrInternal, "postgres storage is nil")
	}

	if len(titles) == 0 {
		slog.Info("empty titles, skip getting coins")
		return []*entities.Coin{}, nil
	}

	query := `
	SELECT DISTINCT ON (title)
		title,
		cost,
		actual_at
	FROM crypto.coins
	WHERE title = ANY($1)
	ORDER BY title, actual_at DESC;
	`

	rows, err := s.pool.Query(ctx, query, titles)
	if err != nil {
		slog.Error("query coins by titles failed", "error", err, "titles", titles)
		return nil, errors.Wrapf(entities.ErrInternal, "query titles error: %v", err)
	}
	defer rows.Close()

	dtoList, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[CoinRowDTO])
	if err != nil {
		slog.Error("collect coin rows failed", "error", err, "titles", titles)
		return nil, errors.Wrap(err, "collect rows error")
	}

	coins := make([]*entities.Coin, 0, len(dtoList))
	if len(dtoList) == 0 {
		slog.Error("coins not found", "error", entities.ErrNotFound, "titles", titles)
		return nil, errors.Wrap(entities.ErrNotFound, "coins not found")
	}

	for _, dto := range dtoList {
		entity, err := entities.NewCoin(dto.Title, dto.Cost, dto.ActualAt)
		if err != nil {
			slog.Error("new coin failed", "error", err, "title", dto.Title, "cost", dto.Cost)
			return nil, errors.Wrap(err, "new coin error")
		}

		coins = append(coins, entity)
	}

	return coins, nil
}

func (s *PostgresStorage) GetAggregatedCoins(
	ctx context.Context,
	titles []string,
	aggregate entities.Aggregate) ([]*entities.Coin, error) {
	if s == nil || s.pool == nil {
		slog.Error("postgres storage is nil", "error", entities.ErrInternal)
		return nil, errors.Wrap(entities.ErrInternal, "postgres storage is nil")
	}

	if len(titles) == 0 {
		slog.Error("titles is empty", "error", entities.ErrInvalidParam)
		return nil, errors.Wrap(entities.ErrInvalidParam, "titles is empty")
	}

	var aggFunc string
	switch strings.ToUpper(string(aggregate)) {
	case min, max, avg:
		aggFunc = strings.ToUpper(string(aggregate))
	default:
		slog.Error("invalid aggregation type", "error", entities.ErrInvalidParam, "aggregate", aggregate)
		return nil, errors.Wrapf(entities.ErrInvalidParam, "invalid aggregation type: %v", aggregate)
	}

	query := `SELECT c.title, ` + aggFunc + `(c.cost) AS cost, CURRENT_DATE AS actual_at
	FROM crypto.coins c
	WHERE c.title = ANY($1)
	AND DATE(c.actual_at) = CURRENT_DATE
	GROUP BY c.title
	ORDER BY c.title`

	rows, err := s.pool.Query(ctx, query, titles)
	if err != nil {
		slog.Error("query aggregated coins failed", "error", err, "titles", titles, "aggregate", aggregate)
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
			slog.Error("scan aggregated coin failed", "error", err)
			return nil, errors.Wrapf(entities.ErrInternal, "scan error: %v", err)
		}

		coin, err := entities.NewCoin(title, cost, actualAt)
		if err != nil {
			slog.Error("new coin failed", "error", err, "title", title, "cost", cost)
			return nil, errors.Wrap(err, "new coin error")
		}
		coins = append(coins, coin)
	}

	if err := rows.Err(); err != nil {
		slog.Error("aggregated coin rows error", "error", err)
		return nil, errors.Wrap(err, "collect rows error")
	}

	if len(coins) == 0 {
		slog.Error("aggregated coins not found", "error", entities.ErrNotFound, "titles", titles, "aggregate", aggregate)
		return nil, errors.Wrap(entities.ErrNotFound, "required titles not found")
	}

	return coins, nil
}
