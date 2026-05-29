package main

import (
	"flag"

	"crypto_service/internal/adapters/config"
	"crypto_service/pkg/application"
)

func main() {
	var path string

	flag.StringVar(&path, "config", "", "path to configuration file")
	flag.Parse()

	cfg := config.NewConfig(path)

	app := application.New(cfg)
	app.Start()
	//./config/coin.yaml
}
