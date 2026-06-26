package models

import (
	"fmt"
	"strings"
)

type Symbol string

const assetSeparator = "-"

func NewSymbol(s string) (Symbol, error) {
	parts := strings.Split(s, assetSeparator)
	if len(parts) != 2 {
		return "", fmt.Errorf("wrong symbol format %s", s)
	}

	return Symbol(s), nil
}

func (s Symbol) String() string { return string(s) }

func (s Symbol) BaseAsset() string {
	parts := strings.Split(s.String(), assetSeparator)
	if len(parts) == 2 {
		return parts[0]
	}

	return ""
}

func (s Symbol) QuoteAsset() string {
	parts := strings.Split(s.String(), assetSeparator)
	if len(parts) == 2 {
		return parts[1]
	}

	return ""
}

type MarketPair struct {
	Symbol         Symbol
	Base           string
	Quote          string
	Exchange       map[string]struct{}
	BasePrecision  int32
	QuotePrecision int32
	TradingEnabled bool
	SecurityType   int32
}

func (mp MarketPair) HasAndReturnAnother(asset string) (bool, string) {
	if mp.Base == asset {
		return true, mp.Quote
	}
	if mp.Quote == asset {
		return true, mp.Base
	}

	return false, ""
}

func NewMarketPair(pair string, base string, quote string, exchanges map[string]struct{}, st int32) *MarketPair {
	return &MarketPair{
		Symbol:         Symbol(pair),
		Base:           base,
		Quote:          quote,
		Exchange:       exchanges,
		SecurityType:   st,
		TradingEnabled: true,
	}
}

type LinkedPairs struct {
	Pairs []Pair
}

func (lp *LinkedPairs) String() string {
	return fmt.Sprintf("LinkedPairs{Pairs: %v}", lp.Pairs)
}

type Pair struct {
	Symbol Symbol
}
