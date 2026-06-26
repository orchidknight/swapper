package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNextStepOrderRoundsFirstSellSubOrderDownToBasePrecision(t *testing.T) {
	availableAmount := mustDecimal(t, "1.23456789")
	swap := NewSwap(&Order{
		ID:              1,
		Type:            OrderTypeSwap,
		Symbol:          "BTC-USDT",
		Side:            SideSell,
		Amount:          availableAmount,
		AvailableAmount: availableAmount,
	}, []*LinkedPairs{{
		Pairs: []Pair{
			{
				Symbol:         "BTC-USDT",
				BasePrecision:  4,
				QuotePrecision: 2,
			},
		},
	}})

	subOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("next step order: %v", err)
	}

	wantAmount := mustDecimal(t, "1.2345")
	if !subOrder.Amount.Equal(wantAmount) {
		t.Fatalf("suborder amount mismatch: got %s, want %s", subOrder.Amount, wantAmount)
	}
	if !subOrder.AvailableAmount.Equal(wantAmount) {
		t.Fatalf("suborder available amount mismatch: got %s, want %s", subOrder.AvailableAmount, wantAmount)
	}
	if subOrder.AvailableAmount.GreaterThan(availableAmount) {
		t.Fatalf("rounded amount exceeds available input: got %s, available %s", subOrder.AvailableAmount, availableAmount)
	}

	completedSubOrder := *subOrder
	completedSubOrder.Status = OrderStatusCompleted
	completedSubOrder.ExecutedAmount = wantAmount
	completedSubOrder.AvgPrice = mustDecimal(t, "100")
	swap.Update(&completedSubOrder)

	wantDust := mustDecimal(t, "0.00006789")
	if !swap.Order.AvailableAmount.Equal(wantDust) {
		t.Fatalf("swap order dust mismatch: got %s, want %s", swap.Order.AvailableAmount, wantDust)
	}
}

func TestNextStepOrderRoundsNextBuySubOrderDownToQuotePrecision(t *testing.T) {
	swap := NewSwap(&Order{
		ID:              2,
		Type:            OrderTypeSwap,
		Symbol:          "BTC-DOGE",
		Side:            SideSell,
		Amount:          mustDecimal(t, "1"),
		AvailableAmount: mustDecimal(t, "1"),
	}, []*LinkedPairs{{
		Pairs: []Pair{
			{
				Symbol:         "BTC-USDT",
				BasePrecision:  8,
				QuotePrecision: 6,
			},
			{
				Symbol:         "DOGE-USDT",
				BasePrecision:  3,
				QuotePrecision: 2,
			},
		},
	}})

	firstSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("first next step order: %v", err)
	}

	completedFirstSubOrder := *firstSubOrder
	completedFirstSubOrder.Status = OrderStatusCompleted
	completedFirstSubOrder.ExecutedAmount = mustDecimal(t, "1")
	completedFirstSubOrder.AvgPrice = mustDecimal(t, "12.345678")
	swap.Update(&completedFirstSubOrder)

	secondSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("second next step order: %v", err)
	}

	wantAvailableTotal := mustDecimal(t, "12.34")
	if !secondSubOrder.AvailableTotal.Equal(wantAvailableTotal) {
		t.Fatalf("suborder available total mismatch: got %s, want %s", secondSubOrder.AvailableTotal, wantAvailableTotal)
	}
	if secondSubOrder.AvailableTotal.GreaterThan(swap.Steps[0].ReceivedAmount) {
		t.Fatalf("rounded total exceeds received input: got %s, received %s", secondSubOrder.AvailableTotal, swap.Steps[0].ReceivedAmount)
	}
}

func TestNextStepOrderRoundsNextSellSubOrderDownToBasePrecision(t *testing.T) {
	swap := NewSwap(&Order{
		ID:              3,
		Type:            OrderTypeSwap,
		Symbol:          "AAA-DDD",
		Side:            SideSell,
		Amount:          mustDecimal(t, "1"),
		AvailableAmount: mustDecimal(t, "1"),
	}, []*LinkedPairs{{
		Pairs: []Pair{
			{
				Symbol:         "AAA-BBB",
				BasePrecision:  8,
				QuotePrecision: 6,
			},
			{
				Symbol:         "CCC-BBB",
				BasePrecision:  8,
				QuotePrecision: 6,
			},
			{
				Symbol:         "CCC-DDD",
				BasePrecision:  3,
				QuotePrecision: 2,
			},
		},
	}})

	firstSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("first next step order: %v", err)
	}

	completedFirstSubOrder := *firstSubOrder
	completedFirstSubOrder.Status = OrderStatusCompleted
	completedFirstSubOrder.ExecutedAmount = mustDecimal(t, "1")
	completedFirstSubOrder.AvgPrice = mustDecimal(t, "2")
	swap.Update(&completedFirstSubOrder)

	secondSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("second next step order: %v", err)
	}

	completedSecondSubOrder := *secondSubOrder
	completedSecondSubOrder.Status = OrderStatusCompleted
	completedSecondSubOrder.ExecutedAmount = mustDecimal(t, "3.456789")
	completedSecondSubOrder.AvgPrice = mustDecimal(t, "0.5")
	swap.Update(&completedSecondSubOrder)

	thirdSubOrder, err := swap.NextStepOrder()
	if err != nil {
		t.Fatalf("third next step order: %v", err)
	}

	wantAvailableAmount := mustDecimal(t, "3.456")
	if !thirdSubOrder.AvailableAmount.Equal(wantAvailableAmount) {
		t.Fatalf("suborder available amount mismatch: got %s, want %s", thirdSubOrder.AvailableAmount, wantAvailableAmount)
	}
	if !thirdSubOrder.AvailableTotal.IsZero() {
		t.Fatalf("suborder available total mismatch: got %s, want zero", thirdSubOrder.AvailableTotal)
	}
	if thirdSubOrder.AvailableAmount.GreaterThan(swap.Steps[1].ReceivedAmount) {
		t.Fatalf("rounded amount exceeds received input: got %s, received %s", thirdSubOrder.AvailableAmount, swap.Steps[1].ReceivedAmount)
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
