# Signing State Machines (SSM) Blockchain v2

Modern implementation of Signing State Machines for Hyperledger Fabric 2.5+ with AI-validated asset registry and stablecoin collateralization.

## Overview

Signing State Machines (SSM) provide a constrained smart contract paradigm based on finite state automata. This v2 implementation modernizes the 2018 codebase with:

- ✅ **Hyperledger Fabric 2.5+ Support** - Contract API
- ✅ **AI Impact Validation** - Real-time transition assessment
- ✅ **Asset Registry** - Comprehensive asset management
- ✅ **Stablecoin Collateralization** - Asset-backed positions
- ✅ **Event Emission** - Event-driven architecture support
- ✅ **Modern Go Modules** - Proper dependency management

## Quick Start

### Prerequisites

- Hyperledger Fabric 2.5+
- Go 1.20+
- Python 3.8+ (for AI validator)
- Docker (for local deployment)

### Build Chaincode

```bash
cd chaincode/go/ssm

# Download dependencies
go mod download

# Build chaincode
go build -o ssm
```

### Deploy Chaincode

```bash
# Package chaincode
peer lifecycle chaincode package ssm.tar.gz \
  --path ./chaincode/go/ssm \
  --lang golang \
  --label ssm_v2_0

# Install on peers
peer lifecycle chaincode install ssm.tar.gz

# Follow standard Fabric 2.5+ deployment process for approve and commit
```

### Start AI Validator (Optional)

```bash
# Install Python dependencies
pip install -r requirements.txt

# Start validator API
python validator_api.py

# API runs on http://0.0.0.0:5000
```

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────┐
│                   Client Application                     │
└────────────────────┬────────────────────────────────────┘
                     │
         ┌───────────┴────────────┐
         │                        │
┌────────▼────────┐    ┌─────────▼──────────┐
│  SSM Chaincode  │    │  AI Validator API  │
│  (Fabric 2.5+)  │◄───┤  (Python/Flask)    │
└─────────────────┘    └────────────────────┘
         │
┌────────▼────────────────────────────────────┐
│     Hyperledger Fabric Blockchain           │
│  (State DB + Ledger + Smart Contracts)      │
└─────────────────────────────────────────────┘
```

### Key Features

#### 1. Signing State Machines
- Finite state automata for smart contracts
- Role-based transitions
- Cryptographic signature verification
- Public and private state data

#### 2. AI-Validated Transitions
- Real-time compatibility scoring (0-1)
- Impact metrics calculation
- Liquidation risk assessment
- Automatic rejection of risky operations
- Configurable validation thresholds

#### 3. Asset Management
- Generic asset registry
- Metadata and verification tracking
- Asset lifecycle management
- State-based asset evolution

#### 4. Stablecoin Collateralization
- Asset-backed stablecoin positions
- Collateralization ratio monitoring
- Health checks and alerts
- Liquidation price tracking
- Multi-stablecoin support (USDC, USDT, DAI)

## API Reference

### Transaction Functions

#### Agent Management
```go
RegisterUser(userJSON, adminName, signature string) error
```
Register a new user agent with cryptographic verification.

#### SSM Management
```go
CreateSSM(ssmJSON, adminName, signature string) error
StartSession(stateJSON, adminName, signature string) error
SetSessionLimit(updateJSON, adminName, signature string) error
```
Create and manage Signing State Machines and sessions.

#### Transition Execution
```go
PerformTransition(action, stateJSON, userName, signature string) error
PerformTransitionWithValidation(action, stateJSON, userName, signature string) error
```
Execute state transitions with optional AI validation.

#### Credits and Permissions
```go
GrantCredits(grantJSON, adminName, signature string) error
```
Manage user API credits and permissions.

### Query Functions

```go
GetSession(sessionID string) (*State, error)
GetSSM(ssmName string) (*SigningStateMachine, error)
GetUser(userName string) (*Agent, error)
GetAdmin(adminName string) (*Agent, error)
GetCredits(userName string) (*Grant, error)
ListByType(entityType string) ([]string, error)
GetSessionHistory(sessionID string) ([]map[string]interface{}, error)
```

## Data Structures

### Agent
```json
{
  "name": "alice",
  "pub": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
}
```

### State Machine
```json
{
  "name": "AssetLifecycle",
  "transitions": [
    {"from": 0, "to": 1, "role": "issuer", "action": "create"},
    {"from": 1, "to": 2, "role": "holder", "action": "collateralize"},
    {"from": 2, "to": 3, "role": "holder", "action": "transfer"}
  ]
}
```

### Session State
```json
{
  "ssm": "AssetLifecycle",
  "session": "asset001",
  "iteration": 0,
  "current": 0,
  "roles": {
    "alice": "issuer",
    "bob": "holder"
  },
  "public": {"description": "Carbon credit asset"},
  "private": {
    "alice": "encrypted_data_1",
    "bob": "encrypted_data_2"
  }
}
```

### Asset Metadata (Extended)
```json
{
  "assetId": "ASSET001",
  "assetType": "carbon_credit",
  "quantity": 1000.0,
  "unit": "tCO2e",
  "attributes": {
    "vintage": 2024,
    "methodology": "VM0042",
    "country": "Brazil"
  },
  "verificationData": {
    "status": "verified",
    "verifiedBy": "Verra",
    "certificateId": "CERT123"
  }
}
```

### Collateral Position
```json
{
  "positionId": "POS001",
  "assetId": "ASSET001",
  "collateralAmount": 15000.0,
  "stablecoinAmount": 10000.0,
  "stablecoinType": "USDC",
  "collateralRatio": 1.5,
  "status": "active",
  "ownerAgent": "alice"
}
```

## Usage Examples

### 1. Register User

```javascript
const user = {
  name: "alice",
  pub: "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
};

await contract.submitTransaction(
  'SSMContract:RegisterUser',
  JSON.stringify(user),
  'admin1',
  signature
);
```

### 2. Create State Machine

```javascript
const ssm = {
  name: "Negotiation",
  transitions: [
    {from: 0, to: 1, role: "initiator", action: "propose"},
    {from: 1, to: 2, role: "validator", action: "accept"},
    {from: 1, to: 3, role: "validator", action: "reject"}
  ]
};

await contract.submitTransaction(
  'SSMContract:CreateSSM',
  JSON.stringify(ssm),
  'admin1',
  signature
);
```

### 3. Start Session

```javascript
const initialState = {
  ssm: "Negotiation",
  session: "deal001",
  iteration: 0,
  current: 0,
  roles: {
    "alice": "initiator",
    "bob": "validator"
  },
  public: "Initial offer: 100 units"
};

await contract.submitTransaction(
  'SSMContract:StartSession',
  JSON.stringify(initialState),
  'admin1',
  signature
);
```

### 4. Perform Transition with AI Validation

```javascript
const update = {
  session: "deal001",
  iteration: 0,
  current: 1,
  public: "Updated offer: 95 units",
  assetData: {
    assetId: "ASSET001",
    quantity: 95.0
  },
  collateralData: {
    collateralAmount: 142.5,
    stablecoinAmount: 95.0,
    stablecoinType: "USDC"
  }
};

await contract.submitTransaction(
  'SSMContract:PerformTransitionWithValidation',
  'propose',
  JSON.stringify(update),
  'alice',
  signature
);
```

### 5. Query Session

```javascript
const session = await contract.evaluateTransaction(
  'SSMContract:GetSession',
  'deal001'
);
console.log(JSON.parse(session));
```

## AI Validator Integration

### Configuration

Set the validator URL (optional):
```bash
export VALIDATOR_URL=http://validator:5000
```

### Validation Metrics

The AI validator calculates:

- **Compatibility Score** (0-1): Overall transition validity
  - ≥ 0.7: Approved
  - < 0.7: Rejected

- **Collateral Ratio**: Asset value / Stablecoin debt
  - ≥ 2.0: Very safe
  - ≥ 1.5: Safe
  - ≥ 1.2: Minimum safe
  - < 1.0: Under-collateralized (rejected)

- **Liquidation Risk** (0-1): Risk of position liquidation
  - < 0.3: Low risk
  - 0.3-0.6: Medium risk
  - > 0.6: High risk

- **Risk Score** (0-1): Overall transaction risk

### Validation Example

```python
# Python client example
import requests

response = requests.post('http://localhost:5000/api/v1/validate/collateral', json={
    "asset_id": "ASSET001",
    "collateral_amount": 150.0,
    "stablecoin_amount": 100.0,
    "stablecoin_type": "USDC"
})

result = response.json()
print(f"Result: {result['validation_result']}")
print(f"Score: {result['compatibility_score']}")
print(f"Recommendations: {result['recommendations']}")
```

## Events

All transactions emit events for integration:

- `UserRegistered` - New user registered
- `SSMCreated` - New state machine created
- `SessionStarted` - New session started
- `SessionLimitSet` - Session iteration limit set
- `CreditsGranted` - User credits updated
- `TransitionPerformed` - State transition executed
- `TransitionPerformedWithValidation` - AI-validated transition executed

## Migration from v1

See [MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md) for detailed migration instructions.

### Key Changes

- Contract API instead of shim API
- New method names (e.g., `RegisterUser` instead of `Invoke("register")`)
- JSON responses instead of pb.Response
- Go modules for dependency management

### Compatibility

✅ **State Data**: Fully compatible
✅ **Key Prefixes**: Unchanged
⚠️ **API Methods**: Requires client updates

## Documentation

- **[MODERNIZATION_SUMMARY.md](./MODERNIZATION_SUMMARY.md)** - Changes overview
- **[SSM_MODERNIZATION_REPORT.md](./SSM_MODERNIZATION_REPORT.md)** - Detailed analysis
- **[MIGRATION_GUIDE.md](./MIGRATION_GUIDE.md)** - Migration instructions
- **[TECHNICAL_IMPLEMENTATION.md](./TECHNICAL_IMPLEMENTATION.md)** - Technical details

## Testing

### Unit Tests

```bash
# Python tests
python test_impact_validator.py

# Go tests (coming soon)
go test ./...
```

### Integration Tests

```bash
# Start validator
python validator_api.py &

# Run examples
python examples/asset_collateral_example.py
```

## Performance

- **Transaction Latency**: ~100-300ms (without AI validation)
- **AI Validation Overhead**: +50-200ms
- **Throughput**: Depends on Fabric network configuration
- **State Size**: Optimized for large-scale deployments

## Security

- **Signature Verification**: RSA-based cryptographic signatures
- **Access Control**: Role-based permissions and grants
- **Data Privacy**: Encrypted private state data
- **Audit Trail**: Complete transaction history
- **AI Validation**: Risk-based transaction filtering

## Deployment Options

### Local Development
```bash
# See deployment/local/README.md
cd deployment/local
./start.sh
```

### Production

- Use Hyperledger Fabric 2.5+ production configuration
- Deploy AI validator as separate service
- Configure TLS for all communications
- Set up monitoring and logging
- Implement backup and disaster recovery

## Troubleshooting

### Chaincode Won't Build

**Issue**: Missing dependencies
```bash
# Solution
go mod download
go mod tidy
```

### AI Validation Fails

**Issue**: Cannot connect to validator
```bash
# Solution
export VALIDATOR_URL=http://your-validator:5000
# Or check validator is running:
curl http://localhost:5000/health
```

### Transaction Rejected

**Issue**: Low compatibility score
```bash
# Check validation response for recommendations
# Adjust collateral ratio or transaction parameters
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

Apache-2.0

## Credits

- **Original SSM Concept**: Luc Yriarte <luc.yriarte@thingagora.org> (2018)
- **Modernization**: 2024
- **AI Integration**: 2024

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- See documentation in `/docs`
- Check examples in `/examples`

## Version

**Current Version**: 2.0.0
**Fabric Compatibility**: 2.5+
**Go Version**: 1.20+
