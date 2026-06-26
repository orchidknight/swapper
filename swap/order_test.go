package swap

import (
	"context"
	"errors"
	"fmt"
	"github.com/orchidknight/swapper/markets"
	"github.com/orchidknight/swapper/models"
	"github.com/shopspring/decimal"
	"log"
	"testing"
)

// nolint
func TestSwapper_ConsumeOrder(t *testing.T) {
	tests := map[string]struct {
		inputOrder         *models.Order
		inputMarketService MarketProvider
		wantReport         *models.SwapperReport
		wantErr            any
	}{
		"reject swap": {
			inputOrder: &models.Order{
				ID:              0,
				Status:          models.OrderStatusNew,
				Type:            models.OrderTypeSwap,
				Symbol:          "BTC-USDT",
				Side:            models.SideSell,
				AvailableAmount: decimal.NewFromFloat(1000),
			},
			inputMarketService: &markets.MarketService{
				Markets: nil,
			},
			wantReport: &models.SwapperReport{
				SubOrderToSend: nil,
				ResultSwapOrder: &models.Order{
					Type:            models.OrderTypeSwap,
					Status:          models.OrderStatusRejected,
					Symbol:          "BTC-USDT",
					AvailableAmount: decimal.NewFromFloat(1000),
					Side:            models.SideSell,
				},
			},
			wantErr: nil,
		},
		"one step swap": {
			inputOrder: &models.Order{
				ID:              0,
				Status:          models.OrderStatusNew,
				Type:            models.OrderTypeSwap,
				Symbol:          "BTC-USDT",
				Side:            models.SideSell,
				AvailableAmount: decimal.NewFromFloat(1000),
			},
			inputMarketService: &markets.MarketService{
				Markets: inputOnePhaseMarkets,
			},
			wantReport: &models.SwapperReport{
				SubOrderToSend: &models.Order{
					Type:            models.OrderTypeMarket,
					Status:          models.OrderStatusNew,
					Symbol:          "BTC-USDT",
					AvailableAmount: decimal.NewFromFloat(1000),
					Side:            models.SideSell,
				},
			},
			wantErr: nil,
		},
		"2 steps swap sell then buy": {
			inputOrder: &models.Order{
				Status:          models.OrderStatusNew,
				Type:            models.OrderTypeSwap,
				Side:            models.SideSell,
				Symbol:          "SHIB-DOGE",
				AvailableAmount: decimal.NewFromFloat(1000),
				Amount:          decimal.NewFromFloat(1000),
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantReport: &models.SwapperReport{
				SubOrderToSend: &models.Order{
					Type:            models.OrderTypeMarket,
					Symbol:          "SHIB-USDT",
					Side:            models.SideSell,
					Status:          models.OrderStatusNew,
					AvailableAmount: decimal.NewFromFloat(1000),
					Amount:          decimal.NewFromFloat(1000),
				},
			},
			wantErr: nil,
		},
		"3 steps swap sell then buy": {
			inputOrder: &models.Order{
				Type:            models.OrderTypeSwap,
				Status:          models.OrderStatusNew,
				Side:            models.SideSell,
				Symbol:          "DOGE-SHIB",
				AvailableAmount: decimal.NewFromFloat(1000),
				Amount:          decimal.NewFromFloat(1000),
			},
			inputMarketService: &markets.MarketService{
				Markets: inputThreePhaseMarkets,
			},
			wantReport: &models.SwapperReport{
				SubOrderToSend: &models.Order{
					Type:            models.OrderTypeMarket,
					Status:          models.OrderStatusNew,
					Symbol:          "DOGE-USDT",
					Side:            models.SideSell,
					AvailableAmount: decimal.NewFromFloat(1000),
					Amount:          decimal.NewFromFloat(1000),
				},
			},
			wantErr: nil,
		},
	}

	ctx := context.Background()
	logMock := NewLogMock()

	for name, tc := range tests {
		s := NewSwapper(tc.inputMarketService, nil, &MockedStorage{}, logMock)

		t.Run(name, func(t *testing.T) {
			gotReport, err := s.ConsumeOrder(ctx, tc.inputOrder)
			fmt.Println(gotReport, err)
			if err != nil {
				if !errors.As(err, tc.wantErr) {
					t.Fatalf("wrong error wanted")
				}
			}

			fmt.Printf("%v\n", gotReport.SubOrderToSend)
			fmt.Printf("%v\n", tc.wantReport.SubOrderToSend)

			if err = orderEquals(gotReport.SubOrderToSend, tc.wantReport.SubOrderToSend); err != nil {
				t.Fatalf("orders do not match: %v", err)
			} else {
				fmt.Println("orders match")
			}

			fmt.Printf("'%s' passed successfully!\n", name)
		})
	}
}

type LogMock struct{}

func NewLogMock() models.Logger {
	return &LogMock{}
}

func (*LogMock) Debug(component string, format string, a ...any) {
	log.Printf(fmt.Sprintf("%-6s | %s", component, format), a...)
}

func (*LogMock) Info(component string, format string, a ...any) {
	log.Printf(fmt.Sprintf("%-6s | %s", component, format), a...)
}

func (*LogMock) Warn(component string, format string, a ...any) {
	log.Printf(fmt.Sprintf("%-6s | %s", component, format), a...)
}

func (*LogMock) Error(component string, format string, a ...any) {
	log.Printf(fmt.Sprintf("%-6s | %s", component, format), a...)
}

func (*LogMock) Fatal(component string, format string, a ...any) {
	log.Printf(fmt.Sprintf("| %-6s |%s", component, format), a...)
}
