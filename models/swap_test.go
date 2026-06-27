package models

import (
	"sync"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewSwap(t *testing.T) {
	tests := map[string]struct {
		inputOrder *Order
		inputSteps []*LinkedPairs
		wantSwap   *Swap
	}{
		"one step swap reversed": {
			inputOrder: &Order{
				ID:     1,
				Type:   OrderTypeSwap,
				Symbol: "USDT-XLM",
				Side:   SideSell,
			},
			inputSteps: []*LinkedPairs{{Pairs: []Pair{{Symbol: "XLM-USDT"}}}},
			wantSwap: &Swap{
				ID:     1,
				Type:   SwapTypeSellThenBuy,
				Status: SwapStatusNew,
				Order: &Order{
					ID:     1,
					Type:   OrderTypeSwap,
					Symbol: "USDT-XLM",
				},
				Steps: []*Step{
					{
						ID:            0,
						Status:        StepStatusNew,
						Side:          SideBuy,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "XLM-USDT",
						ReceivedAsset: "XLM",
						SpentAsset:    "USDT",
					},
				},
			},
		},
		"one step swap direct": {
			inputOrder: &Order{
				ID:     1,
				Type:   OrderTypeSwap,
				Symbol: "XLM-USDT",
				Side:   SideSell,
			},
			inputSteps: []*LinkedPairs{{Pairs: []Pair{{Symbol: "XLM-USDT"}}}},
			wantSwap: &Swap{
				ID:     1,
				Type:   SwapTypeSellThenBuy,
				Status: SwapStatusNew,
				Order: &Order{
					ID:     1,
					Type:   OrderTypeSwap,
					Symbol: "XLM-USDT",
				},
				Steps: []*Step{
					{
						ID:            0,
						Status:        StepStatusNew,
						Side:          SideSell,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "XLM-USDT",
						ReceivedAsset: "USDT",
						SpentAsset:    "XLM",
					},
				},
			},
		},
		"two step swap": {
			inputOrder: &Order{
				ID:     2,
				Type:   OrderTypeSwap,
				Symbol: "SOL-PEPE",
				Side:   SideSell,
			},
			inputSteps: []*LinkedPairs{{Pairs: []Pair{{Symbol: "SOL-USDT"}, {Symbol: "PEPE-USDT"}}}},
			wantSwap: &Swap{
				ID:     2,
				Type:   SwapTypeSellThenBuy,
				Status: SwapStatusNew,
				Order: &Order{
					ID:     2,
					Type:   OrderTypeSwap,
					Symbol: "SOL-PEPE",
				},
				Steps: []*Step{
					{
						ID:            0,
						Status:        StepStatusNew,
						Side:          SideSell,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "SOL-USDT",
						ReceivedAsset: "USDT",
						SpentAsset:    "SOL",
					},
					{
						ID:            1,
						Status:        StepStatusNew,
						Side:          SideBuy,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "PEPE-USDT",
						ReceivedAsset: "PEPE",
						SpentAsset:    "USDT",
					},
				},
			},
		},
		"three step swap from spec": {
			inputOrder: &Order{
				ID:     4,
				Type:   OrderTypeSwap,
				Symbol: "DOGE-SHIB",
				Side:   SideSell,
			},
			inputSteps: []*LinkedPairs{{Pairs: []Pair{{Symbol: "DOGE-USDT"}, {Symbol: "BNB-USDT"}, {Symbol: "SHIB-BNB"}}}},
			wantSwap: &Swap{
				ID:     4,
				Type:   SwapTypeSellThenBuy,
				Status: SwapStatusNew,
				Order: &Order{
					ID:     4,
					Type:   OrderTypeSwap,
					Symbol: "DOGE-SHIB",
				},
				Steps: []*Step{
					{
						ID:            0,
						Status:        StepStatusNew,
						Side:          SideSell,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "DOGE-USDT",
						ReceivedAsset: "USDT",
						SpentAsset:    "DOGE",
					},
					{
						ID:            1,
						Status:        StepStatusNew,
						Side:          SideBuy,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "BNB-USDT",
						ReceivedAsset: "BNB",
						SpentAsset:    "USDT",
					},
					{
						ID:            2,
						Status:        StepStatusNew,
						Side:          SideBuy,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "SHIB-BNB",
						ReceivedAsset: "SHIB",
						SpentAsset:    "BNB",
					},
				},
			},
		},
		"five step swap": {
			inputOrder: &Order{
				ID:     3,
				Type:   OrderTypeSwap,
				Symbol: "A-F",
				Side:   SideSell,
			},
			inputSteps: []*LinkedPairs{{Pairs: []Pair{{Symbol: "A-B"}, {Symbol: "C-B"}, {Symbol: "C-D"}, {Symbol: "E-D"}, {Symbol: "E-F"}}}},
			wantSwap: &Swap{
				ID:     3,
				Type:   SwapTypeSellThenBuy,
				Status: SwapStatusNew,
				Order: &Order{
					ID:     3,
					Type:   OrderTypeSwap,
					Symbol: "A-F",
				},
				Steps: []*Step{
					{
						ID:            0,
						Status:        StepStatusNew,
						Side:          SideSell,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "A-B",
						ReceivedAsset: "B",
						SpentAsset:    "A",
					},
					{
						ID:            1,
						Status:        StepStatusNew,
						Side:          SideBuy,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "C-B",
						ReceivedAsset: "C",
						SpentAsset:    "B",
					},
					{
						ID:            2,
						Status:        StepStatusNew,
						Side:          SideSell,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "C-D",
						ReceivedAsset: "D",
						SpentAsset:    "C",
					},
					{
						ID:            3,
						Status:        StepStatusNew,
						Side:          SideBuy,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "E-D",
						ReceivedAsset: "E",
						SpentAsset:    "D",
					},
					{
						ID:            4,
						Status:        StepStatusNew,
						Side:          SideSell,
						Type:          SwapTypeSellThenBuy,
						Symbol:        "E-F",
						ReceivedAsset: "F",
						SpentAsset:    "E",
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotSwap := NewSwap(tc.inputOrder, tc.inputSteps)
			if gotSwap.String() != tc.wantSwap.String() {
				t.Fatalf("swaps dont match:\n actual:%s\nwant:%s\n", gotSwap, tc.wantSwap)
			}
		})
	}
}

func TestNewSwapReturnsNilForEmptyPaths(t *testing.T) {
	order := &Order{ID: 1, Type: OrderTypeSwap, Symbol: "BTC-USDT", Side: SideSell}

	tests := map[string][]*LinkedPairs{
		"nil paths":        nil,
		"empty paths":      {},
		"nil first path":   {nil},
		"empty first path": {{Pairs: nil}},
	}

	for name, paths := range tests {
		t.Run(name, func(t *testing.T) {
			if got := NewSwap(order, paths); got != nil {
				t.Fatalf("expected nil swap, got %v", got)
			}
		})
	}
}

func TestNewSwapReturnsNilForNilOrder(t *testing.T) {
	if got := NewSwap(nil, []*LinkedPairs{{Pairs: []Pair{{Symbol: "BTC-USDT"}}}}); got != nil {
		t.Fatalf("expected nil swap, got %v", got)
	}
}

func TestNextStepOrderUsesQuoteTotalForReversedFirstStep(t *testing.T) {
	swapOrder := &Order{
		ID:              1,
		Type:            OrderTypeSwap,
		Symbol:          "USDT-XLM",
		Side:            SideSell,
		Amount:          decimal.NewFromInt(100),
		AvailableAmount: decimal.NewFromInt(100),
	}
	swap := NewSwap(swapOrder, []*LinkedPairs{{Pairs: []Pair{{Symbol: "XLM-USDT"}}}})

	subOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("next step order: %v", err)
	}

	if subOrder.Side != SideBuy {
		t.Fatalf("suborder side mismatch: got %s, want %s", subOrder.Side, SideBuy)
	}
	if !subOrder.Amount.IsZero() {
		t.Fatalf("suborder amount mismatch: got %s, want 0", subOrder.Amount)
	}
	if !subOrder.AvailableTotal.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("suborder available total mismatch: got %s, want 100", subOrder.AvailableTotal)
	}
}

func TestNextStepOrderUsesDerivedSideForThreeHopPath(t *testing.T) {
	swapOrder := &Order{
		ID:              1,
		Type:            OrderTypeSwap,
		Symbol:          "DOGE-SHIB",
		Side:            SideSell,
		Amount:          decimal.NewFromInt(100),
		AvailableAmount: decimal.NewFromInt(100),
	}
	swap := NewSwap(swapOrder, []*LinkedPairs{{
		Pairs: []Pair{{Symbol: "DOGE-USDT"}, {Symbol: "BNB-USDT"}, {Symbol: "SHIB-BNB"}},
	}})

	firstSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("first next step order: %v", err)
	}
	completedFirst := *firstSubOrder
	completedFirst.Status = OrderStatusCompleted
	completedFirst.ExecutedAmount = decimal.NewFromInt(100)
	completedFirst.ExecutedTotal = decimal.NewFromInt(200)
	swap.Update(&completedFirst)

	secondSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("second next step order: %v", err)
	}
	completedSecond := *secondSubOrder
	completedSecond.Status = OrderStatusCompleted
	completedSecond.ExecutedAmount = decimal.NewFromInt(10)
	completedSecond.ExecutedTotal = decimal.NewFromInt(200)
	swap.Update(&completedSecond)

	thirdSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("third next step order: %v", err)
	}

	if thirdSubOrder.Side != SideBuy {
		t.Fatalf("third suborder side mismatch: got %s, want %s", thirdSubOrder.Side, SideBuy)
	}
	if thirdSubOrder.Symbol != "SHIB-BNB" {
		t.Fatalf("third suborder symbol mismatch: got %s, want SHIB-BNB", thirdSubOrder.Symbol)
	}
	if !thirdSubOrder.AvailableTotal.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("third suborder available total mismatch: got %s, want 10", thirdSubOrder.AvailableTotal)
	}
}

func TestStepUpdateUsesExecutedTotalWhenProvided(t *testing.T) {
	tests := map[string]struct {
		step      *Step
		wantSpent decimal.Decimal
		wantRecv  decimal.Decimal
	}{
		"sell receives executed total": {
			step: &Step{
				Side: SideSell,
				Type: SwapTypeSellThenBuy,
			},
			wantSpent: decimal.NewFromInt(2),
			wantRecv:  decimal.NewFromInt(5),
		},
		"buy spends executed total": {
			step: &Step{
				Side: SideBuy,
				Type: SwapTypeSellThenBuy,
			},
			wantSpent: decimal.NewFromInt(5),
			wantRecv:  decimal.NewFromInt(2),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			tc.step.Update(&Order{
				Status:         OrderStatusCompleted,
				ExecutedAmount: decimal.NewFromInt(2),
				ExecutedTotal:  decimal.NewFromInt(5),
				AvgPrice:       decimal.Zero,
			})

			if !tc.step.SpentAmount.Equal(tc.wantSpent) {
				t.Fatalf("spent amount mismatch: got %s, want %s", tc.step.SpentAmount, tc.wantSpent)
			}
			if !tc.step.ReceivedAmount.Equal(tc.wantRecv) {
				t.Fatalf("received amount mismatch: got %s, want %s", tc.step.ReceivedAmount, tc.wantRecv)
			}
		})
	}
}

func TestSwapUpdateRejectsZeroExecutedAmountWithoutPanic(t *testing.T) {
	swapOrder := &Order{
		ID:              1,
		Account:         "account",
		Type:            OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            SideSell,
		Amount:          decimal.NewFromInt(1),
		AvailableAmount: decimal.NewFromInt(1),
	}
	swap := NewSwap(swapOrder, []*LinkedPairs{{Pairs: []Pair{{Symbol: "BTC-USDT"}}}})

	subOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("next step order: %v", err)
	}

	// Exchange reports Completed but with zero executed volume (zero matches).
	// AvgPrice = total / amount would divide by zero and panic.
	completed := *subOrder
	completed.Status = OrderStatusCompleted
	completed.ExecutedAmount = decimal.Zero
	completed.AvgPrice = decimal.Zero

	if !swap.Update(&completed) {
		t.Fatal("expected Update to accept the suborder result")
	}

	if swap.Status != SwapStatusRejected {
		t.Fatalf("status mismatch: got %s, want %s", swap.Status, SwapStatusRejected)
	}
	if swap.Order.RejectReason != RejectReasonNoMatches {
		t.Fatalf("reject reason mismatch: got %s, want %s", swap.Order.RejectReason, RejectReasonNoMatches)
	}
}

func TestSwapUpdateAndNextStepOrderAreRaceSafe(t *testing.T) {
	swapOrder := &Order{
		ID:              1,
		Account:         "account",
		Type:            OrderTypeSwap,
		Symbol:          "SOL-PEPE",
		Side:            SideSell,
		Amount:          decimal.NewFromInt(10),
		AvailableAmount: decimal.NewFromInt(10),
	}
	swap := NewSwap(swapOrder, []*LinkedPairs{{
		Pairs: []Pair{
			{Symbol: "SOL-USDT"},
			{Symbol: "PEPE-USDT"},
		},
	}})

	subOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("next step order: %v", err)
	}
	if subOrder == nil {
		t.Fatal("expected first suborder")
	}

	completedSubOrder := *subOrder
	completedSubOrder.Status = OrderStatusCompleted
	completedSubOrder.ExecutedAmount = decimal.NewFromInt(10)
	completedSubOrder.AvgPrice = decimal.NewFromInt(2)

	const goroutines = 8
	const iterations = 1000

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < iterations; j++ {
				order := completedSubOrder
				swap.Update(&order)
			}
		}()

		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < iterations; j++ {
				if _, err := swap.NextStepOrder(); err != nil {
					t.Errorf("next step order: %v", err)
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
