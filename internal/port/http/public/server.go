package public

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"

	"crypto_service/internal/entities"
	"crypto_service/internal/port"
	"crypto_service/pkg/dto"
)

type Aggregate string

const (
	basePath       = "/crypto/v1"
	ratesPath      = "/rates"
	aggregatedPath = "/aggregated"

	AggregateAvg Aggregate = "avg"
	AggregateMin Aggregate = "min"
	AggregateMax Aggregate = "max"

	aggregateQueryParam = "aggregate"
)

type Server struct {
	router  *http.Server
	service port.ServiceProvider
}

func NewServer(service port.ServiceProvider, port string, timeout time.Duration) (*Server, error) {
	switch {
	case service == nil:
		slog.Error("new server failed", "error", entities.ErrInvalidParam, "reason", "service is nil")
		return nil, fmt.Errorf("new server: service is nil: %w", entities.ErrInvalidParam)
	case port == "":
		slog.Error("new server failed", "error", entities.ErrInvalidParam, "reason", "port is empty")
		return nil, fmt.Errorf("new server: port is empty: %w", entities.ErrInvalidParam)
	case timeout <= 0:
		slog.Error("new server failed", "error", entities.ErrInvalidParam, "reason", "timeout must be greater than zero")
		return nil, fmt.Errorf("new server: timeout must be greater than zero: %w", entities.ErrInvalidParam)
	}

	return &Server{
		router: &http.Server{
			ReadTimeout:  timeout,
			WriteTimeout: timeout,
			Addr:         fmt.Sprintf(":%s", port),
		},
		service: service,
	}, nil
}

func (s *Server) Start() error {
	s.registerRoutes()

	go func() {
		if err := s.router.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "error", err)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if err := s.router.Shutdown(ctx); err != nil {
		slog.Error("server shutdown failed", "error", err)
		return err
	}
	return nil
}

func (s *Server) registerRoutes() {
	router := chi.NewRouter()
	router.Use(s.timeoutMiddleware)
	router.Post(fmt.Sprintf("%s%s", basePath, ratesPath), s.actualRates)
	router.Post(fmt.Sprintf("%s%s%s", basePath, ratesPath, aggregatedPath), s.aggregatedRates)
	s.router.Handler = router
}

func (s *Server) actualRates(resp http.ResponseWriter, req *http.Request) {
	slog.Info("requested actual rates")

	resp.Header().Set("Content-Type", "application/json")

	var titlesDTO dto.TitlesDTO
	err := json.NewDecoder(req.Body).Decode(&titlesDTO)

	if err != nil {
		slog.Error("failed to decode request body", "error", err)

		wrappedErr := errors.Wrapf(
			entities.ErrInvalidParam,
			"failed to decode request body: %v",
			err,
		)

		s.errProcessing(wrappedErr, resp)
		return
	}

	coins, err := s.service.GetCoins(req.Context(), titlesDTO.Titles)
	if err != nil {
		slog.Error("get coins failed", "error", err)
		s.errProcessing(err, resp)
		return
	}

	s.coinsProcessing(coins, resp)
}

func (s *Server) aggregatedRates(resp http.ResponseWriter, req *http.Request) {
	slog.Info("requested aggregated rates")

	resp.Header().Set("Content-Type", "application/json")

	rawAggregate := req.URL.Query().Get(aggregateQueryParam)
	parsedAggregate, err := parseAggregate(rawAggregate)
	if err != nil {
		slog.Error("invalid aggregate query param", "aggregate", rawAggregate, "error", err)
		s.errProcessing(err, resp)
		return
	}

	var titlesDTO dto.TitlesDTO
	err = json.NewDecoder(req.Body).Decode(&titlesDTO)
	if err != nil {
		slog.Error("failed to decode request body", "error", err)

		wrappedErr := errors.Wrapf(
			entities.ErrInvalidParam,
			"failed to decode request body: %v",
			err,
		)

		s.errProcessing(wrappedErr, resp)
		return
	}

	coins, err := s.service.GetAggregatedCoins(req.Context(), titlesDTO.Titles, parsedAggregate)
	if err != nil {
		slog.Error("failed to get aggregated coins", "error", err)
		s.errProcessing(err, resp)
		return
	}
	s.coinsProcessing(coins, resp)
}

func (s *Server) coinsProcessing(coins []*entities.Coin, resp http.ResponseWriter) {
	coinsDTO := dto.CoinsDTO{
		Coins: make([]dto.CoinDTO, 0, len(coins)),
	}

	for i, coin := range coins {
		if coin == nil {
			slog.Error("coin is nil", "index", i)
			continue
		}
		coinsDTO.Coins = append(coinsDTO.Coins, dto.CoinDTO{
			Title:    coin.Title(),
			Cost:     coin.Cost(),
			ActualAt: coin.ActualAt(),
		})
	}

	data, err := json.Marshal(&coinsDTO)
	if err != nil {
		slog.Error("failed to marshal coins response", "error", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err := resp.Write(data); err != nil {
		slog.Error("failed to write response", "error", err)
		return
	}
}

func parseAggregate(raw string) (entities.Aggregate, error) {
	normal := strings.ToLower(raw)

	switch normal {
	case string(AggregateMin), string(AggregateMax), string(AggregateAvg):
		return entities.Aggregate(normal), nil
	default:
		return "", errors.Wrapf(entities.ErrInvalidParam, "invalid aggregation type: %s", raw)
	}
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), s.router.WriteTimeout)
		defer cancel()
		req = req.WithContext(ctx)
		next.ServeHTTP(resp, req)
	})

}

func (s *Server) errProcessing(err error, resp http.ResponseWriter) {
	statusCode := http.StatusInternalServerError
	errDTO := dto.ErrorDTO{
		Message:    err.Error(),
		StatusCode: statusCode,
	}
	switch {
	case errors.Is(err, entities.ErrInternal):
		errDTO.StatusCode = http.StatusInternalServerError
	case errors.Is(err, entities.ErrNotFound):
		errDTO.StatusCode = http.StatusNotFound
	case errors.Is(err, entities.ErrInvalidParam):
		errDTO.StatusCode = http.StatusBadRequest
	default:
		errDTO.StatusCode = http.StatusInternalServerError
	}

	data, err := json.Marshal(&errDTO)
	if err != nil {
		slog.Error("failed to marshal error response", "error", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(errDTO.StatusCode)
	if _, err := resp.Write(data); err != nil {
		slog.Error("failed to write response", "error", err)
		return
	}
}
