"""
AI Impact Validator for State Machine Transitions
Validates compatibility and impact of asset state transitions in real-time
"""

import json
import hashlib
import time
from typing import Dict, List, Optional, Tuple
from dataclasses import dataclass, asdict
from datetime import datetime


@dataclass
class TransitionContext:
    """Context information for a state transition"""
    session_id: str
    from_state: int
    to_state: int
    action: str
    role: str
    asset_id: Optional[str] = None
    asset_type: Optional[str] = None
    asset_quantity: Optional[float] = None
    collateral_amount: Optional[float] = None
    stablecoin_amount: Optional[float] = None
    public_data: Optional[Dict] = None
    private_data: Optional[Dict] = None


@dataclass
class ValidationResult:
    """Result of impact validation"""
    validation_id: str
    timestamp: str
    transition_from: int
    transition_to: int
    action: str
    validation_result: str
    compatibility_score: float
    impact_metrics: Dict[str, float]
    recommendations: List[str]
    ai_model: str
    confidence: float
    metadata: Dict


class ImpactValidator:
    """AI-driven validator for state machine transitions"""

    def __init__(self, model_name: str = "impact-validator-v1", min_score: float = 0.7):
        self.model_name = model_name
        self.min_score = min_score
        self.validation_history = []

    def validate_transition(self, context: TransitionContext) -> ValidationResult:
        """
        Validates a state transition with AI-driven impact analysis

        Args:
            context: TransitionContext with all relevant information

        Returns:
            ValidationResult with compatibility score and recommendations
        """
        # Generate validation ID
        validation_id = self._generate_validation_id(context)

        # Calculate compatibility score
        compatibility_score = self._calculate_compatibility(context)

        # Calculate impact metrics
        impact_metrics = self._calculate_impact_metrics(context)

        # Generate recommendations
        recommendations = self._generate_recommendations(context, compatibility_score, impact_metrics)

        # Determine validation result
        validation_result = "approved" if compatibility_score >= self.min_score else "rejected"

        # Calculate confidence
        confidence = self._calculate_confidence(context, impact_metrics)

        # Create validation result
        result = ValidationResult(
            validation_id=validation_id,
            timestamp=datetime.utcnow().isoformat() + "Z",
            transition_from=context.from_state,
            transition_to=context.to_state,
            action=context.action,
            validation_result=validation_result,
            compatibility_score=compatibility_score,
            impact_metrics=impact_metrics,
            recommendations=recommendations,
            ai_model=self.model_name,
            confidence=confidence,
            metadata={
                "session_id": context.session_id,
                "asset_id": context.asset_id,
                "asset_type": context.asset_type
            }
        )

        # Store in history
        self.validation_history.append(result)

        return result

    def _calculate_compatibility(self, context: TransitionContext) -> float:
        """
        Calculate compatibility score for the transition
        This is where actual AI/ML models would be integrated
        """
        score = 0.85  # Base score

        # Check asset quantity consistency
        if context.asset_quantity is not None:
            if context.asset_quantity > 0:
                score += 0.05
            else:
                score -= 0.2

        # Check collateralization ratio
        if context.collateral_amount and context.stablecoin_amount:
            ratio = context.collateral_amount / context.stablecoin_amount
            if ratio >= 1.5:  # Over-collateralized
                score += 0.1
            elif ratio >= 1.2:  # Adequately collateralized
                score += 0.05
            elif ratio < 1.0:  # Under-collateralized
                score -= 0.3

        # Ensure score is between 0 and 1
        return max(0.0, min(1.0, score))

    def _calculate_impact_metrics(self, context: TransitionContext) -> Dict[str, float]:
        """Calculate various impact metrics"""
        metrics = {}

        # Asset utilization rate
        if context.asset_quantity:
            metrics["asset_utilization"] = min(1.0, context.asset_quantity / 1000.0)

        # Collateralization ratio
        if context.collateral_amount and context.stablecoin_amount:
            metrics["collateral_ratio"] = context.collateral_amount / context.stablecoin_amount
            metrics["liquidation_risk"] = self._calculate_liquidation_risk(
                context.collateral_amount, context.stablecoin_amount
            )

        # State transition complexity
        metrics["transition_complexity"] = abs(context.to_state - context.from_state) / 10.0

        # Risk score (0-1, lower is better)
        metrics["risk_score"] = self._calculate_risk_score(context)

        return metrics

    def _calculate_liquidation_risk(self, collateral: float, debt: float) -> float:
        """Calculate liquidation risk (0-1, higher is more risk)"""
        ratio = collateral / debt if debt > 0 else 10.0

        if ratio >= 2.0:
            return 0.1  # Very low risk
        elif ratio >= 1.5:
            return 0.3  # Low risk
        elif ratio >= 1.2:
            return 0.6  # Medium risk
        elif ratio >= 1.0:
            return 0.85  # High risk
        else:
            return 1.0  # Critical risk

    def _calculate_risk_score(self, context: TransitionContext) -> float:
        """Calculate overall risk score for the transition"""
        risk = 0.2  # Base risk

        # Higher state numbers might indicate more complex states
        if context.to_state > 5:
            risk += 0.1

        # Large quantities increase risk
        if context.asset_quantity and context.asset_quantity > 1000:
            risk += 0.2

        # Under-collateralization increases risk significantly
        if context.collateral_amount and context.stablecoin_amount:
            ratio = context.collateral_amount / context.stablecoin_amount
            if ratio < 1.2:
                risk += 0.4

        return min(1.0, risk)

    def _generate_recommendations(self, context: TransitionContext,
                                 score: float, metrics: Dict[str, float]) -> List[str]:
        """Generate actionable recommendations based on validation"""
        recommendations = []

        if score < self.min_score:
            recommendations.append("Transition validation failed - score below threshold")

        if "liquidation_risk" in metrics and metrics["liquidation_risk"] > 0.6:
            recommendations.append("High liquidation risk detected - consider increasing collateral")

        if "collateral_ratio" in metrics and metrics["collateral_ratio"] < 1.2:
            recommendations.append("Collateralization ratio is low - add more collateral")

        if "risk_score" in metrics and metrics["risk_score"] > 0.7:
            recommendations.append("High risk score - review transition parameters")

        if context.asset_quantity and context.asset_quantity < 0:
            recommendations.append("Invalid asset quantity - must be positive")

        if not recommendations:
            recommendations.append("All validation checks passed successfully")

        return recommendations

    def _calculate_confidence(self, context: TransitionContext,
                             metrics: Dict[str, float]) -> float:
        """Calculate confidence level of the validation"""
        confidence = 0.9  # Base confidence

        # More data = higher confidence
        data_points = sum([
            context.asset_quantity is not None,
            context.collateral_amount is not None,
            context.public_data is not None,
            len(metrics) > 3
        ])

        confidence = 0.6 + (data_points * 0.1)

        return min(1.0, confidence)

    def _generate_validation_id(self, context: TransitionContext) -> str:
        """Generate unique validation ID"""
        data = f"{context.session_id}{context.from_state}{context.to_state}{time.time()}"
        hash_obj = hashlib.sha256(data.encode())
        return f"val_{hash_obj.hexdigest()[:16]}"

    def get_validation_history(self, session_id: Optional[str] = None) -> List[ValidationResult]:
        """Get validation history, optionally filtered by session"""
        if session_id:
            return [v for v in self.validation_history if v.metadata.get("session_id") == session_id]
        return self.validation_history

    def to_json(self, result: ValidationResult) -> str:
        """Convert validation result to JSON"""
        return json.dumps(asdict(result), indent=2)


class AssetRegistryValidator:
    """Specialized validator for asset registry operations"""

    def __init__(self):
        self.validator = ImpactValidator(model_name="asset-registry-validator-v1")

    def validate_asset_creation(self, asset_id: str, asset_type: str,
                               quantity: float, attributes: Dict) -> ValidationResult:
        """Validate asset creation"""
        context = TransitionContext(
            session_id=asset_id,
            from_state=0,
            to_state=1,
            action="create",
            role="issuer",
            asset_id=asset_id,
            asset_type=asset_type,
            asset_quantity=quantity,
            public_data=attributes
        )
        return self.validator.validate_transition(context)

    def validate_collateralization(self, asset_id: str, collateral_amount: float,
                                   stablecoin_amount: float, stablecoin_type: str) -> ValidationResult:
        """Validate collateralization of asset"""
        context = TransitionContext(
            session_id=asset_id,
            from_state=1,
            to_state=2,
            action="collateralize",
            role="holder",
            asset_id=asset_id,
            collateral_amount=collateral_amount,
            stablecoin_amount=stablecoin_amount,
            public_data={"stablecoin_type": stablecoin_type}
        )
        return self.validator.validate_transition(context)

    def validate_transfer(self, asset_id: str, from_agent: str,
                         to_agent: str, quantity: float) -> ValidationResult:
        """Validate asset transfer"""
        context = TransitionContext(
            session_id=asset_id,
            from_state=2,
            to_state=3,
            action="transfer",
            role="holder",
            asset_id=asset_id,
            asset_quantity=quantity,
            public_data={"from": from_agent, "to": to_agent}
        )
        return self.validator.validate_transition(context)


def main():
    """Example usage"""
    validator = AssetRegistryValidator()

    # Validate asset creation
    result = validator.validate_asset_creation(
        asset_id="ASSET001",
        asset_type="carbon_credit",
        quantity=100.0,
        attributes={"vintage": 2024, "methodology": "VM0042"}
    )

    print("Asset Creation Validation:")
    print(validator.validator.to_json(result))
    print()

    # Validate collateralization
    result = validator.validate_collateralization(
        asset_id="ASSET001",
        collateral_amount=150.0,
        stablecoin_amount=100.0,
        stablecoin_type="USDC"
    )

    print("Collateralization Validation:")
    print(validator.validator.to_json(result))
    print()

    print(f"Validation Result: {result.validation_result}")
    print(f"Compatibility Score: {result.compatibility_score:.2f}")
    print(f"Recommendations: {result.recommendations}")


if __name__ == "__main__":
    main()
