"""
REST API for AI Impact Validator
Provides HTTP endpoints for the chaincode to validate transitions
"""

from flask import Flask, request, jsonify
from flask_cors import CORS
from impact_validator import ImpactValidator, AssetRegistryValidator, TransitionContext, ValidationResult
from dataclasses import asdict
from typing import Dict, Optional
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize Flask app
app = Flask(__name__)
CORS(app)

# Initialize validators
impact_validator = ImpactValidator(min_score=0.7)
asset_validator = AssetRegistryValidator()


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    return jsonify({
        "status": "healthy",
        "service": "impact-validator-api",
        "version": "1.0.0"
    })


@app.route('/api/v1/validate/transition', methods=['POST'])
def validate_transition():
    """
    Validate a state machine transition

    Request body:
    {
        "session_id": "string",
        "from_state": int,
        "to_state": int,
        "action": "string",
        "role": "string",
        "asset_id": "string (optional)",
        "asset_type": "string (optional)",
        "asset_quantity": float (optional),
        "collateral_amount": float (optional),
        "stablecoin_amount": float (optional),
        "public_data": {} (optional),
        "private_data": {} (optional)
    }
    """
    try:
        data = request.get_json()

        # Validate required fields
        required_fields = ["session_id", "from_state", "to_state", "action", "role"]
        for field in required_fields:
            if field not in data:
                return jsonify({
                    "error": f"Missing required field: {field}"
                }), 400

        # Create transition context
        context = TransitionContext(
            session_id=data["session_id"],
            from_state=data["from_state"],
            to_state=data["to_state"],
            action=data["action"],
            role=data["role"],
            asset_id=data.get("asset_id"),
            asset_type=data.get("asset_type"),
            asset_quantity=data.get("asset_quantity"),
            collateral_amount=data.get("collateral_amount"),
            stablecoin_amount=data.get("stablecoin_amount"),
            public_data=data.get("public_data"),
            private_data=data.get("private_data")
        )

        # Validate transition
        result = impact_validator.validate_transition(context)

        logger.info(f"Validated transition for session {data['session_id']}: {result.validation_result}")

        return jsonify(asdict(result)), 200

    except Exception as e:
        logger.error(f"Error validating transition: {str(e)}")
        return jsonify({
            "error": "Internal server error",
            "message": str(e)
        }), 500


@app.route('/api/v1/validate/asset/create', methods=['POST'])
def validate_asset_creation():
    """
    Validate asset creation

    Request body:
    {
        "asset_id": "string",
        "asset_type": "string",
        "quantity": float,
        "attributes": {}
    }
    """
    try:
        data = request.get_json()

        required_fields = ["asset_id", "asset_type", "quantity"]
        for field in required_fields:
            if field not in data:
                return jsonify({
                    "error": f"Missing required field: {field}"
                }), 400

        result = asset_validator.validate_asset_creation(
            asset_id=data["asset_id"],
            asset_type=data["asset_type"],
            quantity=data["quantity"],
            attributes=data.get("attributes", {})
        )

        logger.info(f"Validated asset creation for {data['asset_id']}: {result.validation_result}")

        return jsonify(asdict(result)), 200

    except Exception as e:
        logger.error(f"Error validating asset creation: {str(e)}")
        return jsonify({
            "error": "Internal server error",
            "message": str(e)
        }), 500


@app.route('/api/v1/validate/collateral', methods=['POST'])
def validate_collateralization():
    """
    Validate collateralization

    Request body:
    {
        "asset_id": "string",
        "collateral_amount": float,
        "stablecoin_amount": float,
        "stablecoin_type": "string"
    }
    """
    try:
        data = request.get_json()

        required_fields = ["asset_id", "collateral_amount", "stablecoin_amount", "stablecoin_type"]
        for field in required_fields:
            if field not in data:
                return jsonify({
                    "error": f"Missing required field: {field}"
                }), 400

        result = asset_validator.validate_collateralization(
            asset_id=data["asset_id"],
            collateral_amount=data["collateral_amount"],
            stablecoin_amount=data["stablecoin_amount"],
            stablecoin_type=data["stablecoin_type"]
        )

        logger.info(f"Validated collateralization for {data['asset_id']}: {result.validation_result}")

        return jsonify(asdict(result)), 200

    except Exception as e:
        logger.error(f"Error validating collateralization: {str(e)}")
        return jsonify({
            "error": "Internal server error",
            "message": str(e)
        }), 500


@app.route('/api/v1/validate/transfer', methods=['POST'])
def validate_transfer():
    """
    Validate asset transfer

    Request body:
    {
        "asset_id": "string",
        "from_agent": "string",
        "to_agent": "string",
        "quantity": float
    }
    """
    try:
        data = request.get_json()

        required_fields = ["asset_id", "from_agent", "to_agent", "quantity"]
        for field in required_fields:
            if field not in data:
                return jsonify({
                    "error": f"Missing required field: {field}"
                }), 400

        result = asset_validator.validate_transfer(
            asset_id=data["asset_id"],
            from_agent=data["from_agent"],
            to_agent=data["to_agent"],
            quantity=data["quantity"]
        )

        logger.info(f"Validated transfer for {data['asset_id']}: {result.validation_result}")

        return jsonify(asdict(result)), 200

    except Exception as e:
        logger.error(f"Error validating transfer: {str(e)}")
        return jsonify({
            "error": "Internal server error",
            "message": str(e)
        }), 500


@app.route('/api/v1/history/<session_id>', methods=['GET'])
def get_validation_history(session_id: str):
    """Get validation history for a session"""
    try:
        history = impact_validator.get_validation_history(session_id)
        return jsonify({
            "session_id": session_id,
            "count": len(history),
            "validations": [asdict(v) for v in history]
        }), 200

    except Exception as e:
        logger.error(f"Error retrieving history: {str(e)}")
        return jsonify({
            "error": "Internal server error",
            "message": str(e)
        }), 500


@app.route('/api/v1/history', methods=['GET'])
def get_all_history():
    """Get all validation history"""
    try:
        history = impact_validator.get_validation_history()
        return jsonify({
            "count": len(history),
            "validations": [asdict(v) for v in history]
        }), 200

    except Exception as e:
        logger.error(f"Error retrieving history: {str(e)}")
        return jsonify({
            "error": "Internal server error",
            "message": str(e)
        }), 500


@app.route('/api/v1/metrics', methods=['GET'])
def get_metrics():
    """Get validator metrics"""
    try:
        history = impact_validator.get_validation_history()

        total_validations = len(history)
        approved = sum(1 for v in history if v.validation_result == "approved")
        rejected = total_validations - approved

        avg_score = sum(v.compatibility_score for v in history) / total_validations if total_validations > 0 else 0
        avg_confidence = sum(v.confidence for v in history) / total_validations if total_validations > 0 else 0

        return jsonify({
            "total_validations": total_validations,
            "approved": approved,
            "rejected": rejected,
            "approval_rate": approved / total_validations if total_validations > 0 else 0,
            "average_compatibility_score": avg_score,
            "average_confidence": avg_confidence
        }), 200

    except Exception as e:
        logger.error(f"Error retrieving metrics: {str(e)}")
        return jsonify({
            "error": "Internal server error",
            "message": str(e)
        }), 500


if __name__ == '__main__':
    logger.info("Starting Impact Validator API on port 5000")
    app.run(host='0.0.0.0', port=5000, debug=True)
