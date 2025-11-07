# Blockchain-SSM Test Suite

Comprehensive test suite for blockchain-ssm including ITMO Article 6 and DeFi functionality.

## 📋 Test Coverage

### Integration Tests

**ITMO Tests** (`integration/itmo_integration_test.js`):
- State record lifecycle
- State transition validation
- Article 6 compliance
- Registry reference validation
- Collateralization logic
- State history tracking
- Error handling

**DeFi Tests** (`integration/defi_integration_test.js`):
- Price oracle (aggregation, circuit breaker, volatility)
- Lending protocol (interest rates, supply/withdraw)
- Borrowing & liquidation (LTV, health factor, liquidations)
- Liquidity pools (AMM, constant product, LP tokens, fees)
- Complete workflows

### Unit Tests

**Go Unit Tests** (`../chaincode/go/ssm/defi_test.go`):
- Price aggregation algorithms
- Interest rate calculations
- Health factor computation
- Liquidation price calculation
- AMM pricing formulas
- Circuit breaker logic
- Benchmarks

## 🚀 Quick Start

### Prerequisites

- Node.js 16+
- Mocha test framework
- Chai assertion library

### Installation

```bash
cd test
npm install
```

### Run All Tests

```bash
npm test
```

### Run Specific Test Suites

```bash
# ITMO tests only
npm run test:itmo

# DeFi tests only
npm run test:defi

# With coverage
npm run test:coverage

# Watch mode
npm run test:watch
```

## 📊 Test Structure

```
test/
├── integration/
│   ├── itmo_integration_test.js    # ITMO functionality tests
│   └── defi_integration_test.js    # DeFi functionality tests
├── package.json                     # Test dependencies
└── README.md                        # This file

../chaincode/go/ssm/
└── defi_test.go                    # Go unit tests
```

## 🧪 Test Categories

### 1. ITMO Article 6 Tests

#### State Record Lifecycle
- Create ITMO state record
- Transition states (issued → authorized → transferred → held → collateral)
- Track corresponding adjustments
- Tokenize states
- Collateralize states

#### State Transition Validation
- Valid transitions
- Invalid transitions
- Transition logic

#### Article 6 Compliance
- Article 6.2 requirements
- Corresponding adjustment completeness
- Double counting prevention

#### Registry Reference Validation
- ITMO reference structure
- Registry URL format
- Serial number validation

#### Collateralization Logic
- Health factor calculation
- Liquidatable position identification
- Liquidation price calculation

### 2. DeFi Tests

#### Price Oracle
- Median price aggregation
- Weighted average calculation
- Mean calculation
- Inactive source filtering
- Circuit breaker triggering
- Volatility calculation (standard deviation)

#### Lending Protocol
- Interest rate model (low/high utilization)
- Supply APY calculation
- LP shares calculation
- Interest accrual

#### Borrowing & Liquidation
- LTV calculations
- Max LTV enforcement
- Health factor tracking
- Price change effects
- Liquidation amounts
- Protocol protection

#### Liquidity Pools (AMM)
- Constant product formula
- Swap output calculation
- Price impact
- LP token calculations (first & subsequent deposits)
- Withdrawal amounts
- Fee distribution
- Slippage protection

#### Complete Workflows
- Full lending workflow (supply → borrow → accrue → repay)
- Full AMM workflow (add → swap → remove)

## 📈 Test Scenarios

### ITMO Scenario Example

```javascript
1. Create state record for BRA-2024-001-0001 (1000 tCO2e)
2. Authorize for international transfer
3. Track CA: Brazil → Switzerland
4. Transfer with CA completion
5. Hold state
6. Tokenize (1000 tokens)
7. Collateralize ($10,000 USDC @ 150% ratio)
8. Verify health factor: 1.5 ✓
```

### DeFi Scenario Example

```javascript
1. Create price oracle (3 sources)
2. Create lending pool (80% optimal utilization)
3. Supply 500 tCO2e
4. Borrow 300 tCO2e (60% utilization)
5. Price crash: $15 → $9
6. Health factor drops < 1.0
7. Liquidate position
8. Protocol protected ✓
```

## ✅ Test Assertions

### ITMO Tests

```javascript
// State transition
expect(transition.toState).to.equal('authorized');

// Health factor
expect(healthFactor).to.be.closeTo(1.7, 0.01);

// Article 6 compliance
expect(ca.transferringPartyCA).to.be.true;
expect(ca.acquiringPartyCA).to.be.true;

// Liquidation
const isLiquidatable = healthFactor < 1.0;
expect(isLiquidatable).to.be.true;
```

### DeFi Tests

```javascript
// Interest rates
expect(borrowAPY).to.be.closeTo(0.05125, 0.0001);

// AMM swap
expect(amountOut).to.be.closeTo(90.67, 0.1);

// Price impact
expect(priceImpact).to.be.closeTo(9.067, 0.1);

// Circuit breaker
expect(shouldTrigger).to.be.true; // 40% change
```

## 🔧 Configuration

### Test Timeout

Default: 30 seconds per test

```javascript
this.timeout(30000);
```

### Mocha Options

```json
{
  "timeout": 30000,
  "slow": 5000,
  "require": ["chai"],
  "reporter": "spec"
}
```

## 📝 Writing New Tests

### Template for ITMO Test

```javascript
describe('New ITMO Feature', function() {
    it('should perform expected behavior', async function() {
        // Arrange
        const stateRecord = {
            stateRecordId: 'STATE_TEST',
            // ...
        };

        // Act
        const result = performOperation(stateRecord);

        // Assert
        expect(result).to.have.property('success');
        expect(result.success).to.be.true;
    });
});
```

### Template for DeFi Test

```javascript
describe('New DeFi Feature', function() {
    it('should calculate correctly', function() {
        // Arrange
        const input = 100;
        const expected = 105;

        // Act
        const result = calculate(input);

        // Assert
        expect(result).to.be.closeTo(expected, 0.01);
    });
});
```

## 🐛 Debugging Tests

### Run Single Test

```bash
mocha integration/itmo_integration_test.js --grep "should create ITMO state record"
```

### Enable Verbose Logging

```bash
DEBUG=* npm test
```

### Check Test Coverage

```bash
npm run test:coverage
```

Coverage report will be in `coverage/` directory.

## 📊 Expected Results

All tests should pass:

```
ITMO Integration Tests
  State Record Lifecycle
    ✓ should create ITMO state record
    ✓ should transition state: issued → authorized
    ✓ should track corresponding adjustment
    ✓ should tokenize state
    ✓ should collateralize state
  ...

DeFi Integration Tests
  Price Oracle Tests
    ✓ should calculate median price correctly
    ✓ should trigger on extreme price changes
  Lending Protocol Tests
    ✓ should calculate rates at low utilization
  ...

85 passing (2.5s)
```

## 🚨 Continuous Integration

### GitHub Actions

Tests run automatically on:
- Every push to any branch
- Every pull request
- Scheduled daily runs

See `.github/workflows/test.yml` for configuration.

## 📚 Resources

- [Mocha Documentation](https://mochajs.org/)
- [Chai Assertion Library](https://www.chaijs.com/)
- [Hyperledger Fabric Testing](https://hyperledger-fabric.readthedocs.io/en/latest/test_network.html)

## 🤝 Contributing

When adding new features:

1. Write tests FIRST (TDD)
2. Ensure all existing tests pass
3. Add integration tests for workflows
4. Update this README with new test descriptions
5. Maintain >80% code coverage

## 📄 License

Apache-2.0
