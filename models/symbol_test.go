package models

import "testing"

func TestNewSymbolRejectsEmptyAssets(t *testing.T) {
	tests := []string{
		"BTC-",
		"-USDT",
		"-",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if got, err := NewSymbol(input); err == nil {
				t.Fatalf("NewSymbol(%q) = %q, want error", input, got)
			}
		})
	}
}
