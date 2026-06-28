package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"crypto_service/internal/cases"
	"crypto_service/internal/entities"
	"crypto_service/toolkit/tracing"
)

const (
	basePath        = "https://api.coingecko.com/"
	simplePricePath = "api/v3/simple/price"

	vsCurrenciesQuery = "vs_currencies"
	symbolsQuery      = "symbols"

	xCgProApiKeyHeader = "x_cg_pro_api_key"

	defaultCostIn  = "usd"
	defaultTimeout = 5 * time.Second
)

var (
	_ cases.CryptoProvider = (*Client)(nil)
)

type Client struct {
	*http.Client
	costIn   string
	apiToken string
}

type ClientOption func(*Client)

func WithCustomCostIn(costIn string) ClientOption {
	return func(c *Client) {
		c.costIn = costIn
	}
}

func WithCustomTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.Timeout = timeout
	}
}

func (c *Client) setOptions(opts ...ClientOption) {
	for _, opt := range opts {
		opt(c)
	}
}

func NewClient(apiToken string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		costIn:   defaultCostIn,
		apiToken: apiToken,
		Client: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	c.setOptions(opts...)

	switch {
	case c.costIn == "":
		slog.Error("new coingecko client failed", "error", entities.ErrInvalidParam, "reason", "costIn not set")
		return nil, errors.Wrap(entities.ErrInvalidParam, "costIn not set")
	case c.Timeout <= 0:
		slog.Error("new coingecko client failed", "error", entities.ErrInvalidParam, "reason", "timeout not set")
		return nil, errors.Wrap(entities.ErrInvalidParam, "timeout must be greater than 0")
	case c.apiToken == "":
		slog.Error("new coingecko client failed", "error", entities.ErrInvalidParam, "reason", "apiToken not set")
		return nil, errors.Wrap(entities.ErrInvalidParam, "apiToken must be filled")
	}

	return c, nil
}

func (c *Client) GetActualCoins(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	ctx, span, cancelFn := tracing.Start(ctx, "Provider.CoinGecko.GetActualCoins")
	defer cancelFn()

	if len(titles) == 0 {
		err := errors.Wrap(entities.ErrInvalidParam, "titles is empty")
		span.SetError(err)
		slog.Error("titles is empty", "error", err)
		return nil, err
	}

	urlRaw, err := c.buildURL(titles)
	if err != nil {
		span.SetError(err)
		slog.Error("build url failed", "error", err, "titles", titles)
		return nil, errors.Wrap(err, "build url failure")
	}

	request, err := c.buildRequest(ctx, urlRaw)
	if err != nil {
		span.SetError(err)
		slog.Error("build request failed", "error", err, "url", urlRaw)
		return nil, errors.Wrap(err, "build request failure")
	}

	response, err := c.doRequest(ctx, request)
	if err != nil {
		span.SetError(err)
		slog.Error("do request failed", "error", err)
		return nil, errors.Wrap(err, "do request failure")
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			span.SetError(err)
			slog.Error("close response body failed", "error", err)
		}
	}()

	coins, err := c.parseCoins(response.Body)
	if err != nil {
		span.SetError(err)
		return nil, err
	}

	return coins, nil
}

func (c *Client) parseCoins(body io.Reader) ([]*entities.Coin, error) {
	var data map[string]map[string]float64

	if body == nil {
		slog.Error("response body is nil", "error", entities.ErrInternal)
		return nil, errors.Wrap(entities.ErrInternal, "incorrect body")
	}

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		slog.Error("decode coingecko response failed", "error", err)
		return nil, errors.Wrapf(entities.ErrInternal, "incorrect decode: %v", err)
	}

	if len(data) == 0 {
		slog.Error("empty coingecko response", "error", entities.ErrNotFound)
		return nil, errors.Wrap(entities.ErrNotFound, "empty coingecko response")
	}

	coins := make([]*entities.Coin, 0, len(data))

	for title, coinData := range data {
		cost, ok := coinData[c.costIn]
		if !ok {
			slog.Error("cost not found in response", "title", title, "cost_in", c.costIn)
			return nil, errors.Wrap(entities.ErrInternal, "cost not found in response")
		}

		coin, err := entities.NewCoin(title, cost, time.Now())
		if err != nil {
			slog.Error("new coin failed", "error", err, "title", title, "cost", cost)
			return nil, errors.Wrap(err, "new coin from rawData")
		}

		coins = append(coins, coin)
	}

	return coins, nil
}

func (c *Client) buildURL(titles []string) (*url.URL, error) {
	urlRaw, err := url.Parse(fmt.Sprintf("%s%s", basePath, simplePricePath))
	if err != nil {
		slog.Error("parse coingecko url failed", "error", err)
		return nil, errors.Wrapf(entities.ErrInternal, "parse error: %v", err)
	}

	query := urlRaw.Query()
	query.Set(vsCurrenciesQuery, c.costIn)
	query.Set(symbolsQuery, strings.Join(titles, ","))
	urlRaw.RawQuery = query.Encode()

	return urlRaw, nil
}

func (c *Client) buildRequest(ctx context.Context, urlRaw *url.URL) (*http.Request, error) {
	if urlRaw == nil {
		slog.Error("url is nil", "error", entities.ErrInvalidParam)
		return nil, errors.Wrap(entities.ErrInvalidParam, "url is nil")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlRaw.String(), nil)
	if err != nil {
		slog.Error("create coingecko request failed", "error", err, "url", urlRaw.String())
		return nil, errors.Wrapf(entities.ErrInternal, "creating request error: %v", err)
	}

	request.Header.Add(xCgProApiKeyHeader, c.apiToken)

	return request, nil
}

func (c *Client) doRequest(ctx context.Context, request *http.Request) (*http.Response, error) {
	ctx, span, cancelFn := tracing.Start(ctx, "Provider.CoinGecko.DoRequest")
	defer cancelFn()

	if request == nil {
		err := errors.Wrap(entities.ErrInvalidParam, "request is nil")
		span.SetError(err)
		slog.Error("request is nil", "error", err)
		return nil, err
	}

	request = request.WithContext(ctx)

	response, err := c.Do(request)
	if err != nil {
		span.SetError(err)
		slog.Error("coingecko request failed", "error", err, "url", request.URL.String())
		return nil, errors.Wrap(err, "response error")
	}

	if response == nil {
		err := errors.Wrap(entities.ErrInternal, "response is nil")
		span.SetError(err)
		slog.Error("coingecko response is nil", "error", err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err := errors.Wrapf(entities.ErrInternal, "incorrect response status code: %d", response.StatusCode)
		span.SetError(err)

		slog.Error(
			"coingecko returned non-ok status",
			"error", err,
			"status_code", response.StatusCode,
			"url", request.URL.String(),
		)

		if closeErr := response.Body.Close(); closeErr != nil {
			span.SetError(closeErr)
			slog.Error("close response body failed", "error", closeErr)
		}

		return nil, err
	}

	return response, nil
}
