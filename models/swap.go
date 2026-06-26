package models

import (
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type SwapStatus string
type StepStatus string
type SwapType string

const (
	StepStatusNew        = "new"
	StepStatusInProgress = "inProgress"
	StepStatusCompleted  = "completed"
	StepStatusRejected   = "rejected"
	StepStatusCanceled   = "canceled"

	SwapStatusNew        = "new"
	SwapStatusInProgress = "inProgress"
	SwapStatusCompleted  = "completed"
	SwapStatusRejected   = "rejected"
	SwapStatusCanceled   = "canceled"

	SwapTypeUnspecified = "unspecified"
	SwapTypeBuyThenSell = "buy-then-sell"
	SwapTypeSellThenBuy = "sell-then-buy"
)

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

func (s *Swap) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return fmt.Sprintf("%d %s %s %v %v", s.ID, s.Type, s.Status, s.SubOrders, s.Steps)
}

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

func (s *Step) String() string {
	return fmt.Sprintf("{%d %s %s %s Spent: %v%s Received: %v%s}", s.ID, s.Status, s.Side, s.Symbol, s.SpentAmount, s.SpentAsset, s.ReceivedAmount, s.ReceivedAsset)
}

// nolint
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

// nolint
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

		fmt.Println("Swap completed!")
		return true
	}

	if o.Status == OrderStatusRejected && s.CurrentStep == 0 && s.CurrentPath != len(s.Paths)-1 {

		s.RejectedSteps = append(s.RejectedSteps, s.Steps[stepIndex])

		s.CurrentPath++
		newPathIndex := s.CurrentPath

		fmt.Printf("GOT REJECT ON FIRST STEP, try next %v, rejected: %v \n", s.Paths[newPathIndex].Pairs, s.RejectedSteps)

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

// nolint
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

type SwapperReport struct {
	SubOrderToSend  *Order
	ResultSwapOrder *Order
}

func (sr *SwapperReport) String() string {
	return fmt.Sprintf("{Order: %v}", sr.SubOrderToSend)
}

func (s *Swap) StepPairs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pairs []string

	for _, step := range s.Steps {
		pairs = append(pairs, step.Symbol.String())
	}

	return pairs
}

type PairStep struct {
	Pair string
	Side Side
}

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
