// Copyright Luc Yriarte <luc.yriarte@thingagora.org> 2018
// Modernized 2024 for Hyperledger Fabric 2.5+
// License: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SSMContract provides functions for managing Signing State Machines
type SSMContract struct {
	contractapi.Contract
}

// Constants for key prefixes
const (
	PrefixAdmin  = "ADMIN"
	PrefixUser   = "USER"
	PrefixGrant  = "GRANT"
	PrefixSSM    = "SSM"
	PrefixState  = "STATE"
	PrefixAsset  = "ASSET"
)

// InitLedger initializes the chaincode with administrators
func (s *SSMContract) InitLedger(ctx contractapi.TransactionContextInterface, adminsJSON string) error {
	var admins []Agent

	// Parse administrators from JSON
	err := json.Unmarshal([]byte(adminsJSON), &admins)
	if err != nil {
		return fmt.Errorf("failed to parse admins JSON: %w", err)
	}

	// Verify and store each administrator
	for i := range admins {
		// Verify public key
		_, err = admins[i].PublicKey()
		if err != nil {
			return fmt.Errorf("invalid public key for admin %s: %w", admins[i].Name, err)
		}

		// Store admin
		key := makeKey(PrefixAdmin, admins[i].Name)
		err = admins[i].Put(ctx.GetStub(), key)
		if err != nil {
			return fmt.Errorf("failed to store admin %s: %w", admins[i].Name, err)
		}
	}

	return nil
}

// ============================================================================
// Transaction Functions - Agent Management
// ============================================================================

// RegisterUser registers a new user agent
func (s *SSMContract) RegisterUser(ctx contractapi.TransactionContextInterface,
	userJSON, adminName, signature string) error {

	// Verify admin signature
	if err := s.verifySignature(ctx, adminName, PrefixAdmin, userJSON, signature); err != nil {
		// Try to verify as user with grants
		if err := s.checkGrants(ctx, userJSON, adminName, signature, "register"); err != nil {
			return err
		}
	}

	var user Agent
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return fmt.Errorf("failed to parse user JSON: %w", err)
	}

	// Verify public key
	if _, err := user.PublicKey(); err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	// Check if user already exists
	key := makeKey(PrefixUser, user.Name)
	if err := s.checkUnique(ctx, key); err != nil {
		return err
	}

	// Store user
	if err := user.Put(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("failed to store user: %w", err)
	}

	// Emit event
	return ctx.GetStub().SetEvent("UserRegistered", []byte(user.Name))
}

// ============================================================================
// Transaction Functions - SSM Management
// ============================================================================

// CreateSSM creates a new Signing State Machine
func (s *SSMContract) CreateSSM(ctx contractapi.TransactionContextInterface,
	ssmJSON, adminName, signature string) error {

	// Verify admin signature
	if err := s.verifySignature(ctx, adminName, PrefixAdmin, ssmJSON, signature); err != nil {
		// Try to verify as user with grants
		if err := s.checkGrants(ctx, ssmJSON, adminName, signature, "create"); err != nil {
			return err
		}
	}

	var ssm SigningStateMachine
	if err := json.Unmarshal([]byte(ssmJSON), &ssm); err != nil {
		return fmt.Errorf("failed to parse SSM JSON: %w", err)
	}

	// Check if SSM already exists
	key := makeKey(PrefixSSM, ssm.Name)
	if err := s.checkUnique(ctx, key); err != nil {
		return err
	}

	// Store SSM
	if err := ssm.Put(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("failed to store SSM: %w", err)
	}

	// Emit event
	return ctx.GetStub().SetEvent("SSMCreated", []byte(ssm.Name))
}

// StartSession starts a new SSM session
func (s *SSMContract) StartSession(ctx contractapi.TransactionContextInterface,
	stateJSON, adminName, signature string) error {

	// Verify admin signature
	if err := s.verifySignature(ctx, adminName, PrefixAdmin, stateJSON, signature); err != nil {
		// Try to verify as user with grants
		if err := s.checkGrants(ctx, stateJSON, adminName, signature, "start"); err != nil {
			return err
		}
	}

	var state State
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Check if session already exists
	key := makeKey(PrefixState, state.Session)
	if err := s.checkUnique(ctx, key); err != nil {
		return err
	}

	// Store initial state
	if err := state.Put(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("failed to store state: %w", err)
	}

	// Emit event
	return ctx.GetStub().SetEvent("SessionStarted", []byte(state.Session))
}

// SetSessionLimit sets or updates the iteration limit for a session
func (s *SSMContract) SetSessionLimit(ctx contractapi.TransactionContextInterface,
	updateJSON, adminName, signature string) error {

	// Verify admin signature (only admins can set limits)
	if err := s.verifySignature(ctx, adminName, PrefixAdmin, updateJSON, signature); err != nil {
		return err
	}

	var update State
	if err := json.Unmarshal([]byte(updateJSON), &update); err != nil {
		return fmt.Errorf("failed to parse update JSON: %w", err)
	}

	// Get current session state
	var session State
	key := makeKey(PrefixState, update.Session)
	if err := session.Get(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Update limit
	if err := session.SetLimit(&update); err != nil {
		return fmt.Errorf("failed to set limit: %w", err)
	}

	// Save updated state
	if err := session.Put(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Emit event
	return ctx.GetStub().SetEvent("SessionLimitSet", []byte(session.Session))
}

// GrantCredits grants or updates API credits for a user
func (s *SSMContract) GrantCredits(ctx contractapi.TransactionContextInterface,
	grantJSON, adminName, signature string) error {

	// Verify admin signature (only admins can grant credits)
	if err := s.verifySignature(ctx, adminName, PrefixAdmin, grantJSON, signature); err != nil {
		return err
	}

	var update Grant
	if err := json.Unmarshal([]byte(grantJSON), &update); err != nil {
		return fmt.Errorf("failed to parse grant JSON: %w", err)
	}

	// Verify user exists
	userKey := makeKey(PrefixUser, update.User)
	var user Agent
	if err := user.Get(ctx.GetStub(), userKey); err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get or create grant
	var grant Grant
	grantKey := makeKey(PrefixGrant, update.User)
	err := grant.Get(ctx.GetStub(), grantKey)
	if err != nil {
		// No existing grant, use the update as current grant
		grant = update
	}

	// Update credits
	if err := grant.SetCredits(&update); err != nil {
		return fmt.Errorf("failed to set credits: %w", err)
	}

	// Save grant
	if err := grant.Put(ctx.GetStub(), grantKey); err != nil {
		return fmt.Errorf("failed to save grant: %w", err)
	}

	// Emit event
	return ctx.GetStub().SetEvent("CreditsGranted", []byte(grant.User))
}

// PerformTransition performs a state transition in an SSM session
func (s *SSMContract) PerformTransition(ctx contractapi.TransactionContextInterface,
	action, stateJSON, userName, signature string) error {

	// Verify user signature
	if err := s.verifySignature(ctx, userName, PrefixUser, action+stateJSON, signature); err != nil {
		return err
	}

	var update State
	if err := json.Unmarshal([]byte(stateJSON), &update); err != nil {
		return fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Get current session state
	var session State
	sessionKey := makeKey(PrefixState, update.Session)
	if err := session.Get(ctx.GetStub(), sessionKey); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get user's role
	role := session.Roles[userName]
	if role == "" {
		return fmt.Errorf("user %s has no role in session %s", userName, session.Session)
	}

	// Get SSM definition
	var ssm SigningStateMachine
	ssmKey := makeKey(PrefixSSM, session.Ssm)
	if err := ssm.Get(ctx.GetStub(), ssmKey); err != nil {
		return fmt.Errorf("SSM not found: %w", err)
	}

	// Determine next state
	nextState := ssm.NextState(session.Current, role, action)
	if nextState == -1 {
		return fmt.Errorf("no valid transition from state %d with role %s and action %s",
			session.Current, role, action)
	}
	update.Current = nextState

	// Perform transition
	if err := session.Perform(&update, role, action); err != nil {
		return fmt.Errorf("transition failed: %w", err)
	}

	// Save updated state
	if err := session.Put(ctx.GetStub(), sessionKey); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Emit event
	eventData, _ := json.Marshal(map[string]interface{}{
		"session": session.Session,
		"from":    session.Origin.From,
		"to":      session.Origin.To,
		"action":  action,
		"role":    role,
	})
	return ctx.GetStub().SetEvent("TransitionPerformed", eventData)
}

// PerformTransitionWithValidation performs a state transition with AI validation
func (s *SSMContract) PerformTransitionWithValidation(ctx contractapi.TransactionContextInterface,
	action, stateJSON, userName, signature string) error {

	// Get validator URL from environment or use default
	validatorURL := os.Getenv("VALIDATOR_URL")
	if validatorURL == "" {
		validatorURL = "http://validator:5000"
	}

	// Verify user signature
	if err := s.verifySignature(ctx, userName, PrefixUser, action+stateJSON, signature); err != nil {
		return err
	}

	var update ExtendedState
	if err := json.Unmarshal([]byte(stateJSON), &update); err != nil {
		return fmt.Errorf("failed to parse state JSON: %w", err)
	}

	// Get current session state
	var session ExtendedState
	sessionKey := makeKey(PrefixState, update.Session)
	if err := session.Get(ctx.GetStub(), sessionKey); err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	// Get user's role
	role := session.Roles[userName]
	if role == "" {
		return fmt.Errorf("user %s has no role in session %s", userName, session.Session)
	}

	// Get SSM definition
	var ssm SigningStateMachine
	ssmKey := makeKey(PrefixSSM, session.Ssm)
	if err := ssm.Get(ctx.GetStub(), ssmKey); err != nil {
		return fmt.Errorf("SSM not found: %w", err)
	}

	// Determine next state
	nextState := ssm.NextState(session.Current, role, action)
	if nextState == -1 {
		return fmt.Errorf("no valid transition from state %d", session.Current)
	}
	update.Current = nextState

	// Perform AI validation
	validator := NewValidatorClient(validatorURL, 5*time.Second)
	req := CreateValidationRequest(
		session.Session,
		session.Current,
		nextState,
		action,
		role,
		session.AssetData,
		session.CollateralData,
	)

	validationResp, err := validator.ValidateWithRetry(req, 3)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if validationResp.ValidationResult != "approved" {
		return fmt.Errorf("transition validation rejected: %v", validationResp.Recommendations)
	}

	// Convert validation response
	validation := ConvertToImpactValidation(validationResp)

	// Perform transition with validation
	if err := session.PerformWithValidation(&update, role, action, validation); err != nil {
		return fmt.Errorf("transition failed: %w", err)
	}

	// Save updated state
	if err := session.Put(ctx.GetStub(), sessionKey); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Emit event
	eventData, _ := json.Marshal(map[string]interface{}{
		"session":             session.Session,
		"from":                session.Origin.From,
		"to":                  session.Origin.To,
		"action":              action,
		"role":                role,
		"compatibility_score": validationResp.CompatibilityScore,
	})
	return ctx.GetStub().SetEvent("TransitionPerformedWithValidation", eventData)
}

// ============================================================================
// Query Functions
// ============================================================================

// GetSession retrieves a session state
func (s *SSMContract) GetSession(ctx contractapi.TransactionContextInterface, sessionID string) (*State, error) {
	var state State
	key := makeKey(PrefixState, sessionID)
	if err := state.Get(ctx.GetStub(), key); err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	return &state, nil
}

// GetSSM retrieves an SSM definition
func (s *SSMContract) GetSSM(ctx contractapi.TransactionContextInterface, ssmName string) (*SigningStateMachine, error) {
	var ssm SigningStateMachine
	key := makeKey(PrefixSSM, ssmName)
	if err := ssm.Get(ctx.GetStub(), key); err != nil {
		return nil, fmt.Errorf("SSM not found: %w", err)
	}
	return &ssm, nil
}

// GetUser retrieves a user agent
func (s *SSMContract) GetUser(ctx contractapi.TransactionContextInterface, userName string) (*Agent, error) {
	var user Agent
	key := makeKey(PrefixUser, userName)
	if err := user.Get(ctx.GetStub(), key); err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &user, nil
}

// GetAdmin retrieves an admin agent
func (s *SSMContract) GetAdmin(ctx contractapi.TransactionContextInterface, adminName string) (*Agent, error) {
	var admin Agent
	key := makeKey(PrefixAdmin, adminName)
	if err := admin.Get(ctx.GetStub(), key); err != nil {
		return nil, fmt.Errorf("admin not found: %w", err)
	}
	return &admin, nil
}

// GetCredits retrieves a user's credits
func (s *SSMContract) GetCredits(ctx contractapi.TransactionContextInterface, userName string) (*Grant, error) {
	var grant Grant
	key := makeKey(PrefixGrant, userName)
	if err := grant.Get(ctx.GetStub(), key); err != nil {
		return nil, fmt.Errorf("credits not found: %w", err)
	}
	return &grant, nil
}

// ListByType lists all entities of a given type
func (s *SSMContract) ListByType(ctx contractapi.TransactionContextInterface, entityType string) ([]string, error) {
	prefix := strings.ToUpper(entityType)
	if entityType == "session" {
		prefix = PrefixState
	}

	iterator, err := ctx.GetStub().GetStateByRange(" ", "~")
	if err != nil {
		return nil, fmt.Errorf("failed to get state iterator: %w", err)
	}
	defer iterator.Close()

	var results []string
	prefixWithSep := prefix + "_"

	for iterator.HasNext() {
		queryResponse, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate: %w", err)
		}

		if strings.HasPrefix(queryResponse.Key, prefixWithSep) {
			id := strings.TrimPrefix(queryResponse.Key, prefixWithSep)
			results = append(results, id)
		}
	}

	return results, nil
}

// GetSessionHistory retrieves the history of a session
func (s *SSMContract) GetSessionHistory(ctx contractapi.TransactionContextInterface, sessionID string) ([]map[string]interface{}, error) {
	key := makeKey(PrefixState, sessionID)

	iterator, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer iterator.Close()

	var history []map[string]interface{}

	for iterator.HasNext() {
		modification, err := iterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate history: %w", err)
		}

		var state State
		if err := json.Unmarshal(modification.Value, &state); err != nil {
			continue
		}

		entry := map[string]interface{}{
			"txId":      modification.TxId,
			"timestamp": modification.Timestamp.AsTime(),
			"state":     state,
		}
		history = append(history, entry)
	}

	return history, nil
}

// ============================================================================
// Utility Functions
// ============================================================================

// verifySignature verifies an agent's signature
func (s *SSMContract) verifySignature(ctx contractapi.TransactionContextInterface,
	agentName, agentType, message, signature string) error {

	var agent Agent
	key := makeKey(agentType, agentName)
	if err := agent.Get(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	return agent.Verify(message, signature)
}

// checkGrants verifies user grants and updates credits
func (s *SSMContract) checkGrants(ctx contractapi.TransactionContextInterface,
	message, userName, signature, api string) error {

	// Verify user signature
	if err := s.verifySignature(ctx, userName, PrefixUser, message, signature); err != nil {
		return err
	}

	// Get user's grant
	var grant Grant
	key := makeKey(PrefixGrant, userName)
	if err := grant.Get(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("no grants found for user: %w", err)
	}

	// Verify and update credits
	if err := grant.ApiGrant(userName, api); err != nil {
		return err
	}

	// Save updated grant
	if err := grant.Put(ctx.GetStub(), key); err != nil {
		return fmt.Errorf("failed to save grant: %w", err)
	}

	return nil
}

// checkUnique ensures a key doesn't already exist
func (s *SSMContract) checkUnique(ctx contractapi.TransactionContextInterface, key string) error {
	data, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	if data != nil {
		return fmt.Errorf("identifier %s already exists", key)
	}
	return nil
}

// makeKey creates a composite key with prefix
func makeKey(prefix, id string) string {
	return prefix + "_" + id
}
