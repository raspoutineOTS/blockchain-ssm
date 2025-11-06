// Copyright Luc Yriarte <luc.yriarte@thingagora.org> 2018
// License: Apache-2.0

package main

import (
	"encoding/json"
	"errors"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

type ExtendedState struct {
	ExtendedStateModel
}

//
// Storable interface implementation
//

func (self *ExtendedState) Put(stub shim.ChaincodeStubInterface, key string) error {
	self.ExtendedStateModel.ObjectType = "extendedState"
	data, err := self.Serialize()
	if err != nil {
		return err
	}
	return stub.PutState(key, data)
}

func (self *ExtendedState) Get(stub shim.ChaincodeStubInterface, key string) error {
	data, err := stub.GetState(key)
	if err != nil {
		return err
	}
	err = self.Deserialize(data)
	if err != nil {
		return err
	}
	self.ExtendedStateModel.ObjectType = ""
	return err
}

//
// Serializable interface implementation
//

func (self *ExtendedState) Serialize() ([]byte, error) {
	return json.Marshal(self.ExtendedStateModel)
}

func (self *ExtendedState) Deserialize(data []byte) error {
	return json.Unmarshal(data, &self.ExtendedStateModel)
}

//
// ExtendedState API implementation
//

// PerformWithValidation performs a transition with AI impact validation
func (self *ExtendedState) PerformWithValidation(update *ExtendedState, role string, action string, validation *ImpactValidation) error {
	// Check if current iteration count passed the limit
	if self.Limit != nil && self.Iteration >= *self.Limit {
		return errors.New("Passed limit iterations count.")
	}

	// Check the proposed update iteration
	if self.Iteration != update.Iteration {
		return errors.New("Invalid iteration number of proposed update.")
	}

	// Validate compatibility score threshold
	if validation != nil && validation.CompatibilityScore < 0.7 {
		return errors.New("Impact validation failed: compatibility score too low")
	}

	// Set origin transition
	self.Origin = &Transition{self.Current, update.Current, role, action}

	// Add validation record
	if validation != nil {
		validation.TransitionFrom = self.Current
		validation.TransitionTo = update.Current
		validation.Action = action
		if self.ImpactValidations == nil {
			self.ImpactValidations = []ImpactValidation{}
		}
		self.ImpactValidations = append(self.ImpactValidations, *validation)
	}

	// Increment iteration
	self.Iteration++

	// Update the current state
	self.Current = update.Current

	// Update public and private data
	self.Public = update.Public
	self.Private = update.Private

	// Update asset and collateral data if provided
	if update.AssetData != nil {
		self.AssetData = update.AssetData
	}
	if update.CollateralData != nil {
		self.CollateralData = update.CollateralData
	}

	return nil
}

// UpdateAssetMetadata updates the asset metadata
func (self *ExtendedState) UpdateAssetMetadata(asset *AssetMetadata) error {
	if asset == nil {
		return errors.New("Asset metadata cannot be nil")
	}
	self.AssetData = asset
	return nil
}

// UpdateCollateralPosition updates the collateral position
func (self *ExtendedState) UpdateCollateralPosition(collateral *CollateralPosition) error {
	if collateral == nil {
		return errors.New("Collateral position cannot be nil")
	}

	// Validate collateral ratio
	if collateral.CollateralRatio <= 0 {
		return errors.New("Collateral ratio must be positive")
	}

	self.CollateralData = collateral
	return nil
}

// CheckCollateralHealth verifies the health of the collateral position
func (self *ExtendedState) CheckCollateralHealth(currentAssetPrice float64) (bool, string) {
	if self.CollateralData == nil {
		return false, "No collateral position"
	}

	if self.AssetData == nil {
		return false, "No asset data"
	}

	// Calculate current value
	currentValue := self.AssetData.Quantity * currentAssetPrice
	requiredValue := self.CollateralData.StablecoinAmount * self.CollateralData.CollateralRatio

	if currentValue < requiredValue {
		return false, "Undercollateralized"
	}

	if currentValue < requiredValue*1.2 {
		return true, "Warning: Near liquidation threshold"
	}

	return true, "Healthy"
}

// GetLatestValidation returns the most recent impact validation
func (self *ExtendedState) GetLatestValidation() *ImpactValidation {
	if len(self.ImpactValidations) == 0 {
		return nil
	}
	return &self.ImpactValidations[len(self.ImpactValidations)-1]
}

// GetValidationHistory returns all impact validations
func (self *ExtendedState) GetValidationHistory() []ImpactValidation {
	return self.ImpactValidations
}
