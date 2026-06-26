package models

// Side identifies whether an order buys or sells the base asset.
type Side string

const (
	// SideUnspecified is the zero-value side.
	SideUnspecified Side = "unspecified"

	// SideBuy means the order buys the base asset.
	SideBuy Side = "buy"

	// SideSell means the order sells the base asset.
	SideSell Side = "sell"
)

// Opposite returns the opposite trading side.
func (s Side) Opposite() Side {
	switch s {
	case SideSell:
		return SideBuy
	case SideBuy:
		return SideSell
	default:
		return SideUnspecified
	}
}
