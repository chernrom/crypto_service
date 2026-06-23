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

	var aggregatedPath string
	if aggregatedType != "" {
		aggregatedPath = fmt.Sprintf("/%s", aggregatedType)
	}

	urlRaw, err := url.Parse(fmt.Sprintf("%s%s%s", basePath, ratesPath, aggregatedPath))
	require.NoError(t, err)

	requestDTO := dto.TitlesDTO{
		Titles: titles,
	}
	data, err := json.Marshal(requestDTO)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlRaw.String(), bytes.NewReader(data))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, body
}

func Test_AcutalCoinSuccess(t *testing.T) {
	t.Parallel()

	titles := []string{"btc", "eth"}

	statusCode, body := sendRequestToCoin(t, "", titles)

	require.Equal(t, http.StatusOK, statusCode)
	require.NotEmpty(t, body)

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
		require.Truef(t, ok, "wrong coin title: %s", coin.Title)
	}
}
