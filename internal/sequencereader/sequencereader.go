// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package sequencereader

import (
	"errors"
)

// SequenceReader defines the simplest reader for sequences.
type SequenceReader[T any] struct {
	s    []T
	idx  int
	size int
}

// New is a constructor for SequenceReader.
func New[T any](seq []T) *SequenceReader[T] {
	return &SequenceReader[T]{
		s:    seq,
		idx:  0,
		size: len(seq),
	}
}

// HasNext returns true is sequence is not ended.
func (sr *SequenceReader[T]) HasNext() bool {
	return sr.idx < sr.size
}

// Next returns next element of the sequence.
func (sr *SequenceReader[T]) Next() (T, error) {
	if !sr.HasNext() {
		return *new(T), errors.New("the sequence is ended")
	}

	pIdx := sr.idx
	sr.idx++

	return sr.s[pIdx], nil
}

// Len returns how many items are left.
func (sr *SequenceReader[T]) Len() int {
	return sr.size - sr.idx
}
