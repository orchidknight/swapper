package markets

import (
	"errors"
	"sort"
	"strings"

	"github.com/orchidknight/swapper/models"
)

const (
	separator = "-"
)

var (
	ErrInvalidSwapPairSameAssets = errors.New("invalid symbol, same base and quote asset")
	ErrInvalidSwapPair           = errors.New("invalid symbol, cant extract base and quote asset")
)

type MarketService struct {
	Markets map[models.Symbol]*models.MarketPair
}

func New(m []*models.MarketPair) (*MarketService, error) {
	markets := make(map[models.Symbol]*models.MarketPair)
	for i := range m {
		markets[m[i].Symbol] = m[i]
	}

	return &MarketService{
		Markets: markets,
	}, nil
}

func (ms *MarketService) GetMarket(s models.Symbol) *models.MarketPair {
	return ms.Markets[s]
}

func (ms *MarketService) GetAllSwapPairs(symbol models.Symbol) ([]*models.LinkedPairs, error) {
	exceptions := make(map[string]struct{})
	parts := strings.Split(symbol.String(), separator)
	if len(parts) < 2 {
		return nil, ErrInvalidSwapPair
	}

	src, dst := parts[0], parts[1]
	if src == dst {
		return nil, ErrInvalidSwapPairSameAssets
	}

	allLinkedPairs := ms.findAllLinks(src, dst, exceptions)
	sortSlicesByLength(allLinkedPairs)

	return allLinkedPairs, nil
}

func (ms *MarketService) linkedAssets(asset string, skip map[string]struct{}) map[string]*models.MarketPair {
	result := make(map[string]*models.MarketPair)
	for _, market := range ms.Markets {
		if !market.TradingEnabled {
			continue
		}
		ok, linkedAsset := market.HasAndReturnAnother(asset)
		if ok {
			if _, ok := skip[market.Symbol.String()]; !ok {
				result[linkedAsset] = market
			}
		}
	}

	return result
}

func (ms *MarketService) findAllLinks(src, dst string, exceptions map[string]struct{}) []*models.LinkedPairs {
	var results []*models.LinkedPairs

	linkedAssets := ms.linkedAssets(dst, exceptions)
	linked, ok := linkedAssets[src]
	if ok {
		results = append(results, &models.LinkedPairs{
			Pairs: []models.Pair{{Symbol: linked.Symbol}},
		})

		return results
	}

	for linkedAsset, pair := range linkedAssets {
		exceptions[pair.Symbol.String()] = struct{}{}
		linkedPairs := ms.findAllLinks(src, linkedAsset, exceptions)
		for _, linkedPair := range linkedPairs {
			linkedPair.Pairs = append(linkedPair.Pairs, models.Pair{Symbol: pair.Symbol})
			results = append(results, linkedPair)
		}
	}

	return results
}

func sortSlicesByLength(slices []*models.LinkedPairs) {
	sort.Slice(slices, func(i, j int) bool {
		return len(slices[i].Pairs) < len(slices[j].Pairs)
	})
}
