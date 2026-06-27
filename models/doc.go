// Package models defines the domain objects and ports used by swapper.
//
// The package contains the public domain types shared by the orchestration,
// market graph, persistence, and logging layers. Order represents both initial
// swap orders and generated market suborders. Swap stores the state machine for
// one active conversion, including selected steps, suborder mapping, current
// status, and final swap-order result fields.
//
// Quantity fields keep base and quote assets separate. Amount, AvailableAmount,
// and ExecutedAmount are base-asset quantities. Total, AvailableTotal, and
// ExecutedTotal are quote-asset quantities. Generated sell suborders use base
// amounts; generated buy suborders use AvailableTotal as the quote-asset market
// buy budget.
//
// MarketPair precision fields describe how suborder quantities are truncated:
// BasePrecision applies to sell amounts, and QuotePrecision applies to buy
// totals. PrecisionUnknown means precision metadata is unavailable and quantities
// are passed through without truncation. Storage and Logger are small
// caller-provided ports; this module does not prescribe a database, exchange, or
// logging implementation.
package models
