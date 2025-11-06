// Copyright Luc Yriarte <luc.yriarte@thingagora.org> 2018
// License: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// ValidatorClient handles communication with the AI validator service
type ValidatorClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// TransitionValidationRequest represents a request to validate a transition
type TransitionValidationRequest struct {
	SessionID        string                 `json:"session_id"`
	FromState        int                    `json:"from_state"`
	ToState          int                    `json:"to_state"`
	Action           string                 `json:"action"`
	Role             string                 `json:"role"`
	AssetID          string                 `json:"asset_id,omitempty"`
	AssetType        string                 `json:"asset_type,omitempty"`
	AssetQuantity    *float64               `json:"asset_quantity,omitempty"`
	CollateralAmount *float64               `json:"collateral_amount,omitempty"`
	StablecoinAmount *float64               `json:"stablecoin_amount,omitempty"`
	PublicData       map[string]interface{} `json:"public_data,omitempty"`
	PrivateData      map[string]interface{} `json:"private_data,omitempty"`
}

// ValidationResponse represents the response from the validator
type ValidationResponse struct {
	ValidationID       string             `json:"validation_id"`
	Timestamp          string             `json:"timestamp"`
	TransitionFrom     int                `json:"transition_from"`
	TransitionTo       int                `json:"transition_to"`
	Action             string             `json:"action"`
	ValidationResult   string             `json:"validation_result"`
	CompatibilityScore float64            `json:"compatibility_score"`
	ImpactMetrics      map[string]float64 `json:"impact_metrics"`
	Recommendations    []string           `json:"recommendations"`
	AIModel            string             `json:"ai_model"`
	Confidence         float64            `json:"confidence"`
	Metadata           map[string]string  `json:"metadata"`
}

// NewValidatorClient creates a new validator client
func NewValidatorClient(baseURL string, timeout time.Duration) *ValidatorClient {
	return &ValidatorClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Timeout: timeout,
	}
}

// ValidateTransition validates a state transition through the AI service
func (vc *ValidatorClient) ValidateTransition(req *TransitionValidationRequest) (*ValidationResponse, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/validate/transition", vc.BaseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := vc.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("validation failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Unmarshal response
	var validationResp ValidationResponse
	if err := json.Unmarshal(body, &validationResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &validationResp, nil
}

// ConvertToImpactValidation converts API response to ImpactValidation struct
func ConvertToImpactValidation(resp *ValidationResponse) *ImpactValidation {
	metadata := make(map[string]interface{})
	for k, v := range resp.Metadata {
		metadata[k] = v
	}

	return &ImpactValidation{
		ValidationID:       resp.ValidationID,
		Timestamp:          resp.Timestamp,
		TransitionFrom:     resp.TransitionFrom,
		TransitionTo:       resp.TransitionTo,
		Action:             resp.Action,
		ValidationResult:   resp.ValidationResult,
		CompatibilityScore: resp.CompatibilityScore,
		ImpactMetrics:      resp.ImpactMetrics,
		Recommendations:    resp.Recommendations,
		AIModel:            resp.AIModel,
		Confidence:         resp.Confidence,
		Metadata:           metadata,
	}
}

// ValidateWithRetry validates with retry logic for resilience
func (vc *ValidatorClient) ValidateWithRetry(req *TransitionValidationRequest, maxRetries int) (*ValidationResponse, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		resp, err := vc.ValidateTransition(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Wait before retrying (exponential backoff)
		if i < maxRetries-1 {
			waitTime := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("validation failed after %d retries: %v", maxRetries, lastErr)
}

// CreateValidationRequest creates a validation request from state context
func CreateValidationRequest(session string, from, to int, action, role string,
	asset *AssetMetadata, collateral *CollateralPosition) *TransitionValidationRequest {

	req := &TransitionValidationRequest{
		SessionID: session,
		FromState: from,
		ToState:   to,
		Action:    action,
		Role:      role,
	}

	if asset != nil {
		req.AssetID = asset.AssetID
		req.AssetType = asset.AssetType
		req.AssetQuantity = &asset.Quantity
	}

	if collateral != nil {
		req.CollateralAmount = &collateral.CollateralAmount
		req.StablecoinAmount = &collateral.StablecoinAmount
	}

	return req
}

// CheckValidatorHealth checks if the validator service is healthy
func (vc *ValidatorClient) CheckValidatorHealth() error {
	url := fmt.Sprintf("%s/health", vc.BaseURL)
	resp, err := vc.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("validator service is unhealthy")
	}

	return nil
}
