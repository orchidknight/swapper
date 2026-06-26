package models

import "testing"

func TestNewIDDoesNotDuplicateDuringBulkGeneration(t *testing.T) {
	const count = 100_000

	seen := make(map[uint64]struct{}, count)
	for range count {
		id := NewID()
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id generated: %d", id)
		}

		seen[id] = struct{}{}
	}
}
