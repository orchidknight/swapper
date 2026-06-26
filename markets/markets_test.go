package markets

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/orchidknight/swapper/models"
)

var (
	inputThreePhaseMarkets = map[models.Symbol]*models.MarketPair{
		"SHIB-BNB":  {Symbol: "SHIB-BNB", Base: "SHIB", Quote: "BNB", TradingEnabled: true},
		"BNB-USDT":  {Symbol: "BNB-USDT", Base: "BNB", Quote: "USDT", TradingEnabled: true},
		"DOGE-USDT": {Symbol: "DOGE-USDT", Base: "DOGE", Quote: "USDT", TradingEnabled: true},
	}
	inputOnePhaseMarkets = map[models.Symbol]*models.MarketPair{
		"BTC-USDT": {Symbol: "BTC-USDT", Base: "BTC", Quote: "USDT", TradingEnabled: true},
	}

	inputTwoPhaseMarkets = map[models.Symbol]*models.MarketPair{
		"SHIB-USDT": {Symbol: "SHIB-USDT", Base: "SHIB", Quote: "USDT", TradingEnabled: true},
		"DOGE-USDT": {Symbol: "DOGE-USDT", Base: "DOGE", Quote: "USDT", TradingEnabled: true},
	}

	// PEPE_USDT FDUSD_USDT SOL_FDUSD BTC_SOL
	inputManyMarkets = map[models.Symbol]*models.MarketPair{
		"PEPE-USDT":  {Symbol: "PEPE-USDT", Base: "PEPE", Quote: "USDT", TradingEnabled: true},
		"FDUSD-USDT": {Symbol: "FDUSD-USDT", Base: "FDUSD", Quote: "USDT", TradingEnabled: true},
		"SOL-FDUSD":  {Symbol: "SOL-FDUSD", Base: "SOL", Quote: "FDUSD", TradingEnabled: true},
		"BTC-SOL":    {Symbol: "BTC-SOL", Base: "BTC", Quote: "SOL", TradingEnabled: true},
		"SOL-USDT":   {Symbol: "SOL-USDT", Base: "SOL", Quote: "USDT", TradingEnabled: true},
	}
)

func TestMarketService_GetSwapPairs(t *testing.T) {
	tests := map[string]struct {
		inputSymbol  models.Symbol
		inputMarkets map[models.Symbol]*models.MarketPair
		wantPairs    []*models.LinkedPairs
		wantErr      error
	}{
		"one step swap": {
			inputSymbol:  "BTC-USDT",
			inputMarkets: inputOnePhaseMarkets,
			wantPairs:    []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "BTC-USDT"}}}},
			wantErr:      nil,
		},
		"three steps swap": {
			inputSymbol:  "DOGE-SHIB",
			inputMarkets: inputThreePhaseMarkets,
			wantPairs:    []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "DOGE-USDT"}, {Symbol: "BNB-USDT"}, {Symbol: "SHIB-BNB"}}}},
			wantErr:      nil,
		},
		"two steps swap": {
			inputSymbol:  "DOGE-SHIB",
			inputMarkets: inputTwoPhaseMarkets,
			wantPairs:    []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "DOGE-USDT"}, {Symbol: "SHIB-USDT"}}}},
			wantErr:      nil,
		},
		"two alternative paths swap": {
			inputSymbol:  "PEPE-SOL",
			inputMarkets: inputManyMarkets,
			wantPairs: []*models.LinkedPairs{
				{Pairs: []models.Pair{{Symbol: "PEPE-USDT"}, {Symbol: "SOL-USDT"}}},
				{Pairs: []models.Pair{{Symbol: "PEPE-USDT"}, {Symbol: "FDUSD-USDT"}, {Symbol: "SOL-FDUSD"}}},
			},
			wantErr: nil,
		},
		"same source and destination swap": {
			inputSymbol:  "PEPE-PEPE",
			inputMarkets: inputManyMarkets,
			wantPairs:    nil,
			wantErr:      ErrInvalidSwapPairSameAssets,
		},
		"invalid symbol swap": {
			inputSymbol:  "PEPE_USDT",
			inputMarkets: inputManyMarkets,
			wantPairs:    nil,
			wantErr:      ErrInvalidSwapPair,
		},
	}

	for name, tc := range tests {
		ms := MarketService{
			Markets: tc.inputMarkets,
		}

		t.Run(name, func(t *testing.T) {
			got, _ := ms.GetAllSwapPairs(tc.inputSymbol)
			diff := cmp.Diff(tc.wantPairs, got)
			if diff != "" {
				fmt.Printf("Got: %s Want: %s\n", got, tc.wantPairs)
				t.Fatalf("%s", diff)
			}
		})
	}
}
