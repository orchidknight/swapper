package swap

import (
	"context"
	"fmt"

	"github.com/orchidknight/swapper/models"
)

var nextStepOrder = (*models.Swap).NextStepOrder

// nolint
func (s *Swapper) ConsumeSubOrderResult(ctx context.Context, order *models.Order) (*models.SwapperReport, error) {
	sr := new(models.SwapperReport)

	swap, err := s.findSwapByOrder(order.ID)
	if err != nil {
		return nil, err
	}

	if swap == nil {
		return nil, nil
	}

	canProceed := swap.Update(order)
	err = s.storage.UpdateSwap(ctx, swap)
	if err != nil {
		s.log.Error("swapper", "UpdateSwap: %v", err)
	}

	if !canProceed {
		return nil, nil
	}

	s.log.Debug("swapper", "got new order response, status: %s", swap.Status)

	switch swap.Status {
	case models.SwapStatusCompleted, models.SwapStatusRejected, models.SwapStatusCanceled:

		s.lock.Lock()
		delete(s.orders, order.ID)
		s.lock.Unlock()

		err = s.storage.DeleteSwap(ctx, swap.ID)
		if err != nil {
			s.log.Error("swapper", "DeleteSwap: %v", err)
		}

		sr.ResultSwapOrder = swap.Order

		return sr, err
	case models.SwapStatusInProgress:
		if swap.Steps[swap.CurrentStep].Status == models.StepStatusCompleted {
			s.lock.Lock()
			defer s.lock.Unlock()

			delete(s.orders, order.ID)
			sr.SubOrderToSend, err = nextStepOrder(swap)
			if err != nil {
				return nil, err
			}

			s.log.Debug("swapper", "step completed, prepare next order to send")
			fmt.Println("order to send: ", sr.SubOrderToSend)

			if sr.SubOrderToSend != nil {
				s.orders[sr.SubOrderToSend.ID] = swap.Order.ID
			}

			return sr, nil
		}
	case models.SwapStatusNew:
		s.lock.Lock()
		defer s.lock.Unlock()

		delete(s.orders, order.ID)
		sr.SubOrderToSend, err = nextStepOrder(swap)
		if err != nil {
			return nil, err
		}
		if sr.SubOrderToSend != nil {
			s.orders[sr.SubOrderToSend.ID] = swap.Order.ID
		}

		return sr, nil
	}

	return sr, err
}
