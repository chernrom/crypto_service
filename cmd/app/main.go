package main

import (
	"context"
	"crypto_service/internal/adapters/provider/coingecko"
	"fmt"
	"log/slog"
	"os"
)

func main() {

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
