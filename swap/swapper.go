package swap

import (
	"context"
	"fmt"
	"github.com/orchidknight/swapper/models"
	"sync"
)

type MarketProvider interface {
	GetAllSwapPairs(symbol models.Symbol) ([]*models.LinkedPairs, error)
	GetMarket(symbol models.Symbol) *models.MarketPair
}

type OrderSender interface {
	SendOrder(ctx context.Context, o *models.Order) error
}

type Swapper struct {
	activeSwaps map[uint64]*models.Swap
	markets     MarketProvider
	storage     models.Storage

	orders map[uint64]uint64
	sender OrderSender

	lock sync.RWMutex
	log  models.Logger
}

func NewSwapper(
	markets MarketProvider,
	orderSender OrderSender,
	storage models.Storage,
	log models.Logger,
) *Swapper {
	return &Swapper{
		markets:     markets,
		lock:        sync.RWMutex{},
		activeSwaps: make(map[uint64]*models.Swap),
		orders:      make(map[uint64]uint64),
		sender:      orderSender,
		storage:     storage,
		log:         log,
	}
}

func (s *Swapper) AllSwapSteps(o *models.Order) ([]*models.LinkedPairs, error) {
	var validSwapSteps []*models.LinkedPairs

	steps, err := s.markets.GetAllSwapPairs(o.Symbol)
	if err != nil {
		return nil, fmt.Errorf("GetAllSwapPairs: can't get swap pairs for %s", o.Symbol)
	}
	for _, lp := range steps {
		if len(lp.Pairs) != 0 {
			s.applyMarketPrecision(lp)
			validSwapSteps = append(validSwapSteps, lp)
		}
	}

	if len(validSwapSteps) == 0 {
		return validSwapSteps, fmt.Errorf("can't find pairs for swap %s", o.Symbol)
	}

	return validSwapSteps, nil
}

func (s *Swapper) applyMarketPrecision(linkedPairs *models.LinkedPairs) {
	for i, pair := range linkedPairs.Pairs {
		market := s.markets.GetMarket(pair.Symbol)
		if market == nil {
			continue
		}

		linkedPairs.Pairs[i].BasePrecision = market.BasePrecision
		linkedPairs.Pairs[i].QuotePrecision = market.QuotePrecision
	}
}

func (s *Swapper) findSwapByOrder(id uint64) (*models.Swap, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	initialOrderID, ok := s.orders[id]
	if ok {
		activeSwap, ok := s.activeSwaps[initialOrderID]
		if ok {
			return activeSwap, nil
		}

		return nil, fmt.Errorf("can't find swap by report order id")
	}

	return nil, nil
}

func (s *Swapper) LoadOrders(ctx context.Context) error {
	activeSwaps, err := s.storage.GetAllSwaps(ctx)
	if err != nil {
		return fmt.Errorf("GetAllSwaps: %v", err)
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, activeSwap := range activeSwaps {
		s.activeSwaps[activeSwap.ID] = activeSwap

		for _, step := range activeSwap.Steps {
			if step.Status == models.SwapStatusInProgress {
				s.orders[step.Order.ID] = activeSwap.ID
			}
		}
	}

	fmt.Printf("Loaded: %d swaps\n", len(activeSwaps))

	return nil
}
