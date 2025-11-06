#!/usr/bin/env python3
"""
Example: Asset-backed Stablecoin Flow
Demonstrates the complete lifecycle of using assets as collateral for stablecoins
"""

import requests
import json
import time


class AssetCollateralExample:
    """Example implementation of asset-backed stablecoin operations"""

    def __init__(self, validator_url="http://localhost:5000"):
        self.validator_url = validator_url
        self.session = requests.Session()

    def check_health(self):
        """Verify validator service is running"""
        response = self.session.get(f"{self.validator_url}/health")
        print(f"✓ Validator health: {response.json()}")
        return response.status_code == 200

    def step1_create_asset(self, asset_id, quantity, attributes):
        """Step 1: Create and register an asset"""
        print(f"\n{'='*60}")
        print(f"STEP 1: Creating Asset {asset_id}")
        print(f"{'='*60}")

        payload = {
            "asset_id": asset_id,
            "asset_type": "carbon_credit",
            "quantity": quantity,
            "attributes": attributes
        }

        response = self.session.post(
            f"{self.validator_url}/api/v1/validate/asset/create",
            json=payload
        )

        result = response.json()
        print(f"Validation Result: {result['validation_result']}")
        print(f"Compatibility Score: {result['compatibility_score']:.2f}")
        print(f"Confidence: {result['confidence']:.2f}")
        print(f"Recommendations: {result['recommendations']}")

        return result

    def step2_collateralize_asset(self, asset_id, collateral_amount,
                                   stablecoin_amount, stablecoin_type):
        """Step 2: Use asset as collateral for stablecoins"""
        print(f"\n{'='*60}")
        print(f"STEP 2: Collateralizing Asset {asset_id}")
        print(f"{'='*60}")

        ratio = collateral_amount / stablecoin_amount
        print(f"Collateral Amount: ${collateral_amount:.2f}")
        print(f"Stablecoin Amount: ${stablecoin_amount:.2f}")
        print(f"Collateralization Ratio: {ratio:.2f}x")

        payload = {
            "asset_id": asset_id,
            "collateral_amount": collateral_amount,
            "stablecoin_amount": stablecoin_amount,
            "stablecoin_type": stablecoin_type
        }

        response = self.session.post(
            f"{self.validator_url}/api/v1/validate/collateral",
            json=payload
        )

        result = response.json()
        print(f"\nValidation Result: {result['validation_result']}")
        print(f"Compatibility Score: {result['compatibility_score']:.2f}")

        if "impact_metrics" in result:
            metrics = result["impact_metrics"]
            if "collateral_ratio" in metrics:
                print(f"Verified Collateral Ratio: {metrics['collateral_ratio']:.2f}x")
            if "liquidation_risk" in metrics:
                risk = metrics["liquidation_risk"]
                risk_level = "HIGH" if risk > 0.6 else "MEDIUM" if risk > 0.3 else "LOW"
                print(f"Liquidation Risk: {risk:.2f} ({risk_level})")

        print(f"Recommendations: {result['recommendations']}")

        return result

    def step3_transfer_asset(self, asset_id, from_agent, to_agent, quantity):
        """Step 3: Transfer asset to another party"""
        print(f"\n{'='*60}")
        print(f"STEP 3: Transferring Asset {asset_id}")
        print(f"{'='*60}")

        print(f"From: {from_agent}")
        print(f"To: {to_agent}")
        print(f"Quantity: {quantity}")

        payload = {
            "asset_id": asset_id,
            "from_agent": from_agent,
            "to_agent": to_agent,
            "quantity": quantity
        }

        response = self.session.post(
            f"{self.validator_url}/api/v1/validate/transfer",
            json=payload
        )

        result = response.json()
        print(f"\nValidation Result: {result['validation_result']}")
        print(f"Compatibility Score: {result['compatibility_score']:.2f}")
        print(f"Recommendations: {result['recommendations']}")

        return result

    def get_metrics(self):
        """Get overall validator metrics"""
        response = self.session.get(f"{self.validator_url}/api/v1/metrics")
        return response.json()


def scenario_healthy_collateralization():
    """Scenario 1: Healthy over-collateralized position"""
    print("\n" + "="*70)
    print("SCENARIO 1: Healthy Over-Collateralized Position")
    print("="*70)

    example = AssetCollateralExample()

    if not example.check_health():
        print("❌ Validator service not available")
        return

    # Step 1: Create carbon credit asset
    example.step1_create_asset(
        asset_id="CC_2024_001",
        quantity=1000.0,  # 1000 tCO2e
        attributes={
            "vintage": 2024,
            "methodology": "VM0042",
            "project": "Amazon Rainforest Conservation",
            "country": "Brazil",
            "verification": "Verra"
        }
    )

    time.sleep(0.5)

    # Step 2: Collateralize at 180% ratio
    # Assuming carbon credits valued at $15 per ton
    asset_value = 1000.0 * 15  # $15,000
    collateral_amount = asset_value
    stablecoin_amount = asset_value / 1.8  # 180% collateralization

    example.step2_collateralize_asset(
        asset_id="CC_2024_001",
        collateral_amount=collateral_amount,
        stablecoin_amount=stablecoin_amount,
        stablecoin_type="USDC"
    )

    time.sleep(0.5)

    # Step 3: Transfer to another party
    example.step3_transfer_asset(
        asset_id="CC_2024_001",
        from_agent="Alice",
        to_agent="Bob",
        quantity=500.0
    )

    # Show metrics
    print(f"\n{'='*60}")
    print("VALIDATOR METRICS")
    print(f"{'='*60}")
    metrics = example.get_metrics()
    print(json.dumps(metrics, indent=2))


def scenario_risky_collateralization():
    """Scenario 2: Risky under-collateralized position"""
    print("\n" + "="*70)
    print("SCENARIO 2: Risky Under-Collateralized Position (Should Fail)")
    print("="*70)

    example = AssetCollateralExample()

    if not example.check_health():
        print("❌ Validator service not available")
        return

    # Create asset
    example.step1_create_asset(
        asset_id="CC_2024_002",
        quantity=500.0,
        attributes={
            "vintage": 2023,
            "methodology": "VM0036",
            "project": "Renewable Energy",
            "country": "India"
        }
    )

    time.sleep(0.5)

    # Attempt to collateralize at only 95% ratio (should fail)
    asset_value = 500.0 * 12  # $6,000
    collateral_amount = asset_value
    stablecoin_amount = asset_value / 0.95  # Only 95% collateralization

    result = example.step2_collateralize_asset(
        asset_id="CC_2024_002",
        collateral_amount=collateral_amount,
        stablecoin_amount=stablecoin_amount,
        stablecoin_type="USDT"
    )

    if result['validation_result'] == 'rejected':
        print("\n✓ Position correctly rejected due to insufficient collateral")


def scenario_edge_case_collateralization():
    """Scenario 3: Edge case at minimum threshold"""
    print("\n" + "="*70)
    print("SCENARIO 3: Edge Case - Minimum Collateralization Threshold")
    print("="*70)

    example = AssetCollateralExample()

    if not example.check_health():
        print("❌ Validator service not available")
        return

    # Create asset
    example.step1_create_asset(
        asset_id="CC_2024_003",
        quantity=200.0,
        attributes={
            "vintage": 2024,
            "methodology": "VM0015"
        }
    )

    time.sleep(0.5)

    # Collateralize at exactly 120% (minimum safe threshold)
    asset_value = 200.0 * 18  # $3,600
    collateral_amount = asset_value
    stablecoin_amount = asset_value / 1.2  # Exactly 120%

    result = example.step2_collateralize_asset(
        asset_id="CC_2024_003",
        collateral_amount=collateral_amount,
        stablecoin_amount=stablecoin_amount,
        stablecoin_type="DAI"
    )

    print("\n✓ Position at minimum threshold - watch for warnings")


def main():
    """Run all scenarios"""
    print("\n" + "="*70)
    print("ASSET-BACKED STABLECOIN EXAMPLES")
    print("Demonstrating AI-Validated Asset Collateralization")
    print("="*70)

    try:
        # Run scenarios
        scenario_healthy_collateralization()
        time.sleep(1)

        scenario_risky_collateralization()
        time.sleep(1)

        scenario_edge_case_collateralization()

        print("\n" + "="*70)
        print("ALL SCENARIOS COMPLETED")
        print("="*70)

    except requests.exceptions.ConnectionError:
        print("\n❌ ERROR: Cannot connect to validator API")
        print("Please start the validator API first:")
        print("  python validator_api.py")
    except Exception as e:
        print(f"\n❌ ERROR: {str(e)}")


if __name__ == "__main__":
    main()
