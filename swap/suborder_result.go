package swap

import (
	"context"
	"errors"

	"github.com/orchidknight/swapper/models"
)

// ConsumeSubOrderResult applies a suborder execution result and returns the next swap action.
//
// Storage failures are logged via the Logger and never returned: a non-nil error
// means the swap itself could not progress (e.g. building the next suborder failed).
func (s *Swapper) ConsumeSubOrderResult(ctx context.Context, order *models.Order) (*models.SwapperReport, error) {
	if order == nil {
		return nil, errors.New("order is nil")
	}

	swap, err := s.findSwapByOrder(order.ID)
	if err != nil {
		return nil, err
	}

	if swap == nil {
		return nil, nil
	}

	if !s.updateSwap(ctx, swap, order) {
		return nil, nil
	}

	s.log.Debug("swapper", "got new order response, status: %s", swap.Status)

	switch swap.Status {
	case models.SwapStatusCompleted, models.SwapStatusRejected, models.SwapStatusCanceled:
		return s.finishSwap(ctx, order.ID, swap)
	case models.SwapStatusInProgress:
		if swap.Steps[swap.CurrentStep].Status == models.StepStatusCompleted {
			sr, err := s.nextSubOrderReport(order.ID, swap)
			if err != nil {
				return nil, err
			}

			s.log.Debug("swapper", "step completed, prepare next order to send: %v", sr.SubOrderToSend)

			return sr, nil
		}
	case models.SwapStatusNew:
		return s.nextSubOrderReport(order.ID, swap)
	}

	return new(models.SwapperReport), nil
}

func (s *Swapper) updateSwap(ctx context.Context, swap *models.Swap, order *models.Order) bool {
	canProceed := swap.Update(order)
	if err := s.storage.UpdateSwap(ctx, swap); err != nil {
		s.log.Error("swapper", "UpdateSwap: %v", err)
	}

	return canProceed
}

func (s *Swapper) finishSwap(ctx context.Context, orderID uint64, swap *models.Swap) (*models.SwapperReport, error) {
	s.lock.Lock()
	delete(s.orders, orderID)
	delete(s.activeSwaps, swap.ID)
	s.lock.Unlock()

	if err := s.storage.DeleteSwap(ctx, swap.ID); err != nil {
		s.log.Error("swapper", "DeleteSwap: %v", err)
	}

	return &models.SwapperReport{ResultSwapOrder: swap.Order}, nil
}

func (s *Swapper) nextSubOrderReport(orderID uint64, swap *models.Swap) (*models.SwapperReport, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	subOrder, err := s.nextStepOrder(swap)
	if err != nil {
		return nil, err
	}

	delete(s.orders, orderID)

	if subOrder != nil {
		s.orders[subOrder.ID] = swap.Order.ID
	}

	return &models.SwapperReport{SubOrderToSend: subOrder}, nil
}
