package markets

import (
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
				t.Fatalf("%s", diff)
			}
		})
	}
}

func TestMarketService_GetSwapPairsDeterministic(t *testing.T) {
	ms := MarketService{
		Markets: map[models.Symbol]*models.MarketPair{
			"AAA-BBB": {Symbol: "AAA-BBB", Base: "AAA", Quote: "BBB", TradingEnabled: true},
			"AAA-CCC": {Symbol: "AAA-CCC", Base: "AAA", Quote: "CCC", TradingEnabled: true},
			"BBB-DDD": {Symbol: "BBB-DDD", Base: "BBB", Quote: "DDD", TradingEnabled: true},
			"CCC-DDD": {Symbol: "CCC-DDD", Base: "CCC", Quote: "DDD", TradingEnabled: true},
		},
	}
	want := []*models.LinkedPairs{
		{Pairs: []models.Pair{{Symbol: "AAA-BBB"}, {Symbol: "BBB-DDD"}}},
		{Pairs: []models.Pair{{Symbol: "AAA-CCC"}, {Symbol: "CCC-DDD"}}},
	}

	for i := 0; i < 50; i++ {
		got, err := ms.GetAllSwapPairs("AAA-DDD")
		if err != nil {
			t.Fatalf("get swap pairs: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("iteration %d returned unstable swap pairs (-want +got):\n%s", i, diff)
		}
	}
}

func TestMarketService_GetSwapPairsIsolatesBranchExceptions(t *testing.T) {
	ms := MarketService{
		Markets: map[models.Symbol]*models.MarketPair{
			"AAA-WWW": {Symbol: "AAA-WWW", Base: "AAA", Quote: "WWW", TradingEnabled: true},
			"BBB-DDD": {Symbol: "BBB-DDD", Base: "BBB", Quote: "DDD", TradingEnabled: true},
			"BBB-XXX": {Symbol: "BBB-XXX", Base: "BBB", Quote: "XXX", TradingEnabled: true},
			"CCC-DDD": {Symbol: "CCC-DDD", Base: "CCC", Quote: "DDD", TradingEnabled: true},
			"CCC-XXX": {Symbol: "CCC-XXX", Base: "CCC", Quote: "XXX", TradingEnabled: true},
			"WWW-XXX": {Symbol: "WWW-XXX", Base: "WWW", Quote: "XXX", TradingEnabled: true},
		},
	}
	want := []*models.LinkedPairs{
		{Pairs: []models.Pair{{Symbol: "AAA-WWW"}, {Symbol: "WWW-XXX"}, {Symbol: "BBB-XXX"}, {Symbol: "BBB-DDD"}}},
		{Pairs: []models.Pair{{Symbol: "AAA-WWW"}, {Symbol: "WWW-XXX"}, {Symbol: "CCC-XXX"}, {Symbol: "CCC-DDD"}}},
	}

	got, err := ms.GetAllSwapPairs("AAA-DDD")
	if err != nil {
		t.Fatalf("get swap pairs: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("swap pairs mismatch (-want +got):\n%s", diff)
	}
}
