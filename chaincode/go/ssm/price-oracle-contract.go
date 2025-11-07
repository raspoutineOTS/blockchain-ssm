// Copyright 2024
// License: Apache-2.0
//
// Price Oracle Chaincode Methods
// Multi-source price aggregation with circuit breaker protection

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Key prefixes for price oracle
const (
	PrefixPriceOracle = "PRICE_ORACLE"
	PrefixPriceHistory = "PRICE_HISTORY"
)

// ============================================================================
// Price Oracle Management
// ============================================================================

// CreatePriceOracle creates a new price oracle for an asset type
func (s *SSMContract) CreatePriceOracle(
	ctx contractapi.TransactionContextInterface,
	oracleJSON, userName, signature string,
) error {
	// Verify admin signature
	if err := s.verifySignature(ctx, userName, PrefixAdmin, oracleJSON, signature); err != nil {
		return fmt.Errorf("only admins can create price oracles: %w", err)
	}

	var oracle PriceOracle
	if err := json.Unmarshal([]byte(oracleJSON), &oracle); err != nil {
		return fmt.Errorf("failed to parse oracle JSON: %w", err)
	}

	// Validate
	if oracle.AssetType == "" {
		return fmt.Errorf("assetType is required")
	}

	// Check if oracle exists
	key := makeKey(PrefixPriceOracle, oracle.AssetType)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to check existing oracle: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("oracle for %s already exists", oracle.AssetType)
	}

	// Initialize
	now := time.Now()
	oracle.LastUpdated = now

	// Default circuit breaker if not set
	if oracle.CircuitBreaker == nil {
		oracle.CircuitBreaker = &CircuitBreaker{
			MaxPriceChange:      0.15, // 15% max change
			IsTriggered:         false,
			CooldownPeriod:      300, // 5 minutes
			ConsecutiveTriggers: 0,
		}
	}

	// Default aggregation method
	if oracle.AggregationMethod == "" {
		oracle.AggregationMethod = "median"
	}

	// Initialize history
	if oracle.PriceHistory == nil {
		oracle.PriceHistory = []PricePoint{}
	}

	// Store oracle
	oracleBytes, err := json.Marshal(oracle)
	if err != nil {
		return fmt.Errorf("failed to marshal oracle: %w", err)
	}

	if err := ctx.GetStub().PutState(key, oracleBytes); err != nil {
		return fmt.Errorf("failed to store oracle: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"assetType":  oracle.AssetType,
		"price":      oracle.CurrentPrice,
		"numSources": len(oracle.PriceSources),
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("PriceOracleCreated", eventBytes)
}

// UpdatePrice updates the price from all sources and aggregates
func (s *SSMContract) UpdatePrice(
	ctx contractapi.TransactionContextInterface,
	assetType, priceUpdatesJSON, userName, signature string,
) error {
	// Verify signature (admin or authorized oracle operator)
	if err := s.verifySignature(ctx, userName, PrefixAdmin, priceUpdatesJSON, signature); err != nil {
		if err := s.verifySignature(ctx, userName, PrefixUser, priceUpdatesJSON, signature); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Get oracle
	oracle, err := s.GetPriceOracle(ctx, assetType)
	if err != nil {
		return err
	}

	// Parse price updates
	var updates []PriceSource
	if err := json.Unmarshal([]byte(priceUpdatesJSON), &updates); err != nil {
		return fmt.Errorf("failed to parse price updates: %w", err)
	}

	// Update sources
	now := time.Now()
	for i := range updates {
		updates[i].LastUpdated = now
	}

	// Merge with existing sources
	sourceMap := make(map[string]*PriceSource)
	for i := range oracle.PriceSources {
		sourceMap[oracle.PriceSources[i].SourceID] = &oracle.PriceSources[i]
	}
	for i := range updates {
		sourceMap[updates[i].SourceID] = &updates[i]
	}

	// Rebuild sources slice
	oracle.PriceSources = []PriceSource{}
	for _, source := range sourceMap {
		oracle.PriceSources = append(oracle.PriceSources, *source)
	}

	// Calculate new aggregated price
	newPrice, err := s.aggregatePrice(oracle)
	if err != nil {
		return fmt.Errorf("failed to aggregate price: %w", err)
	}

	// Check circuit breaker
	if oracle.CurrentPrice > 0 {
		priceChange := math.Abs(newPrice-oracle.CurrentPrice) / oracle.CurrentPrice

		if priceChange > oracle.CircuitBreaker.MaxPriceChange {
			// Circuit breaker triggered!
			oracle.CircuitBreaker.IsTriggered = true
			oracle.CircuitBreaker.TriggerTime = now
			oracle.CircuitBreaker.ConsecutiveTriggers++

			// Don't update price
			eventPayload := map[string]interface{}{
				"assetType":   assetType,
				"oldPrice":    oracle.CurrentPrice,
				"newPrice":    newPrice,
				"priceChange": priceChange,
				"reason":      "circuit_breaker_triggered",
			}
			eventBytes, _ := json.Marshal(eventPayload)
			_ = ctx.GetStub().SetEvent("CircuitBreakerTriggered", eventBytes)

			// Store oracle with triggered circuit breaker
			key := makeKey(PrefixPriceOracle, assetType)
			oracleBytes, _ := json.Marshal(oracle)
			return ctx.GetStub().PutState(key, oracleBytes)
		}

		// Check if circuit breaker can be reset
		if oracle.CircuitBreaker.IsTriggered {
			cooldownElapsed := now.Sub(oracle.CircuitBreaker.TriggerTime).Seconds()
			if cooldownElapsed > float64(oracle.CircuitBreaker.CooldownPeriod) {
				oracle.CircuitBreaker.IsTriggered = false
				oracle.CircuitBreaker.ConsecutiveTriggers = 0
			}
		}
	}

	// Update price
	oldPrice := oracle.CurrentPrice
	oracle.CurrentPrice = newPrice
	oracle.LastUpdated = now

	// Add to history
	pricePoint := PricePoint{
		Timestamp: now,
		Price:     newPrice,
		Source:    "aggregated",
	}
	oracle.PriceHistory = append(oracle.PriceHistory, pricePoint)

	// Keep only last 1000 points
	if len(oracle.PriceHistory) > 1000 {
		oracle.PriceHistory = oracle.PriceHistory[len(oracle.PriceHistory)-1000:]
	}

	// Calculate volatility (standard deviation of recent prices)
	oracle.Volatility = s.calculateVolatility(oracle.PriceHistory)

	// Store updated oracle
	key := makeKey(PrefixPriceOracle, assetType)
	oracleBytes, err := json.Marshal(oracle)
	if err != nil {
		return fmt.Errorf("failed to marshal oracle: %w", err)
	}

	if err := ctx.GetStub().PutState(key, oracleBytes); err != nil {
		return fmt.Errorf("failed to store oracle: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"assetType":  assetType,
		"oldPrice":   oldPrice,
		"newPrice":   newPrice,
		"volatility": oracle.Volatility,
		"numSources": len(oracle.PriceSources),
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("PriceUpdated", eventBytes)
}

// GetPriceOracle retrieves a price oracle
func (s *SSMContract) GetPriceOracle(
	ctx contractapi.TransactionContextInterface,
	assetType string,
) (*PriceOracle, error) {
	key := makeKey(PrefixPriceOracle, assetType)
	oracleBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get oracle: %w", err)
	}
	if oracleBytes == nil {
		return nil, fmt.Errorf("oracle for %s not found", assetType)
	}

	var oracle PriceOracle
	if err := json.Unmarshal(oracleBytes, &oracle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal oracle: %w", err)
	}

	return &oracle, nil
}

// GetCurrentPrice gets the current price for an asset type
func (s *SSMContract) GetCurrentPrice(
	ctx contractapi.TransactionContextInterface,
	assetType string,
) (float64, error) {
	oracle, err := s.GetPriceOracle(ctx, assetType)
	if err != nil {
		return 0, err
	}

	// Check if price is stale
	stalePeriod := time.Duration(oracle.UpdateFrequency*2) * time.Second
	if time.Since(oracle.LastUpdated) > stalePeriod {
		return 0, fmt.Errorf("price is stale (last update: %v)", oracle.LastUpdated)
	}

	// Check circuit breaker
	if oracle.CircuitBreaker.IsTriggered {
		return 0, fmt.Errorf("circuit breaker is triggered")
	}

	return oracle.CurrentPrice, nil
}

// GetPriceHistory gets historical prices
func (s *SSMContract) GetPriceHistory(
	ctx contractapi.TransactionContextInterface,
	assetType string,
	limit int,
) ([]PricePoint, error) {
	oracle, err := s.GetPriceOracle(ctx, assetType)
	if err != nil {
		return nil, err
	}

	history := oracle.PriceHistory
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}

	return history, nil
}

// ============================================================================
// Price Aggregation Logic
// ============================================================================

// aggregatePrice aggregates prices from multiple sources
func (s *SSMContract) aggregatePrice(oracle *PriceOracle) (float64, error) {
	// Filter active sources
	var activeSources []PriceSource
	for _, source := range oracle.PriceSources {
		if source.IsActive && source.Price > 0 {
			activeSources = append(activeSources, source)
		}
	}

	if len(activeSources) == 0 {
		return 0, fmt.Errorf("no active price sources")
	}

	// Aggregate based on method
	switch oracle.AggregationMethod {
	case "median":
		return s.calculateMedianPrice(activeSources), nil
	case "mean":
		return s.calculateMeanPrice(activeSources), nil
	case "weighted_average":
		return s.calculateWeightedAveragePrice(activeSources), nil
	default:
		return s.calculateMedianPrice(activeSources), nil
	}
}

// calculateMedianPrice calculates median price (robust against outliers)
func (s *SSMContract) calculateMedianPrice(sources []PriceSource) float64 {
	prices := make([]float64, len(sources))
	for i, source := range sources {
		prices[i] = source.Price
	}

	sort.Float64s(prices)

	mid := len(prices) / 2
	if len(prices)%2 == 0 {
		return (prices[mid-1] + prices[mid]) / 2.0
	}
	return prices[mid]
}

// calculateMeanPrice calculates simple average
func (s *SSMContract) calculateMeanPrice(sources []PriceSource) float64 {
	sum := 0.0
	for _, source := range sources {
		sum += source.Price
	}
	return sum / float64(len(sources))
}

// calculateWeightedAveragePrice calculates weighted average
func (s *SSMContract) calculateWeightedAveragePrice(sources []PriceSource) float64 {
	weightedSum := 0.0
	totalWeight := 0.0

	for _, source := range sources {
		weight := source.Weight
		if weight <= 0 {
			weight = 1.0 // Default weight
		}
		weightedSum += source.Price * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

// calculateVolatility calculates price volatility (standard deviation)
func (s *SSMContract) calculateVolatility(history []PricePoint) float64 {
	if len(history) < 2 {
		return 0
	}

	// Use recent 100 points or all if less
	n := 100
	if len(history) < n {
		n = len(history)
	}
	recent := history[len(history)-n:]

	// Calculate mean
	sum := 0.0
	for _, point := range recent {
		sum += point.Price
	}
	mean := sum / float64(len(recent))

	// Calculate variance
	variance := 0.0
	for _, point := range recent {
		diff := point.Price - mean
		variance += diff * diff
	}
	variance /= float64(len(recent))

	// Return standard deviation
	return math.Sqrt(variance)
}

// ============================================================================
// Circuit Breaker Management
// ============================================================================

// ResetCircuitBreaker manually resets the circuit breaker (admin only)
func (s *SSMContract) ResetCircuitBreaker(
	ctx contractapi.TransactionContextInterface,
	assetType, userName, signature string,
) error {
	// Verify admin signature
	emptyData := map[string]string{"assetType": assetType}
	dataJSON, _ := json.Marshal(emptyData)
	if err := s.verifySignature(ctx, userName, PrefixAdmin, string(dataJSON), signature); err != nil {
		return fmt.Errorf("only admins can reset circuit breaker: %w", err)
	}

	// Get oracle
	oracle, err := s.GetPriceOracle(ctx, assetType)
	if err != nil {
		return err
	}

	// Reset circuit breaker
	oracle.CircuitBreaker.IsTriggered = false
	oracle.CircuitBreaker.ConsecutiveTriggers = 0

	// Store
	key := makeKey(PrefixPriceOracle, assetType)
	oracleBytes, _ := json.Marshal(oracle)
	if err := ctx.GetStub().PutState(key, oracleBytes); err != nil {
		return fmt.Errorf("failed to store oracle: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"assetType": assetType,
		"resetBy":   userName,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("CircuitBreakerReset", eventBytes)
}

// UpdateCircuitBreakerConfig updates circuit breaker configuration
func (s *SSMContract) UpdateCircuitBreakerConfig(
	ctx contractapi.TransactionContextInterface,
	assetType, configJSON, userName, signature string,
) error {
	// Verify admin signature
	if err := s.verifySignature(ctx, userName, PrefixAdmin, configJSON, signature); err != nil {
		return fmt.Errorf("only admins can update circuit breaker config: %w", err)
	}

	// Get oracle
	oracle, err := s.GetPriceOracle(ctx, assetType)
	if err != nil {
		return err
	}

	// Parse new config
	var newConfig CircuitBreaker
	if err := json.Unmarshal([]byte(configJSON), &newConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Update config (preserve trigger state)
	wasTriggered := oracle.CircuitBreaker.IsTriggered
	triggerTime := oracle.CircuitBreaker.TriggerTime
	consecutiveTriggers := oracle.CircuitBreaker.ConsecutiveTriggers

	oracle.CircuitBreaker = &newConfig
	oracle.CircuitBreaker.IsTriggered = wasTriggered
	oracle.CircuitBreaker.TriggerTime = triggerTime
	oracle.CircuitBreaker.ConsecutiveTriggers = consecutiveTriggers

	// Store
	key := makeKey(PrefixPriceOracle, assetType)
	oracleBytes, _ := json.Marshal(oracle)
	return ctx.GetStub().PutState(key, oracleBytes)
}

// ============================================================================
// Price Source Management
// ============================================================================

// AddPriceSource adds a new price source to an oracle
func (s *SSMContract) AddPriceSource(
	ctx contractapi.TransactionContextInterface,
	assetType, sourceJSON, userName, signature string,
) error {
	// Verify admin signature
	if err := s.verifySignature(ctx, userName, PrefixAdmin, sourceJSON, signature); err != nil {
		return fmt.Errorf("only admins can add price sources: %w", err)
	}

	// Get oracle
	oracle, err := s.GetPriceOracle(ctx, assetType)
	if err != nil {
		return err
	}

	// Parse source
	var source PriceSource
	if err := json.Unmarshal([]byte(sourceJSON), &source); err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	// Check if source already exists
	for _, existing := range oracle.PriceSources {
		if existing.SourceID == source.SourceID {
			return fmt.Errorf("source %s already exists", source.SourceID)
		}
	}

	// Add source
	source.LastUpdated = time.Now()
	if source.Weight == 0 {
		source.Weight = 1.0 // Default weight
	}
	if source.ReliabilityScore == 0 {
		source.ReliabilityScore = 0.5 // Default reliability
	}

	oracle.PriceSources = append(oracle.PriceSources, source)

	// Store
	key := makeKey(PrefixPriceOracle, assetType)
	oracleBytes, _ := json.Marshal(oracle)
	return ctx.GetStub().PutState(key, oracleBytes)
}

// RemovePriceSource removes a price source
func (s *SSMContract) RemovePriceSource(
	ctx contractapi.TransactionContextInterface,
	assetType, sourceID, userName, signature string,
) error {
	// Verify admin signature
	removeData := map[string]string{"assetType": assetType, "sourceID": sourceID}
	dataJSON, _ := json.Marshal(removeData)
	if err := s.verifySignature(ctx, userName, PrefixAdmin, string(dataJSON), signature); err != nil {
		return fmt.Errorf("only admins can remove price sources: %w", err)
	}

	// Get oracle
	oracle, err := s.GetPriceOracle(ctx, assetType)
	if err != nil {
		return err
	}

	// Remove source
	newSources := []PriceSource{}
	found := false
	for _, source := range oracle.PriceSources {
		if source.SourceID != sourceID {
			newSources = append(newSources, source)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("source %s not found", sourceID)
	}

	oracle.PriceSources = newSources

	// Store
	key := makeKey(PrefixPriceOracle, assetType)
	oracleBytes, _ := json.Marshal(oracle)
	return ctx.GetStub().PutState(key, oracleBytes)
}
