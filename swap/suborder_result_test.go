package swap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/orchidknight/swapper/markets"
	"github.com/orchidknight/swapper/models"
	"github.com/shopspring/decimal"
)

type consumeSubOrderResultTestCase struct {
	inputOrder           *models.Order
	inputSubOrderResults []*models.Order
	inputMarketService   MarketProvider
	wantSwapReports      []*models.SwapperReport
	wantErr              string
}

func TestSwapper_ConsumeSubOrderResult(t *testing.T) {
	tests := map[string]consumeSubOrderResultTestCase{
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
		"two phase swap sell then buy: reject second step leaves stranded intermediate asset": {
			inputOrder: &models.Order{
				ID:              1,
				Type:            models.OrderTypeSwap,
				Symbol:          "SHIB-DOGE",
				Side:            models.SideSell,
				AvailableAmount: decimal.NewFromFloat(100),
				Amount:          decimal.NewFromFloat(100),
			},
			inputSubOrderResults: []*models.Order{
				{
					Side:            models.SideSell,
					Status:          models.OrderStatusCompleted,
					Symbol:          "SHIB-USDT",
					ExecutedAmount:  decimal.NewFromFloat(100),
					AvailableAmount: decimal.Zero,
					Amount:          decimal.NewFromFloat(100),
					AvgPrice:        decimal.NewFromFloat(10),
					ExecutedTotal:   decimal.NewFromFloat(1000),
				},
				{
					Side:         models.SideBuy,
					Status:       models.OrderStatusRejected,
					Symbol:       "DOGE-USDT",
					RejectReason: models.RejectReasonNotEnoughLiquidity,
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
						Symbol:         "DOGE-USDT",
						Type:           models.OrderTypeMarket,
						Status:         models.OrderStatusNew,
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
						RejectReason:    models.RejectReasonNotEnoughLiquidity,
						StrandedAmount:  decimal.NewFromFloat(1000),
						StrandedAsset:   "USDT",
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
		t.Run(name, func(t *testing.T) {
			runConsumeSubOrderResultCase(ctx, t, logger, tc)
		})
	}
}

func runConsumeSubOrderResultCase(
	ctx context.Context,
	t *testing.T,
	logger models.Logger,
	tc consumeSubOrderResultTestCase,
) {
	t.Helper()

	s := NewSwapper(tc.inputMarketService, &MockedStorage{}, logger)
	swapReport, err := s.ConsumeOrder(ctx, tc.inputOrder)
	if err != nil {
		t.Fatalf("consume order error: %v", err)
	}

	for i, inputReport := range tc.inputSubOrderResults {
		inputReport.ID = swapReport.SubOrderToSend.ID
		got, err := s.ConsumeSubOrderResult(ctx, inputReport)
		assertError(t, err, tc.wantErr)

		if err = reportEquals(got, tc.wantSwapReports[i]); err != nil {
			t.Fatalf("reports do not match: %v", err)
		}

		if got.SubOrderToSend != nil {
			swapReport.SubOrderToSend = got.SubOrderToSend
		}
	}
}

func TestSwapper_ConsumeSubOrderResultUnlocksAfterNextStepOrderError(t *testing.T) {
	nextStepErr := errors.New("next step order failed")

	inProgressSwap, inProgressOrder := newNextStepOrderErrorInProgressCase(t)
	newSwap, newOrder := newNextStepOrderErrorNewCase(t)

	tests := map[string]struct {
		swap  *models.Swap
		order *models.Order
	}{
		"in progress next step":  {swap: inProgressSwap, order: inProgressOrder},
		"new rerouted next step": {swap: newSwap, order: newOrder},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewSwapper(nil, &MockedStorage{}, NewLogMock())
			s.nextStepOrder = func(*models.Swap) (*models.Order, error) {
				return nil, nextStepErr
			}
			s.activeSwaps[tc.swap.ID] = tc.swap
			s.orders[tc.order.ID] = tc.swap.ID

			_, err := s.ConsumeSubOrderResult(context.Background(), tc.order)
			if !errors.Is(err, nextStepErr) {
				t.Fatalf("got error %v, want %v", err, nextStepErr)
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				_, _ = s.ConsumeSubOrderResult(context.Background(), tc.order)
			}()

			select {
			case <-done:
			case <-time.After(200 * time.Millisecond):
				t.Fatal("ConsumeSubOrderResult blocked after NextStepOrder error")
			}
		})
	}
}

func TestSwapper_ConsumeSubOrderResultValidatesInput(t *testing.T) {
	s := NewSwapper(&markets.MarketService{Markets: inputOnePhaseMarkets}, &MockedStorage{}, NewLogMock())

	report, err := s.ConsumeSubOrderResult(context.Background(), nil)
	assertError(t, err, "order is nil")
	if report != nil {
		t.Fatalf("report mismatch: got %v, want nil", report)
	}
}

func TestSwapper_LoadOrdersRestoresOutstandingSubOrder(t *testing.T) {
	ctx := context.Background()
	storage := newMemoryStorage()
	marketService := &markets.MarketService{Markets: inputOnePhaseMarkets}
	initialSwapper := NewSwapper(marketService, storage, NewLogMock())

	report, err := initialSwapper.ConsumeOrder(ctx, &models.Order{
		ID:              10,
		Type:            models.OrderTypeSwap,
		Status:          models.OrderStatusNew,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	})
	if err != nil {
		t.Fatalf("consume order: %v", err)
	}

	restoredSwapper := NewSwapper(marketService, storage, NewLogMock())
	if loadErr := restoredSwapper.LoadOrders(ctx); loadErr != nil {
		t.Fatalf("load orders: %v", loadErr)
	}

	result, err := restoredSwapper.ConsumeSubOrderResult(ctx, &models.Order{
		ID:             report.SubOrderToSend.ID,
		Type:           models.OrderTypeMarket,
		Status:         models.OrderStatusCompleted,
		Symbol:         "BTC-USDT",
		Side:           models.SideSell,
		ExecutedAmount: decimal.NewFromInt(1),
		ExecutedTotal:  decimal.NewFromInt(100),
		AvgPrice:       decimal.NewFromInt(100),
	})
	if err != nil {
		t.Fatalf("consume suborder result: %v", err)
	}
	if result == nil || result.ResultSwapOrder == nil {
		t.Fatalf("expected completed swap report, got %v", result)
	}
	if result.ResultSwapOrder.Status != models.OrderStatusCompleted {
		t.Fatalf("swap order status mismatch: got %s, want %s", result.ResultSwapOrder.Status, models.OrderStatusCompleted)
	}
}

func TestSwapper_LoadOrdersRebuildsIndexesAtomically(t *testing.T) {
	ctx := context.Background()
	storage := newMemoryStorage()
	s := NewSwapper(&markets.MarketService{Markets: inputOnePhaseMarkets}, storage, NewLogMock())

	staleSwap := models.NewSwap(&models.Order{
		ID:              501,
		Type:            models.OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	}, []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "BTC-USDT"}}}})
	staleSubOrder, err := staleSwap.NextStepOrder()
	if err != nil {
		t.Fatalf("stale next step order: %v", err)
	}
	s.activeSwaps[staleSwap.ID] = staleSwap
	s.orders[staleSubOrder.ID] = staleSwap.ID

	freshSwap := models.NewSwap(&models.Order{
		ID:              502,
		Type:            models.OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(2),
		AvailableAmount: decimal.NewFromInt(2),
	}, []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "BTC-USDT"}}}})
	freshSubOrder, err := freshSwap.NextStepOrder()
	if err != nil {
		t.Fatalf("fresh next step order: %v", err)
	}
	storage.swaps[freshSwap.ID] = freshSwap

	if err := s.LoadOrders(ctx); err != nil {
		t.Fatalf("load orders: %v", err)
	}

	if _, ok := s.activeSwaps[staleSwap.ID]; ok {
		t.Fatal("stale active swap was not removed")
	}
	if _, ok := s.orders[staleSubOrder.ID]; ok {
		t.Fatal("stale suborder mapping was not removed")
	}
	if got := s.orders[freshSubOrder.ID]; got != freshSwap.ID {
		t.Fatalf("fresh suborder mapping mismatch: got %d, want %d", got, freshSwap.ID)
	}
}

func TestSwapper_ConsumeSubOrderResultRemovesFinishedSwapFromMemory(t *testing.T) {
	ctx := context.Background()
	s := NewSwapper(&markets.MarketService{Markets: inputOnePhaseMarkets}, &MockedStorage{}, NewLogMock())

	report, err := s.ConsumeOrder(ctx, &models.Order{
		ID:              11,
		Type:            models.OrderTypeSwap,
		Status:          models.OrderStatusNew,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	})
	if err != nil {
		t.Fatalf("consume order: %v", err)
	}

	_, err = s.ConsumeSubOrderResult(ctx, &models.Order{
		ID:             report.SubOrderToSend.ID,
		Type:           models.OrderTypeMarket,
		Status:         models.OrderStatusCompleted,
		Symbol:         "BTC-USDT",
		Side:           models.SideSell,
		ExecutedAmount: decimal.NewFromInt(1),
		ExecutedTotal:  decimal.NewFromInt(100),
		AvgPrice:       decimal.NewFromInt(100),
	})
	if err != nil {
		t.Fatalf("consume suborder result: %v", err)
	}

	if got := len(s.activeSwaps); got != 0 {
		t.Fatalf("active swaps count mismatch: got %d, want 0", got)
	}
}

func TestSwapper_ConsumeSubOrderResultCancelsSwap(t *testing.T) {
	ctx := context.Background()
	s := NewSwapper(&markets.MarketService{Markets: inputOnePhaseMarkets}, &MockedStorage{}, NewLogMock())

	report, err := s.ConsumeOrder(ctx, &models.Order{
		ID:              12,
		Type:            models.OrderTypeSwap,
		Status:          models.OrderStatusNew,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	})
	if err != nil {
		t.Fatalf("consume order: %v", err)
	}

	result, err := s.ConsumeSubOrderResult(ctx, &models.Order{
		ID:     report.SubOrderToSend.ID,
		Type:   models.OrderTypeMarket,
		Status: models.OrderStatusCanceled,
		Symbol: "BTC-USDT",
		Side:   models.SideSell,
	})
	if err != nil {
		t.Fatalf("consume suborder result: %v", err)
	}
	if result == nil || result.ResultSwapOrder == nil {
		t.Fatalf("expected canceled swap report, got %v", result)
	}
	if result.ResultSwapOrder.Status != models.OrderStatusCanceled {
		t.Fatalf("swap order status mismatch: got %s, want %s", result.ResultSwapOrder.Status, models.OrderStatusCanceled)
	}
	if got := len(s.activeSwaps); got != 0 {
		t.Fatalf("active swaps count mismatch: got %d, want 0", got)
	}
}

func newNextStepOrderErrorInProgressCase(t *testing.T) (*models.Swap, *models.Order) {
	t.Helper()

	swapOrder := &models.Order{
		ID:              101,
		Type:            models.OrderTypeSwap,
		Symbol:          "SOL-PEPE",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(10),
		AvailableAmount: decimal.NewFromInt(10),
	}
	swap := models.NewSwap(swapOrder, []*models.LinkedPairs{
		{Pairs: []models.Pair{{Symbol: "SOL-USDT"}, {Symbol: "PEPE-USDT"}}},
	})
	subOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("create initial sub order: %v", err)
	}

	return swap, &models.Order{
		ID:             subOrder.ID,
		Type:           subOrder.Type,
		Symbol:         subOrder.Symbol,
		Side:           subOrder.Side,
		Status:         models.OrderStatusCompleted,
		ExecutedAmount: decimal.NewFromInt(10),
		AvgPrice:       decimal.NewFromInt(10),
	}
}

func newNextStepOrderErrorNewCase(t *testing.T) (*models.Swap, *models.Order) {
	t.Helper()

	swapOrder := &models.Order{
		ID:              102,
		Type:            models.OrderTypeSwap,
		Symbol:          "SHIB-DOGE",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(100),
		AvailableAmount: decimal.NewFromInt(100),
	}
	swap := models.NewSwap(swapOrder, []*models.LinkedPairs{
		{Pairs: []models.Pair{{Symbol: "SHIB-USDT"}, {Symbol: "DOGE-USDT"}}},
		{Pairs: []models.Pair{{Symbol: "SHIB-USDC"}, {Symbol: "BIN-USDC"}, {Symbol: "BIN-DOGE"}}},
	})
	subOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("create initial sub order: %v", err)
	}

	return swap, &models.Order{
		ID:     subOrder.ID,
		Type:   subOrder.Type,
		Symbol: subOrder.Symbol,
		Side:   subOrder.Side,
		Status: models.OrderStatusRejected,
		Amount: decimal.NewFromInt(100),
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

type memoryStorage struct {
	swaps map[uint64]*models.Swap
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{swaps: make(map[uint64]*models.Swap)}
}

func (s *memoryStorage) SaveSwap(_ context.Context, swap *models.Swap) error {
	s.swaps[swap.ID] = swap

	return nil
}

func (s *memoryStorage) GetAllSwaps(_ context.Context) ([]*models.Swap, error) {
	swaps := make([]*models.Swap, 0, len(s.swaps))
	for _, swap := range s.swaps {
		swaps = append(swaps, swap)
	}

	return swaps, nil
}

func (s *memoryStorage) DeleteSwap(_ context.Context, id uint64) error {
	delete(s.swaps, id)

	return nil
}

func (s *memoryStorage) UpdateSwap(_ context.Context, swap *models.Swap) error {
	s.swaps[swap.ID] = swap

	return nil
}
