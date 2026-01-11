// Copyright Blockchain SSM Lightning Integration 2025
// License: Apache-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// ITMOTaprootBridge provides conversion and synchronization between ITMO and Taproot Assets
type ITMOTaprootBridge struct {
	MappingID        string    `json:"mapping_id"`
	ITMOid           string    `json:"itmo_id"`
	TaprootAssetID   string    `json:"taproot_asset_id"`
	HyperledgerState string    `json:"hyperledger_state"` // Current SSM state
	LightningState   string    `json:"lightning_state"`   // "minted", "distributed", "traded"
	SyncStatus       string    `json:"sync_status"`       // "synced", "pending_sync", "conflict"
	LastSync         time.Time `json:"last_sync"`
	ConflictReason   string    `json:"conflict_reason,omitempty"`
}

// ConversionConfig defines parameters for ITMO to Taproot conversion
type ConversionConfig struct {
	Decimals             int     `json:"decimals"`               // Token decimals (e.g., 3 = 0.001 tCO2e)
	MinimumUnit          float64 `json:"minimum_unit"`           // Minimum tradeable unit in tCO2e
	EnableFractional     bool    `json:"enable_fractional"`      // Allow fractional trading
	MetadataStorageType  string  `json:"metadata_storage_type"`  // "ipfs", "arweave", "hyperledger"
	AutoAnchorOnMint     bool    `json:"auto_anchor_on_mint"`    // Anchor to Bitcoin on minting
	RequireVerification  bool    `json:"require_verification"`   // Require ITMO verification before minting
}

// DefaultConversionConfig returns sensible defaults
func DefaultConversionConfig() ConversionConfig {
	return ConversionConfig{
		Decimals:            3,       // 0.001 tCO2e precision
		MinimumUnit:         0.001,   // 1 kg CO2e
		EnableFractional:    true,
		MetadataStorageType: "ipfs",
		AutoAnchorOnMint:    true,
		RequireVerification: true,
	}
}

// ConvertITMOToTaprootAsset converts an ITMO commodity to a Taproot Asset specification
func ConvertITMOToTaprootAsset(itmo ITMOCommodity, config ConversionConfig) (*ITMOTaprootAsset, error) {
	// Validate ITMO
	if itmo.ID == "" {
		return nil, fmt.Errorf("ITMO ID is required")
	}

	if config.RequireVerification && itmo.VerificationStatus != "Verified" {
		return nil, fmt.Errorf("ITMO must be verified before tokenization (status: %s)", itmo.VerificationStatus)
	}

	// Check minimum unit
	if config.MinimumUnit > 0 && itmo.QuantityTonsCO2e < config.MinimumUnit {
		return nil, fmt.Errorf("ITMO quantity %.3f tCO2e below minimum %.3f tCO2e",
			itmo.QuantityTonsCO2e, config.MinimumUnit)
	}

	// Calculate supply amount
	supplyAmount := int64(itmo.QuantityTonsCO2e * float64(pow10(config.Decimals)))

	// Generate deterministic Taproot Asset ID
	assetID := generateAssetID(itmo)

	// Generate group key for fungibility
	groupKey := generateGroupKey(itmo.ProjectType, itmo.Methodology)

	// Create metadata URI
	var metadataURI string
	switch config.MetadataStorageType {
	case "ipfs":
		metadataURI = fmt.Sprintf("ipfs://itmo/%s", itmo.ID)
	case "arweave":
		metadataURI = fmt.Sprintf("ar://itmo/%s", itmo.ID)
	case "hyperledger":
		metadataURI = fmt.Sprintf("hyperledger://ssm/itmo/%s", itmo.ID)
	default:
		metadataURI = fmt.Sprintf("ipfs://itmo/%s", itmo.ID)
	}

	taprootAsset := &ITMOTaprootAsset{
		ITMOCommodity:  itmo,
		TaprootAssetID: assetID,
		GroupKey:       groupKey,
		SupplyAmount:   supplyAmount,
		Decimals:       config.Decimals,
		MintTimestamp:  time.Now().Unix(),
		MetadataURI:    metadataURI,
	}

	return taprootAsset, nil
}

// SyncTaprootToHyperledger synchronizes Taproot Asset state back to Hyperledger
func SyncTaprootToHyperledger(asset *ITMOTaprootAsset, lightningTransfers []LightningTransfer) (*ITMOTaprootBridge, error) {
	bridge := &ITMOTaprootBridge{
		MappingID:      generateMappingID(asset.ID, asset.TaprootAssetID),
		ITMOid:         asset.ID,
		TaprootAssetID: asset.TaprootAssetID,
		LastSync:       time.Now(),
	}

	// Determine Lightning state based on transfers
	if len(lightningTransfers) == 0 {
		bridge.LightningState = "minted"
	} else {
		// Check if all transfers are completed
		allCompleted := true
		for _, transfer := range lightningTransfers {
			if transfer.Status != "completed" {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			bridge.LightningState = "traded"
		} else {
			bridge.LightningState = "distributed"
		}
	}

	// Calculate ownership from transfers
	ownership := calculateOwnership(asset, lightningTransfers)

	// Create sync report
	bridge.HyperledgerState = fmt.Sprintf("ITMO %s tokenized as %s with %d owners",
		asset.ID, asset.TaprootAssetID, len(ownership))

	// Check for conflicts
	totalDistributed := int64(0)
	for _, amount := range ownership {
		totalDistributed += amount
	}

	if totalDistributed > asset.SupplyAmount {
		bridge.SyncStatus = "conflict"
		bridge.ConflictReason = fmt.Sprintf("Distributed amount %d exceeds supply %d",
			totalDistributed, asset.SupplyAmount)
	} else {
		bridge.SyncStatus = "synced"
	}

	return bridge, nil
}

// calculateOwnership determines current token ownership from transfers
func calculateOwnership(asset *ITMOTaprootAsset, transfers []LightningTransfer) map[string]int64 {
	ownership := make(map[string]int64)

	// Initial minter owns all tokens
	// In production, this would come from the minting transaction
	initialOwner := "minter" // Placeholder
	ownership[initialOwner] = asset.SupplyAmount

	// Apply transfers
	for _, transfer := range transfers {
		if transfer.Status == "completed" {
			amount := int64(transfer.Amount * float64(pow10(asset.Decimals)))

			// Deduct from sender
			if ownership[transfer.FromAgent] >= amount {
				ownership[transfer.FromAgent] -= amount
			}

			// Add to recipient
			ownership[transfer.ToAgent] += amount
		}
	}

	// Remove zero balances
	for agent, balance := range ownership {
		if balance == 0 {
			delete(ownership, agent)
		}
	}

	return ownership
}

// generateAssetID creates a deterministic asset ID from ITMO data
func generateAssetID(itmo ITMOCommodity) string {
	data := fmt.Sprintf("%s_%s_%s_%d_%f",
		itmo.ID,
		itmo.ProjectType,
		itmo.CountryOfOrigin,
		itmo.VintageYear,
		itmo.QuantityTonsCO2e)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // 32 character hex string
}

// generateGroupKey creates a group key for fungible tokens
func generateGroupKey(projectType, methodology string) string {
	data := fmt.Sprintf("%s_%s", projectType, methodology)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

// generateMappingID creates a unique mapping identifier
func generateMappingID(itmoID, assetID string) string {
	data := fmt.Sprintf("%s_%s", itmoID, assetID)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// ValidateITMOForTokenization checks if an ITMO is ready for tokenization
func ValidateITMOForTokenization(itmo ITMOCommodity) (bool, []string) {
	var errors []string

	// Check required fields
	if itmo.ID == "" {
		errors = append(errors, "ITMO ID is required")
	}

	if itmo.QuantityTonsCO2e <= 0 {
		errors = append(errors, fmt.Sprintf("Invalid quantity: %.2f", itmo.QuantityTonsCO2e))
	}

	if itmo.ProjectType == "" {
		errors = append(errors, "Project type is required")
	}

	if itmo.Methodology == "" {
		errors = append(errors, "Methodology is required")
	}

	if itmo.CountryOfOrigin == "" {
		errors = append(errors, "Country of origin is required")
	}

	if itmo.VintageYear < 2000 || itmo.VintageYear > time.Now().Year() {
		errors = append(errors, fmt.Sprintf("Invalid vintage year: %d", itmo.VintageYear))
	}

	// Check verification status
	validStatuses := map[string]bool{
		"Verified":       true,
		"Pending":        false,
		"Failed":         false,
		"Under Review":   false,
	}

	if verified, exists := validStatuses[itmo.VerificationStatus]; !exists {
		errors = append(errors, fmt.Sprintf("Unknown verification status: %s", itmo.VerificationStatus))
	} else if !verified {
		errors = append(errors, fmt.Sprintf("ITMO not verified (status: %s)", itmo.VerificationStatus))
	}

	return len(errors) == 0, errors
}

// CreateMetadataJSON generates JSON metadata for IPFS/Arweave storage
func CreateMetadataJSON(asset *ITMOTaprootAsset) (string, error) {
	metadata := map[string]interface{}{
		"name":        fmt.Sprintf("Carbon Credit ITMO %s", asset.ID),
		"description": fmt.Sprintf("Tokenized carbon credit from %s project (%s methodology)",
			asset.ProjectType, asset.Methodology),
		"image": fmt.Sprintf("ipfs://Qm.../itmo_%s.png", asset.ID),
		"attributes": []map[string]interface{}{
			{
				"trait_type": "Project Type",
				"value":      asset.ProjectType,
			},
			{
				"trait_type": "Methodology",
				"value":      asset.Methodology,
			},
			{
				"trait_type": "Country",
				"value":      asset.CountryOfOrigin,
			},
			{
				"trait_type": "Vintage Year",
				"value":      asset.VintageYear,
			},
			{
				"trait_type": "Quantity (tCO2e)",
				"value":      asset.QuantityTonsCO2e,
				"display_type": "number",
			},
			{
				"trait_type": "Verification Status",
				"value":      asset.VerificationStatus,
			},
		},
		"properties": map[string]interface{}{
			"itmo_id":          asset.ID,
			"taproot_asset_id": asset.TaprootAssetID,
			"group_key":        asset.GroupKey,
			"supply":           asset.SupplyAmount,
			"decimals":         asset.Decimals,
			"mint_timestamp":   asset.MintTimestamp,
		},
	}

	jsonBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// EstimateTokenizationCost estimates the cost of tokenizing an ITMO
func EstimateTokenizationCost(itmo ITMOCommodity) map[string]float64 {
	costs := make(map[string]float64)

	// Bitcoin transaction fee for minting (approximate)
	costs["bitcoin_mint_tx"] = 5.0 // USD

	// IPFS pinning cost (approximate)
	costs["ipfs_metadata"] = 0.10 // USD

	// Lightning channel capacity requirement (in USD, based on asset value)
	// Assume $10 per tCO2e
	costs["lightning_channel_capacity"] = itmo.QuantityTonsCO2e * 10.0

	// Total
	total := 0.0
	for _, cost := range costs {
		total += cost
	}
	costs["total"] = total

	return costs
}

// Helper function (duplicate from lightning-anchor.go for standalone use)
func pow10(n int) int {
	result := 1
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}
