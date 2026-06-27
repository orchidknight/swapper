package swap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/orchidknight/swapper/markets"
	"github.com/orchidknight/swapper/models"
	"github.com/shopspring/decimal"
	"log"
	"testing"
)

func TestSwapper_ConsumeOrder(t *testing.T) {
	tests := map[string]struct {
		inputOrder         *models.Order
		inputMarketService MarketProvider
		wantReport         *models.SwapperReport
		wantErr            string
		wantActiveSwaps    int
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
			wantErr: "AllSwapSteps: can't find pairs for swap BTC-USDT",
		},
		"reject buy swap because unsupported": {
			inputOrder: &models.Order{
				ID:             42,
				Status:         models.OrderStatusNew,
				Type:           models.OrderTypeSwap,
				Symbol:         "SHIB-DOGE",
				Side:           models.SideBuy,
				AvailableTotal: decimal.NewFromFloat(1000),
				Total:          decimal.NewFromFloat(1000),
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantReport: &models.SwapperReport{
				SubOrderToSend: nil,
				ResultSwapOrder: &models.Order{
					ID:             42,
					Type:           models.OrderTypeSwap,
					Status:         models.OrderStatusRejected,
					Symbol:         "SHIB-DOGE",
					AvailableTotal: decimal.NewFromFloat(1000),
					Total:          decimal.NewFromFloat(1000),
					Side:           models.SideBuy,
					RejectReason:   models.RejectReasonBuySwapsNotSupported,
				},
			},
			wantErr: "buy swaps not supported",
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
			wantActiveSwaps: 1,
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
			wantActiveSwaps: 1,
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
			wantActiveSwaps: 1,
		},
	}

	ctx := context.Background()
	logMock := NewLogMock()

	for name, tc := range tests {
		s := NewSwapper(tc.inputMarketService, &MockedStorage{}, logMock)

		t.Run(name, func(t *testing.T) {
			gotReport, err := s.ConsumeOrder(ctx, tc.inputOrder)
			assertError(t, err, tc.wantErr)

			if err = reportEquals(gotReport, tc.wantReport); err != nil {
				t.Fatalf("reports do not match: %v", err)
			}

			if gotActiveSwaps := len(s.activeSwaps); gotActiveSwaps != tc.wantActiveSwaps {
				t.Fatalf("active swaps count mismatch: got %d, want %d", gotActiveSwaps, tc.wantActiveSwaps)
			}
		})
	}
}

func TestSwapper_ConsumeOrderAppliesMarketPrecisionToFirstSubOrder(t *testing.T) {
	swapper := NewSwapper(&markets.MarketService{
		Markets: map[models.Symbol]*models.MarketPair{
			"BTC-USDT": {
				Symbol:         "BTC-USDT",
				Base:           "BTC",
				Quote:          "USDT",
				BasePrecision:  3,
				QuotePrecision: 2,
				TradingEnabled: true,
			},
		},
	}, &MockedStorage{}, NewLogMock())

	report, err := swapper.ConsumeOrder(context.Background(), &models.Order{
		ID:              7,
		Status:          models.OrderStatusNew,
		Type:            models.OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          mustDecimal(t, "1000.123456"),
		AvailableAmount: mustDecimal(t, "1000.123456"),
	})
	if err != nil {
		t.Fatalf("consume order: %v", err)
	}

	wantAmount := mustDecimal(t, "1000.123")
	if !report.SubOrderToSend.Amount.Equal(wantAmount) {
		t.Fatalf("suborder amount mismatch: got %s, want %s", report.SubOrderToSend.Amount, wantAmount)
	}
	if !report.SubOrderToSend.AvailableAmount.Equal(wantAmount) {
		t.Fatalf("suborder available amount mismatch: got %s, want %s", report.SubOrderToSend.AvailableAmount, wantAmount)
	}

	activeSwap := swapper.activeSwaps[7]
	if activeSwap == nil {
		t.Fatal("expected active swap")
	}

	firstStep := activeSwap.Steps[0]
	if firstStep.BasePrecision != 3 {
		t.Fatalf("step base precision mismatch: got %d, want 3", firstStep.BasePrecision)
	}
	if firstStep.QuotePrecision != 2 {
		t.Fatalf("step quote precision mismatch: got %d, want 2", firstStep.QuotePrecision)
	}
}

func TestSwapper_ConsumeOrderReleasesLockWhenNextStepFails(t *testing.T) {
	wantErr := errors.New("next step boom")

	s := NewSwapper(&markets.MarketService{Markets: inputOnePhaseMarkets}, &MockedStorage{}, NewLogMock())
	s.nextStepOrder = func(*models.Swap) (*models.Order, error) {
		return nil, wantErr
	}

	_, err := s.ConsumeOrder(context.Background(), &models.Order{
		ID:              1,
		Status:          models.OrderStatusNew,
		Type:            models.OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	if !s.lock.TryLock() {
		t.Fatal("lock was not released after NextStepOrder error")
	}
	s.lock.Unlock()
}

func TestSwapper_ConsumeOrderValidatesInput(t *testing.T) {
	s := NewSwapper(&markets.MarketService{Markets: inputOnePhaseMarkets}, &MockedStorage{}, NewLogMock())

	report, err := s.ConsumeOrder(context.Background(), nil)
	assertError(t, err, "order is nil")
	if report != nil {
		t.Fatalf("report mismatch: got %v, want nil", report)
	}
}

func TestSwapper_ConsumeOrderRejectsInvalidSwapOrder(t *testing.T) {
	tests := map[string]struct {
		order      *models.Order
		wantErr    string
		wantStatus models.OrderStatus
	}{
		"non swap order": {
			order: &models.Order{
				ID:              1,
				Type:            models.OrderTypeMarket,
				Status:          models.OrderStatusNew,
				Symbol:          "BTC-USDT",
				Side:            models.SideSell,
				Amount:          decimal.NewFromInt(1),
				AvailableAmount: decimal.NewFromInt(1),
			},
			wantErr:    "order type must be Swap",
			wantStatus: models.OrderStatusRejected,
		},
		"unspecified side": {
			order: &models.Order{
				ID:              2,
				Type:            models.OrderTypeSwap,
				Status:          models.OrderStatusNew,
				Symbol:          "BTC-USDT",
				Side:            models.SideUnspecified,
				Amount:          decimal.NewFromInt(1),
				AvailableAmount: decimal.NewFromInt(1),
			},
			wantErr:    "swap side must be sell",
			wantStatus: models.OrderStatusRejected,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewSwapper(&markets.MarketService{Markets: inputOnePhaseMarkets}, &MockedStorage{}, NewLogMock())

			report, err := s.ConsumeOrder(context.Background(), tc.order)
			assertError(t, err, tc.wantErr)

			if report == nil || report.ResultSwapOrder == nil {
				t.Fatalf("expected rejected order report, got %v", report)
			}
			if report.ResultSwapOrder.Status != tc.wantStatus {
				t.Fatalf("status mismatch: got %s, want %s", report.ResultSwapOrder.Status, tc.wantStatus)
			}
		})
	}
}

func assertError(t *testing.T, got error, wantContains string) {
	t.Helper()

	if wantContains == "" {
		if got != nil {
			t.Fatalf("unexpected error: %v", got)
		}

		return
	}

	if got == nil {
		t.Fatalf("expected error containing %q, got nil", wantContains)
	}

	if !strings.Contains(got.Error(), wantContains) {
		t.Fatalf("wrong error: got %q, want substring %q", got.Error(), wantContains)
	}
}

func mustDecimal(t *testing.T, value string) decimal.Decimal {
	t.Helper()

	result, err := decimal.NewFromString(value)
	if err != nil {
		t.Fatalf("parse decimal %q: %v", value, err)
	}

	return result
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
