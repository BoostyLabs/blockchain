// Copyright (C) 2024 Creditor Corp. Group.
// See LICENSE for copying information.

package utils

// Condition defines helping struct to check Condition and invoke
// callback functions returning Condition state and error if any.
type Condition struct {
	b   bool
	err error
}

// Ok return state ot the Condition.
func (c *Condition) Ok() bool {
	return c.b
}

// Error return error if any.
func (c *Condition) Error() error {
	return c.err
}

// Then invokes fn if Condition is true.
func (c *Condition) Then(fn func() error) *Condition {
	if c.b {
		c.err = fn()
	}

	return c
}

// IfLen returns length Condition result.
func IfLen[T any](arr []T, length int) *Condition {
	return &Condition{len(arr) == length, nil}
}
