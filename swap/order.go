package swap

import (
	"context"
	"errors"
	"fmt"

	"github.com/orchidknight/swapper/models"
)

func (s *Swapper) ConsumeOrder(ctx context.Context, order *models.Order) (*models.SwapperReport, error) {
	if order.Side == models.SideBuy {
		order.Reject(models.RejectReasonBuySwapsNotSupported)

		return &models.SwapperReport{
			ResultSwapOrder: order,
		}, errors.New("buy swaps not supported")
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

	s.lock.Lock()

	s.activeSwaps[order.ID] = swap

	orderToSend, err := swap.NextStepOrder()
	if err != nil {
		return nil, err
	}

	s.orders[orderToSend.ID] = order.ID

	err = s.storage.SaveSwap(ctx, swap)
	if err != nil {
		s.log.Error("swapper", "SaveSwap: %v", err)
	}
	s.lock.Unlock()

	return &models.SwapperReport{SubOrderToSend: orderToSend}, err
}
