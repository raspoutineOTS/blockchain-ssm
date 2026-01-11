# The Signing State Machines Blockchain

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Hyperledger Fabric](https://img.shields.io/badge/Hyperledger%20Fabric-2.5+-green.svg)](https://hyperledger-fabric.readthedocs.io/)
[![Lightning Network](https://img.shields.io/badge/Lightning%20Network-Enabled-yellow.svg)](https://lightning.network/)

> A Signing State Machine (SSM) is a smart contract written with a more constrained paradigm than plain programming languages, based on a finite state automaton.

## What's New (v2.0)

This fork extends the original SSM blockchain with **climate finance infrastructure**:

- **ITMO Article 6 Compliance** — State-only tracking for international carbon transfers
- **Lightning Network Integration** — Sub-second settlements, 93% storage optimization
- **DeFi Stack** — Lending, borrowing, AMM pools for carbon credits
- **AI-Validated Asset Registry** — Collateralized stablecoin minting

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Quick Start](#quick-start)
- [ITMO Article 6 System](#itmo-article-6-system)
- [Lightning Network Layer](#lightning-network-layer)
- [DeFi Components](#defi-components)
- [Original SSM Documentation](#original-ssm-documentation)
- [Contributing](#contributing)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Climate Finance Layer                        │
├──────────────────┬──────────────────┬───────────────────────────┤
│   ITMO States    │   DeFi Stack     │   AI Validation           │
│   - Article 6.2  │   - Price Oracle │   - Impact Scoring        │
│   - Article 6.4  │   - Lending Pool │   - Risk Assessment       │
│   - CA Tracking  │   - AMM Pools    │   - Recommendations       │
├──────────────────┴──────────────────┴───────────────────────────┤
│                    Lightning Network (L2)                        │
│         Instant transfers • Micropayments • Taproot Assets       │
├─────────────────────────────────────────────────────────────────┤
│                   Hyperledger Fabric (L1)                        │
│              SSM Chaincode • State Management • Audit            │
├─────────────────────────────────────────────────────────────────┤
│                      Bitcoin Anchoring                           │
│                 Immutable • Decentralized • Secure               │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.19+
- Node.js 18+ (optional, for SDK)
- Python 3.10+ (for validators)

### Local Deployment

```bash
# Clone the repository
git clone https://github.com/raspoutineOTS/blockchain-ssm.git
cd blockchain-ssm

# Deploy local Hyperledger Fabric network
cd deployment/local
./start.sh

# Install SSM chaincode
cd ../../sdk/cli
./install.sh
```

See [local deployment README](deployment/local/README.md) for detailed instructions.

---

## ITMO Article 6 System

State-only tracking for Internationally Transferred Mitigation Outcomes, operating like **SWIFT for carbon credits**.

### Key Features

| Feature | Description |
|---------|-------------|
| **State-Only Tracking** | Track ITMO status without custody — credits stay in national registries |
| **Corresponding Adjustments** | Automatic CA compliance per Paris Agreement |
| **Multi-Registry Support** | Verra, Gold Standard, UNFCCC A6D connectors |
| **Tokenization** | Fractional ownership via State Tokens |
| **Collateralization** | Use states as collateral (min 120% ratio) |

### State Lifecycle

```
Issued → Authorized → Transferred → Held → Retired
    ↓         ↓            ↓          ↓
  [CA-]    [CA+/-]      [CA+/-]    [Final]
```

Full documentation: [ITMO_README.md](ITMO_README.md) | [ITMO_ARTICLE6_ARCHITECTURE.md](ITMO_ARTICLE6_ARCHITECTURE.md)

---

## Lightning Network Layer

Hybrid L1/L2 architecture enabling instant carbon credit transfers.

### Performance Gains

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Settlement | 2-5 sec | < 1 sec | **5x faster** |
| Cost/tx | $0.05-0.50 | < $0.001 | **99% reduction** |
| Min. trade | 1 tCO2e | 0.001 tCO2e | **Micropayments enabled** |
| Storage | 100% on-chain | 7% on-chain | **93% optimization** |

### Compact Hash-Based Anchoring

Instead of storing full state on Bitcoin, we anchor compact hashes:

```go
// 32-byte state anchor vs. full state storage
stateHash := sha256(ssm + session + iteration + currentState + publicDataHash)
```

Full documentation: [LIGHTNING_README.md](LIGHTNING_README.md) | [LIGHTNING_QUICKSTART.md](LIGHTNING_QUICKSTART.md)

---

## DeFi Components

Full DeFi stack operating on ITMO states while actual credits remain in registries.

### Price Oracle

Multi-source aggregation with manipulation protection:

```go
// Circuit breaker triggers if price deviation > 15%
sources: [Verra, GoldStandard, ICE, Exchanges]
methods: [median, mean, weighted_average]
```

### Lending Protocol

| Utilization | Supply APY | Borrow APY |
|-------------|------------|------------|
| 50% | 2.3% | 5.0% |
| 75% | 5.6% | 10.0% |
| 90% | 25.9% | 35.0% |

### AMM Pools

- Uniswap-style constant product (x * y = k)
- 0.3% swap fees to LPs
- Slippage protection included

Full documentation: [DEFI_README.md](DEFI_README.md)

---

## Original SSM Documentation

### Theory of Operations

#### Agents, Roles and Identities

Every agent provides:
- An identifier
- A public key

Agents hold the private key and never share it. In a SSM, transitions are performed by roles, mapped to agents at runtime.

#### States and Transitions

- Each transition is a tuple `<role, action>`
- Agent signs the update with their private key
- SSM validates using the agent's public key
- Contract is fulfilled when entering an acceptance state

### Data Structures

<details>
<summary>Agent</summary>

```json
{
  "name": "Adam",
  "pub": "MIIBIjANBgkqhkiG9w0BAQEFA..."
}
```
</details>

<details>
<summary>SSM State</summary>

```json
{
  "ssm": "Negociation",
  "session": "carsale20190301",
  "iteration": 1,
  "limit": 10,
  "roles": {"Bob": "Validator", "Sam": "Initiator"},
  "current": 2,
  "origin": {"from": 1, "to": 2, "role": "Validator", "action": "Accept"},
  "public": "Used car for 100 dollars.",
  "private": {"Bob": "XXXX", "Sam": "YYYY"}
}
```
</details>

<details>
<summary>Signing State Machine</summary>

```json
{
  "name": "Negociation",
  "transitions": [
    {"from": 0, "to": 1, "role": "Initiator", "action": "Propose"},
    {"from": 1, "to": 2, "role": "Validator", "action": "Accept"},
    {"from": 1, "to": 3, "role": "Validator", "action": "Reject"}
  ]
}
```
</details>

### API Reference

See [SDK Documentation](sdk/cli/README.md) and [CLI Tutorial](sdk/cli/tutorial.md).

---

## Testing

```bash
# Run Go chaincode tests
cd chaincode/go/ssm
go test -v ./...

# Run integration tests
cd test
npm install
npm test

# Run Python validator tests
python -m pytest test_impact_validator.py
```

---

## Project Structure

```
blockchain-ssm/
├── chaincode/go/ssm/          # Core SSM chaincode + extensions
│   ├── ssm-contract.go        # Modernized SSM logic (Fabric 2.5+)
│   ├── itmo-*.go              # ITMO Article 6 implementations
│   ├── lightning-*.go         # Lightning Network integration
│   ├── defi-*.go              # DeFi stack (oracle, lending, AMM)
│   └── hybrid-storage.go      # Compact hash anchoring
├── deployment/local/          # Local Fabric network
├── sdk/cli/                   # Command-line interface
├── registry-connectors/       # Verra, Gold Standard, UNFCCC
├── examples/                  # Usage examples (JS, Python)
├── test/                      # Integration test suites
├── test-network/              # Docker test environment
├── impact_validator.py        # AI validation framework
└── validator_api.py           # REST API for validators
```

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

This project is licensed under the Apache License 2.0 — see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- Original SSM implementation by [Luc Yriarte](https://github.com/lyriarte/blockchain-ssm)
- Hyperledger Fabric community
- Lightning Network developers

---

<p align="center">
  <b>Built for climate finance. Powered by blockchain.</b>
</p>
