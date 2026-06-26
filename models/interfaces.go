package models

import "context"

type Storage interface {
	SaveSwap(ctx context.Context, swap *Swap) error
	GetAllSwaps(ctx context.Context) ([]*Swap, error)
	DeleteSwap(ctx context.Context, id uint64) error
	UpdateSwap(ctx context.Context, swap *Swap) error
}

type Logger interface {
	Debug(component string, format string, a ...any)
	Info(component string, format string, a ...any)
	Warn(component string, format string, a ...any)
	Error(component string, format string, a ...any)
	Fatal(component string, format string, a ...any)
}
