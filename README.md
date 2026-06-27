[![CI](https://github.com/orchidknight/swapper/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/orchidknight/swapper/actions/workflows/ci.yml)

# swapper

`swapper` is a small Go library for building a chain of market suborders from a single swap order.

The library is a core domain component: it does not connect to exchanges, databases, message queues, or HTTP APIs. Callers provide market data, persistence, and logging through small interfaces, then execute returned suborders in their own infrastructure.

## What It Does

A swap order describes an intent to convert one asset into another. `swapper` turns that intent into one market order at a time:

```text
swap order: sell BTC for DOGE
  → suborder 1: sell BTC-USDT
  → suborder 2: buy DOGE-USDT
  → completed swap order
```

The caller owns the side effects:

1. Call `ConsumeOrder` with a `models.Order` of type `OrderTypeSwap`.
2. Send `SwapperReport.SubOrderToSend` to an exchange.
3. Feed each exchange result back through `ConsumeSubOrderResult`.
4. Repeat until `SwapperReport.ResultSwapOrder` is returned.

## Suborder Quantity Contract

`swapper` returns market suborders in the quantity shape expected by exchange adapters:

- Sell suborders use `Amount` and `AvailableAmount` as the base-asset quantity to sell.
- Buy suborders use `AvailableTotal` as the quote-asset budget for a market buy. `Amount` is intentionally left empty for these buy legs; adapters should map `AvailableTotal` to the exchange's quote-budget field, such as `quoteOrderQty` semantics.
- Precision is applied before the suborder is returned. `BasePrecision` truncates sell `Amount` / `AvailableAmount`; `QuotePrecision` truncates buy `AvailableTotal`.

This buy-leg contract is part of the current sell-swap flow. It is separate from user-facing buy swaps by target output amount, which are not implemented and are rejected at the `ConsumeOrder` boundary.

## Input Order Contract

Accepted swap orders must satisfy these invariants:

- `ID` is non-zero and not already active in the `Swapper`.
- `Type` is `OrderTypeSwap`.
- `Side` is `SideSell`; `SideBuy` is explicitly rejected.
- `Status` is empty, `OrderStatusUnspecified`, or `OrderStatusNew`; terminal statuses are rejected.
- `Amount` and `AvailableAmount` are positive, and `AvailableAmount <= Amount`.

## Install

```bash
go get github.com/orchidknight/swapper
```

## Minimal Example

The example below implements the required ports: `MarketProvider`, `Storage`, and `Logger`.

```go
package main

import (
	"context"
	"fmt"

	"github.com/orchidknight/swapper/models"
	"github.com/orchidknight/swapper/swap"
	"github.com/shopspring/decimal"
)

type marketProvider struct{}

func (marketProvider) GetAllSwapPairs(models.Symbol) ([]*models.LinkedPairs, error) {
	return []*models.LinkedPairs{{Pairs: []models.Pair{{Symbol: "BTC-USDT"}}}}, nil
}

func (marketProvider) GetMarket(symbol models.Symbol) *models.MarketPair {
	return &models.MarketPair{
		Symbol:         symbol,
		BasePrecision:  8,
		QuotePrecision: 2,
		TradingEnabled: true,
	}
}

type storage struct {
	swaps map[uint64]*models.Swap
}

func (s *storage) SaveSwap(_ context.Context, activeSwap *models.Swap) error {
	s.swaps[activeSwap.ID] = activeSwap
	return nil
}

func (s *storage) GetAllSwaps(context.Context) ([]*models.Swap, error) {
	result := make([]*models.Swap, 0, len(s.swaps))
	for _, activeSwap := range s.swaps {
		result = append(result, activeSwap)
	}
	return result, nil
}

func (s *storage) DeleteSwap(_ context.Context, id uint64) error {
	delete(s.swaps, id)
	return nil
}

func (s *storage) UpdateSwap(_ context.Context, activeSwap *models.Swap) error {
	s.swaps[activeSwap.ID] = activeSwap
	return nil
}

type logger struct{}

func (logger) Debug(string, string, ...any) {}
func (logger) Info(string, string, ...any)  {}
func (logger) Warn(string, string, ...any)  {}
func (logger) Error(string, string, ...any) {}
func (logger) Fatal(string, string, ...any) {}

func main() {
	ctx := context.Background()
	storage := &storage{swaps: make(map[uint64]*models.Swap)}
	swapper := swap.NewSwapper(marketProvider{}, storage, logger{})

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
}
```

## Scope

- Only sell swaps are supported. Buy swaps by target output amount are explicitly rejected with `RejectReasonBuySwapsNotSupported`; this is a deferred feature from `swapper-spec.md` §10.
- If a later step is rejected after an earlier step already completed, `ResultSwapOrder` is rejected and includes `StrandedAmount` plus `StrandedAsset` for the intermediate funds left on the caller's account. `swapper` does not unwind or compensate that position.
- Terminal partial fills are surfaced through `StrandedAmount` / `StrandedAsset` when the swap cannot continue. A `Completed` suborder with zero effective execution rejects the swap with `RejectReasonNoMatches`.
- Precision handling is applied from market metadata: base precision truncates sell amounts, quote precision truncates buy totals. Unknown precision is represented as `models.PrecisionUnknown` (`-1`) and does not truncate.
- `swapper` is not an exchange adapter and does not place orders itself. It returns suborders through `SwapperReport`.
- `swapper` is not a database layer. `models.Storage` is a caller-provided port for persisting active swaps.
- `swapper` is process-local. ID generation is unique inside one process; multi-process deployments need an external uniqueness strategy or a node-aware ID layer.
- `markets.MarketService` limits path discovery by default to prevent explosive graph traversal. Set `MaxHops` / `MaxPaths` explicitly to tune this for a deployment.

## Public Packages

- `models`: domain types such as orders, swaps, symbols, market pairs, storage, and logger ports.
- `markets`: in-memory market graph implementation that discovers candidate swap paths.
- `swap`: orchestration service that consumes initial swap orders and suborder execution results.

## License

This project is licensed under the terms in [LICENSE](LICENSE).
