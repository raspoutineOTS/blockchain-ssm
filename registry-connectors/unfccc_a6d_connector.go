// Copyright 2024
// License: Apache-2.0
//
// UNFCCC Article 6 Database (A6D) Connector
// Fetches ITMO states (not ITMOs themselves) from UNFCCC infrastructure

package registryconnectors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// UNFCCCA6DConnector connects to UNFCCC Article 6 Database
type UNFCCCA6DConnector struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	RateLimit  *RateLimiter
}

// NewUNFCCCA6DConnector creates a new A6D connector
func NewUNFCCCA6DConnector(apiKey string) *UNFCCCA6DConnector {
	return &UNFCCCA6DConnector{
		BaseURL: "https://a6d.unfccc.int/api/v1", // Hypothetical URL
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		RateLimit: NewRateLimiter(20, time.Minute), // 20 requests per minute
	}
}

// ITMOStateResponse represents the state of an ITMO from A6D
type ITMOStateResponse struct {
	SerialNumber   string    `json:"serialNumber"`
	CurrentState   string    `json:"currentState"`
	OriginCountry  string    `json:"originCountry"`
	CurrentCountry string    `json:"currentCountry"`
	Quantity       float64   `json:"quantity"`
	VintageYear    int       `json:"vintageYear"`
	ProjectID      string    `json:"projectId"`
	Methodology    string    `json:"methodology"`

	// Authorization
	IsAuthorized     bool   `json:"isAuthorized"`
	AuthorizingParty string `json:"authorizingParty,omitempty"`
	AuthorizedAt     *time.Time `json:"authorizedAt,omitempty"`

	// Corresponding Adjustment
	CAStatus              string  `json:"caStatus"`
	TransferringPartyCA   bool    `json:"transferringPartyCA"`
	AcquiringPartyCA      bool    `json:"acquiringPartyCA"`
	FirstTransferYear     int     `json:"firstTransferYear,omitempty"`

	// A6D Transaction
	A6DTransactionID string    `json:"a6dTransactionId,omitempty"`
	LastReportedAt   time.Time `json:"lastReportedAt"`

	// Proof
	ProofHash string `json:"proofHash"`
	ProofURL  string `json:"proofUrl"`

	// Metadata
	LastUpdated time.Time `json:"lastUpdated"`
}

// CorrespondingAdjustmentResponse represents CA information from A6D
type CorrespondingAdjustmentResponse struct {
	TransactionID     string    `json:"transactionId"`
	TransferringParty string    `json:"transferringParty"`
	AcquiringParty    string    `json:"acquiringParty"`
	Quantity          float64   `json:"quantity"`
	FirstTransferYear int       `json:"firstTransferYear"`

	// Status
	Status              string    `json:"status"` // "pending", "partial", "complete"
	TransferringPartyCA bool      `json:"transferringPartyCA"`
	AcquiringPartyCA    bool      `json:"acquiringPartyCA"`

	// Timestamps
	InitiatedAt time.Time  `json:"initiatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`

	// Verification
	VerifiedByUNFCCC bool   `json:"verifiedByUnfccc"`
	ProofURL         string `json:"proofUrl"`
}

// A6DTransactionResponse represents a transaction in A6D
type A6DTransactionResponse struct {
	TransactionID   string    `json:"transactionId"`
	TransactionType string    `json:"transactionType"`

	// Parties
	TransferringParty string `json:"transferringParty,omitempty"`
	AcquiringParty    string `json:"acquiringParty,omitempty"`
	UsingParty        string `json:"usingParty,omitempty"`

	// Details
	Quantity     float64 `json:"quantity"`
	VintageYear  int     `json:"vintageYear"`
	TransferYear int     `json:"transferYear"`

	// Status
	Status      string    `json:"status"`
	ReportedAt  time.Time `json:"reportedAt"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`

	// Links
	ProofURL string `json:"proofUrl"`
}

// StateUpdateNotification represents a state update from A6D
type StateUpdateNotification struct {
	NotificationID string    `json:"notificationId"`
	SerialNumber   string    `json:"serialNumber"`
	UpdateType     string    `json:"updateType"` // "state_change", "ca_update", "authorization"
	PreviousState  string    `json:"previousState,omitempty"`
	NewState       string    `json:"newState"`
	Timestamp      time.Time `json:"timestamp"`
	ProofHash      string    `json:"proofHash"`
}

// FetchITMOState fetches the current state of an ITMO from A6D
func (ac *UNFCCCA6DConnector) FetchITMOState(serialNumber string) (*ITMOStateResponse, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/itmo/%s/state", ac.BaseURL, serialNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var stateResp ITMOStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&stateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &stateResp, nil
}

// VerifyCorrespondingAdjustment verifies CA status in A6D
func (ac *UNFCCCA6DConnector) VerifyCorrespondingAdjustment(
	transferringParty, acquiringParty string,
	quantity float64,
	year int,
) (*CorrespondingAdjustmentResponse, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/ca/verify", ac.BaseURL)

	reqBody := map[string]interface{}{
		"transferring_party": transferringParty,
		"acquiring_party":    acquiringParty,
		"quantity":           quantity,
		"year":               year,
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var caResp CorrespondingAdjustmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&caResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &caResp, nil
}

// GetA6DTransaction retrieves a transaction from A6D
func (ac *UNFCCCA6DConnector) GetA6DTransaction(transactionID string) (*A6DTransactionResponse, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/transactions/%s", ac.BaseURL, transactionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var txResp A6DTransactionResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &txResp, nil
}

// PollStateUpdates polls for state updates since last sync
func (ac *UNFCCCA6DConnector) PollStateUpdates(lastSync time.Time) ([]StateUpdateNotification, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/updates/states?since=%s", ac.BaseURL, lastSync.Format(time.RFC3339))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var updates []StateUpdateNotification
	if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return updates, nil
}

// ReportStateTransition reports a state transition to A6D (if required)
func (ac *UNFCCCA6DConnector) ReportStateTransition(
	serialNumber string,
	fromState, toState string,
	actor string,
	proofHash string,
) (string, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return "", fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/report/transition", ac.BaseURL)

	reqBody := map[string]interface{}{
		"serial_number": serialNumber,
		"from_state":    fromState,
		"to_state":      toState,
		"actor":         actor,
		"proof_hash":    proofHash,
		"timestamp":     time.Now().UTC(),
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		TransactionID string `json:"transaction_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.TransactionID, nil
}

// SearchITMOsByCountry searches for ITMOs by origin or current country
func (ac *UNFCCCA6DConnector) SearchITMOsByCountry(countryCode string, originOrCurrent string) ([]ITMOStateResponse, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/itmo/search?country=%s&type=%s", ac.BaseURL, countryCode, originOrCurrent)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var itmos []ITMOStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&itmos); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return itmos, nil
}

// VerifyProof verifies a proof hash with A6D
func (ac *UNFCCCA6DConnector) VerifyProof(serialNumber, proofHash string) (bool, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return false, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/verify/proof", ac.BaseURL)

	reqBody := map[string]interface{}{
		"serial_number": serialNumber,
		"proof_hash":    proofHash,
	}

	bodyJSON, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Valid, nil
}

// GetCountryCABalance gets a country's corresponding adjustment balance
func (ac *UNFCCCA6DConnector) GetCountryCABalance(countryCode string, year int) (*CountryCABalance, error) {
	if err := ac.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/ca/balance/%s?year=%d", ac.BaseURL, countryCode, year)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+ac.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := ac.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var balance CountryCABalance
	if err := json.NewDecoder(resp.Body).Decode(&balance); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &balance, nil
}

// CountryCABalance represents a country's CA balance
type CountryCABalance struct {
	CountryCode     string  `json:"countryCode"`
	Year            int     `json:"year"`
	TotalAuthorized float64 `json:"totalAuthorized"` // Total ITMOs authorized for transfer out
	TotalTransferred float64 `json:"totalTransferred"` // Total ITMOs transferred out
	TotalAcquired   float64 `json:"totalAcquired"`   // Total ITMOs acquired
	NetBalance      float64 `json:"netBalance"`      // Net balance (acquired - transferred)
	LastUpdated     time.Time `json:"lastUpdated"`
}
