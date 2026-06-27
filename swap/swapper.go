package swap

import (
	"context"
	"fmt"
	"sync"

	"github.com/orchidknight/swapper/models"
)

// MarketProvider supplies market paths and metadata required to build swap steps.
type MarketProvider interface {
	GetAllSwapPairs(symbol models.Symbol) ([]*models.LinkedPairs, error)
	GetMarket(symbol models.Symbol) *models.MarketPair
}

// Swapper orchestrates active swaps and maps suborder results back to their source swap.
//
// Concurrency contract: distinct swaps may be driven concurrently. For a single
// swap the caller must feed suborder results sequentially — at most one suborder
// is outstanding at a time — so ConsumeSubOrderResult is not safe to call
// concurrently for the same swap.
type Swapper struct {
	activeSwaps map[uint64]*models.Swap
	markets     MarketProvider
	storage     models.Storage

	orders map[uint64]uint64

	// nextStepOrder builds the next suborder for a swap. It is a field rather than
	// a direct method call so tests can inject failures; production always uses
	// (*models.Swap).NextStepOrder.
	nextStepOrder func(*models.Swap) (*models.Order, error)

	lock sync.RWMutex
	log  models.Logger
}

// NewSwapper creates a Swapper with caller-provided ports.
func NewSwapper(
	markets MarketProvider,
	storage models.Storage,
	log models.Logger,
) *Swapper {
	if storage == nil {
		storage = noOpStorage{}
	}
	if log == nil {
		log = noOpLogger{}
	}

	return &Swapper{
		markets:       markets,
		lock:          sync.RWMutex{},
		activeSwaps:   make(map[uint64]*models.Swap),
		orders:        make(map[uint64]uint64),
		storage:       storage,
		log:           log,
		nextStepOrder: (*models.Swap).NextStepOrder,
	}
}

// AllSwapSteps returns valid market paths for the order symbol with market precision applied.
func (s *Swapper) AllSwapSteps(o *models.Order) ([]*models.LinkedPairs, error) {
	if o == nil {
		return nil, fmt.Errorf("order is nil")
	}
	if s.markets == nil {
		return nil, fmt.Errorf("market provider is nil")
	}

	var validSwapSteps []*models.LinkedPairs

	steps, err := s.markets.GetAllSwapPairs(o.Symbol)
	if err != nil {
		return nil, fmt.Errorf("GetAllSwapPairs: can't get swap pairs for %s: %w", o.Symbol, err)
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
			// Unknown precision: mark as -1 so amounts pass through untouched
			// instead of being truncated to whole units.
			linkedPairs.Pairs[i].BasePrecision = models.PrecisionUnknown
			linkedPairs.Pairs[i].QuotePrecision = models.PrecisionUnknown

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

// LoadOrders restores active swaps from Storage.
func (s *Swapper) LoadOrders(ctx context.Context) error {
	activeSwaps, err := s.storage.GetAllSwaps(ctx)
	if err != nil {
		return fmt.Errorf("GetAllSwaps: %w", err)
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, activeSwap := range activeSwaps {
		if activeSwap == nil {
			continue
		}

		s.activeSwaps[activeSwap.ID] = activeSwap

		for _, step := range activeSwap.Steps {
			if step == nil || step.Order == nil {
				continue
			}

			if step.Status == models.StepStatusInProgress || step.Status == models.StepStatusNew {
				s.orders[step.Order.ID] = activeSwap.ID
			}
		}
	}

	if s.log != nil {
		s.log.Debug("swapper", "loaded swaps: %d", len(activeSwaps))
	}

	return nil
}

type noOpStorage struct{}

func (noOpStorage) SaveSwap(context.Context, *models.Swap) error {
	return nil
}

func (noOpStorage) GetAllSwaps(context.Context) ([]*models.Swap, error) {
	return nil, nil
}

func (noOpStorage) DeleteSwap(context.Context, uint64) error {
	return nil
}

func (noOpStorage) UpdateSwap(context.Context, *models.Swap) error {
	return nil
}

type noOpLogger struct{}

func (noOpLogger) Debug(string, string, ...any) {}
func (noOpLogger) Info(string, string, ...any)  {}
func (noOpLogger) Warn(string, string, ...any)  {}
func (noOpLogger) Error(string, string, ...any) {}
func (noOpLogger) Fatal(string, string, ...any) {}
