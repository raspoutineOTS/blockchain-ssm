// Copyright Blockchain SSM Lightning Integration 2025
// License: Apache-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// LightningAnchor represents a cryptographic anchor on Bitcoin/Lightning Network
type LightningAnchor struct {
	SessionID        string `json:"session"`         // SSM session identifier
	StateHash        string `json:"state_hash"`      // SHA256 hash of the SSM state
	BitcoinTxID      string `json:"bitcoin_txid"`    // Bitcoin transaction ID (OP_RETURN)
	LightningInvoice string `json:"lightning_invoice"` // Bolt11 Lightning invoice
	TaprootAssetID   string `json:"taproot_asset_id"` // Taproot Asset ID if minted
	Timestamp        int64  `json:"timestamp"`       // Unix timestamp
	BlockHeight      int64  `json:"block_height"`    // Bitcoin block height
	Confirmations    int    `json:"confirmations"`   // Number of confirmations
	AnchorType       string `json:"anchor_type"`     // Type: "hash_only", "taproot_mint", "lightning_transfer"
	Metadata         map[string]string `json:"metadata"` // Additional metadata
}

// LightningAnchorModel for database storage
type LightningAnchorModel struct {
	ObjectType string `json:"docType"` // "lightning_anchor"
	LightningAnchor
}

// ITMOTaprootAsset represents an ITMO commodity tokenized as a Taproot Asset
type ITMOTaprootAsset struct {
	ITMOCommodity                          // Embedded ITMO data
	TaprootAssetID    string              `json:"taproot_asset_id"`
	GroupKey          string              `json:"group_key"`        // For fungibility
	MintTxID          string              `json:"mint_txid"`        // Minting transaction
	MetadataURI       string              `json:"metadata_uri"`     // IPFS/Arweave URI
	SupplyAmount      int64               `json:"supply_amount"`    // Total minted units
	Decimals          int                 `json:"decimals"`         // Precision (e.g., 3 = 0.001 tCO2e)
	MintTimestamp     int64               `json:"mint_timestamp"`
	Anchor            *LightningAnchor    `json:"anchor,omitempty"` // Link to anchor
}

// LightningTransfer represents a transfer on Lightning Network
type LightningTransfer struct {
	AssetID          string  `json:"asset_id"`
	FromAgent        string  `json:"from_agent"`
	ToAgent          string  `json:"to_agent"`
	Amount           float64 `json:"amount"`           // Amount in asset units
	AmountSats       int64   `json:"amount_sats"`      // Amount in satoshis if applicable
	Invoice          string  `json:"invoice"`          // Bolt11 invoice
	PaymentHash      string  `json:"payment_hash"`
	Preimage         string  `json:"preimage"`         // Payment preimage (proof)
	Status           string  `json:"status"`           // "pending", "completed", "failed"
	Timestamp        int64   `json:"timestamp"`
	Fees             int64   `json:"fees"`             // Routing fees in msats
}

// Storable interface implementation for LightningAnchor
func (la *LightningAnchor) Put(stub shim.ChaincodeStubInterface, key string) error {
	model := LightningAnchorModel{
		ObjectType: "lightning_anchor",
		LightningAnchor: *la,
	}
	data, err := json.Marshal(model)
	if err != nil {
		return err
	}
	return stub.PutState(key, data)
}

func (la *LightningAnchor) Get(stub shim.ChaincodeStubInterface, key string) error {
	data, err := stub.GetState(key)
	if err != nil {
		return err
	}
	var model LightningAnchorModel
	err = json.Unmarshal(data, &model)
	if err != nil {
		return err
	}
	*la = model.LightningAnchor
	return nil
}

// AnchorStateOnBitcoin creates a cryptographic anchor of an SSM state on Bitcoin
func AnchorStateOnBitcoin(stub shim.ChaincodeStubInterface, state *State) (*LightningAnchor, error) {
	// Compute SHA256 hash of the state
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize state: %v", err)
	}

	hash := sha256.Sum256(stateJSON)
	stateHash := hex.EncodeToString(hash[:])

	// Create anchor structure
	anchor := &LightningAnchor{
		SessionID:     state.Session,
		StateHash:     stateHash,
		Timestamp:     time.Now().Unix(),
		AnchorType:    "hash_only",
		Confirmations: 0,
		Metadata:      make(map[string]string),
	}

	// Add state metadata
	anchor.Metadata["ssm"] = state.Ssm
	anchor.Metadata["iteration"] = fmt.Sprintf("%d", state.Iteration)
	anchor.Metadata["current_state"] = fmt.Sprintf("%d", state.Current)

	// Store anchor in ledger
	anchorKey := fmt.Sprintf("anchor_%s_%d", state.Session, state.Iteration)
	err = anchor.Put(stub, anchorKey)
	if err != nil {
		return nil, fmt.Errorf("failed to store anchor: %v", err)
	}

	// NOTE: Actual Bitcoin transaction would be sent by external Lightning Gateway
	// This function creates the anchor record that will be updated with txid later

	return anchor, nil
}

// UpdateAnchorWithBitcoinTx updates an anchor with Bitcoin transaction information
func UpdateAnchorWithBitcoinTx(stub shim.ChaincodeStubInterface, sessionID string, iteration int,
	bitcoinTxID string, blockHeight int64) error {

	anchorKey := fmt.Sprintf("anchor_%s_%d", sessionID, iteration)

	var anchor LightningAnchor
	err := anchor.Get(stub, anchorKey)
	if err != nil {
		return fmt.Errorf("anchor not found: %v", err)
	}

	anchor.BitcoinTxID = bitcoinTxID
	anchor.BlockHeight = blockHeight
	anchor.Confirmations = 1 // Initial confirmation

	err = anchor.Put(stub, anchorKey)
	if err != nil {
		return fmt.Errorf("failed to update anchor: %v", err)
	}

	return nil
}

// MintTaprootAsset creates a Taproot Asset from an ITMO commodity
func MintTaprootAsset(stub shim.ChaincodeStubInterface, itmo *ITMOCommodity, decimals int) (*ITMOTaprootAsset, error) {
	// Calculate supply amount based on CO2 quantity and decimals
	// Example: 1000 tCO2e with 3 decimals = 1,000,000 units (0.001 tCO2e per unit)
	supplyAmount := int64(itmo.QuantityTonsCO2e * float64(pow10(decimals)))

	// Generate deterministic asset ID based on ITMO data
	assetData := fmt.Sprintf("%s_%s_%s_%d",
		itmo.ID,
		itmo.ProjectType,
		itmo.CountryOfOrigin,
		itmo.VintageYear)

	hash := sha256.Sum256([]byte(assetData))
	assetID := hex.EncodeToString(hash[:16]) // Use first 16 bytes for ID

	// Create group key for fungibility (all ITMO tokens of same type are fungible)
	groupData := fmt.Sprintf("%s_%s", itmo.ProjectType, itmo.Methodology)
	groupHash := sha256.Sum256([]byte(groupData))
	groupKey := hex.EncodeToString(groupHash[:16])

	taprootAsset := &ITMOTaprootAsset{
		ITMOCommodity:  *itmo,
		TaprootAssetID: assetID,
		GroupKey:       groupKey,
		SupplyAmount:   supplyAmount,
		Decimals:       decimals,
		MintTimestamp:  time.Now().Unix(),
		MetadataURI:    fmt.Sprintf("ipfs://itmo/%s", itmo.ID),
	}

	// Store Taproot Asset metadata in ledger
	assetKey := fmt.Sprintf("taproot_asset_%s", assetID)
	assetJSON, err := json.Marshal(taprootAsset)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize taproot asset: %v", err)
	}

	err = stub.PutState(assetKey, assetJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to store taproot asset: %v", err)
	}

	// NOTE: Actual Taproot Asset minting would be done by Lightning Gateway
	// This function creates the metadata record

	return taprootAsset, nil
}

// TransferOnLightning records a Lightning Network transfer
func TransferOnLightning(stub shim.ChaincodeStubInterface, transfer *LightningTransfer) error {
	// Validate transfer
	if transfer.AssetID == "" || transfer.FromAgent == "" || transfer.ToAgent == "" {
		return fmt.Errorf("invalid transfer: missing required fields")
	}

	transfer.Timestamp = time.Now().Unix()
	transfer.Status = "pending"

	// Store transfer record
	transferKey := fmt.Sprintf("lightning_transfer_%s_%d", transfer.PaymentHash, transfer.Timestamp)
	transferJSON, err := json.Marshal(transfer)
	if err != nil {
		return fmt.Errorf("failed to serialize transfer: %v", err)
	}

	err = stub.PutState(transferKey, transferJSON)
	if err != nil {
		return fmt.Errorf("failed to store transfer: %v", err)
	}

	// NOTE: Actual Lightning payment would be handled by Lightning Gateway
	// This function creates the transfer record

	return nil
}

// VerifyAnchor verifies a Lightning anchor by checking Bitcoin blockchain
func VerifyAnchor(stub shim.ChaincodeStubInterface, sessionID string, iteration int) (bool, error) {
	anchorKey := fmt.Sprintf("anchor_%s_%d", sessionID, iteration)

	var anchor LightningAnchor
	err := anchor.Get(stub, anchorKey)
	if err != nil {
		return false, fmt.Errorf("anchor not found: %v", err)
	}

	// Check if anchor has Bitcoin transaction
	if anchor.BitcoinTxID == "" {
		return false, fmt.Errorf("anchor not yet committed to Bitcoin")
	}

	// Check confirmations (minimum 6 for final confirmation)
	if anchor.Confirmations < 6 {
		return false, fmt.Errorf("insufficient confirmations: %d/6", anchor.Confirmations)
	}

	// Verify state hash matches
	// NOTE: In production, this would query Bitcoin node to verify OP_RETURN data

	return true, nil
}

// GetAnchor retrieves an anchor by session and iteration
func GetAnchor(stub shim.ChaincodeStubInterface, sessionID string, iteration int) (*LightningAnchor, error) {
	anchorKey := fmt.Sprintf("anchor_%s_%d", sessionID, iteration)

	var anchor LightningAnchor
	err := anchor.Get(stub, anchorKey)
	if err != nil {
		return nil, err
	}

	return &anchor, nil
}

// GetTaprootAsset retrieves a Taproot Asset by ID
func GetTaprootAsset(stub shim.ChaincodeStubInterface, assetID string) (*ITMOTaprootAsset, error) {
	assetKey := fmt.Sprintf("taproot_asset_%s", assetID)

	data, err := stub.GetState(assetKey)
	if err != nil {
		return nil, err
	}

	var asset ITMOTaprootAsset
	err = json.Unmarshal(data, &asset)
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

// Helper function for power of 10
func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}

// QueryAnchorsBySession returns all anchors for a given session
func QueryAnchorsBySession(stub shim.ChaincodeStubInterface, sessionID string) ([]*LightningAnchor, error) {
	// Create query string for CouchDB
	queryString := fmt.Sprintf(`{
		"selector": {
			"docType": "lightning_anchor",
			"session": "%s"
		},
		"sort": [{"timestamp": "desc"}]
	}`, sessionID)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var anchors []*LightningAnchor
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var model LightningAnchorModel
		err = json.Unmarshal(queryResponse.Value, &model)
		if err != nil {
			return nil, err
		}

		anchors = append(anchors, &model.LightningAnchor)
	}

	return anchors, nil
}
