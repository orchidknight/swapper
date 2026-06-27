// Package swap orchestrates swap orders and their generated market suborders.
//
// A Swapper accepts an initial models.Order with type models.OrderTypeSwap via
// ConsumeOrder, builds the first executable market suborder, and returns it as
// SwapperReport.SubOrderToSend. The caller sends that suborder to an exchange,
// feeds each exchange result back through ConsumeSubOrderResult, and repeats
// until SwapperReport.ResultSwapOrder contains the completed, rejected, or
// canceled swap order.
//
// The package does not place exchange orders or own persistence. Callers supply
// market data, storage, and logging ports. Storage failures are logged and are
// not returned from ConsumeOrder or ConsumeSubOrderResult.
// ConsumeOrder validates boundary invariants before accepting a swap: non-zero
// unique order ID, sell side, positive quantities, and non-terminal status.
//
// Suborder quantities follow exchange market-order conventions: sell legs carry
// base-asset Amount and AvailableAmount, while buy legs carry AvailableTotal as
// the quote-asset budget. Market precision metadata is applied before suborders
// are returned: BasePrecision truncates sell amounts, and QuotePrecision
// truncates buy totals.
package swap
