package swap

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/orchidknight/swapper/models"
	"github.com/shopspring/decimal"
)

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
