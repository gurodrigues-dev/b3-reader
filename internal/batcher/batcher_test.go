package batcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatch_Int(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6, 7}
	batchSize := 3

	expected := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7},
	}

	result, err := Batch(input, batchSize)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestBatch_ExactDiv(t *testing.T) {
	input := []string{"a", "b", "c", "d"}
	batchSize := 2

	expected := [][]string{
		{"a", "b"},
		{"c", "d"},
	}

	result, err := Batch(input, batchSize)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestBatch_BatchSizeGreaterThanInput(t *testing.T) {
	input := []int{1, 2, 3}
	batchSize := 10

	expected := [][]int{
		{1, 2, 3},
	}

	result, err := Batch(input, batchSize)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestBatch_EmptyInput(t *testing.T) {
	input := []int{}
	batchSize := 3

	result, err := Batch(input, batchSize)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestBatch_InvalidBatchSize(t *testing.T) {
	input := []int{1, 2, 3}
	batchSize := 0

	result, err := Batch(input, batchSize)
	assert.Error(t, err)
	assert.Nil(t, result)
}
