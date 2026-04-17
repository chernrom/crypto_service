package main

import (
	"context"
	"crypto_service/internal/adapters/provider/coingecko"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

const connStr = "postgres://postgres:postgres@localhost:5432/app?sslmode=disable"

func main() {
	var ctx context.Context
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	c, err := coingecko.NewClient("CG-oBGrtsF1bRVUg3RFoWw8uf16")
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	coins, err := c.GetActualCoins(context.Background(), []string{"btc", "eth"})
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	for _, coin := range coins {
		fmt.Println(coin.Title(), coin.Cost(), coin.ActualAt())
	}

}
