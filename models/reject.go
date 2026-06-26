package models

// RejectReason explains why an order or swap was rejected.
type RejectReason string

const (
	// RejectReasonUnspecified is used when no more specific reject reason is available.
	RejectReasonUnspecified = "Unspecified"

	// RejectReasonNotEnoughLiquidity means the market cannot fill the requested amount.
	RejectReasonNotEnoughLiquidity = "Not enough liquidity"

	// RejectReasonNoMatches means matching produced zero executable volume.
	RejectReasonNoMatches = "Order got zero matches"

	// RejectReasonBuySwapsNotSupported means buy swaps by target output are deferred.
	RejectReasonBuySwapsNotSupported = "Buy swaps not supported"
)
