package swap

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/orchidknight/swapper/markets"
	"github.com/orchidknight/swapper/models"
	"github.com/shopspring/decimal"
)

type staticMarketProvider struct {
	swapPairs []*models.LinkedPairs
	markets   map[models.Symbol]*models.MarketPair
	err       error
}

func (smp *staticMarketProvider) GetAllSwapPairs(models.Symbol) ([]*models.LinkedPairs, error) {
	return smp.swapPairs, smp.err
}

func (smp *staticMarketProvider) GetMarket(symbol models.Symbol) *models.MarketPair {
	return smp.markets[symbol]
}

func TestSwapper_AllSwapStepsFiltersEmptyPairs(t *testing.T) {
	validPairs := &models.LinkedPairs{
		Pairs: []models.Pair{{
			Symbol:         "BTC-USDT",
			BasePrecision:  models.PrecisionUnknown,
			QuotePrecision: models.PrecisionUnknown,
		}},
	}
	providerPairs := &models.LinkedPairs{
		Pairs: []models.Pair{{Symbol: "BTC-USDT"}},
	}
	swapper := NewSwapper(&staticMarketProvider{
		swapPairs: []*models.LinkedPairs{
			{Pairs: nil},
			providerPairs,
			{Pairs: []models.Pair{}},
		},
	}, nil, nil)

	got, err := swapper.AllSwapSteps(&models.Order{Symbol: "BTC-USDT"})
	if err != nil {
		t.Fatalf("all swap steps: %v", err)
	}

	want := []*models.LinkedPairs{validPairs}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("swap steps mismatch (-want +got):\n%s", diff)
	}
}

func TestSwapper_AllSwapStepsDoesNotMutateProviderLinkedPairs(t *testing.T) {
	providerPairs := &models.LinkedPairs{
		Pairs: []models.Pair{{
			Symbol:         "BTC-USDT",
			BasePrecision:  99,
			QuotePrecision: 99,
		}},
	}
	swapper := NewSwapper(&staticMarketProvider{
		swapPairs: []*models.LinkedPairs{providerPairs},
		markets: map[models.Symbol]*models.MarketPair{
			"BTC-USDT": {
				Symbol:         "BTC-USDT",
				BasePrecision:  8,
				QuotePrecision: 2,
				TradingEnabled: true,
			},
		},
	}, nil, nil)

	got, err := swapper.AllSwapSteps(&models.Order{Symbol: "BTC-USDT"})
	if err != nil {
		t.Fatalf("all swap steps: %v", err)
	}

	if got[0].Pairs[0].BasePrecision != 8 || got[0].Pairs[0].QuotePrecision != 2 {
		t.Fatalf("returned precision mismatch: got base=%d quote=%d, want base=8 quote=2",
			got[0].Pairs[0].BasePrecision,
			got[0].Pairs[0].QuotePrecision,
		)
	}
	if providerPairs.Pairs[0].BasePrecision != 99 || providerPairs.Pairs[0].QuotePrecision != 99 {
		t.Fatalf("provider pairs were mutated: got base=%d quote=%d, want base=99 quote=99",
			providerPairs.Pairs[0].BasePrecision,
			providerPairs.Pairs[0].QuotePrecision,
		)
	}
}

func TestSwapper_ConsumeOrderDoesNotTruncateWhenMarketMetadataMissing(t *testing.T) {
	amount := mustDecimal(t, "1.23456789")
	swapper := NewSwapper(&staticMarketProvider{
		swapPairs: []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "BTC-USDT"}}}},
	}, &MockedStorage{}, NewLogMock())

	report, err := swapper.ConsumeOrder(context.Background(), &models.Order{
		ID:              1,
		Status:          models.OrderStatusNew,
		Type:            models.OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          amount,
		AvailableAmount: amount,
	})
	if err != nil {
		t.Fatalf("consume order: %v", err)
	}

	// GetMarket returns nil here: precision is unknown, so the amount must be
	// passed through untouched instead of being truncated to whole units.
	if !report.SubOrderToSend.Amount.Equal(amount) {
		t.Fatalf("suborder amount mismatch: got %s, want %s", report.SubOrderToSend.Amount, amount)
	}
	if !report.SubOrderToSend.AvailableAmount.Equal(amount) {
		t.Fatalf("suborder available amount mismatch: got %s, want %s", report.SubOrderToSend.AvailableAmount, amount)
	}
}

func TestSwapper_ConsumeOrderDoesNotTruncateNewMarketPairDefaultPrecision(t *testing.T) {
	amount := mustDecimal(t, "1.23456789")
	marketProvider := markets.New([]*models.MarketPair{
		models.NewMarketPair("BTC-USDT", "BTC", "USDT", nil, 0),
	})
	swapper := NewSwapper(marketProvider, &MockedStorage{}, NewLogMock())

	report, err := swapper.ConsumeOrder(context.Background(), &models.Order{
		ID:              2,
		Status:          models.OrderStatusNew,
		Type:            models.OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          amount,
		AvailableAmount: amount,
	})
	if err != nil {
		t.Fatalf("consume order: %v", err)
	}

	if !report.SubOrderToSend.Amount.Equal(amount) {
		t.Fatalf("suborder amount mismatch: got %s, want %s", report.SubOrderToSend.Amount, amount)
	}
	if !report.SubOrderToSend.AvailableAmount.Equal(amount) {
		t.Fatalf("suborder available amount mismatch: got %s, want %s", report.SubOrderToSend.AvailableAmount, amount)
	}
}

func TestNewSwapperUsesNoOpPortsForNilStorageAndLogger(t *testing.T) {
	swapper := NewSwapper(&staticMarketProvider{
		swapPairs: []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "BTC-USDT"}}}},
	}, nil, nil)

	report, err := swapper.ConsumeOrder(context.Background(), &models.Order{
		ID:              1,
		Status:          models.OrderStatusNew,
		Type:            models.OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	})
	if err != nil {
		t.Fatalf("consume order: %v", err)
	}
	if report == nil || report.SubOrderToSend == nil {
		t.Fatalf("expected suborder report, got %v", report)
	}
}

func TestSwapper_AllSwapStepsRejectsNilMarketProvider(t *testing.T) {
	swapper := NewSwapper(nil, nil, nil)

	_, err := swapper.AllSwapSteps(&models.Order{Symbol: "BTC-USDT"})
	assertError(t, err, "market provider is nil")
}

func orderEquals(got, want *models.Order) error {
	if got == nil && want == nil {
		return nil
	}

	if got == nil {
		return fmt.Errorf("got nil order, want %v", want)
	}

	if want == nil {
		return fmt.Errorf("got %v, want nil order", got)
	}

	gotComparable := *got
	wantComparable := *want

	if want.ID == 0 {
		gotComparable.ID = 0
	}
	if want.Account == "" {
		gotComparable.Account = ""
	}
	if want.RejectReason == "" {
		gotComparable.RejectReason = ""
	}
	if want.CreatedAt.IsZero() {
		gotComparable.CreatedAt = wantComparable.CreatedAt
	}

	if diff := cmp.Diff(wantComparable, gotComparable, cmp.Comparer(decimal.Decimal.Equal)); diff != "" {
		return fmt.Errorf("order mismatch (-want +got):\n%s", diff)
	}

	return nil
}

func reportEquals(got, want *models.SwapperReport) error {
	if got == nil && want == nil {
		return nil
	}

	if got == nil {
		return fmt.Errorf("got nil report, want %v", want)
	}

	if want == nil {
		return fmt.Errorf("got %v, want nil report", got)
	}

	if err := orderEquals(got.SubOrderToSend, want.SubOrderToSend); err != nil {
		return fmt.Errorf("sub order to send: %w", err)
	}

	if err := orderEquals(got.ResultSwapOrder, want.ResultSwapOrder); err != nil {
		return fmt.Errorf("result swap order: %w", err)
	}

	return nil
}

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
		"SHIB-USDC": {Symbol: "SHIB-USDC", Base: "SHIB", Quote: "USDC", TradingEnabled: true},
		"BIN-USDC":  {Symbol: "BIN-USDC", Base: "BIN", Quote: "USDC", TradingEnabled: true},
		"BIN-DOGE":  {Symbol: "BIN-DOGE", Base: "BIN", Quote: "DOGE", TradingEnabled: true},
	}
)
