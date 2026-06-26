package swap

import (
	"fmt"

	"github.com/orchidknight/swapper/models"
)

func orderEquals(got, want *models.Order) error {
	if got.Side != want.Side {
		return fmt.Errorf("wrong side")
	}
	if got.Type != want.Type {
		return fmt.Errorf("wrong type")
	}

	if !got.AvailableAmount.Equal(want.AvailableAmount) {
		return fmt.Errorf("wrong available amount")
	}

	if !got.ExecutedAmount.Equal(want.ExecutedAmount) {
		return fmt.Errorf("wrong executed amount")
	}

	if !got.AvailableTotal.Equal(want.AvailableTotal) {
		return fmt.Errorf("wrong available total")
	}

	if !got.ExecutedTotal.Equal(want.ExecutedTotal) {
		return fmt.Errorf("wrong executed total")
	}

	if !got.AvgPrice.Equal(want.AvgPrice) {
		return fmt.Errorf("wrong avg price")
	}

	if got.Symbol != want.Symbol {
		return fmt.Errorf("wrong symbol")
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
