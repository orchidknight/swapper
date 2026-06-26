package swap

import (
	"context"

	"github.com/orchidknight/swapper/models"
)

var nextStepOrder = (*models.Swap).NextStepOrder

// ConsumeSubOrderResult applies a suborder execution result and returns the next swap action.
func (s *Swapper) ConsumeSubOrderResult(ctx context.Context, order *models.Order) (*models.SwapperReport, error) {
	swap, err := s.findSwapByOrder(order.ID)
	if err != nil {
		return nil, err
	}

	if swap == nil {
		return nil, nil
	}

	canProceed, updateErr := s.updateSwap(ctx, swap, order)
	if !canProceed {
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

	return new(models.SwapperReport), updateErr
}

func (s *Swapper) updateSwap(ctx context.Context, swap *models.Swap, order *models.Order) (bool, error) {
	canProceed := swap.Update(order)
	err := s.storage.UpdateSwap(ctx, swap)
	if err != nil {
		s.log.Error("swapper", "UpdateSwap: %v", err)
	}

	return canProceed, err
}

func (s *Swapper) finishSwap(ctx context.Context, orderID uint64, swap *models.Swap) (*models.SwapperReport, error) {
	s.lock.Lock()
	delete(s.orders, orderID)
	s.lock.Unlock()

	err := s.storage.DeleteSwap(ctx, swap.ID)
	if err != nil {
		s.log.Error("swapper", "DeleteSwap: %v", err)
	}

	return &models.SwapperReport{ResultSwapOrder: swap.Order}, err
}

func (s *Swapper) nextSubOrderReport(orderID uint64, swap *models.Swap) (*models.SwapperReport, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.orders, orderID)

	subOrder, err := nextStepOrder(swap)
	if err != nil {
		return nil, err
	}

	if subOrder != nil {
		s.orders[subOrder.ID] = swap.Order.ID
	}

	return &models.SwapperReport{SubOrderToSend: subOrder}, nil
}
