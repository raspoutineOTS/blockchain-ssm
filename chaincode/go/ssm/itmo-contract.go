// Copyright 2024
// License: Apache-2.0
//
// ITMO Article 6 Chaincode Methods
// State-only tracking for Internationally Transferred Mitigation Outcomes
// Philosophy: Track states, not assets (like SWIFT for carbon credits)

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Key prefixes for ITMO state records
const (
	PrefixITMOState      = "ITMO_STATE"
	PrefixStateToken     = "STATE_TOKEN"
	PrefixStateCollateral = "STATE_COLLATERAL"
	PrefixITMOHistory    = "ITMO_HISTORY"
)

// ============================================================================
// ITMO State Record Management
// ============================================================================

// CreateITMOStateRecord creates a new state record for tracking an ITMO
// The ITMO itself stays in its national/international registry
// We only create a state record to track its status
func (s *SSMContract) CreateITMOStateRecord(
	ctx contractapi.TransactionContextInterface,
	stateRecordJSON, userName, signature string,
) error {
	// Verify user signature
	if err := s.verifySignature(ctx, userName, PrefixUser, stateRecordJSON, signature); err != nil {
		// Try to verify as admin
		if err := s.verifySignature(ctx, userName, PrefixAdmin, stateRecordJSON, signature); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	var stateRecord ITMOStateRecord
	if err := json.Unmarshal([]byte(stateRecordJSON), &stateRecord); err != nil {
		return fmt.Errorf("failed to parse state record JSON: %w", err)
	}

	// Validate required fields
	if stateRecord.StateRecordID == "" {
		return fmt.Errorf("stateRecordId is required")
	}
	if stateRecord.ITMOReference.SerialNumber == "" {
		return fmt.Errorf("ITMO serial number is required")
	}
	if stateRecord.ITMOReference.RegistryID == "" {
		return fmt.Errorf("registry ID is required")
	}

	// Check if state record already exists
	key := makeKey(PrefixITMOState, stateRecord.StateRecordID)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to check existing state record: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("state record %s already exists", stateRecord.StateRecordID)
	}

	// Set timestamps
	now := time.Now()
	stateRecord.CreatedAt = now
	stateRecord.UpdatedAt = now
	stateRecord.StateUpdatedAt = now
	stateRecord.SyncStatus = "synced"

	// Initialize with issued state if not set
	if stateRecord.CurrentState == "" {
		stateRecord.CurrentState = ITMOStateIssued
	}

	// Create initial state transition record
	initialTransition := ITMOStateTransition{
		FromState:     "",
		ToState:       stateRecord.CurrentState,
		Action:        "create",
		TransitionID:  fmt.Sprintf("TXN_%s_0", stateRecord.StateRecordID),
		TransactionID: ctx.GetStub().GetTxID(),
		Timestamp:     now,
		Actor:         userName,
		Role:          "creator",
	}
	stateRecord.StateHistory = []ITMOStateTransition{initialTransition}

	// Store state record
	stateRecordBytes, err := json.Marshal(stateRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal state record: %w", err)
	}

	if err := ctx.GetStub().PutState(key, stateRecordBytes); err != nil {
		return fmt.Errorf("failed to store state record: %w", err)
	}

	// Store in history
	historyKey := makeKey(PrefixITMOHistory, stateRecord.StateRecordID, "0")
	transitionBytes, _ := json.Marshal(initialTransition)
	if err := ctx.GetStub().PutState(historyKey, transitionBytes); err != nil {
		return fmt.Errorf("failed to store history: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"stateRecordId": stateRecord.StateRecordID,
		"serialNumber":  stateRecord.ITMOReference.SerialNumber,
		"registryId":    stateRecord.ITMOReference.RegistryID,
		"currentState":  stateRecord.CurrentState,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("ITMOStateRecordCreated", eventBytes)
}

// GetITMOStateRecord retrieves an ITMO state record
func (s *SSMContract) GetITMOStateRecord(
	ctx contractapi.TransactionContextInterface,
	stateRecordID string,
) (*ITMOStateRecord, error) {
	key := makeKey(PrefixITMOState, stateRecordID)
	stateRecordBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get state record: %w", err)
	}
	if stateRecordBytes == nil {
		return nil, fmt.Errorf("state record %s not found", stateRecordID)
	}

	var stateRecord ITMOStateRecord
	if err := json.Unmarshal(stateRecordBytes, &stateRecord); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state record: %w", err)
	}

	return &stateRecord, nil
}

// PerformITMOStateTransition performs a state transition on an ITMO state record
// This is the core of state-only tracking - we change the state, not the ITMO
func (s *SSMContract) PerformITMOStateTransition(
	ctx contractapi.TransactionContextInterface,
	stateRecordID, action, transitionDataJSON, userName, signature string,
) error {
	// Get existing state record
	stateRecord, err := s.GetITMOStateRecord(ctx, stateRecordID)
	if err != nil {
		return err
	}

	// Verify user signature
	if err := s.verifySignature(ctx, userName, PrefixUser, transitionDataJSON, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Check if user is authorized for this state record
	isAuthorized := false
	for _, authorizedUser := range stateRecord.AuthorizedUsers {
		if authorizedUser == userName {
			isAuthorized = true
			break
		}
	}
	if !isAuthorized && stateRecord.StateController != userName {
		return fmt.Errorf("user %s not authorized for state transitions", userName)
	}

	// Parse transition data
	var transitionData map[string]interface{}
	if err := json.Unmarshal([]byte(transitionDataJSON), &transitionData); err != nil {
		return fmt.Errorf("failed to parse transition data: %w", err)
	}

	// Determine new state based on action
	newState, err := s.determineNewState(stateRecord.CurrentState, action)
	if err != nil {
		return err
	}

	// Create state transition record
	now := time.Now()
	transition := ITMOStateTransition{
		FromState:     stateRecord.CurrentState,
		ToState:       newState,
		Action:        action,
		TransitionID:  fmt.Sprintf("TXN_%s_%d", stateRecordID, len(stateRecord.StateHistory)),
		TransactionID: ctx.GetStub().GetTxID(),
		Timestamp:     now,
		Actor:         userName,
		Role:          "authorized_user",
	}

	// Extract proof hash if provided
	if proofHash, ok := transitionData["proof_hash"].(string); ok {
		transition.ProofHash = proofHash
	}

	// Extract reason if provided
	if reason, ok := transitionData["reason"].(string); ok {
		transition.Reason = reason
	}

	// Update state record
	stateRecord.CurrentState = newState
	stateRecord.StateUpdatedAt = now
	stateRecord.UpdatedAt = now
	stateRecord.StateHistory = append(stateRecord.StateHistory, transition)

	// Store updated state record
	key := makeKey(PrefixITMOState, stateRecordID)
	stateRecordBytes, err := json.Marshal(stateRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal state record: %w", err)
	}

	if err := ctx.GetStub().PutState(key, stateRecordBytes); err != nil {
		return fmt.Errorf("failed to store state record: %w", err)
	}

	// Store in history
	historyKey := makeKey(PrefixITMOHistory, stateRecordID, fmt.Sprintf("%d", len(stateRecord.StateHistory)-1))
	transitionBytes, _ := json.Marshal(transition)
	if err := ctx.GetStub().PutState(historyKey, transitionBytes); err != nil {
		return fmt.Errorf("failed to store history: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"stateRecordId": stateRecordID,
		"fromState":     string(transition.FromState),
		"toState":       string(transition.ToState),
		"action":        action,
		"actor":         userName,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("ITMOStateTransition", eventBytes)
}

// determineNewState determines the new state based on current state and action
func (s *SSMContract) determineNewState(currentState ITMOState, action string) (ITMOState, error) {
	// Define valid state transitions
	transitions := map[string]map[string]ITMOState{
		string(ITMOStateIssued): {
			"authorize": ITMOStateAuthorized,
		},
		string(ITMOStateAuthorized): {
			"transfer":    ITMOStateTransferred,
			"initiate_ca": ITMOStatePendingCA,
		},
		string(ITMOStateTransferred): {
			"hold":   ITMOStateHeld,
			"retire": ITMOStateRetired,
		},
		string(ITMOStateHeld): {
			"collateralize": ITMOStateCollateral,
			"retire":        ITMOStateRetired,
			"transfer":      ITMOStateTransferred,
		},
		string(ITMOStateCollateral): {
			"release_collateral": ITMOStateHeld,
		},
		string(ITMOStatePendingCA): {
			"complete_ca": ITMOStateTransferred,
			"cancel":      ITMOStateAuthorized,
		},
		// Article 6.4 specific transitions
		string(ITMOStateA64Issued): {
			"a64_authorize": ITMOStateA64Authorized,
		},
		string(ITMOStateA64Authorized): {
			"a64_first_transfer": ITMOStateA64FirstTransfer,
		},
		string(ITMOStateA64FirstTransfer): {
			"convert_to_itmo": ITMOStateTransferred,
		},
	}

	validTransitions, ok := transitions[string(currentState)]
	if !ok {
		return "", fmt.Errorf("no transitions defined for state %s", currentState)
	}

	newState, ok := validTransitions[action]
	if !ok {
		return "", fmt.Errorf("invalid action %s for current state %s", action, currentState)
	}

	return newState, nil
}

// ============================================================================
// ITMO State Tokenization
// ============================================================================

// TokenizeITMOState creates a token representing rights to an ITMO state
// Enables fractional ownership and trading of state rights
func (s *SSMContract) TokenizeITMOState(
	ctx contractapi.TransactionContextInterface,
	tokenizationJSON, userName, signature string,
) error {
	// Verify user signature
	if err := s.verifySignature(ctx, userName, PrefixUser, tokenizationJSON, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	var tokenData struct {
		StateRecordID string  `json:"stateRecordId"`
		TotalSupply   float64 `json:"totalSupply"`
	}
	if err := json.Unmarshal([]byte(tokenizationJSON), &tokenData); err != nil {
		return fmt.Errorf("failed to parse tokenization data: %w", err)
	}

	// Get state record
	stateRecord, err := s.GetITMOStateRecord(ctx, tokenData.StateRecordID)
	if err != nil {
		return err
	}

	// Check if already tokenized
	if stateRecord.IsTokenized {
		return fmt.Errorf("state record %s is already tokenized", tokenData.StateRecordID)
	}

	// Check if state allows tokenization (can't tokenize retired or cancelled states)
	if stateRecord.CurrentState == ITMOStateRetired || stateRecord.CurrentState == ITMOStateCancelled {
		return fmt.Errorf("cannot tokenize state in %s status", stateRecord.CurrentState)
	}

	// Create state token
	tokenID := fmt.Sprintf("TOKEN_%s", tokenData.StateRecordID)
	now := time.Now()

	stateToken := StateToken{
		TokenID:           tokenID,
		StateRecordID:     tokenData.StateRecordID,
		TotalSupply:       tokenData.TotalSupply,
		CirculatingSupply: tokenData.TotalSupply,
		LockedSupply:      0,
		BurnedSupply:      0,
		Holders: map[string]StateTokenBalance{
			userName: {
				Amount:       tokenData.TotalSupply,
				AcquiredAt:   now,
				LockedAmount: 0,
				CanTransfer:  true,
			},
		},
		IsFrozen:         false,
		IsCollateralized: false,
		ITMOReference:    stateRecord.ITMOReference,
		TokenizedAt:      now,
		UpdatedAt:        now,
	}

	// Store token
	tokenKey := makeKey(PrefixStateToken, tokenID)
	tokenBytes, err := json.Marshal(stateToken)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := ctx.GetStub().PutState(tokenKey, tokenBytes); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Update state record
	stateRecord.IsTokenized = true
	stateRecord.TokenID = tokenID
	stateRecord.UpdatedAt = now

	stateRecordKey := makeKey(PrefixITMOState, tokenData.StateRecordID)
	stateRecordBytes, _ := json.Marshal(stateRecord)
	if err := ctx.GetStub().PutState(stateRecordKey, stateRecordBytes); err != nil {
		return fmt.Errorf("failed to update state record: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"tokenId":       tokenID,
		"stateRecordId": tokenData.StateRecordID,
		"totalSupply":   tokenData.TotalSupply,
		"owner":         userName,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("ITMOStateTokenized", eventBytes)
}

// GetStateToken retrieves a state token
func (s *SSMContract) GetStateToken(
	ctx contractapi.TransactionContextInterface,
	tokenID string,
) (*StateToken, error) {
	key := makeKey(PrefixStateToken, tokenID)
	tokenBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	if tokenBytes == nil {
		return nil, fmt.Errorf("token %s not found", tokenID)
	}

	var token StateToken
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// ============================================================================
// ITMO State Collateralization
// ============================================================================

// CollateralizeITMOState uses an ITMO state as collateral for stablecoin minting
// This is the key innovation: collateralizing the STATE, not the ITMO itself
func (s *SSMContract) CollateralizeITMOState(
	ctx contractapi.TransactionContextInterface,
	collateralJSON, userName, signature string,
) error {
	// Verify user signature
	if err := s.verifySignature(ctx, userName, PrefixUser, collateralJSON, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	var collateralData StateCollateral
	if err := json.Unmarshal([]byte(collateralJSON), &collateralData); err != nil {
		return fmt.Errorf("failed to parse collateral data: %w", err)
	}

	// Get state record
	stateRecord, err := s.GetITMOStateRecord(ctx, collateralData.StateRecordID)
	if err != nil {
		return err
	}

	// Validate state allows collateralization
	if stateRecord.CurrentState != ITMOStateHeld && stateRecord.CurrentState != ITMOStateAuthorized {
		return fmt.Errorf("cannot collateralize state in %s status", stateRecord.CurrentState)
	}

	// Check if already collateralized
	if stateRecord.IsCollateral {
		return fmt.Errorf("state record %s is already collateralized", collateralData.StateRecordID)
	}

	// Validate collateral parameters
	if collateralData.CollateralQty > stateRecord.ITMOReference.Quantity {
		return fmt.Errorf("collateral quantity exceeds available quantity")
	}

	if collateralData.CollateralRatio < 1.0 {
		return fmt.Errorf("collateral ratio must be >= 1.0")
	}

	// Generate position ID
	collateralData.PositionID = fmt.Sprintf("POS_%s", collateralData.StateRecordID)
	collateralData.Owner = userName
	now := time.Now()
	collateralData.CollateralizedAt = now
	collateralData.LastUpdated = now

	// Calculate health factor
	collateralData.HealthFactor = (collateralData.CurrentPrice * collateralData.CollateralQty) / collateralData.StablecoinMinted

	// Set restrictions (states used as collateral typically can't be transferred or retired)
	collateralData.CanTransferState = false
	collateralData.CanRetireITMO = false

	// Store collateral position
	collateralKey := makeKey(PrefixStateCollateral, collateralData.PositionID)
	collateralBytes, err := json.Marshal(collateralData)
	if err != nil {
		return fmt.Errorf("failed to marshal collateral: %w", err)
	}

	if err := ctx.GetStub().PutState(collateralKey, collateralBytes); err != nil {
		return fmt.Errorf("failed to store collateral: %w", err)
	}

	// Update state record
	stateRecord.IsCollateral = true
	stateRecord.CollateralDetails = &collateralData
	stateRecord.UpdatedAt = now

	// Perform state transition to collateral state
	stateRecord.CurrentState = ITMOStateCollateral

	stateRecordKey := makeKey(PrefixITMOState, collateralData.StateRecordID)
	stateRecordBytes, _ := json.Marshal(stateRecord)
	if err := ctx.GetStub().PutState(stateRecordKey, stateRecordBytes); err != nil {
		return fmt.Errorf("failed to update state record: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId":      collateralData.PositionID,
		"stateRecordId":   collateralData.StateRecordID,
		"collateralQty":   collateralData.CollateralQty,
		"stablecoinMinted": collateralData.StablecoinMinted,
		"stablecoinType":  collateralData.StablecoinType,
		"healthFactor":    collateralData.HealthFactor,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("ITMOStateCollateralized", eventBytes)
}

// ReleaseITMOCollateral releases collateral and returns state to held status
func (s *SSMContract) ReleaseITMOCollateral(
	ctx contractapi.TransactionContextInterface,
	positionID, userName, signature string,
) error {
	// Get collateral position
	collateralKey := makeKey(PrefixStateCollateral, positionID)
	collateralBytes, err := ctx.GetStub().GetState(collateralKey)
	if err != nil {
		return fmt.Errorf("failed to get collateral: %w", err)
	}
	if collateralBytes == nil {
		return fmt.Errorf("collateral position %s not found", positionID)
	}

	var collateral StateCollateral
	if err := json.Unmarshal(collateralBytes, &collateral); err != nil {
		return fmt.Errorf("failed to unmarshal collateral: %w", err)
	}

	// Verify ownership
	if collateral.Owner != userName {
		return fmt.Errorf("only owner can release collateral")
	}

	// Verify signature
	releaseData := map[string]string{"positionId": positionID}
	releaseJSON, _ := json.Marshal(releaseData)
	if err := s.verifySignature(ctx, userName, PrefixUser, string(releaseJSON), signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Get and update state record
	stateRecord, err := s.GetITMOStateRecord(ctx, collateral.StateRecordID)
	if err != nil {
		return err
	}

	stateRecord.IsCollateral = false
	stateRecord.CollateralDetails = nil
	stateRecord.CurrentState = ITMOStateHeld
	stateRecord.UpdatedAt = time.Now()

	stateRecordKey := makeKey(PrefixITMOState, collateral.StateRecordID)
	stateRecordBytes, _ := json.Marshal(stateRecord)
	if err := ctx.GetStub().PutState(stateRecordKey, stateRecordBytes); err != nil {
		return fmt.Errorf("failed to update state record: %w", err)
	}

	// Delete collateral position
	if err := ctx.GetStub().DelState(collateralKey); err != nil {
		return fmt.Errorf("failed to delete collateral: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId":    positionID,
		"stateRecordId": collateral.StateRecordID,
		"releasedBy":    userName,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("ITMOCollateralReleased", eventBytes)
}

// GetCollateralPosition retrieves a collateral position
func (s *SSMContract) GetCollateralPosition(
	ctx contractapi.TransactionContextInterface,
	positionID string,
) (*StateCollateral, error) {
	key := makeKey(PrefixStateCollateral, positionID)
	collateralBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get collateral: %w", err)
	}
	if collateralBytes == nil {
		return nil, fmt.Errorf("collateral position %s not found", positionID)
	}

	var collateral StateCollateral
	if err := json.Unmarshal(collateralBytes, &collateral); err != nil {
		return nil, fmt.Errorf("failed to unmarshal collateral: %w", err)
	}

	return &collateral, nil
}

// ============================================================================
// Corresponding Adjustment Management
// ============================================================================

// UpdateCorrespondingAdjustment updates CA information for an ITMO state
func (s *SSMContract) UpdateCorrespondingAdjustment(
	ctx contractapi.TransactionContextInterface,
	stateRecordID, caJSON, userName, signature string,
) error {
	// Verify signature
	if err := s.verifySignature(ctx, userName, PrefixUser, caJSON, signature); err != nil {
		if err := s.verifySignature(ctx, userName, PrefixAdmin, caJSON, signature); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	var ca CorrespondingAdjustment
	if err := json.Unmarshal([]byte(caJSON), &ca); err != nil {
		return fmt.Errorf("failed to parse CA data: %w", err)
	}

	// Get state record
	stateRecord, err := s.GetITMOStateRecord(ctx, stateRecordID)
	if err != nil {
		return err
	}

	// Update CA
	stateRecord.CorrespondingAdjustment = &ca
	stateRecord.UpdatedAt = time.Now()

	// Store updated state record
	key := makeKey(PrefixITMOState, stateRecordID)
	stateRecordBytes, _ := json.Marshal(stateRecord)
	if err := ctx.GetStub().PutState(key, stateRecordBytes); err != nil {
		return fmt.Errorf("failed to update state record: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"stateRecordId":     stateRecordID,
		"transferringParty": ca.TransferringParty,
		"acquiringParty":    ca.AcquiringParty,
		"adjustmentStatus":  ca.AdjustmentStatus,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("CorrespondingAdjustmentUpdated", eventBytes)
}

// ============================================================================
// Query Functions
// ============================================================================

// GetITMOStateHistory retrieves the complete state transition history
func (s *SSMContract) GetITMOStateHistory(
	ctx contractapi.TransactionContextInterface,
	stateRecordID string,
) ([]ITMOStateTransition, error) {
	stateRecord, err := s.GetITMOStateRecord(ctx, stateRecordID)
	if err != nil {
		return nil, err
	}

	return stateRecord.StateHistory, nil
}

// QueryITMOsByCountry retrieves ITMO state records by country
func (s *SSMContract) QueryITMOsByCountry(
	ctx contractapi.TransactionContextInterface,
	countryCode string,
) ([]*ITMOStateRecord, error) {
	// Create query for ITMOs from or in a specific country
	queryString := fmt.Sprintf(`{
		"selector": {
			"$or": [
				{"itmoReference.originCountry": "%s"},
				{"itmoReference.currentCountry": "%s"}
			]
		}
	}`, countryCode, countryCode)

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resultsIterator.Close()

	var stateRecords []*ITMOStateRecord
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var stateRecord ITMOStateRecord
		if err := json.Unmarshal(queryResponse.Value, &stateRecord); err != nil {
			continue
		}

		stateRecords = append(stateRecords, &stateRecord)
	}

	return stateRecords, nil
}

// QueryCollateralizedITMOs retrieves all ITMO states currently used as collateral
func (s *SSMContract) QueryCollateralizedITMOs(
	ctx contractapi.TransactionContextInterface,
) ([]*ITMOStateRecord, error) {
	queryString := `{
		"selector": {
			"isCollateral": true
		}
	}`

	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resultsIterator.Close()

	var stateRecords []*ITMOStateRecord
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var stateRecord ITMOStateRecord
		if err := json.Unmarshal(queryResponse.Value, &stateRecord); err != nil {
			continue
		}

		stateRecords = append(stateRecords, &stateRecord)
	}

	return stateRecords, nil
}
