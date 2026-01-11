// Copyright Blockchain SSM Lightning Integration 2025
// License: Apache-2.0

package main

import (
	"encoding/json"
	"testing"
)

// TestConvertITMOToTaprootAsset tests ITMO to Taproot Asset conversion
func TestConvertITMOToTaprootAsset(t *testing.T) {
	itmo := ITMOCommodity{
		ID:                 "ITMO-TEST-001",
		QuantityTonsCO2e:   100.5,
		ProjectType:        "Solar Energy",
		Methodology:        "CDM ACM0002",
		CountryOfOrigin:    "Brazil",
		VintageYear:        2024,
		VerificationStatus: "Verified",
	}

	config := DefaultConversionConfig()

	asset, err := ConvertITMOToTaprootAsset(itmo, config)
	if err != nil {
		t.Fatalf("Failed to convert ITMO: %v", err)
	}

	// Verify supply calculation: 100.5 tCO2e * 1000 (3 decimals) = 100,500 units
	expectedSupply := int64(100500)
	if asset.SupplyAmount != expectedSupply {
		t.Errorf("Expected supply %d, got %d", expectedSupply, asset.SupplyAmount)
	}

	// Verify decimals
	if asset.Decimals != 3 {
		t.Errorf("Expected 3 decimals, got %d", asset.Decimals)
	}

	// Verify asset ID is generated
	if asset.TaprootAssetID == "" {
		t.Error("Asset ID should not be empty")
	}

	// Verify group key is generated
	if asset.GroupKey == "" {
		t.Error("Group key should not be empty")
	}

	// Verify metadata URI
	expectedURI := "ipfs://itmo/ITMO-TEST-001"
	if asset.MetadataURI != expectedURI {
		t.Errorf("Expected URI %s, got %s", expectedURI, asset.MetadataURI)
	}
}

// TestValidateITMOForTokenization tests ITMO validation
func TestValidateITMOForTokenization(t *testing.T) {
	tests := []struct {
		name          string
		itmo          ITMOCommodity
		shouldBeValid bool
		expectedError string
	}{
		{
			name: "Valid ITMO",
			itmo: ITMOCommodity{
				ID:                 "ITMO-VALID-001",
				QuantityTonsCO2e:   50.0,
				ProjectType:        "Wind Energy",
				Methodology:        "CDM AM0077",
				CountryOfOrigin:    "Germany",
				VintageYear:        2024,
				VerificationStatus: "Verified",
			},
			shouldBeValid: true,
		},
		{
			name: "Missing ID",
			itmo: ITMOCommodity{
				QuantityTonsCO2e:   50.0,
				ProjectType:        "Wind Energy",
				Methodology:        "CDM AM0077",
				CountryOfOrigin:    "Germany",
				VintageYear:        2024,
				VerificationStatus: "Verified",
			},
			shouldBeValid: false,
			expectedError: "ITMO ID is required",
		},
		{
			name: "Invalid quantity",
			itmo: ITMOCommodity{
				ID:                 "ITMO-INVALID-001",
				QuantityTonsCO2e:   -10.0,
				ProjectType:        "Wind Energy",
				Methodology:        "CDM AM0077",
				CountryOfOrigin:    "Germany",
				VintageYear:        2024,
				VerificationStatus: "Verified",
			},
			shouldBeValid: false,
			expectedError: "Invalid quantity",
		},
		{
			name: "Not verified",
			itmo: ITMOCommodity{
				ID:                 "ITMO-PENDING-001",
				QuantityTonsCO2e:   50.0,
				ProjectType:        "Wind Energy",
				Methodology:        "CDM AM0077",
				CountryOfOrigin:    "Germany",
				VintageYear:        2024,
				VerificationStatus: "Pending",
			},
			shouldBeValid: false,
			expectedError: "ITMO not verified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errors := ValidateITMOForTokenization(tt.itmo)

			if valid != tt.shouldBeValid {
				t.Errorf("Expected valid=%v, got valid=%v. Errors: %v",
					tt.shouldBeValid, valid, errors)
			}

			if !tt.shouldBeValid && len(errors) == 0 {
				t.Error("Expected validation errors, got none")
			}
		})
	}
}

// TestCalculateOwnership tests ownership calculation from transfers
func TestCalculateOwnership(t *testing.T) {
	asset := &ITMOTaprootAsset{
		ITMOCommodity: ITMOCommodity{
			ID:               "ITMO-OWNERSHIP-001",
			QuantityTonsCO2e: 100.0,
		},
		TaprootAssetID: "asset123",
		SupplyAmount:   100000, // 100.000 units
		Decimals:       3,
	}

	transfers := []LightningTransfer{
		{
			FromAgent: "minter",
			ToAgent:   "Alice",
			Amount:    30.0, // 30 tCO2e
			Status:    "completed",
		},
		{
			FromAgent: "Alice",
			ToAgent:   "Bob",
			Amount:    10.0, // 10 tCO2e
			Status:    "completed",
		},
		{
			FromAgent: "minter",
			ToAgent:   "Carol",
			Amount:    20.0, // 20 tCO2e
			Status:    "pending", // Not completed, should not affect ownership
		},
	}

	ownership := calculateOwnership(asset, transfers)

	// Expected:
	// minter: 100,000 - 30,000 = 70,000 units
	// Alice: 30,000 - 10,000 = 20,000 units
	// Bob: 10,000 units
	// Carol: 0 (transfer pending)

	expectedOwnership := map[string]int64{
		"minter": 70000,
		"Alice":  20000,
		"Bob":    10000,
	}

	for agent, expectedAmount := range expectedOwnership {
		actualAmount, exists := ownership[agent]
		if !exists {
			t.Errorf("Expected %s to have ownership, not found", agent)
			continue
		}
		if actualAmount != expectedAmount {
			t.Errorf("Agent %s: expected %d units, got %d",
				agent, expectedAmount, actualAmount)
		}
	}

	// Carol should not be in ownership
	if _, exists := ownership["Carol"]; exists {
		t.Error("Carol should not have ownership (transfer pending)")
	}
}

// TestCreateMetadataJSON tests metadata JSON generation
func TestCreateMetadataJSON(t *testing.T) {
	asset := &ITMOTaprootAsset{
		ITMOCommodity: ITMOCommodity{
			ID:                 "ITMO-META-001",
			QuantityTonsCO2e:   50.0,
			ProjectType:        "Hydroelectric",
			Methodology:        "CDM ACM0123",
			CountryOfOrigin:    "Norway",
			VintageYear:        2024,
			VerificationStatus: "Verified",
		},
		TaprootAssetID: "meta123",
		GroupKey:       "hydro_group",
		SupplyAmount:   50000,
		Decimals:       3,
		MintTimestamp:  1700000000,
	}

	metadataJSON, err := CreateMetadataJSON(asset)
	if err != nil {
		t.Fatalf("Failed to create metadata JSON: %v", err)
	}

	// Parse JSON to verify structure
	var metadata map[string]interface{}
	err = json.Unmarshal([]byte(metadataJSON), &metadata)
	if err != nil {
		t.Fatalf("Failed to parse metadata JSON: %v", err)
	}

	// Verify required fields
	if metadata["name"] == nil {
		t.Error("Metadata should have 'name' field")
	}

	if metadata["description"] == nil {
		t.Error("Metadata should have 'description' field")
	}

	attributes, ok := metadata["attributes"].([]interface{})
	if !ok || len(attributes) == 0 {
		t.Error("Metadata should have 'attributes' array")
	}

	properties, ok := metadata["properties"].(map[string]interface{})
	if !ok {
		t.Error("Metadata should have 'properties' object")
	}

	// Verify properties content
	if properties["itmo_id"] != "ITMO-META-001" {
		t.Errorf("Expected itmo_id 'ITMO-META-001', got %v", properties["itmo_id"])
	}

	if properties["taproot_asset_id"] != "meta123" {
		t.Errorf("Expected taproot_asset_id 'meta123', got %v", properties["taproot_asset_id"])
	}
}

// TestEstimateTokenizationCost tests cost estimation
func TestEstimateTokenizationCost(t *testing.T) {
	itmo := ITMOCommodity{
		ID:               "ITMO-COST-001",
		QuantityTonsCO2e: 1000.0,
	}

	costs := EstimateTokenizationCost(itmo)

	// Verify all cost components exist
	expectedComponents := []string{
		"bitcoin_mint_tx",
		"ipfs_metadata",
		"lightning_channel_capacity",
		"total",
	}

	for _, component := range expectedComponents {
		if _, exists := costs[component]; !exists {
			t.Errorf("Cost component '%s' missing", component)
		}
	}

	// Verify total is sum of components
	expectedTotal := costs["bitcoin_mint_tx"] +
		costs["ipfs_metadata"] +
		costs["lightning_channel_capacity"]

	if costs["total"] != expectedTotal {
		t.Errorf("Total cost mismatch: expected %.2f, got %.2f",
			expectedTotal, costs["total"])
	}

	// Verify lightning capacity scales with ITMO quantity
	// Assumption: $10 per tCO2e
	expectedCapacity := 1000.0 * 10.0
	if costs["lightning_channel_capacity"] != expectedCapacity {
		t.Errorf("Expected capacity %.2f, got %.2f",
			expectedCapacity, costs["lightning_channel_capacity"])
	}
}

// TestSyncTaprootToHyperledger tests synchronization logic
func TestSyncTaprootToHyperledger(t *testing.T) {
	asset := &ITMOTaprootAsset{
		ITMOCommodity: ITMOCommodity{
			ID:               "ITMO-SYNC-001",
			QuantityTonsCO2e: 100.0,
		},
		TaprootAssetID: "sync123",
		SupplyAmount:   100000,
		Decimals:       3,
	}

	transfers := []LightningTransfer{
		{
			FromAgent: "minter",
			ToAgent:   "Alice",
			Amount:    50.0,
			Status:    "completed",
		},
	}

	bridge, err := SyncTaprootToHyperledger(asset, transfers)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify bridge created
	if bridge.ITMOid != "ITMO-SYNC-001" {
		t.Errorf("Expected ITMO ID 'ITMO-SYNC-001', got %s", bridge.ITMOid)
	}

	if bridge.TaprootAssetID != "sync123" {
		t.Errorf("Expected asset ID 'sync123', got %s", bridge.TaprootAssetID)
	}

	// Verify state
	if bridge.LightningState != "traded" {
		t.Errorf("Expected state 'traded', got %s", bridge.LightningState)
	}

	// Verify sync status (should be synced)
	if bridge.SyncStatus != "synced" {
		t.Errorf("Expected sync status 'synced', got %s (reason: %s)",
			bridge.SyncStatus, bridge.ConflictReason)
	}
}

// TestSyncConflictDetection tests conflict detection in sync
func TestSyncConflictDetection(t *testing.T) {
	asset := &ITMOTaprootAsset{
		ITMOCommodity: ITMOCommodity{
			ID:               "ITMO-CONFLICT-001",
			QuantityTonsCO2e: 100.0,
		},
		TaprootAssetID: "conflict123",
		SupplyAmount:   100000, // 100.000 units
		Decimals:       3,
	}

	// Transfers that exceed supply (conflict)
	transfers := []LightningTransfer{
		{
			FromAgent: "minter",
			ToAgent:   "Alice",
			Amount:    80.0,
			Status:    "completed",
		},
		{
			FromAgent: "minter",
			ToAgent:   "Bob",
			Amount:    50.0, // Total: 130 > 100 supply
			Status:    "completed",
		},
	}

	bridge, err := SyncTaprootToHyperledger(asset, transfers)
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Should detect conflict
	if bridge.SyncStatus != "conflict" {
		t.Errorf("Expected sync status 'conflict', got %s", bridge.SyncStatus)
	}

	if bridge.ConflictReason == "" {
		t.Error("Conflict reason should be provided")
	}
}

// TestGenerateAssetID tests deterministic asset ID generation
func TestGenerateAssetID(t *testing.T) {
	itmo1 := ITMOCommodity{
		ID:               "ITMO-001",
		ProjectType:      "Solar",
		CountryOfOrigin:  "US",
		VintageYear:      2024,
		QuantityTonsCO2e: 100.0,
	}

	itmo2 := ITMOCommodity{
		ID:               "ITMO-001",
		ProjectType:      "Solar",
		CountryOfOrigin:  "US",
		VintageYear:      2024,
		QuantityTonsCO2e: 100.0,
	}

	itmo3 := ITMOCommodity{
		ID:               "ITMO-002", // Different ID
		ProjectType:      "Solar",
		CountryOfOrigin:  "US",
		VintageYear:      2024,
		QuantityTonsCO2e: 100.0,
	}

	id1 := generateAssetID(itmo1)
	id2 := generateAssetID(itmo2)
	id3 := generateAssetID(itmo3)

	// Same ITMO should produce same asset ID (deterministic)
	if id1 != id2 {
		t.Error("Same ITMO should produce same asset ID")
	}

	// Different ITMO should produce different asset ID
	if id1 == id3 {
		t.Error("Different ITMO should produce different asset ID")
	}

	// Asset ID should be 32 characters (16 bytes hex)
	if len(id1) != 32 {
		t.Errorf("Asset ID should be 32 characters, got %d", len(id1))
	}
}

// TestGenerateGroupKey tests group key generation
func TestGenerateGroupKey(t *testing.T) {
	key1 := generateGroupKey("Solar", "CDM ACM0002")
	key2 := generateGroupKey("Solar", "CDM ACM0002")
	key3 := generateGroupKey("Wind", "CDM ACM0002")

	// Same inputs should produce same key
	if key1 != key2 {
		t.Error("Same inputs should produce same group key")
	}

	// Different inputs should produce different key
	if key1 == key3 {
		t.Error("Different inputs should produce different group key")
	}

	// Group key should be 32 characters
	if len(key1) != 32 {
		t.Errorf("Group key should be 32 characters, got %d", len(key1))
	}
}

// BenchmarkConvertITMOToTaprootAsset benchmarks conversion performance
func BenchmarkConvertITMOToTaprootAsset(b *testing.B) {
	itmo := ITMOCommodity{
		ID:                 "ITMO-BENCH-001",
		QuantityTonsCO2e:   100.0,
		ProjectType:        "Solar Energy",
		Methodology:        "CDM ACM0002",
		CountryOfOrigin:    "Brazil",
		VintageYear:        2024,
		VerificationStatus: "Verified",
	}

	config := DefaultConversionConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ConvertITMOToTaprootAsset(itmo, config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestCreateCompactAnchor tests compact anchor creation
func TestCreateCompactAnchor(t *testing.T) {
	state := &State{
		Session:   "carbon_credit_001",
		Ssm:       "CarbonCreditLifecycle",
		Iteration: 42,
		Limit:     100,
		Current:   STATE_ACCEPTED,
		Roles: map[string]string{
			"Alice": "Seller",
			"Bob":   "Buyer",
		},
		Public: `{"itmo_id": "ITMO-2025-001", "quantity": 500}`,
		Origin: Transition{
			From:   STATE_PROPOSED,
			To:     STATE_ACCEPTED,
			Role:   "Buyer",
			Action: "Accept",
		},
	}

	sessionCounter := uint16(1)

	anchor, err := CreateCompactAnchor(state, sessionCounter)
	if err != nil {
		t.Fatalf("Failed to create compact anchor: %v", err)
	}

	// Verify hash is generated
	if anchor.H == "" {
		t.Error("State hash should not be empty")
	}
	if len(anchor.H) != 64 {
		t.Errorf("State hash should be 64 hex chars, got %d", len(anchor.H))
	}

	// Verify short session ID
	if anchor.S == "" {
		t.Error("Short session ID should not be empty")
	}
	if len(anchor.S) != 8 {
		t.Errorf("Short session ID should be 8 hex chars, got %d", len(anchor.S))
	}

	// Verify transaction number is encoded
	if anchor.N == 0 {
		t.Error("Transaction number should not be zero")
	}

	// Verify transition code is set
	if anchor.T == 0 {
		t.Error("Transition code should not be zero")
	}

	// Verify timestamp
	if anchor.TS == 0 {
		t.Error("Timestamp should be set")
	}
}

// TestCompactAnchorVerification tests compact anchor verification
func TestCompactAnchorVerification(t *testing.T) {
	state := &State{
		Session:   "test_session",
		Ssm:       "TestSSM",
		Iteration: 10,
		Current:   STATE_VALIDATED,
		Roles:     map[string]string{"Agent1": "Initiator"},
		Public:    "test data",
		Origin:    Transition{From: 1, To: 2, Role: "Initiator", Action: "Validate"},
	}

	sessionCounter := uint16(5)
	anchor, _ := CreateCompactAnchor(state, sessionCounter)

	// Verify anchor matches state
	valid, err := VerifyCompactAnchor(anchor, state)
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}
	if !valid {
		t.Error("Anchor should be valid")
	}

	// Test with modified state (should fail)
	modifiedState := *state
	modifiedState.Iteration = 999

	valid, err = VerifyCompactAnchor(anchor, &modifiedState)
	if err == nil {
		t.Error("Verification should fail with modified state")
	}
	if valid {
		t.Error("Anchor should not be valid for modified state")
	}
}

// TestTransactionNumberEncoding tests transaction number encoding/decoding
func TestTransactionNumberEncoding(t *testing.T) {
	sessionCounter := uint16(42)
	iteration := uint32(1337)
	state := uint8(STATE_ACCEPTED)

	// Encode
	txNum := EncodeTransactionNumber(sessionCounter, iteration, state)

	// Decode
	decodedSession, decodedIter, decodedState, valid := DecodeTransactionNumber(txNum)

	if !valid {
		t.Error("Transaction number checksum should be valid")
	}

	if decodedSession != sessionCounter {
		t.Errorf("Session counter mismatch: expected %d, got %d", sessionCounter, decodedSession)
	}

	if decodedIter != iteration {
		t.Errorf("Iteration mismatch: expected %d, got %d", iteration, decodedIter)
	}

	if decodedState != state {
		t.Errorf("State mismatch: expected %d, got %d", state, decodedState)
	}
}

// TestTransitionEncoding tests transition encoding/decoding
func TestTransitionEncoding(t *testing.T) {
	fromState := uint8(STATE_PROPOSED)
	action := uint8(ACTION_ACCEPT)

	// Encode
	code := EncodeTransition(fromState, action)

	// Decode
	decodedFrom, decodedAction := DecodeTransition(code)

	if decodedFrom != fromState {
		t.Errorf("From state mismatch: expected %d, got %d", fromState, decodedFrom)
	}

	if decodedAction != action {
		t.Errorf("Action mismatch: expected %d, got %d", action, decodedAction)
	}
}

// TestSpaceSavings tests space savings calculation
func TestSpaceSavings(t *testing.T) {
	state := &State{
		Session:   "large_session_id_with_long_name",
		Ssm:       "ComplexStateMachine",
		Iteration: 100,
		Current:   3,
		Roles: map[string]string{
			"Agent1": "Role1",
			"Agent2": "Role2",
			"Agent3": "Role3",
		},
		Public: `{"large": "json", "with": "lots", "of": "data", "fields": [1, 2, 3, 4, 5]}`,
		Origin: Transition{From: 2, To: 3, Role: "Role1", Action: "Process"},
	}

	anchor, _ := CreateCompactAnchor(state, 1)

	fullSize := FullStateSize(state)
	compactSize := CompactAnchorSize(anchor)
	savings := CalculateSpaceSavings(state, anchor)

	t.Logf("Full state size: %d bytes", fullSize)
	t.Logf("Compact anchor size: %d bytes", compactSize)
	t.Logf("Space savings: %.1f%%", savings)

	// Compact anchor should be significantly smaller
	if compactSize >= fullSize {
		t.Error("Compact anchor should be smaller than full state")
	}

	// Should achieve at least 80% savings
	if savings < 80.0 {
		t.Errorf("Expected at least 80%% savings, got %.1f%%", savings)
	}
}

// TestLightningInvoiceCreation tests Lightning invoice creation with compact anchor
func TestLightningInvoiceCreation(t *testing.T) {
	state := &State{
		Session:   "invoice_test",
		Ssm:       "PaymentSSM",
		Iteration: 5,
		Current:   STATE_TRANSFERRED,
		Roles:     map[string]string{"Payer": "Alice", "Payee": "Bob"},
		Public:    "payment data",
		Origin:    Transition{From: 3, To: 4, Role: "Payer", Action: "Transfer"},
	}

	anchor, _ := CreateCompactAnchor(state, 10)

	invoice, err := CreateLightningInvoiceWithCompactAnchor(anchor, 100000, "Carbon credit transfer")
	if err != nil {
		t.Fatalf("Failed to create invoice: %v", err)
	}

	// Verify invoice fields
	if invoice["amount_msat"] != int64(100000) {
		t.Error("Invoice amount mismatch")
	}

	if invoice["description"] != "Carbon credit transfer" {
		t.Error("Invoice description mismatch")
	}

	// Verify metadata contains compact anchor
	metadata, ok := invoice["metadata"].(string)
	if !ok {
		t.Error("Invoice metadata should be a string")
	}

	if len(metadata) == 0 {
		t.Error("Invoice metadata should not be empty")
	}

	// Verify we can deserialize the anchor from metadata
	var parsedAnchor CompactLightningAnchor
	err = json.Unmarshal([]byte(metadata), &parsedAnchor)
	if err != nil {
		t.Errorf("Failed to parse anchor from metadata: %v", err)
	}
}

// TestSessionRegistry tests session counter registry
func TestSessionRegistry(t *testing.T) {
	registry := NewSessionRegistry()

	// Register sessions
	counter1, err := registry.RegisterSession("session_001")
	if err != nil {
		t.Fatalf("Failed to register session: %v", err)
	}

	counter2, err := registry.RegisterSession("session_002")
	if err != nil {
		t.Fatalf("Failed to register session: %v", err)
	}

	// Counters should be different
	if counter1 == counter2 {
		t.Error("Different sessions should have different counters")
	}

	// Re-registering same session should return same counter
	counter1Again, _ := registry.RegisterSession("session_001")
	if counter1Again != counter1 {
		t.Error("Re-registering same session should return same counter")
	}

	// Verify lookup
	retrievedCounter, exists := registry.GetSessionCounter("session_001")
	if !exists {
		t.Error("Session should exist in registry")
	}
	if retrievedCounter != counter1 {
		t.Error("Retrieved counter mismatch")
	}

	// Verify reverse lookup
	retrievedSession, exists := registry.GetSessionID(counter1)
	if !exists {
		t.Error("Counter should exist in registry")
	}
	if retrievedSession != "session_001" {
		t.Error("Retrieved session ID mismatch")
	}
}

// BenchmarkCreateCompactAnchor benchmarks compact anchor creation
func BenchmarkCreateCompactAnchor(b *testing.B) {
	state := &State{
		Session:   "benchmark_session",
		Ssm:       "BenchmarkSSM",
		Iteration: 50,
		Current:   STATE_ACCEPTED,
		Roles:     map[string]string{"Agent": "Role"},
		Public:    "benchmark data",
		Origin:    Transition{From: 1, To: 2, Role: "Role", Action: "Accept"},
	}

	sessionCounter := uint16(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CreateCompactAnchor(state, sessionCounter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVerifyCompactAnchor benchmarks compact anchor verification
func BenchmarkVerifyCompactAnchor(b *testing.B) {
	state := &State{
		Session:   "verify_benchmark",
		Ssm:       "VerifySSM",
		Iteration: 25,
		Current:   STATE_VALIDATED,
		Roles:     map[string]string{"Agent": "Validator"},
		Public:    "verification data",
		Origin:    Transition{From: 1, To: 2, Role: "Validator", Action: "Validate"},
	}

	anchor, _ := CreateCompactAnchor(state, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := VerifyCompactAnchor(anchor, state)
		if err != nil {
			b.Fatal(err)
		}
	}
}
