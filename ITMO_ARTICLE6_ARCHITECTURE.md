# ITMO Article 6 State Tracking Architecture

## Executive Summary

Implementation of a **state-only tracking system** for Internationally Transferred Mitigation Outcomes (ITMO) under Paris Agreement Article 6, using the Signing State Machine (SSM) framework.

### Core Philosophy: State Tracking, Not Asset Transfer

**Key Principle**: Like the SWIFT network for banking, we track **states and transitions** of ITMOs that remain in their national registries, rather than transferring the actual credits onto the blockchain.

```
SWIFT Banking Model:                    ITMO State Machine Model:
Money stays in local banks      →      ITMOs stay in national registries
SWIFT tracks transactions       →      SSM tracks state transitions
Messages = proof of transfer    →      States = proof of status
```

## Article 6 Background

### Article 6.2 (Cooperative Approaches)

Countries can **cooperate** to achieve NDCs (Nationally Determined Contributions) through:
- Bilateral/multilateral trading
- **Corresponding adjustments** (mandatory)
- ITMO transfers between countries
- Transparency and accounting

### Article 6.4 (Mechanism)

New **centralized mechanism** replacing CDM:
- UN-supervised mechanism
- Activity cycle and authorization
- Share of proceeds for adaptation
- Overall mitigation in global emissions

### Key Concepts

**ITMO (Internationally Transferred Mitigation Outcome)**:
- Unit of emission reduction/removal
- 1 ITMO = 1 tonne CO2 equivalent
- Transferred between countries
- Subject to corresponding adjustments

**Corresponding Adjustment (CA)**:
- Mandatory accounting adjustment
- Transferring country: **subtracts** ITMOs from their carbon budget
- Acquiring country: **adds** ITMOs to their carbon budget
- Ensures no double counting
- Tracked in Article 6 Database (A6D)

## Architecture Overview

### State-Only Tracking System

```
┌─────────────────────────────────────────────────────────────┐
│           National Registries (ITMO Source of Truth)         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Verra   │  │   Gold   │  │ National │  │ UNFCCC   │   │
│  │ Registry │  │ Standard │  │ Registry │  │   A6D    │   │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘  └─────┬────┘   │
└────────┼─────────────┼─────────────┼─────────────┼─────────┘
         │             │             │             │
         └─────────────┴─────────────┴─────────────┘
                           │
                  ┌────────▼────────┐
                  │  State Oracle   │  ← Fetches states only
                  │   Connectors    │
                  └────────┬────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│              SSM Blockchain (State Tracking)                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         ITMO State Records (Not the ITMOs!)          │  │
│  │  - Serial number + Registry ID (reference)           │  │
│  │  - Current state (issued, authorized, transferred,   │  │
│  │    held, collateral, retired, cancelled)             │  │
│  │  - Corresponding adjustment status                   │  │
│  │  - State change history                              │  │
│  │  - Proof hashes from source registries              │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │        State Machine Transitions (SSM)               │  │
│  │  State 0 → State 1 (authorize)                       │  │
│  │  State 1 → State 2 (transfer)                        │  │
│  │  State 2 → State 3 (collateralize)                   │  │
│  │  State 3 → State 4 (retire)                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │    State Tokens (Tokenized States for Collateral)   │  │
│  │  - Token ID = StateRecord ID                         │  │
│  │  - Represents "rights to the state"                  │  │
│  │  - Can be used as collateral                         │  │
│  │  - Fractional ownership of state                     │  │
│  └──────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
                           │
                  ┌────────▼────────┐
                  │   Stablecoin    │
                  │ Minting/Burning │
                  └─────────────────┘
```

## ITMO State Model

### State Record Structure

```go
// ITMOStateRecord tracks the state of an ITMO without holding the ITMO itself
type ITMOStateRecord struct {
    // Unique identifier for this state record
    StateRecordID     string    `json:"stateRecordId"`

    // Reference to actual ITMO (stays in national registry)
    ITMOReference     ITMOReference `json:"itmoReference"`

    // Current state
    CurrentState      ITMOState     `json:"currentState"`
    StateUpdatedAt    time.Time     `json:"stateUpdatedAt"`

    // State transition history
    StateHistory      []StateTransition `json:"stateHistory"`

    // Article 6 compliance
    CorrespondingAdjustment *CorrespondingAdjustment `json:"correspondingAdjustment,omitempty"`

    // Proof from source registry
    RegistryProof     RegistryProof `json:"registryProof"`

    // State ownership (who controls state transitions)
    StateController   string        `json:"stateController"`

    // Tokenization
    IsTokenized       bool          `json:"isTokenized"`
    TokenID           string        `json:"tokenId,omitempty"`

    // Collateralization
    IsCollateral      bool          `json:"isCollateral"`
    CollateralDetails *StateCollateral `json:"collateralDetails,omitempty"`

    // Metadata
    CreatedAt         time.Time     `json:"createdAt"`
    LastSyncedAt      time.Time     `json:"lastSyncedAt"`
    NextSyncAt        time.Time     `json:"nextSyncAt"`
}

// ITMOReference points to the actual ITMO in its source registry
type ITMOReference struct {
    // Unique identifiers
    SerialNumber      string    `json:"serialNumber"`      // ITMO serial number
    RegistryID        string    `json:"registryId"`        // Which registry holds it
    RegistryType      string    `json:"registryType"`      // "verra", "gold_standard", "national", "unfccc_a6d"

    // Location
    OriginCountry     string    `json:"originCountry"`     // ISO 3166-1 alpha-3
    CurrentCountry    string    `json:"currentCountry"`    // Where ITMO is currently held

    // ITMO details
    Quantity          float64   `json:"quantity"`          // tCO2e
    VintageYear       int       `json:"vintageYear"`
    ProjectID         string    `json:"projectId"`
    Methodology       string    `json:"methodology"`

    // Article 6 specific
    FirstTransferYear int       `json:"firstTransferYear"` // Year of first international transfer
    AuthorizingParty  string    `json:"authorizingParty"`  // Country code

    // Registry URL (for verification)
    RegistryURL       string    `json:"registryUrl"`
}

// ITMOState represents the state of an ITMO in its lifecycle
type ITMOState string

const (
    // Article 6.2 Cooperative Approaches States
    StateIssued          ITMOState = "issued"           // Issued in origin country
    StateAuthorized      ITMOState = "authorized"       // Authorized for transfer (Article 6.2)
    StateTransferred     ITMOState = "transferred"      // Transferred to another country
    StateHeld            ITMOState = "held"             // Held by an entity
    StateCollateral      ITMOState = "collateral"       // Used as collateral (our innovation)
    StateRetired         ITMOState = "retired"          // Retired towards NDC
    StateCancelled       ITMOState = "cancelled"        // Cancelled

    // Article 6.4 Mechanism States
    StateA64Issued       ITMOState = "a64_issued"       // Issued under Article 6.4
    StateA64Authorized   ITMOState = "a64_authorized"   // Authorized for transfer
    StateA64FirstTransfer ITMOState = "a64_first_transfer" // First international transfer
    StateA64Transferred  ITMOState = "a64_transferred"  // Subsequent transfer
    StateA64Used         ITMOState = "a64_used"         // Used towards NDC
)

// CorrespondingAdjustment tracks the mandatory adjustment for Article 6.2
type CorrespondingAdjustment struct {
    // Adjustment details
    TransferringParty     string    `json:"transferringParty"` // ISO country code
    AcquiringParty        string    `json:"acquiringParty"`    // ISO country code
    Quantity              float64   `json:"quantity"`           // tCO2e
    FirstTransferYear     int       `json:"firstTransferYear"`

    // Status
    AdjustmentStatus      CAStatus  `json:"adjustmentStatus"`
    TransferringPartyCA   bool      `json:"transferringPartyCA"` // CA applied by transferring party
    AcquiringPartyCA      bool      `json:"acquiringPartyCA"`    // CA applied by acquiring party

    // A6D (Article 6 Database) tracking
    A6DReported           bool      `json:"a6dReported"`
    A6DTransactionID      string    `json:"a6dTransactionId,omitempty"`
    A6DReportedAt         time.Time `json:"a6dReportedAt,omitempty"`

    // Proof
    ProofHash             string    `json:"proofHash"`          // Hash of adjustment proof
    ProofURL              string    `json:"proofUrl,omitempty"` // Link to official proof
}

// CAStatus represents corresponding adjustment status
type CAStatus string

const (
    CAPending     CAStatus = "pending"      // Adjustment pending
    CAPartial     CAStatus = "partial"      // One party applied
    CAComplete    CAStatus = "complete"     // Both parties applied
    CADisputed    CAStatus = "disputed"     // Disputed adjustment
    CACancelled   CAStatus = "cancelled"    // Adjustment cancelled
)

// RegistryProof provides cryptographic proof of state from source registry
type RegistryProof struct {
    ProofType         string    `json:"proofType"`         // "hash", "signature", "merkle"
    ProofValue        string    `json:"proofValue"`        // Actual proof
    ProofTimestamp    time.Time `json:"proofTimestamp"`
    RegistrySignature string    `json:"registrySignature"` // Optional registry signature
    VerificationURL   string    `json:"verificationUrl"`   // URL to verify proof
}

// StateCollateral represents state used as collateral
type StateCollateral struct {
    PositionID        string    `json:"positionId"`
    StateRecordID     string    `json:"stateRecordId"`

    // Collateral amount (portion of state used)
    CollateralQty     float64   `json:"collateralQty"`     // tCO2e portion
    TotalQty          float64   `json:"totalQty"`          // Total state quantity

    // Stablecoin details
    StablecoinMinted  float64   `json:"stablecoinMinted"`
    StablecoinType    string    `json:"stablecoinType"`
    CollateralRatio   float64   `json:"collateralRatio"`

    // Risk parameters
    LiquidationPrice  float64   `json:"liquidationPrice"`
    CurrentPrice      float64   `json:"currentPrice"`
    HealthFactor      float64   `json:"healthFactor"`      // >1.0 = healthy

    // Timestamps
    CollateralizedAt  time.Time `json:"collateralizedAt"`
    LastUpdated       time.Time `json:"lastUpdated"`
}

// StateToken represents tokenization of an ITMO state
type StateToken struct {
    TokenID           string              `json:"tokenId"`
    StateRecordID     string              `json:"stateRecordId"`

    // Token economics
    TotalSupply       float64             `json:"totalSupply"`       // Total tokenized quantity
    CirculatingSupply float64             `json:"circulatingSupply"`
    LockedSupply      float64             `json:"lockedSupply"`      // Locked for collateral

    // Fractional ownership of the state
    Holders           map[string]float64  `json:"holders"`           // address -> quantity

    // Token state
    IsFrozen          bool                `json:"isFrozen"`
    IsCollateralized  bool                `json:"isCollateralized"`

    // Compliance
    TransferRules     []TransferRule      `json:"transferRules"`

    // Timestamps
    TokenizedAt       time.Time           `json:"tokenizedAt"`
    LastTransfer      *time.Time          `json:"lastTransfer,omitempty"`
}
```

## State Machine Transitions for ITMO States

### SSM Definition for ITMO Lifecycle

```json
{
  "name": "ITMO_Article6_Lifecycle",
  "transitions": [
    // Article 6.2 Cooperative Approaches
    {"from": 0, "to": 1, "role": "issuer", "action": "issue"},
    {"from": 1, "to": 2, "role": "authority", "action": "authorize"},
    {"from": 2, "to": 3, "role": "transferor", "action": "transfer"},
    {"from": 3, "to": 4, "role": "holder", "action": "collateralize"},
    {"from": 4, "to": 5, "role": "holder", "action": "release_collateral"},
    {"from": 3, "to": 6, "role": "holder", "action": "retire"},
    {"from": 5, "to": 6, "role": "holder", "action": "retire"},
    {"from": 3, "to": 7, "role": "authority", "action": "cancel"},

    // State sync transitions
    {"from": 1, "to": 1, "role": "oracle", "action": "sync_state"},
    {"from": 2, "to": 2, "role": "oracle", "action": "sync_state"},
    {"from": 3, "to": 3, "role": "oracle", "action": "sync_state"},

    // Corresponding adjustment tracking
    {"from": 2, "to": 8, "role": "authority", "action": "apply_ca_transferring"},
    {"from": 8, "to": 3, "role": "authority", "action": "apply_ca_acquiring"},

    // Article 6.4 Mechanism (if needed)
    {"from": 0, "to": 10, "role": "a64_mechanism", "action": "issue_a64"},
    {"from": 10, "to": 11, "role": "a64_mechanism", "action": "first_transfer"},
    {"from": 11, "to": 3, "role": "authority", "action": "convert_to_itmo"}
  ]
}
```

### State Transition Examples

#### Example 1: ITMO Authorization and Transfer

```javascript
// State 0 → State 1: Issue ITMO state record
// ITMO is issued in Verra registry in Brazil
{
  "action": "issue",
  "context": {
    "session": "ITMO_BRA_001",
    "iteration": 0,
    "roles": {
      "oracle": "StateOracle",
      "issuer": "BrazilRegistry",
      "authority": "BrazilGov"
    },
    "current": 0,
    "public": {
      "itmo_reference": {
        "serial_number": "VCS-1234-2024-BR-001",
        "registry_id": "verra",
        "origin_country": "BRA",
        "current_country": "BRA",
        "quantity": 1000.0,
        "vintage_year": 2024
      },
      "current_state": "issued"
    }
  }
}

// State 1 → State 2: Authorize for transfer
// Brazil government authorizes transfer to Switzerland
{
  "action": "authorize",
  "context": {
    "session": "ITMO_BRA_001",
    "iteration": 1,
    "current": 1,
    "public": {
      "current_state": "authorized",
      "corresponding_adjustment": {
        "transferring_party": "BRA",
        "acquiring_party": "CHE",
        "quantity": 1000.0,
        "first_transfer_year": 2024,
        "adjustment_status": "pending"
      }
    }
  }
}

// State 2 → State 3: Transfer with CA
// Transfer occurs, corresponding adjustments applied
{
  "action": "transfer",
  "context": {
    "session": "ITMO_BRA_001",
    "iteration": 2,
    "current": 2,
    "public": {
      "current_state": "transferred",
      "itmo_reference": {
        "current_country": "CHE"  // Now in Switzerland
      },
      "corresponding_adjustment": {
        "adjustment_status": "complete",
        "transferring_party_ca": true,
        "acquiring_party_ca": true,
        "a6d_reported": true,
        "a6d_transaction_id": "A6D-2024-0001234"
      }
    }
  }
}
```

#### Example 2: Collateralize State as Token

```javascript
// State 3 → State 4: Use state as collateral
{
  "action": "collateralize",
  "context": {
    "session": "ITMO_BRA_001",
    "iteration": 3,
    "current": 3,
    "public": {
      "current_state": "collateral",
      "is_tokenized": true,
      "token_id": "TOKEN_ITMO_BRA_001",
      "collateral_details": {
        "position_id": "POS_001",
        "collateral_qty": 1000.0,
        "stablecoin_minted": 15000.0,  // $15 per tCO2e
        "stablecoin_type": "USDC",
        "collateral_ratio": 1.5,
        "health_factor": 1.5
      }
    }
  }
}
```

## UNFCCC Article 6 Database (A6D) Connector

### A6D Connector Implementation

```go
// UNFCCCConnector connects to UNFCCC Article 6 Database
type UNFCCCConnector struct {
    BaseURL    string
    APIKey     string
    HTTPClient *http.Client
}

// FetchITMOState fetches the state of an ITMO from A6D
func (uc *UNFCCCConnector) FetchITMOState(serialNumber string) (*ITMOStateRecord, error) {
    // Query A6D for ITMO state
    url := fmt.Sprintf("%s/api/itmo/%s/state", uc.BaseURL, serialNumber)

    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+uc.APIKey)

    resp, err := uc.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var stateRecord ITMOStateRecord
    json.NewDecoder(resp.Body).Decode(&stateRecord)

    return &stateRecord, nil
}

// VerifyCorrespondingAdjustment verifies CA in A6D
func (uc *UNFCCCConnector) VerifyCorrespondingAdjustment(
    transferringParty, acquiringParty string,
    quantity float64,
    year int,
) (*CorrespondingAdjustment, error) {
    // Query A6D for CA status
    url := fmt.Sprintf("%s/api/ca/verify", uc.BaseURL)

    body := map[string]interface{}{
        "transferring_party": transferringParty,
        "acquiring_party": acquiringParty,
        "quantity": quantity,
        "year": year,
    }

    bodyJSON, _ := json.Marshal(body)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
    req.Header.Set("Authorization", "Bearer "+uc.APIKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := uc.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var ca CorrespondingAdjustment
    json.NewDecoder(resp.Body).Decode(&ca)

    return &ca, nil
}

// ReportStateTransition reports a state transition to A6D (if required)
func (uc *UNFCCCConnector) ReportStateTransition(
    stateRecord *ITMOStateRecord,
) (string, error) {
    // Report transition to A6D
    url := fmt.Sprintf("%s/api/report/transition", uc.BaseURL)

    bodyJSON, _ := json.Marshal(stateRecord)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
    req.Header.Set("Authorization", "Bearer "+uc.APIKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := uc.HTTPClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result struct {
        TransactionID string `json:"transaction_id"`
    }
    json.NewDecoder(resp.Body).Decode(&result)

    return result.TransactionID, nil
}
```

## State Synchronization Service

### State-Only Sync (Like SWIFT Messages)

```python
class ITMOStateSyncService:
    """
    Synchronizes ITMO states (not ITMOs themselves) from national registries
    Similar to SWIFT messaging for banking
    """

    def __init__(self, connectors, blockchain_client):
        self.connectors = connectors  # Registry connectors
        self.blockchain = blockchain_client
        self.message_queue = []

    async def sync_itmo_state(self, itmo_reference):
        """
        Sync the STATE of an ITMO from its source registry

        Key: We fetch state, not the ITMO itself
        """
        connector = self.connectors.get(itmo_reference.registry_type)

        # Fetch current state from source registry
        current_state = await connector.fetch_itmo_state(
            itmo_reference.serial_number
        )

        # Fetch blockchain state record
        state_record = await self.blockchain.get_state_record(
            itmo_reference.serial_number
        )

        # Compare states
        if self._states_differ(current_state, state_record):
            # Create state update message (like SWIFT message)
            message = self._create_state_message(
                current_state,
                state_record
            )

            # Process state transition in SSM
            await self._process_state_transition(message)

    def _create_state_message(self, new_state, old_state):
        """
        Create a state transition message
        Like MT103 in SWIFT for payments
        """
        return {
            "message_type": "STATE_TRANSITION",
            "state_record_id": old_state.state_record_id,
            "from_state": old_state.current_state,
            "to_state": new_state.current_state,
            "timestamp": datetime.utcnow(),
            "proof": new_state.registry_proof,
            "corresponding_adjustment": new_state.corresponding_adjustment
        }

    async def _process_state_transition(self, message):
        """
        Process state transition through SSM
        """
        # Determine SSM action based on state change
        action = self._map_state_to_action(
            message["from_state"],
            message["to_state"]
        )

        # Execute SSM transition
        await self.blockchain.perform_ssm_transition(
            session=message["state_record_id"],
            action=action,
            context=message
        )

    def _map_state_to_action(self, from_state, to_state):
        """Map state changes to SSM actions"""
        state_transitions = {
            ("issued", "authorized"): "authorize",
            ("authorized", "transferred"): "transfer",
            ("transferred", "collateral"): "collateralize",
            ("collateral", "held"): "release_collateral",
            ("held", "retired"): "retire",
        }
        return state_transitions.get((from_state, to_state), "sync_state")
```

## Benefits of State-Only Approach

### Legal & Regulatory Benefits

1. **ITMOs Stay in National Jurisdiction**
   - No legal issues with cross-border asset transfer
   - Complies with national regulations
   - Countries maintain sovereignty

2. **Simplified Compliance**
   - No need to be a registry
   - No custody requirements
   - Pure information system

3. **Article 6 Compliance**
   - CA tracking without ITMO movement
   - A6D integration straightforward
   - Transparent to UNFCCC

### Technical Benefits

1. **Scalability**
   - Only state data on blockchain
   - Much smaller data footprint
   - Faster sync times

2. **SSM Perfect Fit**
   - States = SSM states
   - Transitions = SSM transitions
   - Natural integration

3. **Flexibility**
   - Can track ITMOs from any registry
   - No need to integrate with each registry's transfer system
   - Read-only access sufficient

### Economic Benefits

1. **Lower Costs**
   - No physical ITMO movement costs
   - No registry transfer fees
   - Lower infrastructure costs

2. **Faster Operations**
   - State changes are instant
   - No waiting for registry confirmations
   - Real-time collateralization

3. **Tokenization Innovation**
   - Token = right to state
   - Fractional state ownership
   - Collateral without possession

## Implementation Roadmap

### Phase 1: Foundation (2 weeks)
- [ ] Implement ITMOStateRecord model
- [ ] Create UNFCCC A6D connector (state fetch only)
- [ ] Build state sync service
- [ ] SSM transitions for ITMO states

### Phase 2: CA Tracking (2 weeks)
- [ ] Corresponding adjustment tracking
- [ ] A6D reporting integration
- [ ] CA verification service
- [ ] Dispute resolution flow

### Phase 3: Tokenization (2 weeks)
- [ ] State token creation
- [ ] Fractional state ownership
- [ ] State token transfers
- [ ] Collateralization integration

### Phase 4: Multi-Registry (2 weeks)
- [ ] Verra state connector
- [ ] Gold Standard state connector
- [ ] National registry connectors
- [ ] Unified state format

### Phase 5: Production (2 weeks)
- [ ] Testing & validation
- [ ] Security audit
- [ ] Performance optimization
- [ ] Documentation

**Total: 10 weeks (2.5 months)**

## Comparison: Traditional vs State-Only

| Aspect | Traditional (Asset Transfer) | State-Only (Our Approach) |
|--------|----------------------------|---------------------------|
| **ITMO Location** | On blockchain | In national registry |
| **Blockchain Stores** | Full ITMO data | State reference only |
| **Legal Status** | Custody issues | Pure information |
| **Transfer** | Actual transfer | State transition message |
| **Cost** | High (registry fees) | Low (read-only) |
| **Speed** | Slow (registry deps) | Fast (state update) |
| **Scalability** | Limited | High |
| **Compliance** | Complex | Simplified |
| **SSM Integration** | Awkward | Natural |
| **Token Represents** | Actual ITMO | Right to state |
| **Collateral** | Actual credit | State reference |

## Conclusion

This **state-only tracking architecture** provides:

✅ **Legal Compliance**: ITMOs stay in national registries
✅ **Article 6 Compliance**: Full CA tracking and A6D integration
✅ **SSM Integration**: Perfect fit with state machine philosophy
✅ **Scalability**: Lightweight state-only data
✅ **Innovation**: Tokenize states, not assets
✅ **SWIFT-like**: Messaging about states, not asset movement

**Key Insight**: Like SWIFT doesn't move money but tracks transactions, we don't move ITMOs but track their states. This is legally safer, technically simpler, and economically more efficient.
