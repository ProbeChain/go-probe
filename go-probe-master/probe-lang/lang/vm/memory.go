// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ProbeChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ProbeChain. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"errors"
	"fmt"
)

const (
	// DefaultMemoryLimit is the maximum number of bytes a VM instance may
	// allocate across all live allocations (4 MiB).
	DefaultMemoryLimit uint64 = 4 * 1024 * 1024

	// minAllocSize is the smallest allocation granularity (8 bytes, one
	// 64-bit word).
	minAllocSize uint64 = 8
)

// ErrOutOfMemory is returned when an allocation would exceed the memory limit.
var ErrOutOfMemory = errors.New("vm: out of memory")

// ErrInvalidAddress is returned when a read/write targets an address that is
// outside the bounds of any live allocation.
var ErrInvalidAddress = errors.New("vm: invalid memory address")

// ErrDoubleFree is returned when the caller attempts to free an address that
// was not returned by Alloc or has already been freed.
var ErrDoubleFree = errors.New("vm: double free")

// allocation records a single live memory region.
type allocation struct {
	base uint64 // first byte address (inclusive)
	size uint64 // number of bytes in the region
}

// end returns the first byte past the allocation's last byte.
func (a allocation) end() uint64 { return a.base + a.size }

// Memory is the linear byte-addressable memory model for a PROBE VM instance.
//
// Design:
//   - Allocations are tracked in a map keyed by base address.
//   - The backing store is a single flat byte slice grown lazily.
//   - All accesses are bounds-checked against the corresponding allocation.
//   - A configurable limit caps total allocated bytes to prevent abuse.
//
// The zero value is not usable; use NewMemory.
type Memory struct {
	data    []byte
	allocs  map[uint64]allocation // base → allocation descriptor
	limit   uint64                // max total allocated bytes
	used    uint64                // current total allocated bytes
	nextPtr uint64                // next candidate base address (monotone)
}

// NewMemory creates a Memory instance with the given byte limit.
// If limit is 0, DefaultMemoryLimit is used.
func NewMemory(limit uint64) *Memory {
	if limit == 0 {
		limit = DefaultMemoryLimit
	}
	return &Memory{
		data:   make([]byte, 0, 4096),
		allocs: make(map[uint64]allocation),
		limit:  limit,
	}
}

// Alloc reserves size bytes of memory and returns the base address.
// Returns ErrOutOfMemory if the limit would be exceeded.
// Panics if size is 0.
func (m *Memory) Alloc(size uint64) (uint64, error) {
	if size == 0 {
		return 0, fmt.Errorf("vm: Alloc called with zero size")
	}
	// Round up to minAllocSize for alignment.
	aligned := roundUp(size, minAllocSize)
	if m.used+aligned > m.limit {
		return 0, ErrOutOfMemory
	}

	// Assign a base address from the monotone pointer, then grow the backing
	// slice as needed.
	base := m.nextPtr
	end := base + aligned
	if end > uint64(len(m.data)) {
		// Grow the backing store.
		newCap := max64(end, uint64(cap(m.data))*2)
		if newCap > m.limit*2 {
			newCap = m.limit * 2
		}
		grown := make([]byte, end, newCap)
		copy(grown, m.data)
		m.data = grown
	} else if end > uint64(len(m.data)) {
		m.data = m.data[:end]
	}
	// Ensure data slice covers [0, end).
	if uint64(len(m.data)) < end {
		m.data = m.data[:end]
	}

	// Zero-fill the new region (Go already does this for fresh slices, but
	// a previously freed region may contain stale data).
	for i := base; i < end; i++ {
		m.data[i] = 0
	}

	m.allocs[base] = allocation{base: base, size: aligned}
	m.used += aligned
	m.nextPtr = end
	return base, nil
}

// Free releases the allocation at base.
// Returns ErrDoubleFree if base is not a live allocation base address.
func (m *Memory) Free(base uint64) error {
	a, ok := m.allocs[base]
	if !ok {
		return ErrDoubleFree
	}
	// Scrub the memory to catch use-after-free bugs early.
	for i := a.base; i < a.end(); i++ {
		m.data[i] = 0xCC
	}
	m.used -= a.size
	delete(m.allocs, base)
	return nil
}

// ReadByte returns the byte at address addr.
// Returns ErrInvalidAddress if addr is not within a live allocation.
func (m *Memory) ReadByte(addr uint64) (byte, error) {
	if err := m.checkAccess(addr, 1); err != nil {
		return 0, err
	}
	return m.data[addr], nil
}

// WriteByte writes b to address addr.
// Returns ErrInvalidAddress if addr is not within a live allocation.
func (m *Memory) WriteByte(addr uint64, b byte) error {
	if err := m.checkAccess(addr, 1); err != nil {
		return err
	}
	m.data[addr] = b
	return nil
}

// ReadUint64 reads a 64-bit little-endian word from address addr.
// Returns ErrInvalidAddress if [addr, addr+8) is not within a live allocation.
func (m *Memory) ReadUint64(addr uint64) (uint64, error) {
	if err := m.checkAccess(addr, 8); err != nil {
		return 0, err
	}
	d := m.data[addr:]
	return uint64(d[0]) |
		uint64(d[1])<<8 |
		uint64(d[2])<<16 |
		uint64(d[3])<<24 |
		uint64(d[4])<<32 |
		uint64(d[5])<<40 |
		uint64(d[6])<<48 |
		uint64(d[7])<<56, nil
}

// WriteUint64 writes a 64-bit little-endian word to address addr.
// Returns ErrInvalidAddress if [addr, addr+8) is not within a live allocation.
func (m *Memory) WriteUint64(addr uint64, v uint64) error {
	if err := m.checkAccess(addr, 8); err != nil {
		return err
	}
	d := m.data[addr:]
	d[0] = byte(v)
	d[1] = byte(v >> 8)
	d[2] = byte(v >> 16)
	d[3] = byte(v >> 24)
	d[4] = byte(v >> 32)
	d[5] = byte(v >> 40)
	d[6] = byte(v >> 48)
	d[7] = byte(v >> 56)
	return nil
}

// ReadSlice returns a view of size bytes starting at addr.
// The returned slice is a direct reference into the backing store; the caller
// must not hold it across a subsequent Alloc call (which may grow the backing
// store and reallocate the underlying array).
// Returns ErrInvalidAddress if [addr, addr+size) is not within a live allocation.
func (m *Memory) ReadSlice(addr, size uint64) ([]byte, error) {
	if size == 0 {
		return []byte{}, nil
	}
	if err := m.checkAccess(addr, size); err != nil {
		return nil, err
	}
	return m.data[addr : addr+size], nil
}

// WriteSlice copies len(data) bytes from data into memory at addr.
// Returns ErrInvalidAddress if [addr, addr+len(data)) is not within a live allocation.
func (m *Memory) WriteSlice(addr uint64, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if err := m.checkAccess(addr, uint64(len(data))); err != nil {
		return err
	}
	copy(m.data[addr:], data)
	return nil
}

// Used returns the current number of allocated bytes.
func (m *Memory) Used() uint64 { return m.used }

// Limit returns the configured memory ceiling.
func (m *Memory) Limit() uint64 { return m.limit }

// checkAccess verifies that the range [addr, addr+size) falls within a single
// live allocation.  It returns ErrInvalidAddress on any violation.
func (m *Memory) checkAccess(addr, size uint64) error {
	// addr must fall inside at least one live allocation and that allocation
	// must fully cover [addr, addr+size).
	for _, a := range m.allocs {
		if addr >= a.base && addr+size <= a.end() {
			return nil
		}
	}
	return fmt.Errorf("%w: addr=0x%x size=%d", ErrInvalidAddress, addr, size)
}

// roundUp rounds n up to the nearest multiple of align (which must be a power
// of two).
func roundUp(n, align uint64) uint64 {
	return (n + align - 1) &^ (align - 1)
}

// max64 returns the larger of two uint64 values.
func max64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
