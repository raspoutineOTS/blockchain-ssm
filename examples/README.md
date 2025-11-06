# Asset Collateralization Examples

This directory contains examples demonstrating the AI-validated asset collateralization system.

## Running the Examples

### Prerequisites

1. **Install Python dependencies**:
   ```bash
   pip install -r ../requirements.txt
   ```

2. **Start the validator API**:
   ```bash
   cd ..
   python validator_api.py
   ```
   The API will start on `http://localhost:5000`

3. **Run the examples** (in a separate terminal):
   ```bash
   python asset_collateral_example.py
   ```

## Example Scenarios

### Scenario 1: Healthy Over-Collateralized Position
- Creates a carbon credit asset (1000 tCO2e)
- Collateralizes at 180% ratio
- Mints stablecoins backed by the asset
- Transfers partial asset ownership
- **Expected Result**: All validations pass ✓

### Scenario 2: Risky Under-Collateralized Position
- Creates a carbon credit asset (500 tCO2e)
- Attempts to collateralize at only 95% ratio
- **Expected Result**: Validation fails with recommendations ✗

### Scenario 3: Edge Case - Minimum Threshold
- Creates a carbon credit asset (200 tCO2e)
- Collateralizes at exactly 120% (minimum safe ratio)
- **Expected Result**: Passes with warnings ⚠

## Understanding the Output

### Validation Results
```json
{
  "validation_result": "approved",
  "compatibility_score": 0.95,
  "confidence": 0.9,
  "impact_metrics": {
    "collateral_ratio": 1.8,
    "liquidation_risk": 0.1,
    "risk_score": 0.2
  },
  "recommendations": [
    "All validation checks passed successfully"
  ]
}
```

### Key Metrics

- **Compatibility Score** (0-1): Overall validation score
  - >= 0.7: Approved
  - < 0.7: Rejected

- **Collateral Ratio**: Asset value / Stablecoin debt
  - >= 2.0: Very safe
  - >= 1.5: Safe
  - >= 1.2: Minimum safe
  - < 1.0: Undercollateralized (rejected)

- **Liquidation Risk** (0-1): Risk of liquidation
  - < 0.3: Low risk
  - 0.3-0.6: Medium risk
  - > 0.6: High risk

- **Confidence** (0-1): AI confidence in validation
  - Higher is better

## Customization

### Create Your Own Scenario

```python
from asset_collateral_example import AssetCollateralExample

example = AssetCollateralExample()

# Create custom asset
example.step1_create_asset(
    asset_id="MY_ASSET_001",
    quantity=100.0,
    attributes={
        "type": "renewable_energy_certificate",
        "location": "California"
    }
)

# Collateralize
example.step2_collateralize_asset(
    asset_id="MY_ASSET_001",
    collateral_amount=150.0,
    stablecoin_amount=100.0,
    stablecoin_type="USDC"
)
```

## Integration with Blockchain

These examples demonstrate the validator API independently. To integrate with the Hyperledger Fabric chaincode:

1. Deploy the chaincode with validator integration
2. Configure the validator URL in the chaincode
3. Use the `ExtendedState` type in Go
4. Call `PerformWithValidation()` for transitions

See `TECHNICAL_IMPLEMENTATION.md` for details.
