// Copyright 2024 The ProbeChain Authors
// This file is part of the ProbeChain.

// Package agent provides the standard library for PROBE agent operations.
//
// Agents are first-class entities in the PROBE language with:
//   - Unique identity (address-based)
//   - Reputation scoring (behavior-tracked)
//   - Discovery (network-level)
//   - Message passing (actor model)
package agent

// Identity represents an agent's on-chain identity.
type Identity struct {
	Address    [20]byte // unique agent address
	PublicKey  []byte   // signing public key
	Name       string   // human-readable name
	Version    uint64   // agent version
	CreatedAt  uint64   // block number of creation
}

// Reputation tracks an agent's behavior score.
type Reputation struct {
	Score       uint64  // behavior score (0-1000)
	TotalTasks  uint64  // total tasks completed
	SuccessRate uint64  // success rate (0-100)
	Slashes     uint64  // number of slashing events
}

// Capability describes a service an agent can provide.
type Capability struct {
	Name        string
	Version     string
	Description string
	Cost        uint64 // cost in pico per invocation
}

// Message is an inter-agent message.
type Message struct {
	From    [20]byte
	To      [20]byte
	Payload []byte
	Nonce   uint64
	Value   uint64 // attached PROBE value in pico
}

// Registry provides agent discovery.
type Registry interface {
	Register(id Identity) error
	Lookup(address [20]byte) (*Identity, error)
	FindByCapability(capability string) ([]Identity, error)
	UpdateReputation(address [20]byte, delta int64) error
	GetReputation(address [20]byte) (*Reputation, error)
	Advertise(address [20]byte, cap Capability) error
}
