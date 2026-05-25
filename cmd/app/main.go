package main

import (
	"flag"
	"fmt"

	"crypto_service/internal/adapters/config"
)

func main() {
	var path string

	flag.StringVar(&path, "config", "", "path to configuration file")
	flag.Parse()

	cfg := config.NewConfig(path)
	fmt.Println(cfg.GetConnString())
	//./config/coin.yaml
}
