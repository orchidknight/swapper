package swap

import (
	"context"
	"errors"
	"fmt"

	"github.com/orchidknight/swapper/models"
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
		order.Reject(models.RejectReasonUnspecified)

		return &models.SwapperReport{
			ResultSwapOrder: order,
		}, errors.New("order type must be Swap")
	}
	if order.Side == models.SideBuy {
		order.Reject(models.RejectReasonBuySwapsNotSupported)

		return &models.SwapperReport{
			ResultSwapOrder: order,
		}, errors.New("buy swaps not supported")
	}
	if order.Side != models.SideSell {
		order.Reject(models.RejectReasonUnspecified)

		return &models.SwapperReport{
			ResultSwapOrder: order,
		}, errors.New("swap side must be sell")
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

	s.lock.Lock()
	defer s.lock.Unlock()

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

	s.activeSwaps[order.ID] = swap
	s.orders[orderToSend.ID] = order.ID

	if err := s.storage.SaveSwap(ctx, swap); err != nil {
		s.log.Error("swapper", "SaveSwap: %v", err)
	}

	return &models.SwapperReport{SubOrderToSend: orderToSend}, nil
}
