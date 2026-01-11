// Copyright Blockchain SSM Lightning Integration 2025
// License: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// Lightning-specific SSM actions
const (
	ACTION_ANCHOR_TO_LIGHTNING     = "AnchorToLightning"
	ACTION_MINT_TAPROOT_ASSET      = "MintTaprootAsset"
	ACTION_TRANSFER_VIA_LIGHTNING  = "TransferViaLightning"
	ACTION_VERIFY_LIGHTNING_ANCHOR = "VerifyLightningAnchor"
	ACTION_SETTLE_FROM_LIGHTNING   = "SettleFromLightning"
)

// LightningSSMExtension adds Lightning Network capabilities to SSM
type LightningSSMExtension struct {
	SessionID      string `json:"session"`
	LightningEnabled bool `json:"lightning_enabled"`
	AnchorPolicy   string `json:"anchor_policy"` // "every_transition", "periodic", "manual"
	AnchorInterval int    `json:"anchor_interval"` // For periodic anchoring
	TaprootAssetID string `json:"taproot_asset_id,omitempty"`
	LastAnchor     int    `json:"last_anchor"` // Last iteration that was anchored
}

// PerformLightningAction handles Lightning-specific SSM actions
func PerformLightningAction(stub shim.ChaincodeStubInterface, action string, context State,
	agent string, signature string) pb.Response {

	switch action {
	case ACTION_ANCHOR_TO_LIGHTNING:
		return performAnchorToLightning(stub, context, agent, signature)
	case ACTION_MINT_TAPROOT_ASSET:
		return performMintTaprootAsset(stub, context, agent, signature)
	case ACTION_TRANSFER_VIA_LIGHTNING:
		return performTransferViaLightning(stub, context, agent, signature)
	case ACTION_VERIFY_LIGHTNING_ANCHOR:
		return performVerifyLightningAnchor(stub, context, agent, signature)
	case ACTION_SETTLE_FROM_LIGHTNING:
		return performSettleFromLightning(stub, context, agent, signature)
	default:
		return shim.Error(fmt.Sprintf("Unknown Lightning action: %s", action))
	}
}

// performAnchorToLightning anchors current SSM state to Bitcoin/Lightning
func performAnchorToLightning(stub shim.ChaincodeStubInterface, context State,
	agent string, signature string) pb.Response {

	// Retrieve current state
	var state State
	err := state.Get(stub, context.Session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get state: %s", err))
	}

	// Verify iteration matches
	if state.Iteration != context.Iteration {
		return shim.Error(fmt.Sprintf("Invalid iteration: expected %d, got %d",
			state.Iteration, context.Iteration))
	}

	// Check if agent is authorized (any involved agent can anchor)
	authorized := false
	for agentName := range state.Roles {
		if agentName == agent {
			authorized = true
			break
		}
	}
	if !authorized {
		return shim.Error(fmt.Sprintf("Agent %s not authorized to anchor", agent))
	}

	// Create anchor on Bitcoin
	anchor, err := AnchorStateOnBitcoin(stub, &state)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to anchor state: %s", err))
	}

	// Update Lightning extension metadata
	ext := LightningSSMExtension{
		SessionID:      state.Session,
		LightningEnabled: true,
		LastAnchor:     state.Iteration,
	}
	extKey := fmt.Sprintf("lightning_ext_%s", state.Session)
	extJSON, _ := json.Marshal(ext)
	stub.PutState(extKey, extJSON)

	// Return anchor information
	anchorJSON, _ := json.Marshal(anchor)
	return shim.Success(anchorJSON)
}

// performMintTaprootAsset mints a Taproot Asset for the current SSM session
func performMintTaprootAsset(stub shim.ChaincodeStubInterface, context State,
	agent string, signature string) pb.Response {

	// Retrieve current state
	var state State
	err := state.Get(stub, context.Session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get state: %s", err))
	}

	// Check if ITMO data exists in state
	// Parse public data as ITMO
	var itmo ITMOCommodity
	err = json.Unmarshal([]byte(state.Public), &itmo)
	if err != nil {
		return shim.Error(fmt.Sprintf("State does not contain valid ITMO data: %s", err))
	}

	// Mint Taproot Asset with 3 decimals (0.001 tCO2e precision)
	taprootAsset, err := MintTaprootAsset(stub, &itmo, 3)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to mint Taproot Asset: %s", err))
	}

	// Update Lightning extension with asset ID
	extKey := fmt.Sprintf("lightning_ext_%s", state.Session)
	extData, _ := stub.GetState(extKey)

	var ext LightningSSMExtension
	if extData != nil {
		json.Unmarshal(extData, &ext)
	}
	ext.SessionID = state.Session
	ext.LightningEnabled = true
	ext.TaprootAssetID = taprootAsset.TaprootAssetID

	extJSON, _ := json.Marshal(ext)
	stub.PutState(extKey, extJSON)

	// Return asset information
	assetJSON, _ := json.Marshal(taprootAsset)
	return shim.Success(assetJSON)
}

// performTransferViaLightning executes a transfer on Lightning Network
func performTransferViaLightning(stub shim.ChaincodeStubInterface, context State,
	agent string, signature string) pb.Response {

	// Retrieve current state
	var state State
	err := state.Get(stub, context.Session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get state: %s", err))
	}

	// Parse transfer details from context.Public
	var transfer LightningTransfer
	err = json.Unmarshal([]byte(context.Public), &transfer)
	if err != nil {
		return shim.Error(fmt.Sprintf("Invalid transfer data: %s", err))
	}

	// Verify agent is the sender
	if transfer.FromAgent != agent {
		return shim.Error(fmt.Sprintf("Agent %s not authorized to send from %s",
			agent, transfer.FromAgent))
	}

	// Get Taproot Asset ID from Lightning extension
	extKey := fmt.Sprintf("lightning_ext_%s", state.Session)
	extData, _ := stub.GetState(extKey)
	if extData == nil {
		return shim.Error("Lightning not enabled for this session")
	}

	var ext LightningSSMExtension
	json.Unmarshal(extData, &ext)

	transfer.AssetID = ext.TaprootAssetID

	// Record transfer
	err = TransferOnLightning(stub, &transfer)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to record transfer: %s", err))
	}

	// Update state with transfer info
	state.Iteration++
	transferJSON, _ := json.Marshal(transfer)
	state.Public = string(transferJSON)

	// Store updated state
	err = state.Put(stub, context.Session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to update state: %s", err))
	}

	return shim.Success(transferJSON)
}

// performVerifyLightningAnchor verifies an anchor on Bitcoin
func performVerifyLightningAnchor(stub shim.ChaincodeStubInterface, context State,
	agent string, signature string) pb.Response {

	// Retrieve current state
	var state State
	err := state.Get(stub, context.Session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get state: %s", err))
	}

	// Verify anchor
	verified, err := VerifyAnchor(stub, state.Session, state.Iteration)
	if err != nil {
		return shim.Error(fmt.Sprintf("Verification failed: %s", err))
	}

	result := map[string]interface{}{
		"session":   state.Session,
		"iteration": state.Iteration,
		"verified":  verified,
	}

	resultJSON, _ := json.Marshal(result)
	return shim.Success(resultJSON)
}

// performSettleFromLightning settles a Lightning transfer back to Hyperledger
func performSettleFromLightning(stub shim.ChaincodeStubInterface, context State,
	agent string, signature string) pb.Response {

	// Parse settlement data
	type SettlementData struct {
		PaymentHash string `json:"payment_hash"`
		Preimage    string `json:"preimage"`
		NewOwner    string `json:"new_owner"`
	}

	var settlement SettlementData
	err := json.Unmarshal([]byte(context.Public), &settlement)
	if err != nil {
		return shim.Error(fmt.Sprintf("Invalid settlement data: %s", err))
	}

	// Verify preimage matches payment hash (simplified)
	// In production: hash(preimage) should equal payment_hash

	// Retrieve current state
	var state State
	err = state.Get(stub, context.Session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to get state: %s", err))
	}

	// Update roles to reflect new ownership
	// Find the role of the old owner and transfer to new owner
	var oldRole string
	for agentName, role := range state.Roles {
		if agentName == agent {
			oldRole = role
			break
		}
	}

	if oldRole == "" {
		return shim.Error(fmt.Sprintf("Agent %s has no role in this session", agent))
	}

	// Transfer role to new owner
	delete(state.Roles, agent)
	state.Roles[settlement.NewOwner] = oldRole

	// Update iteration
	state.Iteration++

	// Add settlement info to public data
	settlementInfo := fmt.Sprintf("Settled Lightning transfer %s to %s",
		settlement.PaymentHash, settlement.NewOwner)
	state.Public = settlementInfo

	// Store updated state
	err = state.Put(stub, context.Session)
	if err != nil {
		return shim.Error(fmt.Sprintf("Failed to update state: %s", err))
	}

	return shim.Success([]byte(settlementInfo))
}

// GetLightningExtension retrieves Lightning extension for a session
func GetLightningExtension(stub shim.ChaincodeStubInterface, sessionID string) (*LightningSSMExtension, error) {
	extKey := fmt.Sprintf("lightning_ext_%s", sessionID)
	extData, err := stub.GetState(extKey)
	if err != nil {
		return nil, err
	}

	if extData == nil {
		return nil, fmt.Errorf("Lightning not enabled for session %s", sessionID)
	}

	var ext LightningSSMExtension
	err = json.Unmarshal(extData, &ext)
	if err != nil {
		return nil, err
	}

	return &ext, nil
}

// EnableLightningForSession enables Lightning Network features for an SSM session
func EnableLightningForSession(stub shim.ChaincodeStubInterface, sessionID string,
	anchorPolicy string, anchorInterval int) error {

	ext := LightningSSMExtension{
		SessionID:        sessionID,
		LightningEnabled: true,
		AnchorPolicy:     anchorPolicy,
		AnchorInterval:   anchorInterval,
		LastAnchor:       0,
	}

	extKey := fmt.Sprintf("lightning_ext_%s", sessionID)
	extJSON, err := json.Marshal(ext)
	if err != nil {
		return err
	}

	return stub.PutState(extKey, extJSON)
}

// ShouldAnchor determines if current state should be anchored based on policy
func ShouldAnchor(stub shim.ChaincodeStubInterface, sessionID string, currentIteration int) (bool, error) {
	ext, err := GetLightningExtension(stub, sessionID)
	if err != nil {
		return false, nil // Lightning not enabled
	}

	if !ext.LightningEnabled {
		return false, nil
	}

	switch ext.AnchorPolicy {
	case "every_transition":
		return true, nil
	case "periodic":
		iterationsSinceLastAnchor := currentIteration - ext.LastAnchor
		return iterationsSinceLastAnchor >= ext.AnchorInterval, nil
	case "manual":
		return false, nil
	default:
		return false, fmt.Errorf("unknown anchor policy: %s", ext.AnchorPolicy)
	}
}
