// Copyright 2024
// License: Apache-2.0
//
// Carbon Credit Data Model for Official Registry Integration

package main

import "time"

// CarbonCredit represents a carbon credit from an official registry
type CarbonCredit struct {
	// Universal identifiers
	SerialNumber string `json:"serialNumber"` // Unique across all registries
	RegistryID   string `json:"registryId"`   // Verra, Gold Standard, ACR, CAR, etc.
	ProjectID    string `json:"projectId"`    // Registry-specific project ID
	VintageYear  int    `json:"vintageYear"`  // Year of emission reduction

	// Quantity and units
	Quantity float64 `json:"quantity"` // Amount in tCO2e
	Unit     string  `json:"unit"`     // tCO2e, tCO2, etc.

	// Project details
	ProjectName string `json:"projectName"`
	ProjectType string `json:"projectType"` // Forestry, Renewable Energy, Cookstoves, etc.
	Methodology string `json:"methodology"` // VM0042, ACM0002, GS-CDM, etc.
	Country     string `json:"country"`     // ISO country code
	Region      string `json:"region,omitempty"`

	// Verification and certification
	VerificationBody string    `json:"verificationBody"` // Third-party verifier
	IssuanceDate     time.Time `json:"issuanceDate"`
	ExpiryDate       *time.Time `json:"expiryDate,omitempty"`
	CertificateURL   string     `json:"certificateUrl,omitempty"`

	// Current status
	CurrentState  CreditState       `json:"currentState"`
	StatusHistory []StateTransition `json:"statusHistory"`

	// Ownership
	CurrentOwner     string  `json:"currentOwner"`
	FractionalOwners map[string]float64 `json:"fractionalOwners,omitempty"` // For tokenization

	// Registry synchronization
	LastSyncDate  time.Time `json:"lastSyncDate"`
	RegistryURL   string    `json:"registryUrl"`
	RegistryProof string    `json:"registryProof"` // Hash or signature from registry

	// Additional standards and criteria
	AdditionalCriteria []string `json:"additionalCriteria,omitempty"` // CORSIA, ICAO, etc.
	CobenefitsSDGs     []int    `json:"cobenefitsSDGs,omitempty"`     // UN SDG numbers (1-17)
	CorrespondingAdj   bool     `json:"correspondingAdj,omitempty"`   // Article 6 adjustment

	// Tokenization info
	TokenID       string  `json:"tokenId,omitempty"`       // If tokenized
	IsCollateral  bool    `json:"isCollateral"`            // Used as collateral
	CollateralPct float64 `json:"collateralPct,omitempty"` // Percentage used as collateral
}

// CreditState represents the lifecycle state of a carbon credit
type CreditState string

const (
	StateIssued       CreditState = "issued"       // Newly issued by registry
	StateActive       CreditState = "active"       // Available for trading
	StateHeld         CreditState = "held"         // Held in an account
	StateCollateral   CreditState = "collateral"   // Used as collateral for stablecoins
	StateTransferring CreditState = "transferring" // In transfer process
	StateFrozen       CreditState = "frozen"       // Temporarily frozen
	StateRetired      CreditState = "retired"      // Permanently retired (used)
	StateCancelled    CreditState = "cancelled"    // Cancelled/invalidated
)

// StateTransition tracks changes in credit state
type StateTransition struct {
	FromState     CreditState `json:"fromState"`
	ToState       CreditState `json:"toState"`
	Timestamp     time.Time   `json:"timestamp"`
	TransactionID string      `json:"transactionId"`
	Actor         string      `json:"actor"`
	Reason        string      `json:"reason,omitempty"`
	Quantity      float64     `json:"quantity,omitempty"` // For partial state changes
}

// CarbonCreditToken represents a tokenized carbon credit with state-specific ownership
type CarbonCreditToken struct {
	// Token identification
	TokenID            string      `json:"tokenId"`            // Unique token identifier
	CreditSerialNumber string      `json:"creditSerialNumber"` // Links to CarbonCredit
	State              CreditState `json:"state"`              // Current state of this token

	// Token economics
	TotalSupply       float64 `json:"totalSupply"`       // Total tCO2e represented
	CirculatingSupply float64 `json:"circulatingSupply"` // Currently in circulation
	BurnedSupply      float64 `json:"burnedSupply"`      // Permanently burned (retired)

	// Ownership tracking (fractional)
	Holders map[string]TokenBalance `json:"holders"` // address -> balance info

	// Collateral information (if state == StateCollateral)
	CollateralInfo *TokenCollateral `json:"collateralInfo,omitempty"`

	// Compliance and restrictions
	TransferRestrictions []TransferRule `json:"transferRestrictions,omitempty"`
	Freezable            bool           `json:"freezable"` // Can be frozen by authority
	MinimumHoldPeriod    *time.Duration `json:"minimumHoldPeriod,omitempty"`

	// Metadata
	CreatedAt    time.Time  `json:"createdAt"`
	LastTransfer *time.Time `json:"lastTransfer,omitempty"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// TokenBalance represents a holder's balance with metadata
type TokenBalance struct {
	Amount       float64   `json:"amount"`
	AcquiredAt   time.Time `json:"acquiredAt"`
	LockedUntil  *time.Time `json:"lockedUntil,omitempty"`
	LockedAmount float64    `json:"lockedAmount,omitempty"` // Amount locked for collateral
}

// TokenCollateral stores collateralization details for a token
type TokenCollateral struct {
	PositionID       string    `json:"positionId"`
	CollateralAmount float64   `json:"collateralAmount"` // tCO2e used as collateral
	StablecoinMinted float64   `json:"stablecoinMinted"` // Amount of stablecoin minted
	StablecoinType   string    `json:"stablecoinType"`   // USDC, USDT, DAI, etc.
	CollateralRatio  float64   `json:"collateralRatio"`  // Current ratio (>= 1.0)
	LiquidationPrice float64   `json:"liquidationPrice"` // Price at which liquidation occurs
	CreatedAt        time.Time `json:"createdAt"`
	LastUpdated      time.Time `json:"lastUpdated"`
}

// TransferRule defines restrictions on token transfers
type TransferRule struct {
	RuleID           string         `json:"ruleId"`
	Jurisdiction     string         `json:"jurisdiction"`      // ISO country code or "GLOBAL"
	AllowedActions   []string       `json:"allowedActions"`    // "transfer", "collateralize", "retire"
	RequiredKYCLevel string         `json:"requiredKycLevel"`  // "basic", "enhanced", "institutional"
	MinHoldPeriod    *time.Duration `json:"minHoldPeriod,omitempty"`
	MaxTransferSize  *float64       `json:"maxTransferSize,omitempty"`
	Description      string         `json:"description,omitempty"`
}

// RetirementRecord represents a permanent retirement of carbon credits
type RetirementRecord struct {
	// Retirement identification
	RetirementID       string    `json:"retirementId"`
	CreditSerialNumber string    `json:"creditSerialNumber"`
	TokenID            string    `json:"tokenId,omitempty"` // If tokenized
	Quantity           float64   `json:"quantity"`
	RetirementDate     time.Time `json:"retirementDate"`

	// Beneficiary information
	Beneficiary        string `json:"beneficiary"`        // Entity retiring credits
	BeneficiaryCountry string `json:"beneficiaryCountry"` // ISO country code
	BeneficiaryType    string `json:"beneficiaryType"`    // "individual", "corporate", "government"
	Purpose            string `json:"purpose"`            // Reason for retirement
	ReportingPeriod    string `json:"reportingPeriod,omitempty"`

	// Registry proof
	RegistryID          string `json:"registryId"`
	RegistryProof       string `json:"registryProof"`       // Proof from registry
	RegistryCertificate string `json:"registryCertificate"` // PDF/URL to certificate
	RegistryRetirementID string `json:"registryRetirementId"`

	// On-chain proof
	TransactionID string `json:"transactionId"` // Blockchain transaction
	BlockNumber   uint64 `json:"blockNumber"`
	TokensBurned  float64 `json:"tokensBurned"` // Amount of tokens burned

	// Certificate
	CertificateID   string `json:"certificateId"`
	CertificateHash string `json:"certificateHash"` // SHA-256 of certificate
	CertificateURL  string `json:"certificateUrl,omitempty"`

	// Verification
	VerifiedAt time.Time `json:"verifiedAt"`
	VerifiedBy string    `json:"verifiedBy"` // Verifying authority
}

// RegistryStatus represents the current status from the external registry
type RegistryStatus struct {
	SerialNumber     string      `json:"serialNumber"`
	RegistryID       string      `json:"registryId"`
	Status           CreditState `json:"status"`
	Quantity         float64     `json:"quantity"`
	AvailableQty     float64     `json:"availableQty"`     // Not retired or cancelled
	RetiredQty       float64     `json:"retiredQty"`
	CancelledQty     float64     `json:"cancelledQty"`
	CurrentOwner     string      `json:"currentOwner"`
	LastUpdated      time.Time   `json:"lastUpdated"`
	RegistryURL      string      `json:"registryUrl"`
	ValidationStatus string      `json:"validationStatus"` // "valid", "expired", "under_review"
}

// PriceQuote represents a price quote for carbon credits
type PriceQuote struct {
	ProjectType string    `json:"projectType"` // Type of project
	Vintage     int       `json:"vintage"`     // Vintage year
	Registry    string    `json:"registry"`    // Registry ID
	Price       float64   `json:"price"`       // USD per tCO2e
	Currency    string    `json:"currency"`    // USD, EUR, etc.
	Bid         float64   `json:"bid"`         // Bid price
	Ask         float64   `json:"ask"`         // Ask price
	Volume24h   float64   `json:"volume24h"`   // 24h trading volume
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`     // Exchange or OTC platform
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
}

// MarketStats provides market statistics for a credit type
type MarketStats struct {
	ProjectType       string    `json:"projectType"`
	Vintage           int       `json:"vintage"`
	Registry          string    `json:"registry"`
	AveragePrice      float64   `json:"averagePrice"`
	High24h           float64   `json:"high24h"`
	Low24h            float64   `json:"low24h"`
	Volatility        float64   `json:"volatility"`        // Percentage
	TotalVolume       float64   `json:"totalVolume"`       // Total traded
	OutstandingSupply float64   `json:"outstandingSupply"` // Total available
	LastUpdated       time.Time `json:"lastUpdated"`
}

// ComplianceCheck represents the result of a compliance check
type ComplianceCheck struct {
	CheckID         string           `json:"checkId"`
	UserID          string           `json:"userId"`
	Action          string           `json:"action"`         // "transfer", "collateralize", "retire"
	AssetID         string           `json:"assetId"`        // Credit or token ID
	Jurisdiction    string           `json:"jurisdiction"`   // ISO country code
	Result          ComplianceResult `json:"result"`
	Restrictions    []string         `json:"restrictions,omitempty"`
	RequiredActions []string         `json:"requiredActions,omitempty"`
	KYCLevel        string           `json:"kycLevel"`      // User's KYC level
	AMLStatus       string           `json:"amlStatus"`     // AML screening result
	CheckedAt       time.Time        `json:"checkedAt"`
	ExpiresAt       *time.Time       `json:"expiresAt,omitempty"`
}

// ComplianceResult indicates the outcome of a compliance check
type ComplianceResult string

const (
	ComplianceApproved            ComplianceResult = "approved"
	ComplianceDenied              ComplianceResult = "denied"
	ComplianceRequiresVerification ComplianceResult = "requires_verification"
	CompliancePending             ComplianceResult = "pending"
)

// SyncStatus tracks synchronization with external registries
type SyncStatus struct {
	CreditID     string    `json:"creditId"`
	RegistryID   string    `json:"registryId"`
	LastSyncTime time.Time `json:"lastSyncTime"`
	NextSyncTime time.Time `json:"nextSyncTime"`
	SyncStatus   string    `json:"syncStatus"` // "success", "failed", "pending"
	ErrorMessage string    `json:"errorMessage,omitempty"`
	Conflicts    []string  `json:"conflicts,omitempty"` // List of detected conflicts
}

// ConflictResolution defines how to resolve conflicts between registry and blockchain
type ConflictResolution struct {
	ConflictID   string                 `json:"conflictId"`
	CreditID     string                 `json:"creditId"`
	RegistryID   string                 `json:"registryId"`
	ConflictType string                 `json:"conflictType"` // "quantity", "status", "ownership"
	RegistryData interface{}            `json:"registryData"`
	ChainData    interface{}            `json:"chainData"`
	Resolution   ConflictResolutionType `json:"resolution"`
	ResolvedBy   string                 `json:"resolvedBy"`
	ResolvedAt   time.Time              `json:"resolvedAt"`
	Notes        string                 `json:"notes,omitempty"`
}

// ConflictResolutionType defines the resolution strategy
type ConflictResolutionType string

const (
	RegistryWins   ConflictResolutionType = "registry_wins"   // Registry is source of truth
	BlockchainWins ConflictResolutionType = "blockchain_wins" // Blockchain is authoritative
	ManualReview   ConflictResolutionType = "manual_review"   // Requires human intervention
	Merged         ConflictResolutionType = "merged"          // Data merged from both sources
)
