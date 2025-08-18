package batcher

import (
	"fmt"
)

func Batch[T any](input []T, size int) ([][]T, error) {
	if size <= 0 {
		return nil, fmt.Errorf("batch size must be greater than 0")
	}

	var batches [][]T
	for idx := 0; idx < len(input); idx += size {
		endidx := idx + size
		if endidx > len(input) {
			endidx = len(input)
		}
		batches = append(batches, input[idx:endidx])
	}

	return batches, nil
}
