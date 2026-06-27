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
	// Debug logs diagnostic details useful while tracing swap execution.
	Debug(component string, format string, a ...any)
	// Info logs normal operational messages.
	Info(component string, format string, a ...any)
	// Warn logs recoverable unusual conditions.
	Warn(component string, format string, a ...any)
	// Error logs failures that do not stop the library flow.
	Error(component string, format string, a ...any)
	// Fatal logs a highest-severity message. Implementations are not required to
	// terminate the process; swapper treats Fatal as a log level, not as os.Exit.
	Fatal(component string, format string, a ...any)
}
