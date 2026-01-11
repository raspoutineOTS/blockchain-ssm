// Copyright Blockchain SSM Lightning Integration 2025
// License: Apache-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// StorageLayer represents different storage tiers in the hybrid system
type StorageLayer int

const (
	LAYER_BITCOIN     StorageLayer = iota // Bitcoin blockchain (anchors only)
	LAYER_LIGHTNING                       // Lightning Network (compact metadata)
	LAYER_HYPERLEDGER                     // Hyperledger Fabric (full states)
	LAYER_IPFS                            // IPFS/Arweave (document archive)
)

// HybridStorageStrategy defines the multi-layer storage approach
type HybridStorageStrategy struct {
	BitcoinAnchors     *BitcoinAnchorStore
	LightningMetadata  *LightningMetadataStore
	HyperledgerStates  *HyperledgerStateStore
	IPFSArchive        *IPFSArchiveStore
}

// StorageDecision determines which layer(s) to use for a given state
type StorageDecision struct {
	UseBitcoin     bool
	UseLightning   bool
	UseHyperledger bool
	UseIPFS        bool
	Reason         string
}

// NewHybridStorageStrategy creates a new hybrid storage strategy
func NewHybridStorageStrategy() *HybridStorageStrategy {
	return &HybridStorageStrategy{
		BitcoinAnchors:    NewBitcoinAnchorStore(),
		LightningMetadata: NewLightningMetadataStore(),
		HyperledgerStates: NewHyperledgerStateStore(),
		IPFSArchive:       NewIPFSArchiveStore(),
	}
}

// DecideStorageLayers determines which storage layers to use
func (hss *HybridStorageStrategy) DecideStorageLayers(state *State, context map[string]interface{}) *StorageDecision {
	decision := &StorageDecision{
		UseBitcoin:     false,
		UseLightning:   true, // Always use Lightning for fast transfers
		UseHyperledger: true, // Always use Hyperledger as source of truth
		UseIPFS:        false,
	}

	// Use Bitcoin anchoring for important milestones
	if shouldAnchorOnBitcoin(state, context) {
		decision.UseBitcoin = true
		decision.Reason = "Important milestone or periodic anchor"
	}

	// Use IPFS for large documents
	if hasLargeDocuments(state) {
		decision.UseIPFS = true
		decision.Reason += "; Large documents archived to IPFS"
	}

	return decision
}

// shouldAnchorOnBitcoin determines if state should be anchored on Bitcoin
func shouldAnchorOnBitcoin(state *State, context map[string]interface{}) bool {
	// Anchor every N iterations (e.g., every 10)
	if state.Iteration%10 == 0 {
		return true
	}

	// Anchor on final states
	if isFinalState(state) {
		return true
	}

	// Anchor if explicitly requested
	if context["force_anchor"] == true {
		return true
	}

	return false
}

// hasLargeDocuments checks if state contains large documents
func hasLargeDocuments(state *State) bool {
	// If public data is larger than 10KB, consider archiving
	return len(state.Public) > 10*1024
}

// isFinalState checks if state is in a final/acceptance state
func isFinalState(state *State) bool {
	// This would check against the SSM definition
	// For now, simple heuristic
	return state.Current == STATE_RETIRED || state.Current == STATE_TRANSFERRED
}

// BitcoinAnchorStore manages Bitcoin blockchain anchors
type BitcoinAnchorStore struct {
	anchors map[string]*BitcoinAnchorRecord
}

// BitcoinAnchorRecord represents an anchor on Bitcoin
type BitcoinAnchorRecord struct {
	StateHash      string
	TxID           string
	BlockHeight    int64
	Confirmations  int
	Timestamp      int64
	BatchSize      int // Number of states in this batch
}

// NewBitcoinAnchorStore creates a new Bitcoin anchor store
func NewBitcoinAnchorStore() *BitcoinAnchorStore {
	return &BitcoinAnchorStore{
		anchors: make(map[string]*BitcoinAnchorRecord),
	}
}

// StoreAnchor stores a Bitcoin anchor
func (bas *BitcoinAnchorStore) StoreAnchor(sessionID string, iteration int, record *BitcoinAnchorRecord) error {
	key := fmt.Sprintf("%s:%d", sessionID, iteration)
	bas.anchors[key] = record
	return nil
}

// GetAnchor retrieves a Bitcoin anchor
func (bas *BitcoinAnchorStore) GetAnchor(sessionID string, iteration int) (*BitcoinAnchorRecord, error) {
	key := fmt.Sprintf("%s:%d", sessionID, iteration)
	record, exists := bas.anchors[key]
	if !exists {
		return nil, fmt.Errorf("anchor not found")
	}
	return record, nil
}

// BatchAnchors creates a single Bitcoin transaction for multiple states
func (bas *BitcoinAnchorStore) BatchAnchors(states []*State) (*BitcoinAnchorRecord, error) {
	// Create Merkle root of all state hashes
	var hashes []string
	for _, state := range states {
		hash, _ := GenerateStateHash(state)
		hashes = append(hashes, hash)
	}

	merkleRoot := calculateMerkleRoot(hashes)

	record := &BitcoinAnchorRecord{
		StateHash:     merkleRoot,
		TxID:          "pending", // Would be set after Bitcoin tx
		BlockHeight:   0,
		Confirmations: 0,
		Timestamp:     time.Now().Unix(),
		BatchSize:     len(states),
	}

	return record, nil
}

// calculateMerkleRoot computes Merkle root of hashes
func calculateMerkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return ""
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	// Simple implementation: hash concatenation
	// In production, use proper Merkle tree
	combined := ""
	for _, h := range hashes {
		combined += h
	}

	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// LightningMetadataStore manages Lightning Network metadata
type LightningMetadataStore struct {
	metadata map[string]*LightningMetadataRecord
}

// LightningMetadataRecord represents metadata on Lightning
type LightningMetadataRecord struct {
	CompactAnchor  *CompactLightningAnchor
	Invoice        string
	PaymentHash    string
	Preimage       string
	Status         string
	AmountMsat     int64
	CreatedAt      int64
}

// NewLightningMetadataStore creates a new Lightning metadata store
func NewLightningMetadataStore() *LightningMetadataStore {
	return &LightningMetadataStore{
		metadata: make(map[string]*LightningMetadataRecord),
	}
}

// StoreMetadata stores Lightning metadata
func (lms *LightningMetadataStore) StoreMetadata(paymentHash string, record *LightningMetadataRecord) error {
	lms.metadata[paymentHash] = record
	return nil
}

// GetMetadata retrieves Lightning metadata
func (lms *LightningMetadataStore) GetMetadata(paymentHash string) (*LightningMetadataRecord, error) {
	record, exists := lms.metadata[paymentHash]
	if !exists {
		return nil, fmt.Errorf("metadata not found")
	}
	return record, nil
}

// HyperledgerStateStore manages Hyperledger Fabric states
type HyperledgerStateStore struct {
	states map[string]*State
}

// NewHyperledgerStateStore creates a new Hyperledger state store
func NewHyperledgerStateStore() *HyperledgerStateStore {
	return &HyperledgerStateStore{
		states: make(map[string]*State),
	}
}

// StoreState stores a full state in Hyperledger
func (hss *HyperledgerStateStore) StoreState(state *State) error {
	key := fmt.Sprintf("%s:%d", state.Session, state.Iteration)
	hss.states[key] = state
	return nil
}

// GetState retrieves a full state from Hyperledger
func (hss *HyperledgerStateStore) GetState(sessionID string, iteration int) (*State, error) {
	key := fmt.Sprintf("%s:%d", sessionID, iteration)
	state, exists := hss.states[key]
	if !exists {
		return nil, fmt.Errorf("state not found")
	}
	return state, nil
}

// IPFSArchiveStore manages IPFS/Arweave document archive
type IPFSArchiveStore struct {
	documents map[string]*IPFSDocument
}

// IPFSDocument represents a document on IPFS
type IPFSDocument struct {
	CID         string // IPFS Content Identifier
	Size        int64
	ContentType string
	Timestamp   int64
	Metadata    map[string]string
}

// NewIPFSArchiveStore creates a new IPFS archive store
func NewIPFSArchiveStore() *IPFSArchiveStore {
	return &IPFSArchiveStore{
		documents: make(map[string]*IPFSDocument),
	}
}

// StoreDocument stores a document on IPFS
func (ias *IPFSArchiveStore) StoreDocument(sessionID string, data []byte, contentType string) (*IPFSDocument, error) {
	// In production, this would upload to IPFS
	// For now, simulate with hash
	hash := sha256.Sum256(data)
	cid := "Qm" + hex.EncodeToString(hash[:16])

	doc := &IPFSDocument{
		CID:         cid,
		Size:        int64(len(data)),
		ContentType: contentType,
		Timestamp:   time.Now().Unix(),
		Metadata: map[string]string{
			"session": sessionID,
		},
	}

	ias.documents[cid] = doc
	return doc, nil
}

// GetDocument retrieves a document from IPFS
func (ias *IPFSArchiveStore) GetDocument(cid string) (*IPFSDocument, error) {
	doc, exists := ias.documents[cid]
	if !exists {
		return nil, fmt.Errorf("document not found")
	}
	return doc, nil
}

// HybridStorageWorkflow orchestrates storage across all layers
type HybridStorageWorkflow struct {
	strategy *HybridStorageStrategy
}

// NewHybridStorageWorkflow creates a new hybrid storage workflow
func NewHybridStorageWorkflow() *HybridStorageWorkflow {
	return &HybridStorageWorkflow{
		strategy: NewHybridStorageStrategy(),
	}
}

// StoreState stores a state across appropriate layers
func (hsw *HybridStorageWorkflow) StoreState(state *State, context map[string]interface{}) error {
	// Decide which layers to use
	decision := hsw.strategy.DecideStorageLayers(state, context)

	// Always store in Hyperledger (source of truth)
	if decision.UseHyperledger {
		err := hsw.strategy.HyperledgerStates.StoreState(state)
		if err != nil {
			return fmt.Errorf("failed to store in Hyperledger: %v", err)
		}
	}

	// Store compact metadata on Lightning
	if decision.UseLightning {
		sessionCounter := GetSessionCounter(state.Session)
		compactAnchor, err := CreateCompactAnchor(state, sessionCounter)
		if err != nil {
			return fmt.Errorf("failed to create compact anchor: %v", err)
		}

		record := &LightningMetadataRecord{
			CompactAnchor: compactAnchor,
			Status:        "pending",
			CreatedAt:     time.Now().Unix(),
		}

		// Would create Lightning invoice here
		// For now, just store the metadata
		paymentHash := compactAnchor.H[:16] // Use first 16 chars of hash
		err = hsw.strategy.LightningMetadata.StoreMetadata(paymentHash, record)
		if err != nil {
			return fmt.Errorf("failed to store Lightning metadata: %v", err)
		}
	}

	// Anchor on Bitcoin if needed
	if decision.UseBitcoin {
		stateHash, _ := GenerateStateHash(state)
		record := &BitcoinAnchorRecord{
			StateHash:     stateHash,
			TxID:          "pending",
			Timestamp:     time.Now().Unix(),
			BatchSize:     1,
		}

		err := hsw.strategy.BitcoinAnchors.StoreAnchor(state.Session, state.Iteration, record)
		if err != nil {
			return fmt.Errorf("failed to store Bitcoin anchor: %v", err)
		}
	}

	// Archive large documents to IPFS
	if decision.UseIPFS && len(state.Public) > 0 {
		_, err := hsw.strategy.IPFSArchive.StoreDocument(
			state.Session,
			[]byte(state.Public),
			"application/json",
		)
		if err != nil {
			return fmt.Errorf("failed to archive to IPFS: %v", err)
		}
	}

	return nil
}

// RetrieveState retrieves a state from the most appropriate layer
func (hsw *HybridStorageWorkflow) RetrieveState(sessionID string, iteration int, preferredLayer StorageLayer) (*State, error) {
	// Try preferred layer first
	switch preferredLayer {
	case LAYER_HYPERLEDGER:
		return hsw.strategy.HyperledgerStates.GetState(sessionID, iteration)

	case LAYER_LIGHTNING:
		// Would reconstruct from compact anchor + database lookup
		return nil, fmt.Errorf("Lightning layer reconstruction not implemented")

	case LAYER_BITCOIN:
		// Would verify anchor then fetch full state from Hyperledger
		return nil, fmt.Errorf("Bitcoin layer reconstruction not implemented")

	default:
		// Fall back to Hyperledger
		return hsw.strategy.HyperledgerStates.GetState(sessionID, iteration)
	}
}

// StorageCostAnalysis analyzes storage costs across layers
type StorageCostAnalysis struct {
	HyperledgerCost float64 `json:"hyperledger_cost_usd"`
	LightningCost   float64 `json:"lightning_cost_usd"`
	BitcoinCost     float64 `json:"bitcoin_cost_usd"`
	IPFSCost        float64 `json:"ipfs_cost_usd"`
	TotalCost       float64 `json:"total_cost_usd"`
}

// AnalyzeStorageCosts calculates storage costs
func AnalyzeStorageCosts(state *State, decision *StorageDecision) *StorageCostAnalysis {
	analysis := &StorageCostAnalysis{}

	fullStateSize := FullStateSize(state)

	// Hyperledger: $0.0001 per KB
	if decision.UseHyperledger {
		analysis.HyperledgerCost = float64(fullStateSize) / 1000.0 * 0.0001
	}

	// Lightning: $0.001 per KB (compact anchor is ~100 bytes)
	if decision.UseLightning {
		compactSize := 100 // Approximate size
		analysis.LightningCost = float64(compactSize) / 1000.0 * 0.001
	}

	// Bitcoin: $3-5 per transaction (amortized over batch)
	if decision.UseBitcoin {
		analysis.BitcoinCost = 4.0 / 10.0 // Assume batch of 10
	}

	// IPFS: $0.10 per MB
	if decision.UseIPFS {
		analysis.IPFSCost = float64(fullStateSize) / 1000000.0 * 0.10
	}

	analysis.TotalCost = analysis.HyperledgerCost +
		analysis.LightningCost +
		analysis.BitcoinCost +
		analysis.IPFSCost

	return analysis
}
