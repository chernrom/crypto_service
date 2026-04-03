package entities_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"crypto_service/internal/entities"
)

func TestNewCoin(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		cost       float64
		wantErr    bool
		resErr     error
		errDetails string
	}{
		{
			name:  "success",
			title: "Bitcoin",
			cost:  100,
		},
		{
			name:       "emptyTitle",
			title:      "",
			cost:       100,
			wantErr:    true,
			resErr:     entities.ErrInvalidParam,
			errDetails: "empty name",
		},
		{
			name:       "negativeCost",
			title:      "Bitcoin",
			cost:       -1,
			wantErr:    true,
			resErr:     entities.ErrInvalidParam,
			errDetails: "negative cost",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(it *testing.T) {
			it.Parallel()

			now := time.Now()
			coin, err := entities.NewCoin(tc.title, tc.cost, now)

			if tc.wantErr {
				require.ErrorIs(it, err, tc.resErr)
				require.Contains(it, err.Error(), tc.errDetails)
				require.Nil(it, coin)
				return
			}

			require.NoError(it, err)
			require.NotNil(it, coin)
			require.Equal(it, tc.title, coin.Title())
			require.Equal(it, tc.cost, coin.Cost())
			require.Equal(it, now, coin.ActualAt())
		})
	}
}
