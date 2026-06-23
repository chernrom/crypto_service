//go:build TEST_L2

package l2

import (
	"bytes"
	"context"
	"crypto_service/pkg/dto"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	basePath  = os.Getenv("COIN_BASE_URL")
	ratesPath = os.Getenv("ACTUAL_REQUEST")
)

type TestClient struct {
	*http.Client
}

func NewClient(t *testing.T) *TestClient {
	t.Helper()

	client := &TestClient{
		Client: &http.Client{
			Timeout: time.Second * 5,
		},
	}

	return client
}

func sendRequestToCoin(t *testing.T, aggregatedType string, titles []string) (int, []byte) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	client := NewClient(t)

	requestPath := ratesPath
	if aggregatedType != "" {
		requestPath = fmt.Sprintf("%s/aggregated", ratesPath)
	}

	urlRaw, err := url.Parse(fmt.Sprintf("%s%s", basePath, requestPath))
	require.NoError(t, err)

	if aggregatedType != "" {
		query := urlRaw.Query()
		query.Set("aggregate", aggregatedType)
		urlRaw.RawQuery = query.Encode()
	}

	requestDTO := dto.TitlesDTO{
		Titles: titles,
	}

	data, err := json.Marshal(requestDTO)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlRaw.String(), bytes.NewReader(data))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, body
}

func requireCoinsResponse(t *testing.T, body []byte, titles []string) {
	t.Helper()

	var responseDTO dto.CoinsDTO

	err := json.Unmarshal(body, &responseDTO)
	require.NoError(t, err)

	require.NotEmpty(t, responseDTO.Coins)
	require.Len(t, responseDTO.Coins, len(titles))

	expectedTitles := make(map[string]struct{}, len(titles))

	for _, title := range titles {
		expectedTitles[title] = struct{}{}
	}

	for _, coin := range responseDTO.Coins {
		_, ok := expectedTitles[coin.Title]
		require.Truef(t, ok, "unexpected coin title: %s", coin.Title)

		require.GreaterOrEqual(t, coin.Cost, float64(0))

		delete(expectedTitles, coin.Title)
	}

	require.Empty(t, expectedTitles)
}

func Test_ActualCoinSuccess(t *testing.T) {
	t.Parallel()

	titles := []string{"btc", "eth"}

	statusCode, body := sendRequestToCoin(t, "", titles)

	require.Equal(t, http.StatusOK, statusCode)
	require.NotEmpty(t, body)

	requireCoinsResponse(t, body, titles)
}

func Test_AggregatedCoinSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		aggregateType string
	}{
		{
			name:          "avg",
			aggregateType: "avg",
		},
		{
			name:          "min",
			aggregateType: "min",
		},
		{
			name:          "max",
			aggregateType: "max",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			titles := []string{"btc", "eth"}

			statusCode, body := sendRequestToCoin(t, test.aggregateType, titles)

			require.Equal(t, http.StatusOK, statusCode)
			require.NotEmpty(t, body)

			requireCoinsResponse(t, body, titles)
		})
	}
}

func Test_AggregatedCoinInvalidAggregate(t *testing.T) {
	t.Parallel()

	titles := []string{"btc", "eth"}

	statusCode, body := sendRequestToCoin(t, "sum", titles)

	require.Equal(t, http.StatusBadRequest, statusCode)
	require.NotEmpty(t, body)

	var responseDTO dto.ErrorDTO

	err := json.Unmarshal(body, &responseDTO)
	require.NoError(t, err)

	require.Equal(t, http.StatusBadRequest, responseDTO.StatusCode)
	require.NotEmpty(t, responseDTO.Message)
}
