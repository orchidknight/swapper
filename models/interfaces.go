package models

import "context"

// Storage persists active swaps between order and suborder events.
type Storage interface {
	SaveSwap(ctx context.Context, swap *Swap) error
	GetAllSwaps(ctx context.Context) ([]*Swap, error)
	DeleteSwap(ctx context.Context, id uint64) error
	UpdateSwap(ctx context.Context, swap *Swap) error
}

// Logger receives diagnostic messages from the swap orchestration layer.
type Logger interface {
	Debug(component string, format string, a ...any)
	Info(component string, format string, a ...any)
	Warn(component string, format string, a ...any)
	Error(component string, format string, a ...any)
	Fatal(component string, format string, a ...any)
}
