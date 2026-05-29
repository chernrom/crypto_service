package application

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"

	"crypto_service/internal/adapters/config"
	"crypto_service/internal/adapters/postgres"
	"crypto_service/internal/adapters/provider/coingecko"
	"crypto_service/internal/cases"
	"crypto_service/internal/entities"
	"crypto_service/internal/port"
	"crypto_service/internal/port/http/public"
)

type App struct {
	cfg *config.Config

	server *public.Server

	provider cases.CryptoProvider
	storage  cases.Storage
	service  port.ServiceProvider
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

	app.startAndAwait()
}

func (app *App) startAndAwait() {
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
		slog.Error("public port shutdown error")
		app.panic(err)
	}
	slog.Info("application stopped succesfully")
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

func (app *App) panic(err error) {
	panic(err)
}
