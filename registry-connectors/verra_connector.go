// Copyright 2024
// License: Apache-2.0
//
// Verra Registry Connector
// Connects to Verra Registry API to fetch and verify carbon credit data

package registryconnectors

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// VerraConnector implements connection to Verra Registry
type VerraConnector struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	RateLimit  *RateLimiter
}

// VerraCredit represents a credit from Verra registry
type VerraCredit struct {
	ID                  string    `json:"id"`
	SerialNumber        string    `json:"serialNumber"`
	ProjectID           string    `json:"projectId"`
	ProjectName         string    `json:"projectName"`
	Methodology         string    `json:"methodology"`
	VintageStart        time.Time `json:"vintageStart"`
	VintageEnd          time.Time `json:"vintageEnd"`
	Quantity            float64   `json:"quantitytCO2e"`
	Status              string    `json:"status"`
	IssuanceDate        time.Time `json:"issuanceDate"`
	RetiredQuantity     float64   `json:"retiredQuantity"`
	CancelledQuantity   float64   `json:"cancelledQuantity"`
	Country             string    `json:"country"`
	Region              string    `json:"region"`
	ProjectType         string    `json:"projectType"`
	VerificationBody    string    `json:"verificationBody"`
	CorrespondingAdj    bool      `json:"correspondingAdjustment"`
	AdditionalCriteria  []string  `json:"additionalCertifications"`
	RegistryLink        string    `json:"registryLink"`
}

// VerraRetirement represents a retirement record from Verra
type VerraRetirement struct {
	RetirementID        string    `json:"retirementId"`
	SerialNumber        string    `json:"serialNumber"`
	Quantity            float64   `json:"quantity"`
	RetirementDate      time.Time `json:"retirementDate"`
	Beneficiary         string    `json:"beneficiary"`
	BeneficiaryLocation string    `json:"beneficiaryLocation"`
	RetirementReason    string    `json:"retirementReason"`
	CertificateURL      string    `json:"certificateUrl"`
	Note                string    `json:"note"`
}

// NewVerraConnector creates a new Verra registry connector
func NewVerraConnector(apiKey string) *VerraConnector {
	return &VerraConnector{
		BaseURL: "https://registry.verra.org/api/v1",
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		RateLimit: NewRateLimiter(10, time.Minute), // 10 requests per minute
	}
}

// FetchCredit fetches a carbon credit from Verra registry
func (vc *VerraConnector) FetchCredit(serialNumber string) (*VerraCredit, error) {
	// Rate limiting
	if err := vc.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Build request
	url := fmt.Sprintf("%s/credits/%s", vc.BaseURL, serialNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	req.Header.Set("Authorization", "Bearer "+vc.APIKey)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := vc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var credit VerraCredit
	if err := json.NewDecoder(resp.Body).Decode(&credit); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &credit, nil
}

// VerifyStatus verifies the current status of a credit
func (vc *VerraConnector) VerifyStatus(serialNumber string) (string, error) {
	credit, err := vc.FetchCredit(serialNumber)
	if err != nil {
		return "", err
	}
	return credit.Status, nil
}

// FetchRetirements fetches all retirements for a credit
func (vc *VerraConnector) FetchRetirements(serialNumber string) ([]VerraRetirement, error) {
	if err := vc.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/credits/%s/retirements", vc.BaseURL, serialNumber)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+vc.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := vc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var retirements []VerraRetirement
	if err := json.NewDecoder(resp.Body).Decode(&retirements); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return retirements, nil
}

// InitiateRetirement initiates a retirement on Verra registry
func (vc *VerraConnector) InitiateRetirement(params RetirementParams) (*VerraRetirement, error) {
	if err := vc.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Build request body
	body, err := json.Marshal(map[string]interface{}{
		"serialNumber": params.SerialNumber,
		"quantity":     params.Quantity,
		"beneficiary":  params.Beneficiary,
		"reason":       params.Reason,
		"note":         params.Note,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/retirements", vc.BaseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+vc.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := vc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var retirement VerraRetirement
	if err := json.NewDecoder(resp.Body).Decode(&retirement); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &retirement, nil
}

// SearchProjects searches for projects by criteria
func (vc *VerraConnector) SearchProjects(criteria SearchCriteria) ([]ProjectInfo, error) {
	if err := vc.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Build query parameters
	url := fmt.Sprintf("%s/projects?type=%s&country=%s&vintage=%d",
		vc.BaseURL,
		criteria.ProjectType,
		criteria.Country,
		criteria.Vintage,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+vc.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := vc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var projects []ProjectInfo
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return projects, nil
}

// PollUpdates polls for updates since last sync
func (vc *VerraConnector) PollUpdates(lastSync time.Time) ([]CreditUpdate, error) {
	if err := vc.RateLimit.Wait(); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	url := fmt.Sprintf("%s/updates?since=%s", vc.BaseURL, lastSync.Format(time.RFC3339))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+vc.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := vc.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var updates []CreditUpdate
	if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return updates, nil
}

// Helper types

type RetirementParams struct {
	SerialNumber string
	Quantity     float64
	Beneficiary  string
	Reason       string
	Note         string
}

type SearchCriteria struct {
	ProjectType string
	Country     string
	Vintage     int
	Methodology string
}

type ProjectInfo struct {
	ProjectID       string   `json:"projectId"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Country         string   `json:"country"`
	Methodology     string   `json:"methodology"`
	Status          string   `json:"status"`
	TotalCredits    float64  `json:"totalCredits"`
	AvailableCredits float64 `json:"availableCredits"`
	VintageYears    []int    `json:"vintageYears"`
}

type CreditUpdate struct {
	SerialNumber string    `json:"serialNumber"`
	UpdateType   string    `json:"updateType"` // "issuance", "retirement", "cancellation", "transfer"
	Quantity     float64   `json:"quantity"`
	Timestamp    time.Time `json:"timestamp"`
	Details      map[string]interface{} `json:"details"`
}

// RateLimiter implements simple rate limiting
type RateLimiter struct {
	requests int
	window   time.Duration
	tokens   chan struct{}
}

func NewRateLimiter(requests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: requests,
		window:   window,
		tokens:   make(chan struct{}, requests),
	}

	// Fill initial tokens
	for i := 0; i < requests; i++ {
		rl.tokens <- struct{}{}
	}

	// Refill tokens periodically
	go func() {
		ticker := time.NewTicker(window / time.Duration(requests))
		for range ticker.C {
			select {
			case rl.tokens <- struct{}{}:
			default:
			}
		}
	}()

	return rl
}

func (rl *RateLimiter) Wait() error {
	<-rl.tokens
	return nil
}
