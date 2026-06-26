package models

import (
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

// SwapStatus identifies the current lifecycle state of a swap.
type SwapStatus string

// StepStatus identifies the current lifecycle state of a swap step.
type StepStatus string

// SwapType describes the order of generated market operations.
type SwapType string

const (
	// StepStatusNew means the step has not started execution.
	StepStatusNew = "new"
	// StepStatusInProgress means the step has a partially executed suborder.
	StepStatusInProgress = "inProgress"
	// StepStatusCompleted means the step has completed.
	StepStatusCompleted = "completed"
	// StepStatusRejected means the step was rejected.
	StepStatusRejected = "rejected"
	// StepStatusCanceled means the step was canceled.
	StepStatusCanceled = "canceled"

	// SwapStatusNew means the swap is waiting for its next suborder.
	SwapStatusNew = "new"
	// SwapStatusInProgress means at least one suborder is active or partially completed.
	SwapStatusInProgress = "inProgress"
	// SwapStatusCompleted means all swap steps completed.
	SwapStatusCompleted = "completed"
	// SwapStatusRejected means the swap cannot continue.
	SwapStatusRejected = "rejected"
	// SwapStatusCanceled means the swap was canceled.
	SwapStatusCanceled = "canceled"

	// SwapTypeUnspecified is the zero-value swap type.
	SwapTypeUnspecified = "unspecified"
	// SwapTypeBuyThenSell is reserved for buy swaps and is currently rejected.
	SwapTypeBuyThenSell = "buy-then-sell"
	// SwapTypeSellThenBuy is the supported sell-swap execution mode.
	SwapTypeSellThenBuy = "sell-then-buy"
)

// Swap tracks the state of a swap order and its generated suborders.
type Swap struct {
	ID            uint64
	Type          SwapType
	Status        SwapStatus
	Order         *Order
	SubOrders     map[uint64]int
	Steps         []*Step
	CurrentStep   int
	RejectReason  RejectReason
	RejectedSteps []*Step
	Paths         []*LinkedPairs
	CurrentPath   int

	mu        sync.RWMutex
	CreatedAt time.Time
}

// String returns a debug representation of the swap.
func (s *Swap) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf("%d %s %s %v %v", s.ID, s.Type, s.Status, s.SubOrders, s.Steps)
}

// Step describes one market operation in a swap path.
type Step struct {
	ID             int             `json:"id"`
	Status         StepStatus      `json:"status"`
	Side           Side            `json:"side"`
	Order          *Order          `json:"order"`
	Type           SwapType        `json:"type"`
	Symbol         Symbol          `json:"symbol"`
	ReceivedAmount decimal.Decimal `json:"receivedAmount"`
	ReceivedAsset  string          `json:"receivedAsset"`
	SpentAmount    decimal.Decimal `json:"spentAmount"`
	SpentAsset     string          `json:"spentAsset"`
	BasePrecision  int32           `json:"basePrecision"`
	QuotePrecision int32           `json:"quotePrecision"`
}

// String returns a debug representation of the step.
func (s *Step) String() string {
	return fmt.Sprintf("{%d %s %s %s Spent: %v%s Received: %v%s}", s.ID, s.Status, s.Side, s.Symbol, s.SpentAmount, s.SpentAsset, s.ReceivedAmount, s.ReceivedAsset)
}

// Update applies an exchange order update to the step.
func (s *Step) Update(o *Order) {
	s.Order = o
	switch o.Status {
	case OrderStatusCompleted:
		s.Status = StepStatusCompleted
	case OrderStatusPartiallyCompleted:
		s.Status = StepStatusInProgress
	case OrderStatusRejected:
		s.Status = StepStatusRejected
	case OrderStatusCanceled:
		s.Status = StepStatusCanceled
	}

	if s.Type == SwapTypeBuyThenSell {
		// not supported, see spec §10
		s.Status = StepStatusRejected

		return
	}

	switch s.Side {
	case SideBuy:
		s.ReceivedAmount = o.ExecutedAmount
		s.SpentAmount = o.ExecutedAmount.Mul(o.AvgPrice)
	case SideSell:
		s.ReceivedAmount = o.ExecutedAmount.Mul(o.AvgPrice)
		s.SpentAmount = o.ExecutedAmount
	}
}

// Update applies a suborder result to the swap and reports whether it was accepted.
func (s *Swap) Update(o *Order) bool {
	if o == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stepIndex, ok := s.SubOrders[o.ID]

	if !ok {
		return false
	}

	s.Steps[stepIndex].Update(o)
	s.CurrentStep = stepIndex

	if s.Type == SwapTypeBuyThenSell {
		// not supported, see spec §10
		s.Status = SwapStatusRejected
		s.Order.Reject(RejectReasonBuySwapsNotSupported)
		s.RejectReason = RejectReasonBuySwapsNotSupported

		return true
	}

	lastStep := s.Steps[len(s.Steps)-1]
	firstStep := s.Steps[0]

	if lastStep.Status == StepStatusCompleted {
		s.Status = SwapStatusCompleted
		s.Order.Status = OrderStatusCompleted

		// Для sell-свапа amount — потраченный входящий актив, total — полученный конечный актив.
		// Buy-свапы не поддержаны, см. spec §10.
		switch s.Order.Side {
		case SideSell:
			s.Order.ExecutedAmount = firstStep.SpentAmount
			s.Order.ExecutedTotal = lastStep.ReceivedAmount
			s.Order.AvailableAmount = s.Order.AvailableAmount.Sub(s.Order.ExecutedAmount)
			s.Order.AvgPrice = s.Order.ExecutedTotal.Div(s.Order.ExecutedAmount)
			s.Order.Price = s.Order.AvgPrice
		case SideBuy:
			// not supported, see spec §10
			s.Status = SwapStatusRejected
			s.Order.Reject(RejectReasonBuySwapsNotSupported)
			s.RejectReason = RejectReasonBuySwapsNotSupported

			return true
		}

		return true
	}

	if o.Status == OrderStatusRejected && s.CurrentStep == 0 && s.CurrentPath != len(s.Paths)-1 {
		s.RejectedSteps = append(s.RejectedSteps, s.Steps[stepIndex])

		s.CurrentPath++
		newPathIndex := s.CurrentPath

		if newPathIndex >= len(s.Paths) {
			s.Status = SwapStatusRejected
			s.Order.Status = OrderStatusRejected
			s.RejectReason = o.RejectReason

			return true
		}

		s.Steps = swapSteps(s.Order.Symbol.String(), s.Paths[newPathIndex].Pairs, s.Type)
		s.CurrentStep = 0
		s.Status = SwapStatusNew

		return true
	}

	if o.Status == OrderStatusRejected {
		s.Status = SwapStatusRejected
		s.Order.Status = OrderStatusRejected

		return true
	}

	s.Status = SwapStatusInProgress

	return true
}

// NextStepOrder creates the next executable market suborder for the swap.
func (s *Swap) NextStepOrder() (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, step := range s.Steps {
		if step.Status != StepStatusNew {
			continue
		}

		suborderID := NewID()

		if i == 0 {
			amount := step.truncateBase(s.Order.Amount)
			availableAmount := step.truncateBase(s.Order.AvailableAmount)
			order := &Order{
				ID:              suborderID,
				Account:         s.Order.Account,
				Symbol:          step.Symbol,
				Type:            OrderTypeMarket,
				Side:            step.Side,
				Status:          OrderStatusNew,
				Amount:          amount,
				AvailableAmount: availableAmount,
				CreatedAt:       time.Now().UTC(),
			}

			s.SubOrders[order.ID] = i
			s.Steps[i].Order = order

			return order, nil
		}

		prevStep := s.Steps[i-1]
		orderSide := prevStep.Side.Opposite()

		order := &Order{
			ID:        suborderID,
			Account:   s.Order.Account,
			Symbol:    step.Symbol,
			Type:      OrderTypeMarket,
			Side:      orderSide,
			Status:    OrderStatusNew,
			CreatedAt: time.Now().UTC(),
		}
		switch orderSide {
		case SideSell:
			order.Amount = step.truncateBase(prevStep.ReceivedAmount)
			order.AvailableAmount = order.Amount
		case SideBuy:
			order.AvailableTotal = step.truncateQuote(prevStep.ReceivedAmount)
		}

		s.SubOrders[order.ID] = i
		s.Steps[i].Order = order
		s.CurrentStep = i

		return order, nil
	}

	return nil, nil
}

func (s *Step) truncateBase(value decimal.Decimal) decimal.Decimal {
	return truncatePrecision(value, s.BasePrecision)
}

func (s *Step) truncateQuote(value decimal.Decimal) decimal.Decimal {
	return truncatePrecision(value, s.QuotePrecision)
}

func truncatePrecision(value decimal.Decimal, precision int32) decimal.Decimal {
	if precision < 0 {
		return value
	}

	return value.Truncate(precision)
}

// SwapperReport is returned after consuming a swap order or suborder result.
type SwapperReport struct {
	SubOrderToSend  *Order
	ResultSwapOrder *Order
}

// String returns a debug representation of the report.
func (sr *SwapperReport) String() string {
	return fmt.Sprintf("{Order: %v}", sr.SubOrderToSend)
}

// StepPairs returns the ordered market symbols in the current swap path.
func (s *Swap) StepPairs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pairs []string

	for _, step := range s.Steps {
		pairs = append(pairs, step.Symbol.String())
	}

	return pairs
}

// PairStep describes a market symbol and side pair.
type PairStep struct {
	Pair string
	Side Side
}

// NewSwap creates a swap state machine for an order and candidate paths.
func NewSwap(o *Order, allSteps []*LinkedPairs) *Swap {
	swapType := swapType(o)

	s := &Swap{
		ID:            o.ID,
		Status:        SwapStatusNew,
		Type:          swapType,
		Order:         o,
		Steps:         swapSteps(o.Symbol.String(), allSteps[0].Pairs, swapType),
		SubOrders:     make(map[uint64]int),
		CurrentStep:   0,
		RejectedSteps: make([]*Step, 0),
		Paths:         allSteps,
		CurrentPath:   0,
		CreatedAt:     time.Now().UTC(),
	}

	return s
}

func swapType(o *Order) SwapType {
	switch o.Side {
	case SideSell:
		return SwapTypeSellThenBuy
	case SideBuy:
		return SwapTypeBuyThenSell
	default:
		return SwapTypeUnspecified
	}
}

func swapSteps(initialSymbol string, s []Pair, swapType SwapType) []*Step {
	if swapType == SwapTypeBuyThenSell {
		// not supported, see spec §10
		return nil
	}

	if len(s) == 1 {
		if initialSymbol == s[0].Symbol.String() {
			pair := s[0]
			symbol := pair.Symbol
			step := &Step{
				ID:             0,
				Status:         StepStatusNew,
				Side:           SideSell,
				Type:           SwapTypeSellThenBuy,
				Symbol:         symbol,
				ReceivedAsset:  symbol.QuoteAsset(),
				SpentAsset:     symbol.BaseAsset(),
				BasePrecision:  pair.BasePrecision,
				QuotePrecision: pair.QuotePrecision,
			}

			return []*Step{step}
		}

		pair := s[0]
		symbol := pair.Symbol
		step := &Step{
			ID:             0,
			Status:         StepStatusNew,
			Side:           SideBuy,
			Type:           SwapTypeSellThenBuy,
			Symbol:         symbol,
			ReceivedAsset:  symbol.BaseAsset(),
			SpentAsset:     symbol.QuoteAsset(),
			BasePrecision:  pair.BasePrecision,
			QuotePrecision: pair.QuotePrecision,
		}

		return []*Step{step}
	}

	steps := make([]*Step, 0, len(s))

	var nextSide Side
	var receivedAsset, spentAsset string

	switch swapType {
	case SwapTypeSellThenBuy:
		nextSide = SideSell
	default:
		return nil
	}

	for i, step := range s {
		symbol := step.Symbol
		if nextSide == SideBuy {
			receivedAsset = symbol.BaseAsset()
			spentAsset = symbol.QuoteAsset()
		} else {
			receivedAsset = symbol.QuoteAsset()
			spentAsset = symbol.BaseAsset()
		}
		steps = append(steps, &Step{
			ID:             i,
			Status:         StepStatusNew,
			Side:           nextSide,
			Symbol:         symbol,
			Type:           swapType,
			SpentAmount:    decimal.Zero,
			ReceivedAsset:  receivedAsset,
			SpentAsset:     spentAsset,
			BasePrecision:  step.BasePrecision,
			QuotePrecision: step.QuotePrecision,
		})

		nextSide = nextSide.Opposite()
	}

	return steps
}
