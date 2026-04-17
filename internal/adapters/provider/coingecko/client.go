package coingecko

import (
	"context"
	"crypto_service/internal/cases"
	"crypto_service/internal/entities"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
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

	if c.costIn == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "costIn not set")
	}

	if c.Timeout == 0 {
		return nil, errors.Wrap(entities.ErrInvalidParam, "timeout must be greater than 0")
	}

	if c.apiToken == "" {
		return nil, errors.Wrap(entities.ErrInvalidParam, "apiToken must be filled")
	}

	return c, nil
}

func (c *Client) GetActualCoins(ctx context.Context, titles []string) ([]*entities.Coin, error) {
	urlRaw, err := c.buildURL(titles)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "urlRaw error: %v", err)
	}

	request, err := c.buildRequest(ctx, urlRaw)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "build request error: %v", err)
	}

	response, err := c.doRequest(request)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "do request error: %v", err)
	}

	defer response.Body.Close()

	return c.parseCoins(response.Body)
}

func (c *Client) parseCoins(body io.ReadCloser) ([]*entities.Coin, error) {
	var data map[string]map[string]float64

	if body == nil {
		return nil, errors.Wrap(entities.ErrInternal, "incorrect body")
	}

	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "incorrect decode: %v", err)
	}

	coins := make([]*entities.Coin, 0, len(data))

	for title, coinData := range data {
		cost, ok := coinData[c.costIn]
		if !ok {
			return nil, errors.Wrap(entities.ErrInternal, "cost not found in response")
		}

		coin, err := entities.NewCoin(title, cost, time.Now())
		if err != nil {
			return nil, errors.Wrap(err, "new coin from rawData")
		}

		coins = append(coins, coin)
	}

	return coins, nil
}

func (c *Client) buildURL(titles []string) (*url.URL, error) {
	urlRaw, err := url.Parse(fmt.Sprintf("%s%s", basePath, simplePricePath))
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "parse error: %v", err)
	}

	query := urlRaw.Query()
	query.Set(vsCurrenciesQuery, c.costIn)
	query.Set(symbolsQuery, strings.Join(titles, ","))
	urlRaw.RawQuery = query.Encode()

	return urlRaw, nil
}

func (c *Client) buildRequest(ctx context.Context, urlRaw *url.URL) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, urlRaw.String(), nil)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "creating request error: %v", err)
	}
	request.Header.Add(xCgProApiKeyHeader, c.apiToken)

	return request, nil
}

func (c *Client) doRequest(request *http.Request) (*http.Response, error) {
	response, err := c.Do(request)
	if err != nil {
		return nil, errors.Wrapf(entities.ErrInternal, "response error: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(entities.ErrInternal, "incorrect response status code: %d", response.StatusCode)
	}

	return response, nil
}
