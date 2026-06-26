package models

import (
	"fmt"
	"strings"
)

// Symbol identifies a market pair in BASE-QUOTE form.
type Symbol string

const assetSeparator = "-"

// NewSymbol validates and returns a Symbol in BASE-QUOTE form.
func NewSymbol(s string) (Symbol, error) {
	parts := strings.Split(s, assetSeparator)
	if len(parts) != 2 {
		return "", fmt.Errorf("wrong symbol format %s", s)
	}

	return Symbol(s), nil
}

// String returns the raw symbol text.
func (s Symbol) String() string { return string(s) }

// BaseAsset returns the asset before the separator.
func (s Symbol) BaseAsset() string {
	parts := strings.Split(s.String(), assetSeparator)
	if len(parts) == 2 {
		return parts[0]
	}

	return ""
}

// QuoteAsset returns the asset after the separator.
func (s Symbol) QuoteAsset() string {
	parts := strings.Split(s.String(), assetSeparator)
	if len(parts) == 2 {
		return parts[1]
	}

	return ""
}

// MarketPair describes a tradable market and its precision metadata.
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

// HasAndReturnAnother reports whether asset belongs to the pair and returns the opposite asset.
func (mp MarketPair) HasAndReturnAnother(asset string) (bool, string) {
	if mp.Base == asset {
		return true, mp.Quote
	}
	if mp.Quote == asset {
		return true, mp.Base
	}

	return false, ""
}

// NewMarketPair constructs an enabled MarketPair from raw metadata.
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

// LinkedPairs represents an ordered market path for converting one asset to another.
type LinkedPairs struct {
	Pairs []Pair
}

// String returns a debug representation of the linked pairs.
func (lp *LinkedPairs) String() string {
	return fmt.Sprintf("LinkedPairs{Pairs: %v}", lp.Pairs)
}

// Pair is one market step in a linked swap path.
type Pair struct {
	Symbol         Symbol
	BasePrecision  int32
	QuotePrecision int32
}

// String returns a debug representation of the pair.
func (p Pair) String() string {
	return fmt.Sprintf("{%s}", p.Symbol)
}
