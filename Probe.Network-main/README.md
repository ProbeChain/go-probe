# Probe.Network

Official documentation repository for **ProbeChain** — a high-performance blockchain platform built on Proof-of-Behavior (PoB) consensus with 400ms block times, native post-quantum cryptography, and the PROBE Language.

## Documents

| Document | Description |
|----------|-------------|
| [ProbeChain V2.0 Whitepaper (PDF)](ProbeChain_V2.0_Whitepaper_EN.pdf) | V2.0 technical whitepaper — PoB consensus, StellarSpeed, PROBE Language, VM architecture, Stellar-Class Resilience |
| [ProbeChain V2.0 Whitepaper (Markdown)](ProbeChain_V2.0_Whitepaper_EN.md) | Same content in Markdown format |
| [ProbeChain V1.004 Whitepaper](ProbeChain%20V1.004-EN.pdf) | Original V1 whitepaper |
| [MetaMask Connection Guide](How%20to%20Connecting%20MetaMask%20to%20ProbeChain%20Mainnet.pdf) | How to connect MetaMask to ProbeChain Mainnet |

## Network Overview

| Parameter | Value |
|-----------|-------|
| Chain ID | 142857 |
| Consensus | Proof-of-Behavior (PoB) |
| Block Time | 400 ms (StellarSpeed) |
| Token | PROBE |
| Decimals | 18 |
| Total Supply | 10,000,000,000 PROBE |
| Smallest Unit | Pico (1 PROBE = 10^18 Pico) |

## Key Features

- **Proof-of-Behavior (PoB)** — Five-dimension behavioral scoring (liveness, correctness, cooperation, consistency, signal sovereignty) replaces energy-intensive mining
- **StellarSpeed** — 400ms pipelined block production with sub-second transaction finality
- **Post-Quantum Cryptography** — Native Dilithium (ML-DSA) signatures at the consensus layer; Falcon-512, SLH-DSA VM opcodes
- **PROBE Language** — Agent-first smart contract language with linear types, register-based VM (256 registers, 64-bit words), and native PQC opcodes
- **Stellar-Class Resilience** — AtomicTime nanosecond timestamps, Rydberg atomic receivers, RF block propagation (HF through THz)
- **Superlight DEX** — Native on-chain decentralized exchange with dedicated transaction type

## Links

- **Source Code:** [github.com/ProbeChain/go-probe](https://github.com/ProbeChain/go-probe)
- **Block Explorer:** [scan.probechain.org](https://scan.probechain.org/home)
- **Documentation:** [doc.probechain.org](https://doc.probechain.org/)

## License

MIT License. See [LICENSE](LICENSE) for details.
