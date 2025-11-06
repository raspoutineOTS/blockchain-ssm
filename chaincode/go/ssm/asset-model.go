// Copyright Luc Yriarte <luc.yriarte@thingagora.org> 2018
// License: Apache-2.0

package main

import "time"

// AssetMetadata represents generic asset metadata that can be attached to a state
type AssetMetadata struct {
	AssetID          string                 `json:"assetId"`
	AssetType        string                 `json:"assetType"`
	Quantity         float64                `json:"quantity"`
	Unit             string                 `json:"unit"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	VerificationData *VerificationData      `json:"verificationData,omitempty"`
	Timestamp        string                 `json:"timestamp"`
}

// VerificationData stores verification and certification information
type VerificationData struct {
	Status         string            `json:"status"`
	VerifiedBy     string            `json:"verifiedBy,omitempty"`
	VerifiedAt     string            `json:"verifiedAt,omitempty"`
	CertificateID  string            `json:"certificateId,omitempty"`
	Methodology    string            `json:"methodology,omitempty"`
	ValidationHash string            `json:"validationHash,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// CollateralPosition represents a stablecoin collateral position backed by assets
type CollateralPosition struct {
	PositionID        string  `json:"positionId"`
	AssetID           string  `json:"assetId"`
	CollateralAmount  float64 `json:"collateralAmount"`
	StablecoinAmount  float64 `json:"stablecoinAmount"`
	StablecoinType    string  `json:"stablecoinType"`
	CollateralRatio   float64 `json:"collateralRatio"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"createdAt"`
	LastUpdated       string  `json:"lastUpdated"`
	LiquidationPrice  float64 `json:"liquidationPrice,omitempty"`
	OwnerAgent        string  `json:"ownerAgent"`
}

// ImpactValidation represents AI-driven impact validation results
type ImpactValidation struct {
	ValidationID     string                 `json:"validationId"`
	Timestamp        string                 `json:"timestamp"`
	TransitionFrom   int                    `json:"transitionFrom"`
	TransitionTo     int                    `json:"transitionTo"`
	Action           string                 `json:"action"`
	ValidationResult string                 `json:"validationResult"`
	CompatibilityScore float64              `json:"compatibilityScore"`
	ImpactMetrics    map[string]float64     `json:"impactMetrics,omitempty"`
	Recommendations  []string               `json:"recommendations,omitempty"`
	AIModel          string                 `json:"aiModel"`
	Confidence       float64                `json:"confidence"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ExtendedStateModel extends StateModel with asset and collateral information
type ExtendedStateModel struct {
	StateModel
	AssetData         *AssetMetadata        `json:"assetData,omitempty"`
	CollateralData    *CollateralPosition   `json:"collateralData,omitempty"`
	ImpactValidations []ImpactValidation    `json:"impactValidations,omitempty"`
}

// NewAssetMetadata creates a new asset metadata instance
func NewAssetMetadata(assetID, assetType string, quantity float64, unit string) *AssetMetadata {
	return &AssetMetadata{
		AssetID:   assetID,
		AssetType: assetType,
		Quantity:  quantity,
		Unit:      unit,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Attributes: make(map[string]interface{}),
	}
}

// NewCollateralPosition creates a new collateral position
func NewCollateralPosition(positionID, assetID, ownerAgent, stablecoinType string,
	collateralAmount, stablecoinAmount, collateralRatio float64) *CollateralPosition {
	now := time.Now().UTC().Format(time.RFC3339)
	return &CollateralPosition{
		PositionID:       positionID,
		AssetID:          assetID,
		CollateralAmount: collateralAmount,
		StablecoinAmount: stablecoinAmount,
		StablecoinType:   stablecoinType,
		CollateralRatio:  collateralRatio,
		Status:           "active",
		CreatedAt:        now,
		LastUpdated:      now,
		OwnerAgent:       ownerAgent,
	}
}

// NewImpactValidation creates a new impact validation record
func NewImpactValidation(transitionFrom, transitionTo int, action, aiModel string) *ImpactValidation {
	return &ImpactValidation{
		ValidationID:   generateValidationID(),
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		TransitionFrom: transitionFrom,
		TransitionTo:   transitionTo,
		Action:         action,
		AIModel:        aiModel,
		ImpactMetrics:  make(map[string]float64),
		Metadata:       make(map[string]interface{}),
	}
}

// generateValidationID generates a unique validation ID
func generateValidationID() string {
	return "val_" + time.Now().Format("20060102150405")
}
