# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This repository contains two projects for the **ProbeChain** blockchain network:

- **go-probe-master/** — Full Go implementation of the Probe blockchain client (forked from go-ethereum/geth). Module: `github.com/probeum/go-probeum`. Go 1.15+.
- **Probe.Network-main/** — Documentation and PDFs for ProbeChain network setup (not a code project).

All development work happens in `go-probe-master/`.

## Build & Development Commands

All commands run from `go-probe-master/`:

```bash
make gprobe          # Build the main client → ./build/bin/gprobe
make all             # Build all cmd/ executables
make test            # Build all, then run full test suite
make lint            # Run golangci-lint
make clean           # Clean build cache and binaries
make devtools        # Install code generation tools (stringer, abigen, protoc-gen-go, etc.)
```

Run a single package's tests:
```bash
cd go-probe-master && go test ./core/...
cd go-probe-master && go test ./p2p/...
cd go-probe-master && go test -run TestSpecificName ./pkg/...
```

Build output goes to `go-probe-master/build/bin/`. The build system uses `build/ci.go` as its orchestrator (invoked via `go run`).

## Architecture

### Layered Design

```
CLI (cmd/gprobe)  →  Node lifecycle (node/)  →  Protocol (probe/, p2p/)
                                               →  Consensus (consensus/)
                                               →  Core chain (core/)
                                               →  Storage (probedb/)
```

### Key Packages

| Package | Purpose |
|---------|---------|
| `cmd/gprobe/` | Main client entry point — CLI app using `urfave/cli` |
| `core/` | Blockchain rules: `blockchain.go` (chain mgmt), `tx_pool.go` (mempool), `state_processor.go`, `state_transition.go` |
| `probe/` | Probe protocol handlers, sync, peer management |
| `consensus/` | Pluggable consensus engines: `probeash/` (PoW), `clique/` (PoA), `greatri/` (custom variant) |
| `p2p/` | Devp2p networking: node discovery (v4/v5), RLPx transport, DNS discovery |
| `accounts/` | Keystore, HD wallets, USB hardware wallet support |
| `rpc/` | JSON-RPC server (HTTP, WebSocket, IPC) |
| `internal/probeapi/` | Shared API implementations backing public/private RPC namespaces |
| `les/` | Light Ethereum Subprotocol (light client) |
| `miner/` | Block production |
| `trie/` | Merkle Patricia trie |
| `rlp/` | RLP encoding/decoding |
| `crypto/` | secp256k1, blake2b, BLS12-381, bn256 |
| `params/` | Network parameters, version info (`version.go`) |

### Other Executables (cmd/)

`abigen`, `bootnode`, `checkpoint-admin`, `clef` (signer), `devp2p`, `evm`, `faucet`, `probekey`, `puppprobe` (network simulator), `rlpdump`, `p2psim`.

### Chain Configuration

- **ChainID:** 1205
- **Consensus:** DPOS with 15-second block period (see `genesis.json`)
- **Genesis config** defines validators via enode URLs

## Linting

Configured in `.golangci.yml`. Enabled linters: `deadcode`, `goconst`, `goimports`, `gosimple`, `govet`, `ineffassign`, `misspell`, `unconvert`, `varcheck`. The file `core/genesis_alloc.go` is excluded from linting.

## Naming Conventions

This is an Ethereum fork — "geth" → "gprobe", "eth" → "probe", "Ethereum" → "Probeum". When adding code, follow the existing naming: use `probe`/`Probe`/`gprobe` prefixes, not `eth`/`Eth`/`geth`.

## License

GPL v3 / LGPL v3 (see `COPYING` and `COPYING.LESSER`).
