package swap

import (
	"context"
	"github.com/orchidknight/swapper/markets"
	"github.com/shopspring/decimal"
	"testing"

	"github.com/orchidknight/swapper/models"
)

// nolint
func TestSwapper_ConsumeSubOrderResult(t *testing.T) {
	tests := map[string]struct {
		inputOrder           *models.Order
		inputSubOrderResults []*models.Order
		inputMarketService   MarketProvider
		inputSwap            *models.Swap
		wantSwapReports      []*models.SwapperReport
		wantErr              string
	}{
		"two phase swap sell then buy: success first step ": {
			inputOrder: &models.Order{
				ID:              1,
				Type:            models.OrderTypeSwap,
				Symbol:          "SHIB-DOGE",
				Side:            models.SideSell,
				AvailableAmount: decimal.NewFromFloat(10),
				Amount:          decimal.NewFromFloat(10),
			},
			inputSubOrderResults: []*models.Order{
				{
					Side:            models.SideSell,
					Status:          models.OrderStatusCompleted,
					Symbol:          "SHIB-USDT",
					ExecutedAmount:  decimal.NewFromInt(10),
					AvailableAmount: decimal.Zero,
					Amount:          decimal.NewFromFloat(10),
					Price:           decimal.NewFromInt(100),
					AvgPrice:        decimal.NewFromInt(100),
					ExecutedTotal:   decimal.NewFromInt(1000),
				},
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantSwapReports: []*models.SwapperReport{
				{
					SubOrderToSend: &models.Order{
						Side:           models.SideBuy,
						AvailableTotal: decimal.NewFromInt(1000),
						Symbol:         "DOGE-USDT",
						Type:           models.OrderTypeMarket,
						Status:         models.OrderStatusNew,
					},
				},
			},
		},
		"two phase swap sell then buy: reject first step, rerouting": {
			inputOrder: &models.Order{
				ID:              1,
				Type:            models.OrderTypeSwap,
				Symbol:          "SHIB-DOGE",
				Side:            models.SideSell,
				Status:          models.OrderStatusNew,
				AvailableAmount: decimal.NewFromFloat(100),
				Amount:          decimal.NewFromFloat(100),
			},
			inputSubOrderResults: []*models.Order{
				{
					Side:            models.SideSell,
					Status:          models.OrderStatusRejected,
					Symbol:          "SHIB-USDT",
					Amount:          decimal.NewFromFloat(100),
					AvailableAmount: decimal.NewFromFloat(100),
				},
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantSwapReports: []*models.SwapperReport{
				{

					SubOrderToSend: &models.Order{
						Side:            models.SideSell,
						Amount:          decimal.NewFromFloat(100),
						AvailableAmount: decimal.NewFromFloat(100),
						Symbol:          "SHIB-USDC",
						Type:            models.OrderTypeMarket,
						Status:          models.OrderStatusNew,
					},
				},
			},
		},
		"two phase swap sell then buy: reject all first step, final reject": {
			inputOrder: &models.Order{
				ID:              1,
				Type:            models.OrderTypeSwap,
				Symbol:          "SHIB-DOGE",
				Status:          models.OrderStatusNew,
				Side:            models.SideSell,
				Amount:          decimal.NewFromFloat(100),
				AvailableAmount: decimal.NewFromFloat(100),
			},
			inputSubOrderResults: []*models.Order{
				{
					Side:            models.SideSell,
					Status:          models.OrderStatusRejected,
					Symbol:          "SHIB-USDT",
					Amount:          decimal.NewFromFloat(100),
					AvailableAmount: decimal.NewFromFloat(100),
				},
				{
					Side:            models.SideSell,
					Status:          models.OrderStatusRejected,
					Symbol:          "SHIB-USDC",
					Amount:          decimal.NewFromFloat(100),
					AvailableAmount: decimal.NewFromFloat(100),
				},
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantSwapReports: []*models.SwapperReport{
				{
					SubOrderToSend: &models.Order{
						Side:            models.SideSell,
						Amount:          decimal.NewFromFloat(100),
						AvailableAmount: decimal.NewFromFloat(100),
						Symbol:          "SHIB-USDC",
						Type:            models.OrderTypeMarket,
						Status:          models.OrderStatusNew,
					},
				},
				{
					ResultSwapOrder: &models.Order{
						ID:              1,
						Status:          models.OrderStatusRejected,
						Symbol:          "SHIB-DOGE",
						Type:            models.OrderTypeSwap,
						Side:            models.SideSell,
						Amount:          decimal.NewFromFloat(100),
						AvailableAmount: decimal.NewFromFloat(100),
					},
					SubOrderToSend: nil,
				},
			},
		},
		"two phase swap sell then buy: success first step with 100 amount": {
			inputOrder: &models.Order{
				ID:              1,
				Type:            models.OrderTypeSwap,
				Symbol:          "SHIB-DOGE",
				Side:            models.SideSell,
				Status:          models.OrderStatusNew,
				Amount:          decimal.NewFromFloat(100),
				AvailableAmount: decimal.NewFromFloat(100),
			},
			inputSubOrderResults: []*models.Order{
				{
					Side:            models.SideSell,
					Status:          models.OrderStatusCompleted,
					Symbol:          "SHIB-USDT",
					Price:           decimal.NewFromFloat(10),
					AvgPrice:        decimal.NewFromFloat(10),
					AvailableAmount: decimal.Zero,
					ExecutedAmount:  decimal.NewFromFloat(100),
					ExecutedTotal:   decimal.NewFromFloat(1000),
				},
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantSwapReports: []*models.SwapperReport{
				{
					SubOrderToSend: &models.Order{
						Side:           models.SideBuy,
						AvailableTotal: decimal.NewFromFloat(1000),
						Status:         models.OrderStatusNew,
						Symbol:         "DOGE-USDT",
						Type:           models.OrderTypeMarket,
					},
				},
			},
		},
		"two phase swap sell then buy: partially completed ": {
			inputOrder: &models.Order{
				ID:              1,
				Type:            models.OrderTypeSwap,
				Symbol:          "SHIB-DOGE",
				Side:            models.SideSell,
				Status:          models.OrderStatusNew,
				Amount:          decimal.NewFromFloat(100),
				AvailableAmount: decimal.NewFromFloat(100),
			},
			inputSubOrderResults: []*models.Order{
				{
					Type:   models.OrderTypeMarket,
					Symbol: "SHIB-USDT",
					Side:   models.SideSell,
					Status: models.OrderStatusPartiallyCompleted,
				},
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantSwapReports: []*models.SwapperReport{
				{
					SubOrderToSend:  nil,
					ResultSwapOrder: nil,
				},
			},
		},
		"two phase swap sell then buy: completed": {
			inputOrder: &models.Order{
				ID:              1,
				Type:            models.OrderTypeSwap,
				Symbol:          "SHIB-DOGE",
				Status:          models.OrderStatusNew,
				Side:            models.SideSell,
				Amount:          decimal.NewFromFloat(100),
				AvailableAmount: decimal.NewFromFloat(100),
			},
			inputSubOrderResults: []*models.Order{
				{
					Side:            models.SideSell,
					Type:            models.OrderTypeMarket,
					Status:          models.OrderStatusCompleted,
					Symbol:          "SHIB-USDT",
					AvailableAmount: decimal.Zero,
					Amount:          decimal.NewFromFloat(100),
					Price:           decimal.NewFromFloat(10),
					AvgPrice:        decimal.NewFromFloat(10),
					ExecutedTotal:   decimal.NewFromFloat(1000),
					ExecutedAmount:  decimal.NewFromFloat(100),
				},
				{
					Side:           models.SideBuy,
					Type:           models.OrderTypeMarket,
					Status:         models.OrderStatusCompleted,
					Symbol:         "DOGE-USDT",
					AvailableTotal: decimal.Zero,
					Price:          decimal.NewFromFloat(20),
					AvgPrice:       decimal.NewFromFloat(20),
					ExecutedTotal:  decimal.NewFromFloat(1000),
					ExecutedAmount: decimal.NewFromFloat(50),
				},
			},
			inputMarketService: &markets.MarketService{
				Markets: inputTwoPhaseMarkets,
			},
			wantSwapReports: []*models.SwapperReport{
				{
					SubOrderToSend: &models.Order{
						Status:         models.OrderStatusNew,
						Side:           models.SideBuy,
						Symbol:         "DOGE-USDT",
						Type:           models.OrderTypeMarket,
						AvailableTotal: decimal.NewFromFloat(1000),
					},
				},
				{
					SubOrderToSend: nil,
					ResultSwapOrder: &models.Order{
						ID:              1,
						Symbol:          "SHIB-DOGE",
						Status:          models.OrderStatusCompleted,
						Type:            models.OrderTypeSwap,
						Side:            models.SideSell,
						Amount:          decimal.NewFromFloat(100),
						AvailableAmount: decimal.Zero,
						ExecutedAmount:  decimal.NewFromFloat(100),
						ExecutedTotal:   decimal.NewFromFloat(50),
						Price:           decimal.NewFromFloat(0.5),
						AvgPrice:        decimal.NewFromFloat(0.5),
					},
				},
			},
		},
	}

	ctx := context.Background()
	logger := NewLogMock()

	for name, tc := range tests {
		s := NewSwapper(tc.inputMarketService, nil, &MockedStorage{}, logger)

		t.Run(name, func(t *testing.T) {
			swapReport, err := s.ConsumeOrder(ctx, tc.inputOrder)
			if err != nil {
				t.Fatalf("consume order error: %v", err)
			}

			var got *models.SwapperReport
			for i, inputReport := range tc.inputSubOrderResults {
				inputReport.ID = swapReport.SubOrderToSend.ID
				got, err = s.ConsumeSubOrderResult(ctx, inputReport)
				assertError(t, err, tc.wantErr)

				if err = reportEquals(got, tc.wantSwapReports[i]); err != nil {
					t.Fatalf("reports do not match: %v", err)
				}

				if got.SubOrderToSend != nil {
					swapReport.SubOrderToSend = got.SubOrderToSend
				}
			}
		})
	}
}

type MockedStorage struct {
}

func (*MockedStorage) SaveSwap(_ context.Context, _ *models.Swap) error {
	return nil
}

func (*MockedStorage) GetAllSwaps(_ context.Context) ([]*models.Swap, error) {
	return nil, nil
}
func (*MockedStorage) DeleteSwap(_ context.Context, _ uint64) error {
	return nil
}
func (*MockedStorage) UpdateSwap(_ context.Context, _ *models.Swap) error {
	return nil
}
