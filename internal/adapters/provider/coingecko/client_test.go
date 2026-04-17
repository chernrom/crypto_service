package coingecko

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	apiToken = "test-token"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		apiToken    string
		opts        []ClientOption
		wantErr     bool
		wantCostIn  string
		wantTimeout time.Duration
	}{
		{
			name:        "success",
			apiToken:    apiToken,
			opts:        nil,
			wantErr:     false,
			wantCostIn:  defaultCostIn,
			wantTimeout: defaultTimeout,
		},
		{
			name:     "empty api token",
			apiToken: "",
			opts:     nil,
			wantErr:  true,
		},
		{
			name:     "time out is 0",
			apiToken: apiToken,
			opts: []ClientOption{
				WithCustomTimeout(0),
			},
			wantErr: true,
		},
		{
			name:     "costIn is empty",
			apiToken: apiToken,
			opts: []ClientOption{
				WithCustomCostIn(""),
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			client, err := NewClient(tc.apiToken, tc.opts...)

			if tc.wantErr {
				require.Error(it, err)
				require.Nil(it, client)
				return
			}

			require.NoError(it, err)
			require.NotNil(it, client)
			require.Equal(it, tc.wantCostIn, client.costIn)
			require.Equal(it, tc.wantTimeout, client.Timeout)

		})
	}

}
