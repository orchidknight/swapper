package swap

import (
	"context"
	"errors"
	"fmt"

	"github.com/orchidknight/swapper/models"
	"github.com/shopspring/decimal"
)

// ConsumeOrder starts a swap order and returns the first market suborder to execute.
//
// Storage failures are logged via the Logger and never returned: a non-nil error
// means the swap order was not accepted (e.g. buy swap, no path, or building the
// first suborder failed), in which case SubOrderToSend is nil.
func (s *Swapper) ConsumeOrder(ctx context.Context, order *models.Order) (*models.SwapperReport, error) {
	if order == nil {
		return nil, errors.New("order is nil")
	}
	if order.Type != models.OrderTypeSwap {
		return rejectOrder(order, models.RejectReasonUnspecified, errors.New("order type must be Swap"))
	}
	if order.Side == models.SideBuy {
		return rejectOrder(order, models.RejectReasonBuySwapsNotSupported, errors.New("buy swaps not supported"))
	}
	if order.Side != models.SideSell {
		return rejectOrder(order, models.RejectReasonUnspecified, errors.New("swap side must be sell"))
	}
	if err := validateSellSwapOrder(order); err != nil {
		return rejectOrder(order, models.RejectReasonInvalidOrder, err)
	}
	if s.hasActiveSwap(order.ID) {
		return rejectOrder(order, models.RejectReasonInvalidOrder, fmt.Errorf("active swap already exists for order id %d", order.ID))
	}

	steps, err := s.AllSwapSteps(order)
	if err != nil {
		order.Reject(models.RejectReasonUnspecified)

		return &models.SwapperReport{
			ResultSwapOrder: order,
		}, fmt.Errorf("AllSwapSteps: %w", err)
	}

	s.log.Debug("swapper", "swap steps: %v", steps)

	swap := models.NewSwap(order, steps)
	if swap == nil {
		order.Reject(models.RejectReasonUnspecified)

		return &models.SwapperReport{
			ResultSwapOrder: order,
		}, errors.New("invalid swap path")
	}

	orderToSend, err := s.nextStepOrder(swap)
	if err != nil {
		return nil, fmt.Errorf("NextStepOrder: %w", err)
	}
	if orderToSend == nil {
		order.Reject(models.RejectReasonUnspecified)

		return &models.SwapperReport{
			ResultSwapOrder: order,
		}, errors.New("NextStepOrder: no suborder to send")
	}

	s.lock.Lock()
	if _, exists := s.activeSwaps[order.ID]; exists {
		s.lock.Unlock()

		return rejectOrder(order, models.RejectReasonInvalidOrder, fmt.Errorf("active swap already exists for order id %d", order.ID))
	}
	s.activeSwaps[order.ID] = swap
	s.orders[orderToSend.ID] = order.ID
	s.lock.Unlock()

	if err := s.storage.SaveSwap(ctx, swap); err != nil {
		s.log.Error("swapper", "SaveSwap: %v", err)
	}

	return &models.SwapperReport{SubOrderToSend: orderToSend}, nil
}

func rejectOrder(order *models.Order, reason models.RejectReason, err error) (*models.SwapperReport, error) {
	order.Reject(reason)

	return &models.SwapperReport{
		ResultSwapOrder: order,
	}, err
}

func validateSellSwapOrder(order *models.Order) error {
	if order.ID == 0 {
		return errors.New("order id must be non-zero")
	}
	if order.Status != "" && order.Status != models.OrderStatusUnspecified && order.Status != models.OrderStatusNew {
		return fmt.Errorf("order status must be New, got %s", order.Status)
	}
	if !order.Amount.GreaterThan(decimal.Zero) {
		return errors.New("amount must be positive")
	}
	if !order.AvailableAmount.GreaterThan(decimal.Zero) {
		return errors.New("available amount must be positive")
	}
	if order.AvailableAmount.GreaterThan(order.Amount) {
		return errors.New("available amount exceeds amount")
	}

	return nil
}

func (s *Swapper) hasActiveSwap(orderID uint64) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	_, exists := s.activeSwaps[orderID]

	return exists
}
