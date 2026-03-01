// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Linear type checker for the PROBE language.
//
// The checker enforces the following invariants for every function scope:
//
//  1. A resource binding must be consumed exactly once (via move or drop).
//  2. A resource binding cannot be used after it has been moved (use-after-move).
//  3. A resource binding that goes out of scope without being consumed is an error.
//
// These rules are checked independently of the compiler so that safety holds
// even if the compiler emits incorrect bytecode — a critical property for a
// smart-contract execution environment.
package types

import "fmt"

// LinearErrorCode classifies a linear-type violation.
type LinearErrorCode int

const (
	// ErrUseAfterMove is returned when a binding that has already been moved
	// is used a second time.
	ErrUseAfterMove LinearErrorCode = iota

	// ErrUnconsumedResource is returned when a linear binding leaves scope
	// without being moved or explicitly dropped.
	ErrUnconsumedResource

	// ErrDropNonResource is returned when drop is called on a non-linear binding.
	ErrDropNonResource

	// ErrUnknownBinding is returned when a name is referenced that was never
	// bound in this scope.
	ErrUnknownBinding
)

func (c LinearErrorCode) String() string {
	switch c {
	case ErrUseAfterMove:
		return "use-after-move"
	case ErrUnconsumedResource:
		return "unconsumed-resource"
	case ErrDropNonResource:
		return "drop-non-resource"
	case ErrUnknownBinding:
		return "unknown-binding"
	default:
		return fmt.Sprintf("linear-error(%d)", int(c))
	}
}

// LinearError records a single linear-type violation.
type LinearError struct {
	Code    LinearErrorCode
	Name    string // the binding involved
	Message string
}

func (e *LinearError) Error() string {
	return fmt.Sprintf("linear type error [%s] for %q: %s", e.Code, e.Name, e.Message)
}

// bindingState tracks the consumption state of a single binding.
type bindingState struct {
	typ    Type
	moved  bool // true once the value has been moved or dropped
}

// LinearChecker verifies linear type discipline within a single function scope.
//
// Usage:
//
//	lc := NewLinearChecker()
//	lc.Bind("coin", coinResourceType)  // introduce binding
//	if err := lc.Use("coin"); err != nil { ... }  // move it
//	errs := lc.CheckAllConsumed()      // must be empty
type LinearChecker struct {
	bindings map[string]*bindingState
}

// NewLinearChecker returns a fresh checker with an empty scope.
func NewLinearChecker() *LinearChecker {
	return &LinearChecker{
		bindings: make(map[string]*bindingState),
	}
}

// Bind introduces a new binding with the given name and type.
// If a binding with the same name already exists it is silently replaced
// (shadowing), which is safe as long as the previous binding was consumed.
func (lc *LinearChecker) Bind(name string, typ Type) {
	lc.bindings[name] = &bindingState{typ: typ, moved: false}
}

// Use marks the binding named name as consumed (moved).
//
// Returns an error if:
//   - The name is not bound in this scope (ErrUnknownBinding).
//   - The binding has already been moved (ErrUseAfterMove).
//
// For non-linear types Use is always successful; calling it multiple times on
// a non-linear binding is permitted because copyable values can be used freely.
func (lc *LinearChecker) Use(name string) error {
	b, ok := lc.bindings[name]
	if !ok {
		return &LinearError{
			Code:    ErrUnknownBinding,
			Name:    name,
			Message: fmt.Sprintf("no binding named %q in current scope", name),
		}
	}

	// Non-linear types can be used any number of times.
	if !b.typ.IsLinear() {
		return nil
	}

	if b.moved {
		return &LinearError{
			Code:    ErrUseAfterMove,
			Name:    name,
			Message: fmt.Sprintf("%q has already been moved; cannot use after move", name),
		}
	}

	b.moved = true
	return nil
}

// Drop explicitly destroys the resource named name.
//
// Returns an error if:
//   - The name is not bound in this scope (ErrUnknownBinding).
//   - The binding is not a linear resource (ErrDropNonResource) — dropping a
//     non-resource is always a programmer mistake in the PROBE language.
//   - The binding has already been moved (ErrUseAfterMove).
func (lc *LinearChecker) Drop(name string) error {
	b, ok := lc.bindings[name]
	if !ok {
		return &LinearError{
			Code:    ErrUnknownBinding,
			Name:    name,
			Message: fmt.Sprintf("no binding named %q in current scope", name),
		}
	}

	if !b.typ.IsLinear() {
		return &LinearError{
			Code:    ErrDropNonResource,
			Name:    name,
			Message: fmt.Sprintf("%q has type %s which is not a linear resource; drop is unnecessary", name, b.typ),
		}
	}

	if b.moved {
		return &LinearError{
			Code:    ErrUseAfterMove,
			Name:    name,
			Message: fmt.Sprintf("%q has already been moved; cannot drop after move", name),
		}
	}

	b.moved = true
	return nil
}

// CheckAllConsumed verifies that every linear binding in scope has been
// consumed (moved or dropped). It returns one LinearError per violation.
//
// Call this at the end of a function or block scope.
func (lc *LinearChecker) CheckAllConsumed() []LinearError {
	var errs []LinearError
	for name, b := range lc.bindings {
		if b.typ.IsLinear() && !b.moved {
			errs = append(errs, LinearError{
				Code:    ErrUnconsumedResource,
				Name:    name,
				Message: fmt.Sprintf("resource %q of type %s was never consumed (add `move` or `drop`)", name, b.typ),
			})
		}
	}
	return errs
}

// ---- FnScope ---------------------------------------------------------------

// FnScope models a function body for the purposes of linear checking.
// It holds the checker and the sequence of operations performed.
type FnScope struct {
	Name    string
	Checker *LinearChecker
}

// NewFnScope creates a FnScope for function name.
func NewFnScope(name string) *FnScope {
	return &FnScope{
		Name:    name,
		Checker: NewLinearChecker(),
	}
}

// CheckFunction runs the full linear check on fn and returns all violations.
// This is the top-level entry point used by the verifier.
func (lc *LinearChecker) CheckFunction(fn *FnScope) []LinearError {
	// Transfer the fn's checker state into lc for final consumption check.
	// In a real verifier the checker would be populated during code walking;
	// here we simply delegate to the fn's own checker.
	return fn.Checker.CheckAllConsumed()
}
