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
	for _, linkedPairs := range steps {
		if linkedPairs != nil && len(linkedPairs.Pairs) != 0 {
			validSwapSteps = append(validSwapSteps, s.applyMarketPrecision(linkedPairs))
		}
	}

	if len(validSwapSteps) == 0 {
		return validSwapSteps, fmt.Errorf("can't find pairs for swap %s", o.Symbol)
	}

	return validSwapSteps, nil
}

func (s *Swapper) applyMarketPrecision(linkedPairs *models.LinkedPairs) *models.LinkedPairs {
	linkedPairsWithPrecision := &models.LinkedPairs{
		Pairs: append([]models.Pair(nil), linkedPairs.Pairs...),
	}

	for i, pair := range linkedPairsWithPrecision.Pairs {
		market := s.markets.GetMarket(pair.Symbol)
		if market == nil {
			// Unknown precision: mark as -1 so amounts pass through untouched
			// instead of being truncated to whole units.
			linkedPairsWithPrecision.Pairs[i].BasePrecision = models.PrecisionUnknown
			linkedPairsWithPrecision.Pairs[i].QuotePrecision = models.PrecisionUnknown

			continue
		}

		linkedPairsWithPrecision.Pairs[i].BasePrecision = market.BasePrecision
		linkedPairsWithPrecision.Pairs[i].QuotePrecision = market.QuotePrecision
	}

	return linkedPairsWithPrecision
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
//
// The in-memory indexes are rebuilt atomically from storage. Only in-progress
// step orders are treated as outstanding suborders.
func (s *Swapper) LoadOrders(ctx context.Context) error {
	activeSwaps, err := s.storage.GetAllSwaps(ctx)
	if err != nil {
		return fmt.Errorf("GetAllSwaps: %w", err)
	}

	loadedActiveSwaps, loadedOrders, err := buildLoadedSwapIndexes(activeSwaps)
	if err != nil {
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.activeSwaps = loadedActiveSwaps
	s.orders = loadedOrders

	if s.log != nil {
		s.log.Debug("swapper", "loaded swaps: %d", len(activeSwaps))
	}

	return nil
}

func buildLoadedSwapIndexes(activeSwaps []*models.Swap) (map[uint64]*models.Swap, map[uint64]uint64, error) {
	loadedActiveSwaps := make(map[uint64]*models.Swap, len(activeSwaps))
	loadedOrders := make(map[uint64]uint64)
	for _, activeSwap := range activeSwaps {
		if err := indexLoadedSwap(loadedActiveSwaps, loadedOrders, activeSwap); err != nil {
			return nil, nil, err
		}
	}

	return loadedActiveSwaps, loadedOrders, nil
}

func indexLoadedSwap(
	loadedActiveSwaps map[uint64]*models.Swap,
	loadedOrders map[uint64]uint64,
	activeSwap *models.Swap,
) error {
	if activeSwap == nil {
		return nil
	}
	if activeSwap.ID == 0 {
		return fmt.Errorf("LoadOrders: swap id must be non-zero")
	}
	if _, exists := loadedActiveSwaps[activeSwap.ID]; exists {
		return fmt.Errorf("LoadOrders: duplicate swap id %d", activeSwap.ID)
	}

	loadedActiveSwaps[activeSwap.ID] = activeSwap

	return indexOutstandingOrders(loadedOrders, activeSwap)
}

func indexOutstandingOrders(loadedOrders map[uint64]uint64, activeSwap *models.Swap) error {
	for _, step := range activeSwap.Steps {
		if err := indexOutstandingStep(loadedOrders, activeSwap.ID, step); err != nil {
			return err
		}
	}

	return nil
}

func indexOutstandingStep(loadedOrders map[uint64]uint64, swapID uint64, step *models.Step) error {
	if step == nil || step.Order == nil || step.Status != models.StepStatusInProgress {
		return nil
	}
	if step.Order.ID == 0 {
		return fmt.Errorf("LoadOrders: suborder id must be non-zero for swap %d", swapID)
	}
	if existingSwapID, exists := loadedOrders[step.Order.ID]; exists && existingSwapID != swapID {
		return fmt.Errorf("LoadOrders: duplicate suborder id %d", step.Order.ID)
	}

	loadedOrders[step.Order.ID] = swapID

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
