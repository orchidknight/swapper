package markets

import (
	"errors"
	"sort"
	"strings"

	"github.com/orchidknight/swapper/models"
)

const (
	separator           = "-"
	defaultMaxPathHops  = 8
	defaultMaxPathCount = 256
)

var (
	// ErrInvalidSwapPairSameAssets means a requested swap symbol uses the same source and destination asset.
	ErrInvalidSwapPairSameAssets = errors.New("invalid symbol, same base and quote asset")

	// ErrInvalidSwapPair means a requested swap symbol cannot be split into base and quote assets.
	ErrInvalidSwapPair = errors.New("invalid symbol, cant extract base and quote asset")
)

// MarketService stores available markets and finds candidate paths between assets.
type MarketService struct {
	Markets  map[models.Symbol]*models.MarketPair
	MaxHops  int
	MaxPaths int
}

type linkedMarket struct {
	asset  string
	market *models.MarketPair
}

type pathSearchState struct {
	maxHops  int
	maxPaths int
	paths    int
}

// New builds a MarketService from a list of market pairs.
func New(m []*models.MarketPair) *MarketService {
	markets := make(map[models.Symbol]*models.MarketPair)
	for i := range m {
		if m[i] == nil {
			continue
		}

		markets[m[i].Symbol] = m[i]
	}

	return &MarketService{
		Markets: markets,
	}
}

// GetMarket returns market metadata for the given symbol.
func (ms *MarketService) GetMarket(s models.Symbol) *models.MarketPair {
	return ms.Markets[s]
}

// GetAllSwapPairs returns all deterministic market paths for a swap symbol.
func (ms *MarketService) GetAllSwapPairs(symbol models.Symbol) ([]*models.LinkedPairs, error) {
	exceptions := make(map[string]struct{})
	parts := strings.Split(symbol.String(), separator)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, ErrInvalidSwapPair
	}

	src, dst := parts[0], parts[1]
	if src == dst {
		return nil, ErrInvalidSwapPairSameAssets
	}

	allLinkedPairs := ms.findAllLinks(src, dst, exceptions, ms.newPathSearchState())
	sortLinkedPairs(allLinkedPairs)

	return allLinkedPairs, nil
}

func (ms *MarketService) linkedAssets(asset string, skip map[string]struct{}) []linkedMarket {
	linkedMarkets := make(map[string]*models.MarketPair)
	for _, market := range ms.Markets {
		if market == nil {
			continue
		}
		if !market.TradingEnabled {
			continue
		}
		isLinked, linkedAsset := market.HasAndReturnAnother(asset)
		if !isLinked {
			continue
		}
		if _, skipped := skip[market.Symbol.String()]; skipped {
			continue
		}

		currentMarket, exists := linkedMarkets[linkedAsset]
		if !exists || market.Symbol.String() < currentMarket.Symbol.String() {
			linkedMarkets[linkedAsset] = market
		}
	}

	result := make([]linkedMarket, 0, len(linkedMarkets))
	for linkedAsset, market := range linkedMarkets {
		result = append(result, linkedMarket{
			asset:  linkedAsset,
			market: market,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].asset != result[j].asset {
			return result[i].asset < result[j].asset
		}

		return result[i].market.Symbol.String() < result[j].market.Symbol.String()
	})

	return result
}

func (ms *MarketService) findAllLinks(src, dst string, exceptions map[string]struct{}, state *pathSearchState) []*models.LinkedPairs {
	if state.pathLimitReached() {
		return nil
	}

	linkedAssets := ms.linkedAssets(dst, exceptions)
	results := directLinkedPairs(src, linkedAssets, len(exceptions)+1, state)
	results = append(results, ms.branchLinkedPairs(src, linkedAssets, exceptions, state)...)

	return results
}

func directLinkedPairs(src string, linkedAssets []linkedMarket, pathLength int, state *pathSearchState) []*models.LinkedPairs {
	var results []*models.LinkedPairs
	for _, linkedAsset := range linkedAssets {
		if state.pathLimitReached() {
			break
		}
		if linkedAsset.asset != src {
			continue
		}
		if !state.canUsePathLength(pathLength) {
			continue
		}

		state.paths++
		results = append(results, &models.LinkedPairs{
			Pairs: []models.Pair{{Symbol: linkedAsset.market.Symbol}},
		})
	}

	return results
}

func (ms *MarketService) branchLinkedPairs(
	src string,
	linkedAssets []linkedMarket,
	exceptions map[string]struct{},
	state *pathSearchState,
) []*models.LinkedPairs {
	var results []*models.LinkedPairs
	for _, linkedAsset := range linkedAssets {
		if state.pathLimitReached() {
			break
		}
		if linkedAsset.asset == src {
			continue
		}

		branchExceptions := copyExceptions(exceptions)
		branchExceptions[linkedAsset.market.Symbol.String()] = struct{}{}
		if !state.canRecurse(len(branchExceptions)) {
			continue
		}

		linkedPairs := ms.findAllLinks(src, linkedAsset.asset, branchExceptions, state)
		for _, linkedPair := range linkedPairs {
			linkedPair.Pairs = append(linkedPair.Pairs, models.Pair{Symbol: linkedAsset.market.Symbol})
			results = append(results, linkedPair)
		}
	}

	return results
}

func (ms *MarketService) newPathSearchState() *pathSearchState {
	return &pathSearchState{
		maxHops:  normalizedLimit(ms.MaxHops, defaultMaxPathHops),
		maxPaths: normalizedLimit(ms.MaxPaths, defaultMaxPathCount),
	}
}

func normalizedLimit(value, defaultValue int) int {
	if value < 0 {
		return 0
	}
	if value == 0 {
		return defaultValue
	}

	return value
}

func (s *pathSearchState) canUsePathLength(length int) bool {
	return s.maxHops == 0 || length <= s.maxHops
}

func (s *pathSearchState) canRecurse(currentLength int) bool {
	return s.maxHops == 0 || currentLength < s.maxHops
}

func (s *pathSearchState) pathLimitReached() bool {
	return s.maxPaths > 0 && s.paths >= s.maxPaths
}

func copyExceptions(exceptions map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{}, len(exceptions))
	for exception := range exceptions {
		result[exception] = struct{}{}
	}

	return result
}

func sortLinkedPairs(slices []*models.LinkedPairs) {
	sort.Slice(slices, func(i, j int) bool {
		if len(slices[i].Pairs) != len(slices[j].Pairs) {
			return len(slices[i].Pairs) < len(slices[j].Pairs)
		}

		return linkedPairsLess(slices[i], slices[j])
	})
}

func linkedPairsLess(left, right *models.LinkedPairs) bool {
	if left == nil || right == nil {
		return left != nil
	}

	for i := 0; i < len(left.Pairs) && i < len(right.Pairs); i++ {
		leftSymbol := left.Pairs[i].Symbol.String()
		rightSymbol := right.Pairs[i].Symbol.String()
		if leftSymbol != rightSymbol {
			return leftSymbol < rightSymbol
		}
	}

	return len(left.Pairs) < len(right.Pairs)
}
