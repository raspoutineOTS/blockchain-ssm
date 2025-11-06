# Migration Guide: SSM Chaincode v1 to v2

## Overview

This guide helps you migrate from the legacy SSM chaincode (2018) to the modernized version compatible with Hyperledger Fabric 2.5+.

## What Changed

### Architecture Changes

#### 1. API Framework
**Before (v1)**:
```go
import "github.com/hyperledger/fabric/core/chaincode/shim"

func (s *SSMChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
    return shim.Success(nil)
}

func (s *SSMChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
    function, args := stub.GetFunctionAndParameters()
    // Manual routing...
}
```

**After (v2)**:
```go
import "github.com/hyperledger/fabric-contract-api-go/contractapi"

type SSMContract struct {
    contractapi.Contract
}

func (s *SSMContract) RegisterUser(ctx contractapi.TransactionContextInterface,
    userJSON, adminName, signature string) error {
    // Implementation
}
```

#### 2. Response Handling
**Before (v1)**:
- Returns `pb.Response`
- Uses `shim.Success()` and `shim.Error()`

**After (v2)**:
- Returns `(interface{}, error)` or just `error`
- Automatic JSON serialization
- Better error handling

#### 3. Module Management
**Before (v1)**:
- No `go.mod` file
- Manual dependency management

**After (v2)**:
- Proper `go.mod` with versioned dependencies
- Go 1.20+ support

### Function Name Changes

| v1 Function | v2 Function | Notes |
|-------------|-------------|-------|
| `Init` | `InitLedger` | Now part of contract |
| `Invoke` (with "register") | `RegisterUser` | Direct method call |
| `Invoke` (with "create") | `CreateSSM` | Direct method call |
| `Invoke` (with "start") | `StartSession` | Direct method call |
| `Invoke` (with "limit") | `SetSessionLimit` | Direct method call |
| `Invoke` (with "grant") | `GrantCredits` | Direct method call |
| `Invoke` (with "perform") | `PerformTransition` | Direct method call |
| `Invoke` (with "session") | `GetSession` | Direct method call |
| `Invoke` (with "ssm") | `GetSSM` | Direct method call |
| `Invoke` (with "user") | `GetUser` | Direct method call |
| `Invoke` (with "admin") | `GetAdmin` | Direct method call |
| `Invoke` (with "credits") | `GetCredits` | Direct method call |
| `Invoke` (with "list") | `ListByType` | Direct method call |
| `Invoke` (with "log") | `GetSessionHistory` | Direct method call |

### New Features in v2

1. **AI-Validated Transitions**
   ```go
   PerformTransitionWithValidation(ctx, action, stateJSON, userName, signature)
   ```
   - Integrates with AI validator service
   - Real-time impact assessment
   - Compatibility scoring

2. **Asset Management**
   - Extended state model with asset metadata
   - Collateral position tracking
   - Stablecoin integration

3. **Event Emission**
   - All transactions now emit events
   - Better integration with event-driven architectures

4. **Better Error Messages**
   - Structured errors with context
   - Wrapped errors for debugging

5. **History with Timestamps**
   - Session history includes timestamps
   - Better audit trail

## Migration Steps

### Step 1: Update Dependencies

Create or update `go.mod`:
```bash
cd chaincode/go/ssm
go mod init github.com/your-org/blockchain-ssm/chaincode/go/ssm
go get github.com/hyperledger/fabric-contract-api-go@v1.2.1
go mod tidy
```

### Step 2: Choose Migration Path

#### Option A: Side-by-side Deployment (Recommended)
- Deploy v2 as a new chaincode
- Gradually migrate sessions from v1 to v2
- Keep v1 running for existing sessions

#### Option B: In-place Upgrade
- Stop v1 chaincode
- Deploy v2 with same chaincode ID
- All existing state data remains compatible

### Step 3: Update Client Applications

**Before (v1)**: Invoke with function name as argument
```javascript
// v1 client code
await contract.submitTransaction('perform', action, stateJSON, userName, signature);
```

**After (v2)**: Direct method invocation
```javascript
// v2 client code
await contract.submitTransaction('PerformTransition', action, stateJSON, userName, signature);
```

### Step 4: Update Integration Scripts

If you have scripts that invoke the chaincode:

**Before (v1)**:
```bash
peer chaincode invoke -n ssm -c '{"Args":["register", "{json}", "admin1", "sig"]}'
```

**After (v2)**:
```bash
peer chaincode invoke -n ssm -c '{"Args":["SSMContract:RegisterUser", "{json}", "admin1", "sig"]}'
```

## Backwards Compatibility

### State Data
✅ **Fully Compatible** - All existing state data works with v2
- Agent data
- SSM definitions
- State/Session data
- Grants

### API Signatures
⚠️ **Mostly Compatible** - Function signatures are similar but not identical
- Same parameters, different method names
- Response format is JSON (not pb.Response)

### Key Prefixes
✅ **Fully Compatible** - Same key prefixes used
- `ADMIN_`, `USER_`, `GRANT_`, `SSM_`, `STATE_`

## Testing the Migration

### 1. Deploy v2 Chaincode
```bash
# Package the chaincode
peer lifecycle chaincode package ssm_v2.tar.gz \
  --path ./chaincode/go/ssm \
  --lang golang \
  --label ssm_v2

# Install on peers
peer lifecycle chaincode install ssm_v2.tar.gz

# Approve and commit
# ... (follow your normal chaincode deployment process)
```

### 2. Test Queries (Read-only)
```bash
# Should work with existing data
peer chaincode query -n ssm_v2 -c '{"Args":["SSMContract:GetUser", "alice"]}'
peer chaincode query -n ssm_v2 -c '{"Args":["SSMContract:ListByType", "user"]}'
```

### 3. Test Transactions
```bash
# Register a new user
peer chaincode invoke -n ssm_v2 -c '{"Args":[
  "SSMContract:RegisterUser",
  "{\"name\":\"testuser\",\"pub\":\"...\"}",
  "admin1",
  "signature..."
]}'
```

## Rollback Plan

If you need to rollback to v1:

1. **Stop v2 chaincode** (if in-place upgrade)
2. **Redeploy v1 chaincode** with original code
3. **Update client applications** to use v1 API

**Note**: If you used v2-specific features (AI validation, events), that data will remain in the ledger but won't affect v1 operations.

## Common Issues

### Issue 1: Import Errors
**Error**: `cannot find package "github.com/hyperledger/fabric/core/chaincode/shim"`

**Solution**: Update imports to use Contract API:
```go
import "github.com/hyperledger/fabric-contract-api-go/contractapi"
```

### Issue 2: Method Not Found
**Error**: `could not find chaincode method 'register'`

**Solution**: Use new method names with contract prefix:
```
"register" -> "SSMContract:RegisterUser"
```

### Issue 3: Response Format Mismatch
**Error**: Client expects `pb.Response` but gets JSON

**Solution**: Update client code to parse JSON responses instead of protobuf.

## Performance Considerations

### v2 Improvements
- **Faster routing**: Direct method calls (no string matching)
- **Better caching**: Contract API has built-in optimizations
- **Event efficiency**: Native event support

### v2 Overhead
- **AI validation**: Adds 50-200ms per validated transition (optional feature)
- **JSON serialization**: Slightly more overhead than protobuf (negligible)

## Feature Comparison

| Feature | v1 | v2 |
|---------|----|----|
| Basic SSM operations | ✅ | ✅ |
| Agent management | ✅ | ✅ |
| Signature verification | ✅ | ✅ |
| Grant system | ✅ | ✅ |
| Session history | ✅ | ✅ (with timestamps) |
| AI validation | ❌ | ✅ |
| Asset management | ❌ | ✅ |
| Collateral tracking | ❌ | ✅ |
| Event emission | ❌ | ✅ |
| Rich queries | ❌ | ✅ (future) |
| Pagination | ❌ | ✅ (future) |

## Support

For issues or questions:
1. Check [TECHNICAL_IMPLEMENTATION.md](./TECHNICAL_IMPLEMENTATION.md)
2. Review [SSM_MODERNIZATION_REPORT.md](./SSM_MODERNIZATION_REPORT.md)
3. Open an issue on GitHub

## Next Steps

After successful migration:
1. ✅ Monitor chaincode performance
2. ✅ Update documentation for users
3. ✅ Consider enabling AI validation
4. ✅ Implement rich queries (if needed)
5. ✅ Add pagination for large result sets

## Version Compatibility Matrix

| SSM Version | Fabric Version | Go Version | Status |
|-------------|----------------|------------|--------|
| v1 (2018) | 1.0 - 1.4 | 1.10+ | Legacy |
| v2 (2024) | 2.2+ | 1.19+ | Current |
| v2 (2024) | 2.5+ | 1.20+ | Recommended |
