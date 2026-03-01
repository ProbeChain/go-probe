// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package math provides array operations for the PROBE standard library.
//
// Inspired by J/APL-style array programming, this package provides
// high-level operations on typed arrays that compile to efficient
// register-based VM operations.
package math

// U64Array is a typed array of uint64 values.
type U64Array struct {
	Data []uint64
}

// NewU64Array creates a new array with the given values.
func NewU64Array(vals ...uint64) *U64Array {
	data := make([]uint64, len(vals))
	copy(data, vals)
	return &U64Array{Data: data}
}

// Len returns the length of the array.
func (a *U64Array) Len() int {
	return len(a.Data)
}

// Sum returns the sum of all elements (reduce +).
func (a *U64Array) Sum() uint64 {
	var s uint64
	for _, v := range a.Data {
		s += v
	}
	return s
}

// Map applies a function to each element (monadic map).
func (a *U64Array) Map(f func(uint64) uint64) *U64Array {
	result := make([]uint64, len(a.Data))
	for i, v := range a.Data {
		result[i] = f(v)
	}
	return &U64Array{Data: result}
}

// Zip combines two arrays element-wise (dyadic zip).
func (a *U64Array) Zip(b *U64Array, f func(uint64, uint64) uint64) *U64Array {
	n := len(a.Data)
	if len(b.Data) < n {
		n = len(b.Data)
	}
	result := make([]uint64, n)
	for i := 0; i < n; i++ {
		result[i] = f(a.Data[i], b.Data[i])
	}
	return &U64Array{Data: result}
}

// Filter returns elements matching a predicate.
func (a *U64Array) Filter(f func(uint64) bool) *U64Array {
	var result []uint64
	for _, v := range a.Data {
		if f(v) {
			result = append(result, v)
		}
	}
	return &U64Array{Data: result}
}

// Reduce folds the array with a binary function.
func (a *U64Array) Reduce(init uint64, f func(uint64, uint64) uint64) uint64 {
	acc := init
	for _, v := range a.Data {
		acc = f(acc, v)
	}
	return acc
}

// Iota creates an array [0, 1, 2, ..., n-1] (J-style iota).
func Iota(n int) *U64Array {
	data := make([]uint64, n)
	for i := range data {
		data[i] = uint64(i)
	}
	return &U64Array{Data: data}
}

// Dot computes the dot product of two arrays.
func Dot(a, b *U64Array) uint64 {
	return a.Zip(b, func(x, y uint64) uint64 { return x * y }).Sum()
}
