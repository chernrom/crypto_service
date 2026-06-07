package cases

import (
	"context"
	"log/slog"
	"slices"

	"github.com/pkg/errors"

	"crypto_service/internal/entities"
)

type Service struct {
	provider CryptoProvider
	storage  Storage
}

func NewService(provider CryptoProvider, storage Storage) (*Service, error) {
	if provider == nil {
		slog.Error("new service failed", "error", entities.ErrInvalidParam, "reason", "provider is nil")
		return nil, errors.Wrap(entities.ErrInvalidParam, "provider is nil")
	}

	if storage == nil {
		return nil, errors.Wrap(entities.ErrInvalidParam, "storage is nil")
	}

	return &Service{
		provider: provider,
		storage:  storage,
	}, nil
}

func (s *Service) ensureCoinsExist(ctx context.Context, titles []string) error {
	existingTitles, err := s.storage.GetAllTitles(ctx)
	if err != nil {
		slog.Error(
			"get all titles failed",
			"error", err,
			"titles", titles,
		)

		return errors.Wrap(err, "get all titles failure")
	}

	missingTitles := make([]string, 0)

	for _, title := range titles {
		if !slices.Contains(existingTitles, title) {
			missingTitles = append(missingTitles, title)
		}
	}

	if len(missingTitles) > 0 {
		missingCoins, err := s.provider.GetActualCoins(ctx, missingTitles)
		if err != nil {
			slog.Error(
				"get actual coins failed",
				"error", err,
				"titles", missingTitles,
			)

			return errors.Wrap(err, "get actual rates failure")
		}

		if err := s.storage.Store(ctx, missingCoins); err != nil {
			slog.Error(
				"store failed",
				"error", err,
				"coins", missingCoins,
			)

			return errors.Wrap(err, "store failure")
		}
	}
	return nil
}

func (s *Service) GetCoins(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	err := s.ensureCoinsExist(ctx, titles)
	if err != nil {
		slog.Error("ensure coins exist failed", "error", err, "titles", titles)
		return nil, errors.Wrap(err, "ensure coins exist failure")
	}

	coins, err := s.storage.GetCoinsByTitles(ctx, titles)
	if err != nil {
		slog.Error(
			"get coins by titles failed",
			"error", err,
			"titles", titles,
		)

		return nil, errors.Wrap(err, "get last coins failure")
	}

	return coins, nil
}

func (s *Service) GetAggregatedCoins(
	ctx context.Context,
	titles []string,
	aggregate entities.Aggregate) ([]*entities.Coin, error) {

	err := s.ensureCoinsExist(ctx, titles)
	if err != nil {
		slog.Error("ensure coins exist failed", "error", err, "titles", titles, "aggregate", aggregate)
		return nil, errors.Wrap(err, "ensure coins exist failure")
	}

	aggregatedCoins, err := s.storage.GetAggregatedCoins(ctx, titles, aggregate)
	if err != nil {
		slog.Error(
			"get aggregated coins failed",
			"error", err,
			"titles", titles,
			"aggregate", aggregate,
		)

		return nil, errors.Wrap(err, "get aggregated coins failure")
	}

	return aggregatedCoins, nil
}

func (s *Service) ActualizeCoins(ctx context.Context) error {
	titles, err := s.storage.GetAllTitles(ctx)
	if err != nil {
		slog.Error(
			"get all titles failed",
			"error", err,
			"titles", titles,
		)

		return errors.Wrap(err, "get all titles failure")
	}

	if len(titles) == 0 {
		return nil
	}

	actualCoins, err := s.provider.GetActualCoins(ctx, titles)
	if err != nil {
		slog.Error(
			"get actual coins failed",
			"error", err,
			"titles", titles,
		)

		return errors.Wrap(err, "get actual coins failure")
	}

	if err := s.storage.Store(ctx, actualCoins); err != nil {
		slog.Error(
			"store failed",
			"error", err,
			"coins", actualCoins,
		)

		return errors.Wrap(err, "store failure")
	}

	return nil
}
