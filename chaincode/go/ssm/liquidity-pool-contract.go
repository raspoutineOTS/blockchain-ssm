// Copyright 2024
// License: Apache-2.0
//
// Liquidity Pool Chaincode Methods
// AMM-style pools for trading ITMO state tokens

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Key prefixes for liquidity pools
const (
	PrefixLiqPool    = "LIQ_POOL"
	PrefixLPPosition = "LP_POSITION"
	PrefixSwapTx     = "SWAP_TX"
)

// ============================================================================
// Liquidity Pool Management
// ============================================================================

// CreateLiquidityPool creates a new AMM-style liquidity pool
func (s *SSMContract) CreateLiquidityPool(
	ctx contractapi.TransactionContextInterface,
	poolJSON, userName, signature string,
) error {
	// Verify admin signature
	if err := s.verifySignature(ctx, userName, PrefixAdmin, poolJSON, signature); err != nil {
		return fmt.Errorf("only admins can create liquidity pools: %w", err)
	}

	var pool LiquidityPool
	if err := json.Unmarshal([]byte(poolJSON), &pool); err != nil {
		return fmt.Errorf("failed to parse pool JSON: %w", err)
	}

	// Validate
	if pool.PoolID == "" || pool.TokenA == "" || pool.TokenB == "" {
		return fmt.Errorf("poolId, tokenA, and tokenB are required")
	}

	// Check if exists
	key := makeKey(PrefixLiqPool, pool.PoolID)
	existing, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to check existing pool: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("pool %s already exists", pool.PoolID)
	}

	// Initialize
	now := time.Now()
	pool.CreatedAt = now
	pool.UpdatedAt = now
	pool.ReserveA = 0
	pool.ReserveB = 0
	pool.K = 0
	pool.TotalLPTokens = 0
	pool.Volume24h = 0
	pool.VolumeAllTime = 0
	pool.FeesEarned = 0

	// Default fees
	if pool.SwapFee == 0 {
		pool.SwapFee = 0.003 // 0.3%
	}
	if pool.ProtocolFee == 0 {
		pool.ProtocolFee = 0.0005 // 0.05%
	}

	// Initialize LP token holders
	if pool.LPTokenHolders == nil {
		pool.LPTokenHolders = make(map[string]float64)
	}

	// Store
	poolBytes, err := json.Marshal(pool)
	if err != nil {
		return fmt.Errorf("failed to marshal pool: %w", err)
	}

	if err := ctx.GetStub().PutState(key, poolBytes); err != nil {
		return fmt.Errorf("failed to store pool: %w", err)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"poolId": pool.PoolID,
		"tokenA": pool.TokenA,
		"tokenB": pool.TokenB,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("LiquidityPoolCreated", eventBytes)
}

// GetLiquidityPool retrieves a liquidity pool
func (s *SSMContract) GetLiquidityPool(
	ctx contractapi.TransactionContextInterface,
	poolID string,
) (*LiquidityPool, error) {
	key := makeKey(PrefixLiqPool, poolID)
	poolBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool: %w", err)
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool %s not found", poolID)
	}

	var pool LiquidityPool
	if err := json.Unmarshal(poolBytes, &pool); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %w", err)
	}

	return &pool, nil
}

// ============================================================================
// Add Liquidity
// ============================================================================

// AddLiquidity adds liquidity to pool and mints LP tokens
func (s *SSMContract) AddLiquidity(
	ctx contractapi.TransactionContextInterface,
	liquidityJSON, userName, signature string,
) error {
	// Verify signature
	if err := s.verifySignature(ctx, userName, PrefixUser, liquidityJSON, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	var liqData struct {
		PoolID   string  `json:"poolId"`
		AmountA  float64 `json:"amountA"`
		AmountB  float64 `json:"amountB"`
		MinLPTokens float64 `json:"minLpTokens"` // Slippage protection
	}
	if err := json.Unmarshal([]byte(liquidityJSON), &liqData); err != nil {
		return fmt.Errorf("failed to parse liquidity data: %w", err)
	}

	// Get pool
	pool, err := s.GetLiquidityPool(ctx, liqData.PoolID)
	if err != nil {
		return err
	}

	// Calculate LP tokens to mint
	var lpTokens float64
	now := time.Now()

	if pool.TotalLPTokens == 0 {
		// First liquidity provider: LP tokens = sqrt(amountA * amountB)
		lpTokens = math.Sqrt(liqData.AmountA * liqData.AmountB)
	} else {
		// Subsequent providers: maintain pool ratio
		// LP tokens = (amountA / reserveA) * totalLPTokens
		lpTokensA := (liqData.AmountA / pool.ReserveA) * pool.TotalLPTokens
		lpTokensB := (liqData.AmountB / pool.ReserveB) * pool.TotalLPTokens

		// Use minimum to maintain ratio
		lpTokens = math.Min(lpTokensA, lpTokensB)
	}

	// Check slippage
	if lpTokens < liqData.MinLPTokens {
		return fmt.Errorf("slippage exceeded: got %.2f LP tokens, minimum %.2f", lpTokens, liqData.MinLPTokens)
	}

	// Create/update LP position
	positionID := fmt.Sprintf("LP_%s_%s", liqData.PoolID, userName)

	// Check existing position
	posKey := makeKey(PrefixLPPosition, positionID)
	posBytes, _ := ctx.GetStub().GetState(posKey)

	var position LiquidityProviderPosition
	if posBytes != nil {
		// Update existing position
		json.Unmarshal(posBytes, &position)
		position.LPTokens += lpTokens
		position.DepositedTokenA += liqData.AmountA
		position.DepositedTokenB += liqData.AmountB
		position.UpdatedAt = now
	} else {
		// New position
		position = LiquidityProviderPosition{
			PositionID:      positionID,
			PoolID:          liqData.PoolID,
			Provider:        userName,
			LPTokens:        lpTokens,
			DepositedTokenA: liqData.AmountA,
			DepositedTokenB: liqData.AmountB,
			DepositValueUSD: 0, // Would be calculated with oracle
			FeesEarnedA:     0,
			FeesEarnedB:     0,
			DepositedAt:     now,
			UpdatedAt:       now,
		}
	}

	// Update pool
	pool.ReserveA += liqData.AmountA
	pool.ReserveB += liqData.AmountB
	pool.K = pool.ReserveA * pool.ReserveB
	pool.Price = pool.ReserveB / pool.ReserveA
	pool.TotalLPTokens += lpTokens
	pool.LPTokenHolders[userName] = position.LPTokens
	pool.UpdatedAt = now

	// Calculate TVL (would use oracle in real implementation)
	pool.TVL = pool.ReserveA + pool.ReserveB // Simplified

	// Calculate share of pool
	position.ShareOfPool = position.LPTokens / pool.TotalLPTokens

	// Calculate current claimable amounts
	position.CurrentTokenA = position.ShareOfPool * pool.ReserveA
	position.CurrentTokenB = position.ShareOfPool * pool.ReserveB
	position.CurrentValueUSD = position.CurrentTokenA + position.CurrentTokenB // Simplified

	// Store position
	posBytes, _ = json.Marshal(position)
	_ = ctx.GetStub().PutState(posKey, posBytes)

	// Store pool
	poolKey := makeKey(PrefixLiqPool, liqData.PoolID)
	poolBytes, _ := json.Marshal(pool)
	_ = ctx.GetStub().PutState(poolKey, poolBytes)

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId": positionID,
		"poolId":     liqData.PoolID,
		"provider":   userName,
		"amountA":    liqData.AmountA,
		"amountB":    liqData.AmountB,
		"lpTokens":   lpTokens,
		"shareOfPool": position.ShareOfPool,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("LiquidityAdded", eventBytes)
}

// RemoveLiquidity removes liquidity and burns LP tokens
func (s *SSMContract) RemoveLiquidity(
	ctx contractapi.TransactionContextInterface,
	positionID string,
	lpTokensToBurn float64,
	userName, signature string,
) error {
	// Verify signature
	removeData := map[string]interface{}{
		"positionId":     positionID,
		"lpTokensToBurn": lpTokensToBurn,
	}
	dataJSON, _ := json.Marshal(removeData)
	if err := s.verifySignature(ctx, userName, PrefixUser, string(dataJSON), signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Get position
	posKey := makeKey(PrefixLPPosition, positionID)
	posBytes, err := ctx.GetStub().GetState(posKey)
	if err != nil || posBytes == nil {
		return fmt.Errorf("position not found")
	}

	var position LiquidityProviderPosition
	if err := json.Unmarshal(posBytes, &position); err != nil {
		return fmt.Errorf("failed to unmarshal position: %w", err)
	}

	// Verify ownership
	if position.Provider != userName {
		return fmt.Errorf("not position owner")
	}

	if lpTokensToBurn > position.LPTokens {
		return fmt.Errorf("insufficient LP tokens")
	}

	// Get pool
	pool, err := s.GetLiquidityPool(ctx, position.PoolID)
	if err != nil {
		return err
	}

	// Calculate amounts to withdraw
	shareToWithdraw := lpTokensToBurn / pool.TotalLPTokens
	amountA := shareToWithdraw * pool.ReserveA
	amountB := shareToWithdraw * pool.ReserveB

	// Update position
	position.LPTokens -= lpTokensToBurn
	position.UpdatedAt = time.Now()

	if position.LPTokens <= 0.001 { // Close if essentially empty
		// Delete position
		_ = ctx.GetStub().DelState(posKey)
		delete(pool.LPTokenHolders, userName)
	} else {
		// Update position
		position.ShareOfPool = position.LPTokens / (pool.TotalLPTokens - lpTokensToBurn)
		posBytes, _ = json.Marshal(position)
		_ = ctx.GetStub().PutState(posKey, posBytes)
		pool.LPTokenHolders[userName] = position.LPTokens
	}

	// Update pool
	pool.ReserveA -= amountA
	pool.ReserveB -= amountB
	pool.K = pool.ReserveA * pool.ReserveB
	if pool.ReserveA > 0 {
		pool.Price = pool.ReserveB / pool.ReserveA
	}
	pool.TotalLPTokens -= lpTokensToBurn
	pool.UpdatedAt = time.Now()

	// Store pool
	poolKey := makeKey(PrefixLiqPool, position.PoolID)
	poolBytes, _ := json.Marshal(pool)
	_ = ctx.GetStub().PutState(poolKey, poolBytes)

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId":     positionID,
		"provider":       userName,
		"lpTokensBurned": lpTokensToBurn,
		"amountAOut":     amountA,
		"amountBOut":     amountB,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("LiquidityRemoved", eventBytes)
}

// ============================================================================
// Swap Operations
// ============================================================================

// Swap executes a token swap using constant product formula
func (s *SSMContract) Swap(
	ctx contractapi.TransactionContextInterface,
	swapJSON, userName, signature string,
) error {
	// Verify signature
	if err := s.verifySignature(ctx, userName, PrefixUser, swapJSON, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	var swapData struct {
		PoolID       string  `json:"poolId"`
		TokenIn      string  `json:"tokenIn"`      // "tokenA" or "tokenB"
		AmountIn     float64 `json:"amountIn"`
		MinAmountOut float64 `json:"minAmountOut"` // Slippage protection
	}
	if err := json.Unmarshal([]byte(swapJSON), &swapData); err != nil {
		return fmt.Errorf("failed to parse swap data: %w", err)
	}

	// Get pool
	pool, err := s.GetLiquidityPool(ctx, swapData.PoolID)
	if err != nil {
		return err
	}

	// Determine swap direction
	var (
		tokenOut        string
		reserveIn       float64
		reserveOut      float64
		amountOut       float64
		priceImpact     float64
	)

	if swapData.TokenIn == "tokenA" {
		tokenOut = "tokenB"
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else if swapData.TokenIn == "tokenB" {
		tokenOut = "tokenA"
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	} else {
		return fmt.Errorf("invalid tokenIn: must be 'tokenA' or 'tokenB'")
	}

	// Calculate amount out using constant product formula
	// amountOut = (reserveOut * amountIn) / (reserveIn + amountIn)
	// With fees: amountInWithFee = amountIn * (1 - fee)

	amountInWithFee := swapData.AmountIn * (1 - pool.SwapFee - pool.ProtocolFee)
	amountOut = (reserveOut * amountInWithFee) / (reserveIn + amountInWithFee)

	// Check slippage
	if amountOut < swapData.MinAmountOut {
		return fmt.Errorf("slippage exceeded: got %.6f, minimum %.6f", amountOut, swapData.MinAmountOut)
	}

	// Calculate price impact
	// Price impact = (amountOut / reserveOut) * 100
	priceImpact = (amountOut / reserveOut) * 100

	// Calculate execution price
	executionPrice := swapData.AmountIn / amountOut

	// Calculate fees
	swapFeeAmount := swapData.AmountIn * pool.SwapFee
	protocolFeeAmount := swapData.AmountIn * pool.ProtocolFee

	// Update reserves
	if swapData.TokenIn == "tokenA" {
		pool.ReserveA += swapData.AmountIn
		pool.ReserveB -= amountOut
	} else {
		pool.ReserveB += swapData.AmountIn
		pool.ReserveA -= amountOut
	}

	// Update K (should remain constant, but adjust for fees)
	pool.K = pool.ReserveA * pool.ReserveB

	// Update price
	pool.Price = pool.ReserveB / pool.ReserveA

	// Update volume
	pool.VolumeAllTime += swapData.AmountIn
	pool.Volume24h += swapData.AmountIn // Would need time-based cleanup

	// Update fees earned
	pool.FeesEarned += swapFeeAmount + protocolFeeAmount

	pool.UpdatedAt = time.Now()

	// Create swap transaction record
	txID := fmt.Sprintf("SWAP_%s_%d", swapData.PoolID, time.Now().UnixNano())
	now := time.Now()

	swapTx := SwapTransaction{
		TransactionID:  txID,
		PoolID:         swapData.PoolID,
		Trader:         userName,
		TokenIn:        swapData.TokenIn,
		TokenOut:       tokenOut,
		AmountIn:       swapData.AmountIn,
		AmountOut:      amountOut,
		Price:          executionPrice,
		PriceImpact:    priceImpact,
		SwapFee:        swapFeeAmount,
		ProtocolFee:    protocolFeeAmount,
		MinAmountOut:   swapData.MinAmountOut,
		ActualSlippage: (swapData.MinAmountOut - amountOut) / swapData.MinAmountOut * 100,
		ExecutedAt:     now,
	}

	// Store swap transaction
	txKey := makeKey(PrefixSwapTx, txID)
	txBytes, _ := json.Marshal(swapTx)
	_ = ctx.GetStub().PutState(txKey, txBytes)

	// Store updated pool
	poolKey := makeKey(PrefixLiqPool, swapData.PoolID)
	poolBytes, _ := json.Marshal(pool)
	_ = ctx.GetStub().PutState(poolKey, poolBytes)

	// Distribute fees to LP holders (proportionally)
	s.distributeFees(ctx, pool, swapFeeAmount, swapData.TokenIn)

	// Emit event
	eventPayload := map[string]interface{}{
		"transactionId": txID,
		"poolId":        swapData.PoolID,
		"trader":        userName,
		"tokenIn":       swapData.TokenIn,
		"tokenOut":      tokenOut,
		"amountIn":      swapData.AmountIn,
		"amountOut":     amountOut,
		"price":         executionPrice,
		"priceImpact":   priceImpact,
		"fee":           swapFeeAmount,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("SwapExecuted", eventBytes)
}

// ============================================================================
// Price Calculation
// ============================================================================

// GetSwapQuote calculates expected output for a swap without executing
func (s *SSMContract) GetSwapQuote(
	ctx contractapi.TransactionContextInterface,
	poolID, tokenIn string,
	amountIn float64,
) (float64, float64, error) {
	// Get pool
	pool, err := s.GetLiquidityPool(ctx, poolID)
	if err != nil {
		return 0, 0, err
	}

	// Determine reserves
	var reserveIn, reserveOut float64
	if tokenIn == "tokenA" {
		reserveIn = pool.ReserveA
		reserveOut = pool.ReserveB
	} else if tokenIn == "tokenB" {
		reserveIn = pool.ReserveB
		reserveOut = pool.ReserveA
	} else {
		return 0, 0, fmt.Errorf("invalid tokenIn")
	}

	// Calculate amount out
	amountInWithFee := amountIn * (1 - pool.SwapFee - pool.ProtocolFee)
	amountOut := (reserveOut * amountInWithFee) / (reserveIn + amountInWithFee)

	// Calculate price impact
	priceImpact := (amountOut / reserveOut) * 100

	return amountOut, priceImpact, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// distributeFees distributes swap fees to LP token holders
func (s *SSMContract) distributeFees(
	ctx contractapi.TransactionContextInterface,
	pool *LiquidityPool,
	feeAmount float64,
	feeToken string,
) error {
	// In a real implementation, this would update each LP position
	// For now, fees accrue in the pool reserves

	// Fees are automatically distributed proportionally when LPs withdraw
	// because we use the reserve-based withdrawal calculation

	return nil
}

// calculateImpermanentLoss calculates IL for an LP position
func (s *SSMContract) calculateImpermanentLoss(
	initialPriceRatio, currentPriceRatio float64,
) float64 {
	// IL formula: 2 * sqrt(priceRatio) / (1 + priceRatio) - 1
	priceChange := currentPriceRatio / initialPriceRatio
	il := 2*math.Sqrt(priceChange)/(1+priceChange) - 1

	return il * 100 // Return as percentage
}

// ============================================================================
// Query Functions
// ============================================================================

// GetLPPosition retrieves an LP position
func (s *SSMContract) GetLPPosition(
	ctx contractapi.TransactionContextInterface,
	positionID string,
) (*LiquidityProviderPosition, error) {
	key := makeKey(PrefixLPPosition, positionID)
	posBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get position: %w", err)
	}
	if posBytes == nil {
		return nil, fmt.Errorf("position %s not found", positionID)
	}

	var position LiquidityProviderPosition
	if err := json.Unmarshal(posBytes, &position); err != nil {
		return nil, fmt.Errorf("failed to unmarshal position: %w", err)
	}

	return &position, nil
}

// GetSwapTransaction retrieves a swap transaction
func (s *SSMContract) GetSwapTransaction(
	ctx contractapi.TransactionContextInterface,
	txID string,
) (*SwapTransaction, error) {
	key := makeKey(PrefixSwapTx, txID)
	txBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if txBytes == nil {
		return nil, fmt.Errorf("transaction %s not found", txID)
	}

	var tx SwapTransaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	return &tx, nil
}

// GetPoolStats retrieves pool statistics
func (s *SSMContract) GetPoolStats(
	ctx contractapi.TransactionContextInterface,
	poolID string,
) (map[string]interface{}, error) {
	pool, err := s.GetLiquidityPool(ctx, poolID)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"poolId":         pool.PoolID,
		"reserveA":       pool.ReserveA,
		"reserveB":       pool.ReserveB,
		"price":          pool.Price,
		"tvl":            pool.TVL,
		"volume24h":      pool.Volume24h,
		"volumeAllTime":  pool.VolumeAllTime,
		"feesEarned":     pool.FeesEarned,
		"totalLPTokens":  pool.TotalLPTokens,
		"numLPHolders":   len(pool.LPTokenHolders),
		"apy":            pool.APY,
		"lastUpdated":    pool.UpdatedAt,
	}

	return stats, nil
}
