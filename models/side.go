package models

type Side string

const (
	SideUnspecified Side = "unspecified"
	SideBuy         Side = "buy"
	SideSell        Side = "sell"
)

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
