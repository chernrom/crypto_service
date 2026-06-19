//go:build TEST_L2

package l2

import (
	"bytes"
	"context"
	"crypto_service/pkg/dto"
	"encoding/json"
	"fmt"
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

func sendRequestToCoin(t *testing.T, aggregatedType string, titles []string) *http.Response {
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
	dto := dto.TitlesDTO{
		Titles: titles,
	}
	data, err := json.Marshal(dto)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlRaw.String(), bytes.NewReader(data))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp
}

func Test_AcutalCoinSuccess(t *testing.T) {
	t.Parallel()

	titles := []string{"btc", "eth"}
	resp := sendRequestToCoin(t, "", titles)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
