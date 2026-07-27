package cases

import (
	"context"
	"crypto_service/internal/entities"
	"log/slog"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const (
	getCoinsOperation           = "get_coins"
	getAggregatedCoinsOperation = "get_aggregated_coins"
)

type CachedService struct {
	service ServiceProvider
	cache   CacheProvider
}

func NewCachedService(service ServiceProvider, cache CacheProvider) (*CachedService, error) {
	if service == nil {
		slog.Error("service provider failed", "error", entities.ErrInvalidParam, "reason", "service provider is nil")
		return nil, errors.Wrap(entities.ErrInvalidParam, "service provider is nil")
	}

	if cache == nil {
		slog.Error("cache provider failed", "error", entities.ErrInvalidParam, "reason", "cache provider is nil")
		return nil, errors.Wrap(entities.ErrInvalidParam, "cache provider is nil")
	}

	return &CachedService{
		service: service,
		cache:   cache,
	}, nil
}

func (c *CachedService) ActualizeCoins(ctx context.Context) error {
	err := c.service.ActualizeCoins(ctx)
	if err != nil {
		return err
	}
	c.cache.Flush(ctx)
	return nil
}

func (c *CachedService) GetAggregatedCoins(ctx context.Context, titles []string, aggregate entities.Aggregate) ([]*entities.Coin, error) {
	coins, err := c.cache.GetCoins(ctx, "get_aggregated_coins"+string(aggregate), titles)
	if err != nil {
		slog.Error("cache error", "err", err)
	}
	if coins != nil {
		return coins, nil
	}

	savedCoins, err := c.service.GetAggregatedCoins(ctx, titles, aggregate)
	if err != nil {
		return nil, err
	}
	c.cache.Store(ctx, "get_aggregated_coins"+string(aggregate), strings.Join(titles, ":"), savedCoins)

	return savedCoins, nil
}

func (c *CachedService) GetCoins(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	if len(titles) == 0 {
		err := errors.Wrap(entities.ErrInvalidParam, "titles is empty")
		slog.Error("get cached coins failed", "error", err)
		return nil, err
	}

	cacheTitles := normalizeTitles(titles)

	coins, err := c.cache.GetCoins(
		ctx,
		getCoinsOperation,
		cacheTitles,
	)
	if err != nil {
		// Redis не должен ломать основной запрос.
		slog.Error(
			"get coins from cache failed",
			"error", err,
			"titles", cacheTitles,
		)
	}
	if coins != nil {
		return coins, nil
	}

	coins, err = c.service.GetCoins(ctx, titles)
	if err != nil {
		return nil, err
	}

	cacheKey := strings.Join(cacheTitles, ":")

	if err := c.cache.Store(
		ctx,
		getCoinsOperation,
		cacheKey,
		coins,
	); err != nil {
		slog.Error(
			"store coins in cache failed",
			"error", err,
			"titles", cacheTitles,
		)
	}

	return coins, nil
}

func normalizeTitles(titles []string) []string {
	result := append([]string(nil), titles...)
	sort.Strings(result)

	return result
}
