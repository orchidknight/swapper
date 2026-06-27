// Package markets provides an in-memory market graph for discovering swap paths.
//
// MarketService implements swap.MarketProvider over a map of models.MarketPair
// values. It discovers deterministic candidate paths between the base and quote
// assets of a swap symbol and exposes market metadata for each pair in the path.
// Path discovery is bounded by MaxHops and MaxPaths; zero values use safe
// package defaults, while negative values disable the corresponding limit.
//
// The swap package uses that metadata when preparing suborders. BasePrecision is
// applied to sell-leg base amounts, and QuotePrecision is applied to buy-leg
// quote totals. TradingEnabled controls whether a pair participates in path
// discovery; disabled or nil pairs are ignored.
package markets
