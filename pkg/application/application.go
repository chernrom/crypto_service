package application

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-co-op/gocron/v2"
	"github.com/pkg/errors"

	"crypto_service/internal/adapters/config"
	"crypto_service/internal/adapters/postgres"
	"crypto_service/internal/adapters/provider/coingecko"
	"crypto_service/internal/cases"
	"crypto_service/internal/entities"
	"crypto_service/internal/port"
	"crypto_service/internal/port/http/public"
	"crypto_service/toolkit/tracing"
)

type App struct {
	cfg *config.Config

	server *public.Server

	provider cases.CryptoProvider
	storage  cases.Storage
	service  port.ServiceProvider

	cron gocron.Scheduler
}

func New(cfg *config.Config) *App {
	if cfg == nil {
		panic(errors.Wrap(entities.ErrInvalidParam, "config is not set"))
	}
	return &App{
		cfg: cfg,
	}
}

func (app *App) Start() {
	app.initCryptoProvider()
	app.initStorage()
	app.initService()
	app.initPublicPort()
	app.initCron()

	app.startAndAwait()
}

func (app *App) startAndAwait() {
	app.cron.Start()
	slog.Info("actualize cron job started")

	if err := app.startHTTPPublic(); err != nil {
		app.panic(err)
	}
	slog.Info("public http port started")
	osMon := make(chan os.Signal, 1)
	signal.Notify(osMon, syscall.SIGINT, syscall.SIGTERM)
	<-osMon
	slog.Info("shutdown signal was received")

	ctx, cancel := context.WithTimeout(context.Background(), app.cfg.GetClientTimeout())
	defer cancel()

	if err := app.server.Stop(ctx); err != nil {
		slog.Error("public port shutdown error", "error", err)
		app.panic(err)
	}

	ctx, cancel = context.WithTimeout(context.Background(), app.cfg.GetActualizeIntervalContextTimeout())
	defer cancel()
	if err := app.cron.ShutdownWithContext(ctx); err != nil {
		slog.Error("actualize cron job shutdown error", "error", err)
		app.panic(err)
	}

	slog.Info("application stopped successfully")
}

func (app *App) startHTTPPublic() error {
	if err := app.server.Start(); err != nil {
		return err
	}
	return nil
}

func (app *App) initPublicPort() {
	slog.Info("init public port")

	port := app.cfg.GetPublicHttpPort()
	timeout := app.cfg.GetPublicHttpPortTimeout()
	server, err := public.NewServer(app.service, port, timeout)
	if err != nil {
		app.panic(err)
	}
	app.server = server
}

func (app *App) initService() {
	slog.Info("init service provider")

	service, err := cases.NewService(app.provider, app.storage)
	if err != nil {
		app.panic(err)
	}
	app.service = service
}

func (app *App) initStorage() {
	slog.Info("init storage")

	connStr := app.cfg.GetConnString()
	storage, err := postgres.NewPostgresStorage(connStr)
	if err != nil {
		app.panic(err)
	}
	app.storage = storage
}

func (app *App) initCryptoProvider() {
	slog.Info("crypto provider init")

	token := os.Getenv("COIN_GECKO_TOKEN")
	timeout := app.cfg.GetClientTimeout()
	client, err := coingecko.NewClient(token, coingecko.WithCustomTimeout(timeout))
	if err != nil {
		app.panic(err)
	}
	app.provider = client
}

func (app *App) initCron() {
	s, err := gocron.NewScheduler()
	if err != nil {
		app.panic(err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(
			app.cfg.GetActualizeInterval(),
		),
		gocron.NewTask(func() {
			app.actualizeRates()
		}),
	)
	if err != nil {
		app.panic(err)
	}
	app.cron = s
}

func (app *App) actualizeRates() {
	ctx, span, cancelTrace := tracing.Start(context.Background(), "Application.ActualizeRates")
	defer cancelTrace()

	ctx, cancel := context.WithTimeout(ctx, app.cfg.GetActualizeIntervalContextTimeout())
	defer cancel()

	if err := app.service.ActualizeCoins(ctx); err != nil {
		span.SetError(err)
		slog.Warn("actualize cron job failure", "err", err)
	}
}

func (app *App) panic(err error) {
	slog.Error("application panic", "error", err)
	panic(err)
}
