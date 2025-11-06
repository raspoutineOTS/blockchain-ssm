// Copyright 2024
// License: Apache-2.0
//
// ITMO Article 6 State Tracking Model
// State-only tracking system for Internationally Transferred Mitigation Outcomes
// Philosophy: Track states, not assets (like SWIFT for carbon credits)

package main

import "time"

// ITMOStateRecord tracks the STATE of an ITMO without possessing the ITMO itself
// The actual ITMO remains in its national/international registry
// We only track and tokenize the STATE
type ITMOStateRecord struct {
	// State Record Identification
	StateRecordID string `json:"stateRecordId"` // Unique ID for this state record
	SessionID     string `json:"sessionId"`     // SSM session ID

	// Reference to actual ITMO (which stays in registry)
	ITMOReference ITMOReference `json:"itmoReference"`

	// Current State (what we track)
	CurrentState   ITMOState `json:"currentState"`
	StateUpdatedAt time.Time `json:"stateUpdatedAt"`

	// State History (SSM transitions)
	StateHistory []ITMOStateTransition `json:"stateHistory"`

	// Article 6 Compliance
	CorrespondingAdjustment *CorrespondingAdjustment `json:"correspondingAdjustment,omitempty"`
	Article6Type            string                   `json:"article6Type"` // "6.2" or "6.4"

	// Proof from Source Registry
	RegistryProof RegistryProof `json:"registryProof"`

	// State Control (who can transition state)
	StateController string   `json:"stateController"` // Entity controlling state transitions
	AuthorizedUsers []string `json:"authorizedUsers"` // Users who can perform transitions

	// Tokenization (state as token)
	IsTokenized bool   `json:"isTokenized"`
	TokenID     string `json:"tokenId,omitempty"`

	// Collateralization (using state as collateral)
	IsCollateral      bool             `json:"isCollateral"`
	CollateralDetails *StateCollateral `json:"collateralDetails,omitempty"`

	// Sync Information
	LastSyncedAt time.Time `json:"lastSyncedAt"` // Last sync with source registry
	NextSyncAt   time.Time `json:"nextSyncAt"`   // Next scheduled sync
	SyncStatus   string    `json:"syncStatus"`   // "synced", "pending", "error"

	// Metadata
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ITMOReference contains reference information to the actual ITMO
// The ITMO itself stays in the source registry
type ITMOReference struct {
	// Core Identifiers
	SerialNumber string `json:"serialNumber"` // Unique serial number from registry
	RegistryID   string `json:"registryId"`   // Registry holding the ITMO
	RegistryType string `json:"registryType"` // "verra", "gold_standard", "national", "unfccc_a6d"

	// Geographic Information
	OriginCountry  string `json:"originCountry"`  // ISO 3166-1 alpha-3 (e.g., "BRA")
	CurrentCountry string `json:"currentCountry"` // Where ITMO is currently held

	// ITMO Details
	Quantity    float64 `json:"quantity"`    // tCO2e
	VintageYear int     `json:"vintageYear"` // Year of emission reduction
	ProjectID   string  `json:"projectId"`   // Source project
	ProjectName string  `json:"projectName,omitempty"`
	Methodology string  `json:"methodology"` // e.g., "VM0042", "ACM0002"
	Sector      string  `json:"sector"`      // e.g., "Forestry", "Renewable Energy"

	// Article 6 Specific
	FirstTransferYear int    `json:"firstTransferYear,omitempty"` // Year of first int'l transfer
	AuthorizingParty  string `json:"authorizingParty,omitempty"`  // Country authorizing (ISO code)

	// Verification
	RegistryURL     string    `json:"registryUrl"`               // URL to verify in source registry
	LastVerifiedAt  time.Time `json:"lastVerifiedAt"`
	VerificationHash string   `json:"verificationHash,omitempty"` // Hash for integrity check
}

// ITMOState represents the lifecycle state of an ITMO
type ITMOState string

const (
	// Article 6.2 Cooperative Approaches - Core States
	ITMOStateIssued      ITMOState = "issued"      // Issued in origin country registry
	ITMOStateAuthorized  ITMOState = "authorized"  // Authorized for international transfer
	ITMOStateTransferred ITMOState = "transferred" // Transferred between countries
	ITMOStateHeld        ITMOState = "held"        // Held by entity
	ITMOStateCollateral  ITMOState = "collateral"  // Used as collateral (innovation)
	ITMOStateRetired     ITMOState = "retired"     // Retired towards NDC
	ITMOStateCancelled   ITMOState = "cancelled"   // Cancelled/invalidated

	// Article 6.4 Mechanism - Specific States
	ITMOStateA64Issued       ITMOState = "a64_issued"        // Issued under Art 6.4 mechanism
	ITMOStateA64Authorized   ITMOState = "a64_authorized"    // Authorized under Art 6.4
	ITMOStateA64FirstTransfer ITMOState = "a64_first_transfer" // First international transfer
	ITMOStateA64Used         ITMOState = "a64_used"          // Used towards NDC

	// Intermediate/Pending States
	ITMOStatePendingAuth     ITMOState = "pending_auth"     // Pending authorization
	ITMOStatePendingTransfer ITMOState = "pending_transfer" // Pending transfer
	ITMOStatePendingCA       ITMOState = "pending_ca"       // Pending corresponding adjustment

	// Disputed/Error States
	ITMOStateDisputed  ITMOState = "disputed"   // Under dispute
	ITMOStateSuspended ITMOState = "suspended"  // Temporarily suspended
	ITMOStateError     ITMOState = "error"      // Error state
)

// ITMOStateTransition represents a state change in the SSM
type ITMOStateTransition struct {
	// Transition Details
	FromState ITMOState `json:"fromState"`
	ToState   ITMOState `json:"toState"`
	Action    string    `json:"action"` // SSM action name

	// Execution Info
	TransitionID  string    `json:"transitionId"`
	TransactionID string    `json:"transactionId"` // Blockchain transaction
	Timestamp     time.Time `json:"timestamp"`

	// Actor Information
	Actor string `json:"actor"` // Who performed the transition
	Role  string `json:"role"`  // SSM role

	// State Change Details
	QuantityAffected float64 `json:"quantityAffected,omitempty"` // For partial transitions
	Reason           string  `json:"reason,omitempty"`

	// Proof
	ProofHash string `json:"proofHash,omitempty"` // Hash of transition proof
}

// CorrespondingAdjustment tracks mandatory adjustments under Article 6.2
// This ensures no double counting of emission reductions
type CorrespondingAdjustment struct {
	// Parties Involved
	TransferringParty string `json:"transferringParty"` // ISO 3166-1 alpha-3 (subtracts)
	AcquiringParty    string `json:"acquiringParty"`    // ISO 3166-1 alpha-3 (adds)

	// Adjustment Details
	Quantity          float64 `json:"quantity"`          // tCO2e transferred
	FirstTransferYear int     `json:"firstTransferYear"` // Year of first transfer

	// Status Tracking
	AdjustmentStatus CAStatus `json:"adjustmentStatus"`
	TransferringPartyCA bool  `json:"transferringPartyCA"` // Has transferring party applied CA?
	AcquiringPartyCA    bool  `json:"acquiringPartyCA"`    // Has acquiring party applied CA?

	// A6D (Article 6 Database) Integration
	A6DReported      bool      `json:"a6dReported"`                // Reported to UNFCCC A6D?
	A6DTransactionID string    `json:"a6dTransactionId,omitempty"` // A6D transaction reference
	A6DReportedAt    time.Time `json:"a6dReportedAt,omitempty"`

	// Proof and Verification
	ProofHash       string `json:"proofHash"`             // Hash of adjustment proof
	ProofURL        string `json:"proofUrl,omitempty"`    // Link to official proof
	VerifiedByUNFCCC bool  `json:"verifiedByUnfccc"`      // Verified by UNFCCC?

	// Timestamps
	InitiatedAt time.Time  `json:"initiatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// CAStatus represents the status of a corresponding adjustment
type CAStatus string

const (
	CAPending   CAStatus = "pending"   // Adjustment initiated, not complete
	CAPartial   CAStatus = "partial"   // One party has applied adjustment
	CAComplete  CAStatus = "complete"  // Both parties have applied adjustment
	CADisputed  CAStatus = "disputed"  // Disputed adjustment
	CACancelled CAStatus = "cancelled" // Adjustment cancelled
	CAFailed    CAStatus = "failed"    // Adjustment failed
)

// RegistryProof provides cryptographic proof from source registry
// Proves the state exists in the source registry
type RegistryProof struct {
	// Proof Type
	ProofType string `json:"proofType"` // "hash", "signature", "merkle_proof", "api_response"

	// Proof Data
	ProofValue    string    `json:"proofValue"`    // Actual proof (hash, signature, etc.)
	ProofTimestamp time.Time `json:"proofTimestamp"` // When proof was generated

	// Registry Information
	RegistryID        string `json:"registryId"`                  // Which registry provided proof
	RegistrySignature string `json:"registrySignature,omitempty"` // Optional digital signature
	RegistryPublicKey string `json:"registryPublicKey,omitempty"` // For signature verification

	// Verification
	VerificationURL string `json:"verificationUrl"` // URL to verify proof independently
	VerifiedAt      *time.Time `json:"verifiedAt,omitempty"`
	VerifiedBy      string     `json:"verifiedBy,omitempty"`
}

// StateCollateral represents a state record used as collateral
// Key: We collateralize the STATE, not the ITMO itself
type StateCollateral struct {
	// Position Identification
	PositionID    string `json:"positionId"`
	StateRecordID string `json:"stateRecordId"`

	// Collateral Amounts
	CollateralQty float64 `json:"collateralQty"` // tCO2e portion used as collateral
	TotalQty      float64 `json:"totalQty"`      // Total quantity in state record
	CollateralPct float64 `json:"collateralPct"` // Percentage collateralized (0-100)

	// Stablecoin Details
	StablecoinMinted float64 `json:"stablecoinMinted"` // Amount of stablecoin minted
	StablecoinType   string  `json:"stablecoinType"`   // "USDC", "USDT", "DAI", etc.
	CollateralRatio  float64 `json:"collateralRatio"`  // Ratio (e.g., 1.5 = 150%)

	// Risk Parameters
	LiquidationPrice float64 `json:"liquidationPrice"` // Price at which liquidation occurs
	CurrentPrice     float64 `json:"currentPrice"`     // Current market price
	HealthFactor     float64 `json:"healthFactor"`     // Health (>1.0 = healthy)

	// Owner
	Owner string `json:"owner"` // Address of position owner

	// Restrictions
	CanTransferState bool `json:"canTransferState"` // Can state be transferred while collateralized?
	CanRetireITMO    bool `json:"canRetireItmo"`    // Can underlying ITMO be retired?

	// Timestamps
	CollateralizedAt time.Time `json:"collateralizedAt"`
	LastUpdated      time.Time `json:"lastUpdated"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
}

// StateToken represents tokenization of an ITMO state record
// Token = ownership/control rights over the state, not the ITMO itself
type StateToken struct {
	// Token Identification
	TokenID       string `json:"tokenId"`
	StateRecordID string `json:"stateRecordId"` // Links to ITMOStateRecord

	// Token Supply
	TotalSupply       float64 `json:"totalSupply"`       // Total tokenized quantity
	CirculatingSupply float64 `json:"circulatingSupply"` // Currently circulating
	LockedSupply      float64 `json:"lockedSupply"`      // Locked (for collateral, etc.)
	BurnedSupply      float64 `json:"burnedSupply"`      // Permanently burned

	// Fractional Ownership
	// Key: Multiple parties can own fractions of the state
	Holders map[string]StateTokenBalance `json:"holders"` // address -> balance

	// Token State
	IsFrozen         bool `json:"isFrozen"`         // Token transfers frozen?
	IsCollateralized bool `json:"isCollateralized"` // Any tokens used as collateral?

	// Compliance
	TransferRules     []TransferRule `json:"transferRules"`
	MinimumHoldPeriod *time.Duration `json:"minimumHoldPeriod,omitempty"`

	// Underlying ITMO Reference
	ITMOReference ITMOReference `json:"itmoReference"`

	// Timestamps
	TokenizedAt  time.Time  `json:"tokenizedAt"`
	LastTransfer *time.Time `json:"lastTransfer,omitempty"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// StateTokenBalance represents a holder's balance of state tokens
type StateTokenBalance struct {
	Amount       float64    `json:"amount"`                 // Token amount held
	AcquiredAt   time.Time  `json:"acquiredAt"`
	LockedAmount float64    `json:"lockedAmount,omitempty"` // Amount locked for collateral
	LockedUntil  *time.Time `json:"lockedUntil,omitempty"`
	CanTransfer  bool       `json:"canTransfer"`            // Transfer allowed?
}

// TransferRule defines rules for state token transfers
type TransferRule struct {
	RuleID           string         `json:"ruleId"`
	Jurisdiction     string         `json:"jurisdiction"`      // ISO country code or "GLOBAL"
	AllowedActions   []string       `json:"allowedActions"`    // "transfer", "collateralize", "retire"
	RequiredKYCLevel string         `json:"requiredKycLevel"`  // "none", "basic", "enhanced"
	MinHoldPeriod    *time.Duration `json:"minHoldPeriod,omitempty"`
	MaxTransferSize  *float64       `json:"maxTransferSize,omitempty"`
	RequiresApproval bool           `json:"requiresApproval"` // Requires authority approval?
	Description      string         `json:"description,omitempty"`
}

// StateMessage represents a state update message (like SWIFT message)
type StateMessage struct {
	// Message Type (similar to SWIFT MT types)
	MessageType string `json:"messageType"` // "STATE_SYNC", "STATE_TRANSITION", "CA_UPDATE"
	MessageID   string `json:"messageId"`

	// State Information
	StateRecordID string    `json:"stateRecordId"`
	FromState     ITMOState `json:"fromState,omitempty"`
	ToState       ITMOState `json:"toState"`

	// Source Information
	SourceRegistry string    `json:"sourceRegistry"`
	Timestamp      time.Time `json:"timestamp"`

	// Proof
	Proof RegistryProof `json:"proof"`

	// Additional Data
	CorrespondingAdjustment *CorrespondingAdjustment `json:"correspondingAdjustment,omitempty"`
	Metadata                map[string]interface{}   `json:"metadata,omitempty"`

	// Processing Status
	Processed   bool      `json:"processed"`
	ProcessedAt *time.Time `json:"processedAt,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// Article6Compliance tracks overall Article 6 compliance for a state record
type Article6Compliance struct {
	StateRecordID string `json:"stateRecordId"`

	// Compliance Checks
	IsAuthorized           bool `json:"isAuthorized"`           // Authorized by originating party?
	CAApplied              bool `json:"caApplied"`              // Corresponding adjustment applied?
	A6DReported            bool `json:"a6dReported"`            // Reported to A6D?
	NoDoubleCount          bool `json:"noDoubleCount"`          // No double counting?
	TransparencyMet        bool `json:"transparencyMet"`        // Transparency requirements met?
	ShareOfProceedsApplied bool `json:"shareOfProceedsApplied"` // SOP applied (if Art 6.4)?

	// Overall Compliance
	IsCompliant     bool      `json:"isCompliant"`
	ComplianceScore float64   `json:"complianceScore"` // 0-100
	LastCheckedAt   time.Time `json:"lastCheckedAt"`

	// Issues
	ComplianceIssues []string `json:"complianceIssues,omitempty"`

	// Auditor
	VerifiedBy string     `json:"verifiedBy,omitempty"`
	VerifiedAt *time.Time `json:"verifiedAt,omitempty"`
}

// A6DTransaction represents a transaction in UNFCCC Article 6 Database
type A6DTransaction struct {
	TransactionID string    `json:"transactionId"` // A6D transaction ID
	TransactionType string  `json:"transactionType"` // "authorization", "first_transfer", "use"

	// Parties
	TransferringParty string `json:"transferringParty,omitempty"`
	AcquiringParty    string `json:"acquiringParty,omitempty"`
	UsingParty        string `json:"usingParty,omitempty"`

	// ITMO Details
	ITMOQuantity  float64 `json:"itmoQuantity"`
	VintageYear   int     `json:"vintageYear"`
	TransferYear  int     `json:"transferYear"`

	// Status
	Status       string    `json:"status"` // "pending", "completed", "rejected"
	ReportedAt   time.Time `json:"reportedAt"`
	ConfirmedAt  *time.Time `json:"confirmedAt,omitempty"`

	// Links
	A6DURL       string `json:"a6dUrl"`       // URL in A6D system
	ProofURL     string `json:"proofUrl,omitempty"`
}
