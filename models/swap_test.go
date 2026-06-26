package models

import (
	"fmt"
	"testing"
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
			fmt.Println("got swap: ", gotSwap)
			fmt.Println("want swap: ", tc.wantSwap)
			if gotSwap.String() != tc.wantSwap.String() {
				t.Fatalf("swaps dont match:\n actual:%s\nwant:%s\n", gotSwap, tc.wantSwap)
			}
		})
	}
}
