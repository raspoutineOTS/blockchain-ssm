# Lightning Network Integration - Quick Start Guide

## 🚀 Introduction

This guide helps you quickly get started with Lightning Network integration for SSM carbon credits.

---

## 📋 Prerequisites

### Required Infrastructure

1. **Bitcoin Node** (Testnet for development)
```bash
# Install Bitcoin Core
wget https://bitcoincore.org/bin/bitcoin-core-25.0/bitcoin-25.0-x86_64-linux-gnu.tar.gz
tar -xvf bitcoin-25.0-x86_64-linux-gnu.tar.gz
cd bitcoin-25.0/bin

# Configure testnet
cat > ~/.bitcoin/bitcoin.conf <<EOF
testnet=1
server=1
rpcuser=bitcoinrpc
rpcpassword=YOUR_SECURE_PASSWORD
txindex=1
zmqpubrawblock=tcp://127.0.0.1:28332
zmqpubrawtx=tcp://127.0.0.1:28333
EOF

# Start Bitcoin node
./bitcoind -daemon
```

2. **LND (Lightning Network Daemon)**
```bash
# Install LND
wget https://github.com/lightningnetwork/lnd/releases/download/v0.17.0/lnd-linux-amd64-v0.17.0.tar.gz
tar -xvf lnd-linux-amd64-v0.17.0.tar.gz
cd lnd-linux-amd64-v0.17.0

# Configure LND
cat > ~/.lnd/lnd.conf <<EOF
[Application Options]
alias=SSM-Lightning-Node
listen=0.0.0.0:9735

[Bitcoin]
bitcoin.active=true
bitcoin.testnet=true
bitcoin.node=bitcoind

[Bitcoind]
bitcoind.rpchost=localhost
bitcoind.rpcuser=bitcoinrpc
bitcoind.rpcpass=YOUR_SECURE_PASSWORD
bitcoind.zmqpubrawblock=tcp://127.0.0.1:28332
bitcoind.zmqpubrawtx=tcp://127.0.0.1:28333
EOF

# Start LND
./lnd
```

3. **Taproot Assets Daemon**
```bash
# Install tapd
wget https://github.com/lightninglabs/taproot-assets/releases/download/v0.6.0/tapd-linux-amd64-v0.6.0.tar.gz
tar -xvf tapd-linux-amd64-v0.6.0.tar.gz

# Start tapd (depends on LND)
./tapd --network=testnet
```

4. **Hyperledger Fabric** (existing)
```bash
# Your existing Hyperledger installation
cd deployment/local
./bootstrap.sh
```

---

## 🔧 Configuration

### 1. Create a Lightning Wallet

```bash
# Create LND wallet
lncli create

# Get a Bitcoin address to fund the wallet
lncli newaddress p2wkh

# Get testnet coins
# Visit: https://testnet-faucet.mempool.co/
```

### 2. Open a Lightning Channel

```bash
# Connect to a peer
lncli connect 03xxx...@lightning-node.example.com:9735

# Open a channel with 1,000,000 sats
lncli openchannel --node_key=03xxx... --local_amt=1000000
```

### 3. Deploy SSM Chaincode with Lightning

```bash
cd chaincode/go/ssm

# Chaincode now includes:
# - ssm.go (existing)
# - lightning-anchor.go (new)
# - ssm-lightning.go (new)

# Install chaincode
peer chaincode install -n ssm -v 2.0 -p ./
peer chaincode instantiate -n ssm -v 2.0 -C mychannel -c '{"Args":["init"]}'
```

---

## 💻 Usage Examples

### Example 1: Create and Anchor a Carbon Credit

```javascript
// 1. Create an ITMO
const itmo = {
  ID: "ITMO-2025-WIND-DE-001",
  QuantityTonsCO2e: 500.0,
  ProjectType: "Wind Energy",
  Methodology: "CDM AM0077",
  CountryOfOrigin: "Germany",
  VintageYear: 2024,
  VerificationStatus: "Verified"
};

// 2. Start SSM session
const session = {
  ssm: "CarbonCreditLifecycle",
  session: "wind_germany_001",
  roles: {
    "EnergyCompany": "Alice",
    "Buyer": "Bob"
  },
  public: JSON.stringify(itmo)
};

peer.chaincode.invoke({
  fcn: "start",
  args: [JSON.stringify(session), "Alice", signature]
});

// 3. Enable Lightning for this session
peer.chaincode.invoke({
  fcn: "enableLightning",
  args: ["wind_germany_001", "every_transition", "0"]
});

// 4. Anchor state on Bitcoin
peer.chaincode.invoke({
  fcn: "perform",
  args: ["AnchorToLightning", JSON.stringify(context), "Alice", signature]
});
```

### Example 2: Mint Taproot Asset

```javascript
// Mint a Taproot Asset from ITMO
peer.chaincode.invoke({
  fcn: "perform",
  args: [
    "MintTaprootAsset",
    JSON.stringify({
      session: "wind_germany_001",
      iteration: 0,
      public: JSON.stringify(itmo)
    }),
    "Alice",
    signature
  ]
});

// Result:
// {
//   "taproot_asset_id": "f1e2d3c4b5a69788",
//   "supply_amount": 500000,  // 500.000 units (0.001 tCO2e precision)
//   "decimals": 3
// }
```

### Example 3: Lightning Transfer

```bash
# 1. Generate Lightning invoice (recipient Bob's side)
lncli addinvoice --amt_msat=100000 --memo="Carbon credit 0.1 tCO2e"

# Output:
# payment_request: lnbc1000n1p3xr...
# payment_hash: abc123def456...

# 2. Record transfer in SSM
peer.chaincode.invoke({
  fcn: "perform",
  args: [
    "TransferViaLightning",
    JSON.stringify({
      session: "wind_germany_001",
      iteration: 1,
      public: JSON.stringify({
        asset_id: "f1e2d3c4b5a69788",
        from_agent: "Alice",
        to_agent: "Bob",
        amount: 0.1,
        invoice: "lnbc1000n1p3xr..."
      })
    }),
    "Alice",
    signature
  ]
});

# 3. Pay invoice (Alice's side)
lncli payinvoice lnbc1000n1p3xr...

# Transaction completed in < 1 second!
```

### Example 4: Verify an Anchor

```javascript
// Verify that a state has been anchored on Bitcoin
peer.chaincode.query({
  fcn: "perform",
  args: [
    "VerifyLightningAnchor",
    JSON.stringify({
      session: "wind_germany_001",
      iteration: 0
    }),
    "TUV_SUD",
    signature
  ]
});

// Result:
// {
//   "session": "wind_germany_001",
//   "iteration": 0,
//   "verified": true,
//   "bitcoin_txid": "7a8b9c1d2e3f4a5b6c7d8e9f...",
//   "block_height": 2500123,
//   "confirmations": 6
// }
```

---

## 🔍 Complete Use Cases

### Use Case 1: Lightning Marketplace for Carbon Credits

```javascript
// Scenario: A marketplace where users can buy
// fractions of carbon credits with stablecoins

// 1. Seller: Mint 1000 tCO2e as Taproot Asset
const mintResult = await mintTaprootAsset(itmo);
// → 1,000,000 units (0.001 tCO2e each)

// 2. Buyer: Purchase 5.5 tCO2e for 55 USDT
const purchaseInvoice = await generateLightningInvoice({
  amount_msat: 55000000, // 55 USDT in millisatoshis
  description: "5.5 tCO2e carbon credits",
  asset_id: mintResult.taproot_asset_id,
  asset_amount: 5500 // 5.5 tCO2e = 5500 units
});

// 3. Atomic payment: USDT ↔ Carbon Credits
const result = await atomicSwap({
  buyer_pays: "55 USDT (via Lightning)",
  seller_delivers: "5.5 tCO2e Taproot Asset",
  invoice: purchaseInvoice
});

// ✅ Transaction completed in < 1 sec, fees < $0.001
```

### Use Case 2: Multi-Registry Traceability

```javascript
// Verify that a credit hasn't been sold elsewhere

// 1. Query Hyperledger SSM
const ssmState = await querySSMSession("wind_germany_001");

// 2. Verify Bitcoin anchor
const bitcoinAnchor = await verifyBitcoinAnchor(
  ssmState.session,
  ssmState.iteration
);

// 3. Query external registries via oracles
const externalRegistries = [
  "Verra",
  "Gold Standard",
  "Climate Action Reserve"
];

const verifications = await Promise.all(
  externalRegistries.map(registry =>
    oracleVerify(itmo.ID, registry)
  )
);

// 4. Consolidated result
const report = {
  itmo_id: itmo.ID,
  ssm_status: ssmState.current,
  bitcoin_anchor: bitcoinAnchor.verified,
  bitcoin_txid: bitcoinAnchor.txid,
  external_registries: verifications,
  double_counting_detected: false
};
```

---

## 📊 Monitoring and Observability

### Lightning Node Dashboard

```bash
# Install RTL (Ride The Lightning)
npm install -g rtl

# Configure RTL
rtl --lnnode=LND --configpath=/home/user/.lnd

# Access dashboard
# http://localhost:3000
```

### Important Metrics

```javascript
// 1. Lightning node status
lncli getinfo

// 2. Active channels
lncli listchannels

// 3. Balance
lncli walletbalance
lncli channelbalance

// 4. Payment history
lncli listpayments

// 5. Minted Taproot Assets
tapcli assets list

// 6. SSM Anchors
peer.chaincode.query({
  fcn: "queryAnchorsBySession",
  args: ["wind_germany_001"]
});
```

---

## 🛠️ Troubleshooting

### Problem: Lightning channel closed

```bash
# Check channels
lncli listchannels

# Reopen a channel
lncli openchannel --node_key=03xxx... --local_amt=1000000
```

### Problem: Taproot Asset mint fails

```bash
# Verify tapd is connected to LND
tapcli getinfo

# Check LND wallet
lncli walletbalance

# Re-sync tapd
tapcli stop
tapcli start --network=testnet
```

### Problem: Bitcoin anchor not confirmed

```bash
# Check mempool
bitcoin-cli getrawmempool

# Increase fees (RBF)
bitcoin-cli bumpfee <txid>
```

---

## 🔐 Security

### Best Practices

1. **Private Keys**
   - Use Hardware Security Modules (HSM) for LND keys
   - Encrypted backup of LND seeds

2. **Watchtowers**
   ```bash
   # Enable watchtower for 24/7 monitoring
   lncli tower info
   lncli wtclient add <tower_pubkey>@<tower_host>
   ```

3. **Multi-signatures**
   - Require multiple signatures for mints > 1000 tCO2e
   - Use MuSig2 for aggregated signatures

4. **Audit Trails**
   - All anchors are public on Bitcoin
   - Immutable logs on Hyperledger
   - Systematic cross-chain verification

---

## 📚 Resources

### Documentation
- [Complete Integration Plan](./LIGHTNING_NETWORK_INTEGRATION_PLAN.md)
- [JSON Examples](./examples/lightning-integration-example.json)
- [LND Documentation](https://docs.lightning.engineering/)
- [Taproot Assets Guide](https://docs.lightning.engineering/the-lightning-network/taproot-assets)

### Tools
- [Polar](https://lightningpolar.com/) - Local Lightning network
- [ThunderHub](https://www.thunderhub.io/) - LND web interface
- [Mempool.space](https://mempool.space/testnet) - Bitcoin testnet explorer

### Support
- GitHub Issues: [blockchain-ssm/issues](https://github.com/blockchain-ssm/issues)
- Lightning Dev Slack: lightning.engineering/slack

---

## ✅ Production Checklist

Before deploying to production:

- [ ] Bitcoin mainnet node configured and synced
- [ ] LND node with multiple channels (redundancy)
- [ ] Watchtowers configured (min. 2)
- [ ] Automated LND key backups
- [ ] Monitoring with alerts (Prometheus + Grafana)
- [ ] Load testing (1000+ tx/sec)
- [ ] External security audit
- [ ] Complete internal documentation
- [ ] Ops team training
- [ ] Disaster recovery plan

---

**Version**: 1.0
**Last Updated**: 2025-11-09
**Author**: Blockchain SSM Lightning Integration Team
