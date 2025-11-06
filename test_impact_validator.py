"""
Unit tests for Impact Validator
"""

import unittest
from impact_validator import (
    ImpactValidator,
    AssetRegistryValidator,
    TransitionContext,
    ValidationResult
)


class TestImpactValidator(unittest.TestCase):
    """Test cases for ImpactValidator"""

    def setUp(self):
        """Set up test fixtures"""
        self.validator = ImpactValidator(min_score=0.7)

    def test_valid_transition(self):
        """Test a valid transition with good parameters"""
        context = TransitionContext(
            session_id="test_session_001",
            from_state=0,
            to_state=1,
            action="create",
            role="issuer",
            asset_id="ASSET001",
            asset_type="carbon_credit",
            asset_quantity=100.0,
            collateral_amount=150.0,
            stablecoin_amount=100.0
        )

        result = self.validator.validate_transition(context)

        self.assertIsNotNone(result)
        self.assertEqual(result.validation_result, "approved")
        self.assertGreaterEqual(result.compatibility_score, 0.7)
        self.assertGreater(result.confidence, 0.0)
        self.assertLessEqual(result.confidence, 1.0)

    def test_under_collateralized_transition(self):
        """Test a transition with insufficient collateral"""
        context = TransitionContext(
            session_id="test_session_002",
            from_state=1,
            to_state=2,
            action="collateralize",
            role="holder",
            asset_id="ASSET002",
            asset_type="carbon_credit",
            asset_quantity=100.0,
            collateral_amount=80.0,  # Under-collateralized
            stablecoin_amount=100.0
        )

        result = self.validator.validate_transition(context)

        self.assertIsNotNone(result)
        # Should be rejected due to low collateral
        self.assertEqual(result.validation_result, "rejected")
        self.assertIn("liquidation_risk", result.impact_metrics)
        self.assertGreater(result.impact_metrics["liquidation_risk"], 0.6)

    def test_negative_quantity_transition(self):
        """Test a transition with negative asset quantity"""
        context = TransitionContext(
            session_id="test_session_003",
            from_state=2,
            to_state=3,
            action="transfer",
            role="holder",
            asset_id="ASSET003",
            asset_type="carbon_credit",
            asset_quantity=-50.0  # Invalid negative quantity
        )

        result = self.validator.validate_transition(context)

        self.assertIsNotNone(result)
        self.assertEqual(result.validation_result, "rejected")
        self.assertIn("Invalid asset quantity", " ".join(result.recommendations))

    def test_impact_metrics_calculation(self):
        """Test that impact metrics are properly calculated"""
        context = TransitionContext(
            session_id="test_session_004",
            from_state=0,
            to_state=5,
            action="complex_transition",
            role="validator",
            asset_quantity=500.0,
            collateral_amount=200.0,
            stablecoin_amount=100.0
        )

        result = self.validator.validate_transition(context)

        self.assertIsNotNone(result.impact_metrics)
        self.assertIn("collateral_ratio", result.impact_metrics)
        self.assertIn("liquidation_risk", result.impact_metrics)
        self.assertIn("risk_score", result.impact_metrics)
        self.assertIn("transition_complexity", result.impact_metrics)

        # Verify collateral ratio calculation
        expected_ratio = 200.0 / 100.0
        self.assertAlmostEqual(
            result.impact_metrics["collateral_ratio"],
            expected_ratio,
            places=2
        )

    def test_validation_history(self):
        """Test that validation history is properly stored"""
        # Perform multiple validations
        for i in range(3):
            context = TransitionContext(
                session_id=f"session_{i}",
                from_state=i,
                to_state=i + 1,
                action="test",
                role="tester"
            )
            self.validator.validate_transition(context)

        history = self.validator.get_validation_history()
        self.assertEqual(len(history), 3)

        # Test filtering by session
        session_history = self.validator.get_validation_history("session_1")
        self.assertEqual(len(session_history), 1)
        self.assertEqual(session_history[0].metadata["session_id"], "session_1")


class TestAssetRegistryValidator(unittest.TestCase):
    """Test cases for AssetRegistryValidator"""

    def setUp(self):
        """Set up test fixtures"""
        self.validator = AssetRegistryValidator()

    def test_asset_creation_validation(self):
        """Test asset creation validation"""
        result = self.validator.validate_asset_creation(
            asset_id="ASSET_TEST_001",
            asset_type="carbon_credit",
            quantity=250.0,
            attributes={
                "vintage": 2024,
                "methodology": "VM0042",
                "country": "Brazil"
            }
        )

        self.assertIsNotNone(result)
        self.assertEqual(result.action, "create")
        self.assertEqual(result.transition_from, 0)
        self.assertEqual(result.transition_to, 1)
        self.assertEqual(result.validation_result, "approved")

    def test_collateralization_validation_success(self):
        """Test successful collateralization validation"""
        result = self.validator.validate_collateralization(
            asset_id="ASSET_TEST_002",
            collateral_amount=180.0,
            stablecoin_amount=100.0,
            stablecoin_type="USDC"
        )

        self.assertIsNotNone(result)
        self.assertEqual(result.action, "collateralize")
        self.assertEqual(result.validation_result, "approved")
        self.assertIn("collateral_ratio", result.impact_metrics)
        self.assertGreaterEqual(result.impact_metrics["collateral_ratio"], 1.5)

    def test_collateralization_validation_failure(self):
        """Test failed collateralization due to low ratio"""
        result = self.validator.validate_collateralization(
            asset_id="ASSET_TEST_003",
            collateral_amount=90.0,  # Ratio of 0.9 - under-collateralized
            stablecoin_amount=100.0,
            stablecoin_type="USDT"
        )

        self.assertIsNotNone(result)
        self.assertEqual(result.validation_result, "rejected")
        self.assertIn("liquidation_risk", result.impact_metrics)
        # Should have high liquidation risk
        self.assertGreater(result.impact_metrics["liquidation_risk"], 0.8)

    def test_transfer_validation(self):
        """Test asset transfer validation"""
        result = self.validator.validate_transfer(
            asset_id="ASSET_TEST_004",
            from_agent="Alice",
            to_agent="Bob",
            quantity=75.0
        )

        self.assertIsNotNone(result)
        self.assertEqual(result.action, "transfer")
        self.assertEqual(result.transition_from, 2)
        self.assertEqual(result.transition_to, 3)
        self.assertIn("from", result.metadata)
        self.assertIn("to", result.metadata)

    def test_validation_recommendations(self):
        """Test that appropriate recommendations are generated"""
        # Test with good parameters
        result = self.validator.validate_collateralization(
            asset_id="ASSET_TEST_005",
            collateral_amount=200.0,
            stablecoin_amount=100.0,
            stablecoin_type="USDC"
        )

        self.assertIn("All validation checks passed successfully", result.recommendations)

        # Test with risky parameters
        result = self.validator.validate_collateralization(
            asset_id="ASSET_TEST_006",
            collateral_amount=110.0,  # Just above 1.0 ratio
            stablecoin_amount=100.0,
            stablecoin_type="USDC"
        )

        # Should have warnings
        self.assertTrue(any("risk" in rec.lower() or "collateral" in rec.lower()
                          for rec in result.recommendations))


class TestValidationResult(unittest.TestCase):
    """Test ValidationResult data structure"""

    def test_validation_result_structure(self):
        """Test that ValidationResult contains all required fields"""
        validator = ImpactValidator()
        context = TransitionContext(
            session_id="test",
            from_state=0,
            to_state=1,
            action="test",
            role="tester"
        )

        result = validator.validate_transition(context)

        # Check all required fields exist
        self.assertIsNotNone(result.validation_id)
        self.assertIsNotNone(result.timestamp)
        self.assertIsNotNone(result.transition_from)
        self.assertIsNotNone(result.transition_to)
        self.assertIsNotNone(result.action)
        self.assertIsNotNone(result.validation_result)
        self.assertIsNotNone(result.compatibility_score)
        self.assertIsNotNone(result.impact_metrics)
        self.assertIsNotNone(result.recommendations)
        self.assertIsNotNone(result.ai_model)
        self.assertIsNotNone(result.confidence)
        self.assertIsNotNone(result.metadata)

        # Check types
        self.assertIsInstance(result.validation_id, str)
        self.assertIsInstance(result.compatibility_score, float)
        self.assertIsInstance(result.impact_metrics, dict)
        self.assertIsInstance(result.recommendations, list)


def run_tests():
    """Run all tests"""
    unittest.main(argv=[''], verbosity=2, exit=False)


if __name__ == '__main__':
    run_tests()
