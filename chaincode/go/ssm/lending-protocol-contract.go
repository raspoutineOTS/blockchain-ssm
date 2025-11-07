// Copyright 2024
// License: Apache-2.0
//
// Lending Protocol Chaincode Methods
// Supply, borrow, repay, and liquidation for ITMO states

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Key prefixes for lending protocol
const (
	PrefixLendingPool     = "LENDING_POOL"
	PrefixLendingPosition = "LENDING_POSITION"
	PrefixBorrowPosition  = "BORROW_POSITION"
	PrefixLiquidation     = "LIQUIDATION"
)

// ============================================================================
// Lending Pool Management
// ============================================================================

// CreateLendingPool creates a new lending pool
func (s *SSMContract) CreateLendingPool(
	ctx contractapi.TransactionContextInterface,
	poolJSON, userName, signature string,
) error {
	// Verify admin signature
	if err := s.verifySignature(ctx, userName, PrefixAdmin, poolJSON, signature); err != nil {
		return fmt.Errorf("only admins can create lending pools: %w", err)
	}

	var pool LendingPool
	if err := json.Unmarshal([]byte(poolJSON), &pool); err != nil {
		return fmt.Errorf("failed to parse pool JSON: %w", err)
	}

	// Validate
	if pool.PoolID == "" {
		return fmt.Errorf("poolId is required")
	}

	// Check if exists
	key := makeKey(PrefixLendingPool, pool.PoolID)
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
	pool.TotalDeposited = 0
	pool.TotalBorrowed = 0
	pool.AvailableLiquidity = 0
	pool.UtilizationRate = 0
	pool.ReserveBalance = 0

	// Default parameters if not set
	if pool.MaxLoanToValue == 0 {
		pool.MaxLoanToValue = 0.75 // 75% LTV
	}
	if pool.LiquidationThreshold == 0 {
		pool.LiquidationThreshold = 0.85 // 85%
	}
	if pool.LiquidationPenalty == 0 {
		pool.LiquidationPenalty = 0.10 // 10%
	}
	if pool.ReserveFactor == 0 {
		pool.ReserveFactor = 0.10 // 10%
	}
	if pool.OptimalUtilization == 0 {
		pool.OptimalUtilization = 0.80 // 80%
	}
	if pool.BaseRate == 0 {
		pool.BaseRate = 0.02 // 2%
	}
	if pool.Slope1 == 0 {
		pool.Slope1 = 0.05 // 5%
	}
	if pool.Slope2 == 0 {
		pool.Slope2 = 0.50 // 50% (steep after optimal)
	}

	// Calculate initial rates
	pool.SupplyAPY, pool.BorrowAPY = s.calculateInterestRates(&pool)

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
		"poolId":    pool.PoolID,
		"assetType": pool.AssetType,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("LendingPoolCreated", eventBytes)
}

// GetLendingPool retrieves a lending pool
func (s *SSMContract) GetLendingPool(
	ctx contractapi.TransactionContextInterface,
	poolID string,
) (*LendingPool, error) {
	key := makeKey(PrefixLendingPool, poolID)
	poolBytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool: %w", err)
	}
	if poolBytes == nil {
		return nil, fmt.Errorf("pool %s not found", poolID)
	}

	var pool LendingPool
	if err := json.Unmarshal(poolBytes, &pool); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pool: %w", err)
	}

	return &pool, nil
}

// ============================================================================
// Supply (Lending) Operations
// ============================================================================

// Supply deposits ITMO state into lending pool to earn interest
func (s *SSMContract) Supply(
	ctx contractapi.TransactionContextInterface,
	supplyJSON, userName, signature string,
) error {
	// Verify signature
	if err := s.verifySignature(ctx, userName, PrefixUser, supplyJSON, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	var supplyData struct {
		PoolID        string  `json:"poolId"`
		StateRecordID string  `json:"stateRecordId"`
		Amount        float64 `json:"amount"`
	}
	if err := json.Unmarshal([]byte(supplyJSON), &supplyData); err != nil {
		return fmt.Errorf("failed to parse supply data: %w", err)
	}

	// Get pool
	pool, err := s.GetLendingPool(ctx, supplyData.PoolID)
	if err != nil {
		return err
	}

	// Get state record
	stateRecord, err := s.GetITMOStateRecord(ctx, supplyData.StateRecordID)
	if err != nil {
		return err
	}

	// Validate state
	if stateRecord.IsCollateral {
		return fmt.Errorf("state is already collateralized")
	}
	if stateRecord.StateController != userName {
		return fmt.Errorf("user not authorized for this state")
	}
	if supplyData.Amount > stateRecord.ITMOReference.Quantity {
		return fmt.Errorf("amount exceeds available quantity")
	}

	// Create lending position
	positionID := fmt.Sprintf("LEND_%s_%s", supplyData.PoolID, supplyData.StateRecordID)
	now := time.Now()

	// Calculate shares (if pool empty, 1:1, otherwise proportional)
	var shares float64
	if pool.TotalDeposited == 0 {
		shares = supplyData.Amount
	} else {
		totalShares := pool.TotalDeposited // Simplified: shares = deposits
		shares = (supplyData.Amount / pool.TotalDeposited) * totalShares
	}

	position := LendingPosition{
		PositionID:         positionID,
		PoolID:             supplyData.PoolID,
		Lender:             userName,
		StateRecordID:      supplyData.StateRecordID,
		DepositedAmount:    supplyData.Amount,
		SharesOwned:        shares,
		CurrentValue:       supplyData.Amount,
		AccruedInterest:    0,
		InterestRate:       pool.SupplyAPY,
		IsActive:           true,
		CanWithdraw:        true,
		DepositedAt:        now,
		LastInterestUpdate: now,
		UpdatedAt:          now,
	}

	// Update pool
	pool.TotalDeposited += supplyData.Amount
	pool.AvailableLiquidity += supplyData.Amount
	pool.UtilizationRate = pool.TotalBorrowed / pool.TotalDeposited
	pool.SupplyAPY, pool.BorrowAPY = s.calculateInterestRates(pool)
	pool.UpdatedAt = now

	// Store position
	posKey := makeKey(PrefixLendingPosition, positionID)
	posBytes, _ := json.Marshal(position)
	if err := ctx.GetStub().PutState(posKey, posBytes); err != nil {
		return fmt.Errorf("failed to store position: %w", err)
	}

	// Store updated pool
	poolKey := makeKey(PrefixLendingPool, supplyData.PoolID)
	poolBytes, _ := json.Marshal(pool)
	if err := ctx.GetStub().PutState(poolKey, poolBytes); err != nil {
		return fmt.Errorf("failed to update pool: %w", err)
	}

	// Mark state as lent
	stateRecord.IsCollateral = true
	stateRecordKey := makeKey(PrefixITMOState, supplyData.StateRecordID)
	stateRecordBytes, _ := json.Marshal(stateRecord)
	_ = ctx.GetStub().PutState(stateRecordKey, stateRecordBytes)

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId":    positionID,
		"poolId":        supplyData.PoolID,
		"lender":        userName,
		"amount":        supplyData.Amount,
		"newSupplyAPY":  pool.SupplyAPY,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("SupplyDeposited", eventBytes)
}

// Withdraw withdraws supplied ITMO state from pool
func (s *SSMContract) Withdraw(
	ctx contractapi.TransactionContextInterface,
	positionID, userName, signature string,
) error {
	// Verify signature
	withdrawData := map[string]string{"positionId": positionID}
	dataJSON, _ := json.Marshal(withdrawData)
	if err := s.verifySignature(ctx, userName, PrefixUser, string(dataJSON), signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Get position
	posKey := makeKey(PrefixLendingPosition, positionID)
	posBytes, err := ctx.GetStub().GetState(posKey)
	if err != nil || posBytes == nil {
		return fmt.Errorf("position not found")
	}

	var position LendingPosition
	if err := json.Unmarshal(posBytes, &position); err != nil {
		return fmt.Errorf("failed to unmarshal position: %w", err)
	}

	// Verify ownership
	if position.Lender != userName {
		return fmt.Errorf("not position owner")
	}

	if !position.CanWithdraw {
		return fmt.Errorf("position cannot be withdrawn (used as collateral)")
	}

	// Get pool
	pool, err := s.GetLendingPool(ctx, position.PoolID)
	if err != nil {
		return err
	}

	// Calculate accrued interest
	interest := s.calculateAccruedInterest(&position, pool.SupplyAPY)
	totalWithdraw := position.DepositedAmount + interest

	// Check liquidity
	if totalWithdraw > pool.AvailableLiquidity {
		return fmt.Errorf("insufficient liquidity in pool")
	}

	// Update pool
	pool.TotalDeposited -= position.DepositedAmount
	pool.AvailableLiquidity -= totalWithdraw
	if pool.TotalDeposited > 0 {
		pool.UtilizationRate = pool.TotalBorrowed / pool.TotalDeposited
	} else {
		pool.UtilizationRate = 0
	}
	pool.SupplyAPY, pool.BorrowAPY = s.calculateInterestRates(pool)
	pool.UpdatedAt = time.Now()

	// Mark position as closed
	position.IsActive = false
	position.AccruedInterest = interest
	position.CurrentValue = totalWithdraw
	position.UpdatedAt = time.Now()

	// Store updates
	posBytes, _ = json.Marshal(position)
	_ = ctx.GetStub().PutState(posKey, posBytes)

	poolKey := makeKey(PrefixLendingPool, position.PoolID)
	poolBytes, _ := json.Marshal(pool)
	_ = ctx.GetStub().PutState(poolKey, poolBytes)

	// Release state
	stateRecord, _ := s.GetITMOStateRecord(ctx, position.StateRecordID)
	if stateRecord != nil {
		stateRecord.IsCollateral = false
		stateRecordKey := makeKey(PrefixITMOState, position.StateRecordID)
		stateRecordBytes, _ := json.Marshal(stateRecord)
		_ = ctx.GetStub().PutState(stateRecordKey, stateRecordBytes)
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId":     positionID,
		"lender":         userName,
		"principalAmount": position.DepositedAmount,
		"interestEarned": interest,
		"totalWithdrawn": totalWithdraw,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("SupplyWithdrawn", eventBytes)
}

// ============================================================================
// Borrow Operations
// ============================================================================

// Borrow borrows from lending pool using ITMO states as collateral
func (s *SSMContract) Borrow(
	ctx contractapi.TransactionContextInterface,
	borrowJSON, userName, signature string,
) error {
	// Verify signature
	if err := s.verifySignature(ctx, userName, PrefixUser, borrowJSON, signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	var borrowData struct {
		PoolID             string   `json:"poolId"`
		BorrowAmount       float64  `json:"borrowAmount"`
		CollateralStateIDs []string `json:"collateralStateIds"`
	}
	if err := json.Unmarshal([]byte(borrowJSON), &borrowData); err != nil {
		return fmt.Errorf("failed to parse borrow data: %w", err)
	}

	// Get pool
	pool, err := s.GetLendingPool(ctx, borrowData.PoolID)
	if err != nil {
		return err
	}

	// Check liquidity
	if borrowData.BorrowAmount > pool.AvailableLiquidity {
		return fmt.Errorf("insufficient liquidity in pool")
	}

	// Calculate collateral value
	collateralAmount := 0.0
	for _, stateID := range borrowData.CollateralStateIDs {
		stateRecord, err := s.GetITMOStateRecord(ctx, stateID)
		if err != nil {
			return fmt.Errorf("collateral state %s not found: %w", stateID, err)
		}
		if stateRecord.StateController != userName {
			return fmt.Errorf("user not authorized for state %s", stateID)
		}
		if stateRecord.IsCollateral {
			return fmt.Errorf("state %s already collateralized", stateID)
		}
		collateralAmount += stateRecord.ITMOReference.Quantity
	}

	// Get carbon price
	carbonPrice, err := s.GetCurrentPrice(ctx, pool.AssetType)
	if err != nil {
		return fmt.Errorf("failed to get carbon price: %w", err)
	}

	collateralValue := collateralAmount * carbonPrice
	borrowValue := borrowData.BorrowAmount * carbonPrice

	// Check LTV
	ltv := borrowValue / collateralValue
	if ltv > pool.MaxLoanToValue {
		return fmt.Errorf("LTV %.2f%% exceeds max %.2f%%", ltv*100, pool.MaxLoanToValue*100)
	}

	// Calculate health factor
	healthFactor := (collateralValue * pool.LiquidationThreshold) / borrowValue

	// Create borrow position
	positionID := fmt.Sprintf("BORROW_%s_%s_%d", borrowData.PoolID, userName, time.Now().Unix())
	now := time.Now()

	position := BorrowingPosition{
		PositionID:          positionID,
		PoolID:              borrowData.PoolID,
		Borrower:            userName,
		BorrowedAmount:      borrowData.BorrowAmount,
		BorrowedValue:       borrowValue,
		CollateralStateIDs:  borrowData.CollateralStateIDs,
		CollateralAmount:    collateralAmount,
		CollateralValue:     collateralValue,
		AccruedInterest:     0,
		TotalDebt:           borrowValue,
		InterestRate:        pool.BorrowAPY,
		LoanToValue:         ltv,
		HealthFactor:        healthFactor,
		LiquidationPrice:    s.calculateLiquidationPrice(collateralAmount, borrowValue, pool.LiquidationThreshold),
		IsActive:            true,
		IsLiquidatable:      false,
		BorrowedAt:          now,
		LastInterestUpdate:  now,
		UpdatedAt:           now,
	}

	// Update pool
	pool.TotalBorrowed += borrowData.BorrowAmount
	pool.AvailableLiquidity -= borrowData.BorrowAmount
	pool.UtilizationRate = pool.TotalBorrowed / pool.TotalDeposited
	pool.SupplyAPY, pool.BorrowAPY = s.calculateInterestRates(pool)
	pool.UpdatedAt = now

	// Store position
	posKey := makeKey(PrefixBorrowPosition, positionID)
	posBytes, _ := json.Marshal(position)
	if err := ctx.GetStub().PutState(posKey, posBytes); err != nil {
		return fmt.Errorf("failed to store position: %w", err)
	}

	// Store updated pool
	poolKey := makeKey(PrefixLendingPool, borrowData.PoolID)
	poolBytes, _ := json.Marshal(pool)
	_ = ctx.GetStub().PutState(poolKey, poolBytes)

	// Mark collateral states
	for _, stateID := range borrowData.CollateralStateIDs {
		stateRecord, _ := s.GetITMOStateRecord(ctx, stateID)
		if stateRecord != nil {
			stateRecord.IsCollateral = true
			stateRecordKey := makeKey(PrefixITMOState, stateID)
			stateRecordBytes, _ := json.Marshal(stateRecord)
			_ = ctx.GetStub().PutState(stateRecordKey, stateRecordBytes)
		}
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId":     positionID,
		"poolId":         borrowData.PoolID,
		"borrower":       userName,
		"borrowAmount":   borrowData.BorrowAmount,
		"collateralValue": collateralValue,
		"ltv":            ltv,
		"healthFactor":   healthFactor,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("BorrowExecuted", eventBytes)
}

// Repay repays borrowed amount
func (s *SSMContract) Repay(
	ctx contractapi.TransactionContextInterface,
	positionID string,
	repayAmount float64,
	userName, signature string,
) error {
	// Verify signature
	repayData := map[string]interface{}{
		"positionId":  positionID,
		"repayAmount": repayAmount,
	}
	dataJSON, _ := json.Marshal(repayData)
	if err := s.verifySignature(ctx, userName, PrefixUser, string(dataJSON), signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Get position
	posKey := makeKey(PrefixBorrowPosition, positionID)
	posBytes, err := ctx.GetStub().GetState(posKey)
	if err != nil || posBytes == nil {
		return fmt.Errorf("position not found")
	}

	var position BorrowingPosition
	if err := json.Unmarshal(posBytes, &position); err != nil {
		return fmt.Errorf("failed to unmarshal position: %w", err)
	}

	// Verify ownership
	if position.Borrower != userName {
		return fmt.Errorf("not position owner")
	}

	// Get pool
	pool, err := s.GetLendingPool(ctx, position.PoolID)
	if err != nil {
		return err
	}

	// Calculate interest
	interest := s.calculateBorrowInterest(&position, pool.BorrowAPY)
	totalDebt := position.BorrowedAmount + interest

	// Check repayAmount
	if repayAmount > totalDebt {
		repayAmount = totalDebt // Can't repay more than owed
	}

	// Calculate new debt
	newDebt := totalDebt - repayAmount
	fullyRepaid := newDebt <= 0.01 // Allow small rounding errors

	// Update position
	if fullyRepaid {
		position.IsActive = false
		position.BorrowedAmount = 0
		position.TotalDebt = 0
	} else {
		position.BorrowedAmount = newDebt
		position.TotalDebt = newDebt
		// Recalculate health factor
		carbonPrice, _ := s.GetCurrentPrice(ctx, pool.AssetType)
		collateralValue := position.CollateralAmount * carbonPrice
		position.HealthFactor = (collateralValue * pool.LiquidationThreshold) / newDebt
		position.LoanToValue = newDebt / collateralValue
	}
	position.LastInterestUpdate = time.Now()
	position.UpdatedAt = time.Now()

	// Update pool
	pool.TotalBorrowed -= repayAmount
	pool.AvailableLiquidity += repayAmount

	// Reserve calculation (portion of interest goes to reserve)
	reserveAmount := interest * pool.ReserveFactor
	pool.ReserveBalance += reserveAmount

	pool.UtilizationRate = pool.TotalBorrowed / pool.TotalDeposited
	pool.SupplyAPY, pool.BorrowAPY = s.calculateInterestRates(pool)
	pool.UpdatedAt = time.Now()

	// Store updates
	posBytes, _ = json.Marshal(position)
	_ = ctx.GetStub().PutState(posKey, posBytes)

	poolKey := makeKey(PrefixLendingPool, position.PoolID)
	poolBytes, _ := json.Marshal(pool)
	_ = ctx.GetStub().PutState(poolKey, poolBytes)

	// Release collateral if fully repaid
	if fullyRepaid {
		for _, stateID := range position.CollateralStateIDs {
			stateRecord, _ := s.GetITMOStateRecord(ctx, stateID)
			if stateRecord != nil {
				stateRecord.IsCollateral = false
				stateRecordKey := makeKey(PrefixITMOState, stateID)
				stateRecordBytes, _ := json.Marshal(stateRecord)
				_ = ctx.GetStub().PutState(stateRecordKey, stateRecordBytes)
			}
		}
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"positionId":  positionID,
		"borrower":    userName,
		"repayAmount": repayAmount,
		"interest":    interest,
		"newDebt":     newDebt,
		"fullyRepaid": fullyRepaid,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("DebtRepaid", eventBytes)
}

// ============================================================================
// Liquidation
// ============================================================================

// Liquidate liquidates an unhealthy borrow position
func (s *SSMContract) Liquidate(
	ctx contractapi.TransactionContextInterface,
	positionID, userName, signature string,
) error {
	// Verify signature
	liquidateData := map[string]string{"positionId": positionID}
	dataJSON, _ := json.Marshal(liquidateData)
	if err := s.verifySignature(ctx, userName, PrefixUser, string(dataJSON), signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Get position
	posKey := makeKey(PrefixBorrowPosition, positionID)
	posBytes, err := ctx.GetStub().GetState(posKey)
	if err != nil || posBytes == nil {
		return fmt.Errorf("position not found")
	}

	var position BorrowingPosition
	if err := json.Unmarshal(posBytes, &position); err != nil {
		return fmt.Errorf("failed to unmarshal position: %w", err)
	}

	// Get pool
	pool, err := s.GetLendingPool(ctx, position.PoolID)
	if err != nil {
		return err
	}

	// Get current price
	carbonPrice, err := s.GetCurrentPrice(ctx, pool.AssetType)
	if err != nil {
		return fmt.Errorf("failed to get carbon price: %w", err)
	}

	// Recalculate health factor with current price
	collateralValue := position.CollateralAmount * carbonPrice
	interest := s.calculateBorrowInterest(&position, pool.BorrowAPY)
	totalDebt := position.BorrowedAmount + interest
	healthFactor := (collateralValue * pool.LiquidationThreshold) / totalDebt

	// Check if liquidatable (health factor < 1.0)
	if healthFactor >= 1.0 {
		return fmt.Errorf("position is healthy (HF: %.3f), cannot liquidate", healthFactor)
	}

	// Calculate liquidation amounts
	liquidationBonus := pool.LiquidationPenalty
	collateralToSeize := (totalDebt * (1 + liquidationBonus)) / carbonPrice

	if collateralToSeize > position.CollateralAmount {
		collateralToSeize = position.CollateralAmount
	}

	// Create liquidation event
	eventID := fmt.Sprintf("LIQ_%s_%d", positionID, time.Now().Unix())
	now := time.Now()

	liquidationEvent := LiquidationEvent{
		EventID:          eventID,
		BorrowPositionID: positionID,
		Borrower:         position.Borrower,
		Liquidator:       userName,
		DebtRepaid:       totalDebt,
		CollateralSeized: collateralToSeize,
		LiquidationBonus: collateralToSeize - (totalDebt / carbonPrice),
		CarbonPrice:      carbonPrice,
		HealthFactor:     healthFactor,
		LiquidatedAt:     now,
		TransactionID:    ctx.GetStub().GetTxID(),
	}

	// Update position
	position.IsActive = false
	position.IsLiquidatable = true
	position.UpdatedAt = now

	// Update pool
	pool.TotalBorrowed -= position.BorrowedAmount
	pool.AvailableLiquidity += totalDebt
	pool.UtilizationRate = pool.TotalBorrowed / pool.TotalDeposited
	pool.SupplyAPY, pool.BorrowAPY = s.calculateInterestRates(pool)
	pool.UpdatedAt = now

	// Store liquidation event
	liqKey := makeKey(PrefixLiquidation, eventID)
	liqBytes, _ := json.Marshal(liquidationEvent)
	_ = ctx.GetStub().PutState(liqKey, liqBytes)

	// Store updated position
	posBytes, _ = json.Marshal(position)
	_ = ctx.GetStub().PutState(posKey, posBytes)

	// Store updated pool
	poolKey := makeKey(PrefixLendingPool, position.PoolID)
	poolBytes, _ := json.Marshal(pool)
	_ = ctx.GetStub().PutState(poolKey, poolBytes)

	// Transfer collateral states to liquidator
	// (In real implementation, would transfer state tokens/ownership)
	for _, stateID := range position.CollateralStateIDs {
		stateRecord, _ := s.GetITMOStateRecord(ctx, stateID)
		if stateRecord != nil {
			stateRecord.IsCollateral = false
			stateRecord.StateController = userName // Transfer to liquidator
			stateRecordKey := makeKey(PrefixITMOState, stateID)
			stateRecordBytes, _ := json.Marshal(stateRecord)
			_ = ctx.GetStub().PutState(stateRecordKey, stateRecordBytes)
		}
	}

	// Emit event
	eventPayload := map[string]interface{}{
		"eventId":          eventID,
		"positionId":       positionID,
		"borrower":         position.Borrower,
		"liquidator":       userName,
		"debtRepaid":       totalDebt,
		"collateralSeized": collateralToSeize,
		"liquidationBonus": liquidationEvent.LiquidationBonus,
		"healthFactor":     healthFactor,
	}
	eventBytes, _ := json.Marshal(eventPayload)
	return ctx.GetStub().SetEvent("PositionLiquidated", eventBytes)
}

// ============================================================================
// Helper Functions
// ============================================================================

// calculateInterestRates calculates supply and borrow APY based on utilization
func (s *SSMContract) calculateInterestRates(pool *LendingPool) (supplyAPY, borrowAPY float64) {
	utilization := pool.UtilizationRate

	// Calculate borrow rate using kinked interest rate model
	if utilization <= pool.OptimalUtilization {
		borrowAPY = pool.BaseRate + (utilization/pool.OptimalUtilization)*pool.Slope1
	} else {
		excess := utilization - pool.OptimalUtilization
		maxExcess := 1.0 - pool.OptimalUtilization
		borrowAPY = pool.BaseRate + pool.Slope1 + (excess/maxExcess)*pool.Slope2
	}

	// Supply rate = borrow rate * utilization * (1 - reserve factor)
	supplyAPY = borrowAPY * utilization * (1 - pool.ReserveFactor)

	return supplyAPY, borrowAPY
}

// calculateAccruedInterest calculates accrued interest for lending position
func (s *SSMContract) calculateAccruedInterest(position *LendingPosition, currentAPY float64) float64 {
	timeElapsed := time.Since(position.LastInterestUpdate).Seconds()
	secondsPerYear := float64(365 * 24 * 60 * 60)

	// Simple interest: Principal * Rate * Time
	interest := position.DepositedAmount * currentAPY * (timeElapsed / secondsPerYear)

	return interest
}

// calculateBorrowInterest calculates accrued interest for borrow position
func (s *SSMContract) calculateBorrowInterest(position *BorrowingPosition, currentAPY float64) float64 {
	timeElapsed := time.Since(position.LastInterestUpdate).Seconds()
	secondsPerYear := float64(365 * 24 * 60 * 60)

	// Simple interest: Principal * Rate * Time
	interest := position.BorrowedAmount * currentAPY * (timeElapsed / secondsPerYear)

	return interest
}

// calculateLiquidationPrice calculates price at which position becomes liquidatable
func (s *SSMContract) calculateLiquidationPrice(collateralAmount, borrowValue, liquidationThreshold float64) float64 {
	// Price at which: collateral * price * liquidationThreshold = debt
	return borrowValue / (collateralAmount * liquidationThreshold)
}

// UpdateBorrowPositionHealth updates health factor for a borrow position
func (s *SSMContract) UpdateBorrowPositionHealth(
	ctx contractapi.TransactionContextInterface,
	positionID string,
) error {
	// Get position
	posKey := makeKey(PrefixBorrowPosition, positionID)
	posBytes, err := ctx.GetStub().GetState(posKey)
	if err != nil || posBytes == nil {
		return fmt.Errorf("position not found")
	}

	var position BorrowingPosition
	if err := json.Unmarshal(posBytes, &position); err != nil {
		return fmt.Errorf("failed to unmarshal position: %w", err)
	}

	// Get pool
	pool, err := s.GetLendingPool(ctx, position.PoolID)
	if err != nil {
		return err
	}

	// Get current price
	carbonPrice, err := s.GetCurrentPrice(ctx, pool.AssetType)
	if err != nil {
		return err
	}

	// Calculate new health factor
	collateralValue := position.CollateralAmount * carbonPrice
	interest := s.calculateBorrowInterest(&position, pool.BorrowAPY)
	totalDebt := position.BorrowedAmount + interest

	position.HealthFactor = (collateralValue * pool.LiquidationThreshold) / totalDebt
	position.IsLiquidatable = position.HealthFactor < 1.0
	position.LoanToValue = totalDebt / collateralValue
	position.UpdatedAt = time.Now()

	// Store
	posBytes, _ = json.Marshal(position)
	return ctx.GetStub().PutState(posKey, posBytes)
}
