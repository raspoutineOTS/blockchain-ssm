# Lightning Network Integration - Guide de Démarrage Rapide

## 🚀 Introduction

Ce guide vous permet de démarrer rapidement avec l'intégration Lightning Network pour les crédits carbone SSM.

---

## 📋 Prérequis

### Infrastructure Requise

1. **Bitcoin Node** (Testnet pour développement)
```bash
# Installation Bitcoin Core
wget https://bitcoincore.org/bin/bitcoin-core-25.0/bitcoin-25.0-x86_64-linux-gnu.tar.gz
tar -xvf bitcoin-25.0-x86_64-linux-gnu.tar.gz
cd bitcoin-25.0/bin

# Configuration testnet
cat > ~/.bitcoin/bitcoin.conf <<EOF
testnet=1
server=1
rpcuser=bitcoinrpc
rpcpassword=YOUR_SECURE_PASSWORD
txindex=1
zmqpubrawblock=tcp://127.0.0.1:28332
zmqpubrawtx=tcp://127.0.0.1:28333
EOF

# Démarrer Bitcoin node
./bitcoind -daemon
```

2. **LND (Lightning Network Daemon)**
```bash
# Installation LND
wget https://github.com/lightningnetwork/lnd/releases/download/v0.17.0/lnd-linux-amd64-v0.17.0.tar.gz
tar -xvf lnd-linux-amd64-v0.17.0.tar.gz
cd lnd-linux-amd64-v0.17.0

# Configuration LND
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

# Démarrer LND
./lnd
```

3. **Taproot Assets Daemon**
```bash
# Installation tapd
wget https://github.com/lightninglabs/taproot-assets/releases/download/v0.6.0/tapd-linux-amd64-v0.6.0.tar.gz
tar -xvf tapd-linux-amd64-v0.6.0.tar.gz

# Démarrer tapd (dépend de LND)
./tapd --network=testnet
```

4. **Hyperledger Fabric** (existant)
```bash
# Votre installation Hyperledger existante
cd deployment/local
./bootstrap.sh
```

---

## 🔧 Configuration

### 1. Créer un Wallet Lightning

```bash
# Créer wallet LND
lncli create

# Obtenir une adresse Bitcoin pour financer le wallet
lncli newaddress p2wkh

# Obtenir des testnet coins
# Visitez: https://testnet-faucet.mempool.co/
```

### 2. Ouvrir un Channel Lightning

```bash
# Se connecter à un peer
lncli connect 03xxx...@lightning-node.example.com:9735

# Ouvrir un channel avec 1,000,000 sats
lncli openchannel --node_key=03xxx... --local_amt=1000000
```

### 3. Déployer le Chaincode SSM avec Lightning

```bash
cd chaincode/go/ssm

# Le chaincode inclut maintenant:
# - ssm.go (existant)
# - lightning-anchor.go (nouveau)
# - ssm-lightning.go (nouveau)

# Installer le chaincode
peer chaincode install -n ssm -v 2.0 -p ./
peer chaincode instantiate -n ssm -v 2.0 -C mychannel -c '{"Args":["init"]}'
```

---

## 💻 Exemples d'Utilisation

### Exemple 1: Créer et Ancrer un Crédit Carbone

```javascript
// 1. Créer un ITMO
const itmo = {
  ID: "ITMO-2025-WIND-DE-001",
  QuantityTonsCO2e: 500.0,
  ProjectType: "Wind Energy",
  Methodology: "CDM AM0077",
  CountryOfOrigin: "Germany",
  VintageYear: 2024,
  VerificationStatus: "Verified"
};

// 2. Démarrer une session SSM
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

// 3. Activer Lightning pour cette session
peer.chaincode.invoke({
  fcn: "enableLightning",
  args: ["wind_germany_001", "every_transition", "0"]
});

// 4. Ancrer l'état sur Bitcoin
peer.chaincode.invoke({
  fcn: "perform",
  args: ["AnchorToLightning", JSON.stringify(context), "Alice", signature]
});
```

### Exemple 2: Mint Taproot Asset

```javascript
// Minter un Taproot Asset depuis l'ITMO
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

// Résultat:
// {
//   "taproot_asset_id": "f1e2d3c4b5a69788",
//   "supply_amount": 500000,  // 500.000 units (0.001 tCO2e precision)
//   "decimals": 3
// }
```

### Exemple 3: Transfert Lightning

```bash
# 1. Générer une invoice Lightning (côté destinataire Bob)
lncli addinvoice --amt_msat=100000 --memo="Carbon credit 0.1 tCO2e"

# Output:
# payment_request: lnbc1000n1p3xr...
# payment_hash: abc123def456...

# 2. Enregistrer le transfert dans SSM
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

# 3. Payer l'invoice (côté Alice)
lncli payinvoice lnbc1000n1p3xr...

# Transaction complétée en < 1 seconde!
```

### Exemple 4: Vérifier un Ancrage

```javascript
// Vérifier qu'un état a été ancré sur Bitcoin
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

// Résultat:
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

## 🔍 Cas d'Usage Complets

### Cas 1: Marketplace Lightning pour Crédits Carbone

```javascript
// Scénario: Une marketplace où les utilisateurs peuvent acheter
// des fractions de crédits carbone avec des stablecoins

// 1. Vendeur: Minter 1000 tCO2e en Taproot Asset
const mintResult = await mintTaprootAsset(itmo);
// → 1,000,000 unités (0.001 tCO2e chacune)

// 2. Acheteur: Acheter 5.5 tCO2e pour 55 USDT
const purchaseInvoice = await generateLightningInvoice({
  amount_msat: 55000000, // 55 USDT en millisatoshis
  description: "5.5 tCO2e carbon credits",
  asset_id: mintResult.taproot_asset_id,
  asset_amount: 5500 // 5.5 tCO2e = 5500 units
});

// 3. Paiement atomique: USDT ↔ Carbon Credits
const result = await atomicSwap({
  buyer_pays: "55 USDT (via Lightning)",
  seller_delivers: "5.5 tCO2e Taproot Asset",
  invoice: purchaseInvoice
});

// ✅ Transaction complétée en < 1 sec, frais < $0.001
```

### Cas 2: Traçabilité Multi-Registres

```javascript
// Vérifier qu'un crédit n'a pas été vendu ailleurs

// 1. Consulter Hyperledger SSM
const ssmState = await querySSMSession("wind_germany_001");

// 2. Vérifier l'ancrage Bitcoin
const bitcoinAnchor = await verifyBitcoinAnchor(
  ssmState.session,
  ssmState.iteration
);

// 3. Consulter registres externes via oracles
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

// 4. Résultat consolidé
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

## 📊 Monitoring et Observabilité

### Dashboard Lightning Node

```bash
# Installer RTL (Ride The Lightning)
npm install -g rtl

# Configurer RTL
rtl --lnnode=LND --configpath=/home/user/.lnd

# Accéder au dashboard
# http://localhost:3000
```

### Métriques Importantes

```javascript
// 1. Statut du node Lightning
lncli getinfo

// 2. Channels actifs
lncli listchannels

// 3. Balance
lncli walletbalance
lncli channelbalance

// 4. Historique des paiements
lncli listpayments

// 5. Taproot Assets mintés
tapcli assets list

// 6. Anchors SSM
peer.chaincode.query({
  fcn: "queryAnchorsBySession",
  args: ["wind_germany_001"]
});
```

---

## 🛠️ Troubleshooting

### Problème: Lightning channel fermé

```bash
# Vérifier les channels
lncli listchannels

# Réouvrir un channel
lncli openchannel --node_key=03xxx... --local_amt=1000000
```

### Problème: Taproot Asset mint échoue

```bash
# Vérifier que tapd est connecté à LND
tapcli getinfo

# Vérifier le wallet LND
lncli walletbalance

# Re-sync tapd
tapcli stop
tapcli start --network=testnet
```

### Problème: Anchor Bitcoin non confirmé

```bash
# Vérifier le mempool
bitcoin-cli getrawmempool

# Augmenter les frais (RBF)
bitcoin-cli bumpfee <txid>
```

---

## 🔐 Sécurité

### Best Practices

1. **Clés Privées**
   - Utilisez Hardware Security Modules (HSM) pour les clés LND
   - Backup encrypted des seeds LND

2. **Watchtowers**
   ```bash
   # Activer watchtower pour surveillance 24/7
   lncli tower info
   lncli wtclient add <tower_pubkey>@<tower_host>
   ```

3. **Multi-signatures**
   - Exiger plusieurs signatures pour mints > 1000 tCO2e
   - Utiliser MuSig2 pour signatures agrégées

4. **Audit Trails**
   - Tous les anchors sont publics sur Bitcoin
   - Logs immuables sur Hyperledger
   - Vérification cross-chain systématique

---

## 📚 Ressources

### Documentation
- [Plan d'Intégration Complet](./LIGHTNING_NETWORK_INTEGRATION_PLAN.md)
- [Exemple JSON](./examples/lightning-integration-example.json)
- [LND Documentation](https://docs.lightning.engineering/)
- [Taproot Assets Guide](https://docs.lightning.engineering/the-lightning-network/taproot-assets)

### Outils
- [Polar](https://lightningpolar.com/) - Réseau Lightning local
- [ThunderHub](https://www.thunderhub.io/) - Interface web LND
- [Mempool.space](https://mempool.space/testnet) - Explorateur Bitcoin testnet

### Support
- GitHub Issues: [blockchain-ssm/issues](https://github.com/blockchain-ssm/issues)
- Lightning Dev Slack: lightning.engineering/slack

---

## ✅ Checklist de Production

Avant de déployer en production:

- [ ] Bitcoin mainnet node configuré et synchronisé
- [ ] LND node avec plusieurs channels (redondance)
- [ ] Watchtowers configurés (min. 2)
- [ ] Backup automatisé des clés LND
- [ ] Monitoring avec alertes (Prometheus + Grafana)
- [ ] Tests de charge (1000+ tx/sec)
- [ ] Audit de sécurité externe
- [ ] Documentation interne complète
- [ ] Formation de l'équipe ops
- [ ] Plan de disaster recovery

---

**Version**: 1.0
**Dernière mise à jour**: 2025-11-09
**Auteur**: Blockchain SSM Lightning Integration Team
