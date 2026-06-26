package swap_test

import (
	"context"
	"fmt"

	"github.com/orchidknight/swapper/models"
	"github.com/orchidknight/swapper/swap"
	"github.com/shopspring/decimal"
)

type exampleMarketProvider struct{}

func (exampleMarketProvider) GetAllSwapPairs(models.Symbol) ([]*models.LinkedPairs, error) {
	return []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "BTC-USDT"}}}}, nil
}

func (exampleMarketProvider) GetMarket(symbol models.Symbol) *models.MarketPair {
	return &models.MarketPair{
		Symbol:         symbol,
		BasePrecision:  8,
		QuotePrecision: 2,
		TradingEnabled: true,
	}
}

type exampleStorage struct {
	swaps map[uint64]*models.Swap
}

func (s *exampleStorage) SaveSwap(_ context.Context, activeSwap *models.Swap) error {
	s.swaps[activeSwap.ID] = activeSwap

	return nil
}

func (s *exampleStorage) GetAllSwaps(context.Context) ([]*models.Swap, error) {
	result := make([]*models.Swap, 0, len(s.swaps))
	for _, activeSwap := range s.swaps {
		result = append(result, activeSwap)
	}

	return result, nil
}

func (s *exampleStorage) DeleteSwap(_ context.Context, id uint64) error {
	delete(s.swaps, id)

	return nil
}

func (s *exampleStorage) UpdateSwap(_ context.Context, activeSwap *models.Swap) error {
	s.swaps[activeSwap.ID] = activeSwap

	return nil
}

type exampleLogger struct{}

func (exampleLogger) Debug(string, string, ...any) {}
func (exampleLogger) Info(string, string, ...any)  {}
func (exampleLogger) Warn(string, string, ...any)  {}
func (exampleLogger) Error(string, string, ...any) {}
func (exampleLogger) Fatal(string, string, ...any) {}

func ExampleSwapper_sellSwap() {
	ctx := context.Background()
	storage := &exampleStorage{swaps: make(map[uint64]*models.Swap)}
	swapper := swap.NewSwapper(exampleMarketProvider{}, storage, exampleLogger{})

	report, err := swapper.ConsumeOrder(ctx, &models.Order{
		ID:              models.NewID(),
		Type:            models.OrderTypeSwap,
		Status:          models.OrderStatusNew,
		Symbol:          "BTC-USDT",
		Side:            models.SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	})
	if err != nil {
		panic(err)
	}

	suborder := report.SubOrderToSend
	fmt.Println(suborder.Symbol)

	result, err := swapper.ConsumeSubOrderResult(ctx, &models.Order{
		ID:             suborder.ID,
		Type:           suborder.Type,
		Status:         models.OrderStatusCompleted,
		Symbol:         suborder.Symbol,
		Side:           suborder.Side,
		ExecutedAmount: decimal.NewFromInt(1),
		AvgPrice:       decimal.NewFromInt(100),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(result.ResultSwapOrder.Status)

	// Output:
	// BTC-USDT
	// Completed
}
