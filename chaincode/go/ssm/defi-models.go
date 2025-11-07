// Copyright 2024
// License: Apache-2.0
//
// DeFi Advanced Models for ITMO States
// Lending, Borrowing, Liquidity Pools, and Price Oracle

package main

import (
	"time"
)

// ============================================================================
// Price Oracle Models
// ============================================================================

// PriceOracle aggregates carbon credit prices from multiple sources
type PriceOracle struct {
	AssetType         string              // "itmo", "verra_vcu", "gold_standard"
	CurrentPrice      float64             // USD per tCO2e
	PriceSources      []PriceSource       // Multiple sources for aggregation
	AggregationMethod string              // "median", "mean", "weighted_average"
	LastUpdated       time.Time
	UpdateFrequency   int                 // Seconds between updates
	PriceHistory      []PricePoint        // Historical prices
	Volatility        float64             // Price volatility (std dev)
	CircuitBreaker    *CircuitBreaker     // Protection against price manipulation
}

// PriceSource represents a single price data source
type PriceSource struct {
	SourceID       string    // "verra_api", "gold_standard", "exchange_X"
	Price          float64   // Price from this source
	Weight         float64   // Weight in aggregation (0-1)
	LastUpdated    time.Time
	IsActive       bool      // Is this source currently active?
	ReliabilityScore float64 // 0-1, based on historical accuracy
}

// PricePoint represents a historical price point
type PricePoint struct {
	Timestamp time.Time
	Price     float64
	Volume    float64 // Trading volume at this price
	Source    string
}

// CircuitBreaker protects against extreme price movements
type CircuitBreaker struct {
	MaxPriceChange      float64   // Max % change allowed per update (e.g., 0.15 = 15%)
	IsTriggered         bool      // Is circuit breaker currently active?
	TriggerTime         time.Time
	CooldownPeriod      int       // Seconds before reset
	ConsecutiveTriggers int       // Number of consecutive triggers
}

// ============================================================================
// Lending Protocol Models
// ============================================================================

// LendingPool represents a pool where users can lend ITMO states
type LendingPool struct {
	PoolID              string
	AssetType           string  // "itmo_state", "verra_vcu"
	TotalDeposited      float64 // Total tCO2e deposited
	TotalBorrowed       float64 // Total tCO2e borrowed
	AvailableLiquidity  float64 // Available to borrow
	UtilizationRate     float64 // Borrowed / Deposited (0-1)

	// Interest rates (APY as decimal, e.g., 0.05 = 5%)
	SupplyAPY           float64 // APY for lenders
	BorrowAPY           float64 // APY for borrowers

	// Interest rate model parameters
	BaseRate            float64 // Base interest rate
	OptimalUtilization  float64 // Target utilization (e.g., 0.8 = 80%)
	Slope1              float64 // Rate increase before optimal
	Slope2              float64 // Rate increase after optimal (steep)

	// Pool configuration
	MaxLoanToValue      float64 // Max LTV ratio (e.g., 0.75 = 75%)
	LiquidationThreshold float64 // Liquidation at this LTV (e.g., 0.85)
	LiquidationPenalty  float64 // Penalty for liquidation (e.g., 0.1 = 10%)

	// Reserve
	ReserveFactor       float64 // % of interest going to reserve
	ReserveBalance      float64 // Current reserve balance

	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// LendingPosition represents a user's lending (supply) position
type LendingPosition struct {
	PositionID          string
	PoolID              string
	Lender              string    // User address
	StateRecordID       string    // ITMO state being lent

	// Amounts
	DepositedAmount     float64   // tCO2e deposited
	SharesOwned         float64   // Pool shares owned
	CurrentValue        float64   // Current value with interest

	// Interest tracking
	AccruedInterest     float64   // Interest earned so far
	InterestRate        float64   // Rate at deposit time

	// Status
	IsActive            bool
	CanWithdraw         bool      // Can withdraw if not used as collateral

	DepositedAt         time.Time
	LastInterestUpdate  time.Time
	UpdatedAt           time.Time
}

// BorrowingPosition represents a user's borrowing position
type BorrowingPosition struct {
	PositionID          string
	PoolID              string
	Borrower            string    // User address

	// Borrowed amount
	BorrowedAmount      float64   // tCO2e borrowed
	BorrowedValue       float64   // USD value at borrow time

	// Collateral
	CollateralStateIDs  []string  // ITMO states used as collateral
	CollateralAmount    float64   // Total tCO2e as collateral
	CollateralValue     float64   // Current USD value of collateral

	// Debt tracking
	AccruedInterest     float64   // Interest owed
	TotalDebt           float64   // Principal + interest
	InterestRate        float64   // Current borrow rate

	// Risk metrics
	LoanToValue         float64   // Current LTV ratio
	HealthFactor        float64   // (Collateral * LiqThreshold) / Debt
	LiquidationPrice    float64   // Price triggering liquidation

	// Status
	IsActive            bool
	IsLiquidatable      bool      // Can be liquidated?

	BorrowedAt          time.Time
	LastInterestUpdate  time.Time
	UpdatedAt           time.Time
}

// LiquidationEvent represents a liquidation
type LiquidationEvent struct {
	EventID             string
	BorrowPositionID    string
	Borrower            string
	Liquidator          string    // Who performed liquidation

	// Amounts
	DebtRepaid          float64   // Debt repaid by liquidator
	CollateralSeized    float64   // Collateral given to liquidator
	LiquidationBonus    float64   // Bonus received by liquidator

	// Prices at liquidation
	CarbonPrice         float64
	HealthFactor        float64

	LiquidatedAt        time.Time
	TransactionID       string
}

// ============================================================================
// Liquidity Pool Models (AMM-style)
// ============================================================================

// LiquidityPool represents an AMM-style pool for trading ITMO states
type LiquidityPool struct {
	PoolID              string

	// Pair
	TokenA              string    // e.g., "ITMO_STATE_TOKEN"
	TokenB              string    // e.g., "USDC"

	// Reserves
	ReserveA            float64   // Amount of token A
	ReserveB            float64   // Amount of token B

	// Pricing (constant product: x * y = k)
	K                   float64   // Constant product
	Price               float64   // Current price (B/A)

	// LP tokens
	TotalLPTokens       float64   // Total LP tokens issued
	LPTokenHolders      map[string]float64 // Address -> LP tokens

	// Fees
	SwapFee             float64   // Swap fee (e.g., 0.003 = 0.3%)
	ProtocolFee         float64   // Protocol fee (e.g., 0.0005)

	// Volume tracking
	Volume24h           float64   // 24h trading volume
	VolumeAllTime       float64   // All-time volume
	FeesEarned          float64   // Total fees earned

	// Pool stats
	TVL                 float64   // Total Value Locked (USD)
	APY                 float64   // Annual percentage yield for LPs

	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// LiquidityProviderPosition represents an LP's position
type LiquidityProviderPosition struct {
	PositionID          string
	PoolID              string
	Provider            string    // LP address

	// Position
	LPTokens            float64   // LP tokens owned
	ShareOfPool         float64   // % of pool owned (0-1)

	// Original deposit
	DepositedTokenA     float64
	DepositedTokenB     float64
	DepositValueUSD     float64

	// Current value
	CurrentTokenA       float64   // Claimable token A
	CurrentTokenB       float64   // Claimable token B
	CurrentValueUSD     float64

	// Earnings
	FeesEarnedA         float64   // Fees earned in token A
	FeesEarnedB         float64   // Fees earned in token B
	ImpermanentLoss     float64   // IL as % (negative = loss)

	DepositedAt         time.Time
	UpdatedAt           time.Time
}

// SwapTransaction represents a token swap
type SwapTransaction struct {
	TransactionID       string
	PoolID              string
	Trader              string

	// Swap details
	TokenIn             string
	TokenOut            string
	AmountIn            float64
	AmountOut           float64

	// Pricing
	Price               float64   // Execution price
	PriceImpact         float64   // Price impact %

	// Fees
	SwapFee             float64
	ProtocolFee         float64

	// Slippage protection
	MinAmountOut        float64   // Minimum amount user accepts
	ActualSlippage      float64   // Actual slippage %

	ExecutedAt          time.Time
	BlockNumber         uint64
}

// ============================================================================
// Yield Farming / Staking Models
// ============================================================================

// StakingPool represents a pool for staking ITMO state tokens
type StakingPool struct {
	PoolID              string

	// Staking token
	StakingToken        string    // Token being staked
	TotalStaked         float64   // Total amount staked

	// Rewards
	RewardToken         string    // Token distributed as reward
	RewardRate          float64   // Rewards per second
	RewardPerTokenStored float64  // Accumulated rewards per token

	// Pool parameters
	LockPeriod          int       // Minimum lock period (seconds)
	EarlyWithdrawPenalty float64  // Penalty for early withdrawal

	// APY
	CurrentAPY          float64   // Current APY

	// Status
	IsActive            bool
	StartTime           time.Time
	EndTime             time.Time

	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// StakingPosition represents a user's staking position
type StakingPosition struct {
	PositionID          string
	PoolID              string
	Staker              string

	// Staked amount
	StakedAmount        float64
	StakedValue         float64   // USD value at stake time

	// Rewards
	RewardsEarned       float64   // Total rewards earned
	RewardsClaimed      float64   // Rewards already claimed
	PendingRewards      float64   // Unclaimed rewards
	RewardDebt          float64   // For reward calculation

	// Lock status
	LockedUntil         time.Time
	CanWithdraw         bool

	StakedAt            time.Time
	LastRewardClaim     time.Time
	UpdatedAt           time.Time
}

// ============================================================================
// Derivatives Models
// ============================================================================

// CarbonFuture represents a futures contract on carbon credits
type CarbonFuture struct {
	ContractID          string

	// Contract specs
	UnderlyingAsset     string    // "ITMO_STATE", "VERRA_VCU"
	ContractSize        float64   // tCO2e per contract
	ExpiryDate          time.Time
	SettlementType      string    // "cash", "physical"

	// Pricing
	CurrentPrice        float64   // Current futures price
	SpotPrice           float64   // Current spot price
	Basis               float64   // Futures - Spot

	// Open interest
	OpenInterest        float64   // Total contracts open
	Volume              float64   // Trading volume

	CreatedAt           time.Time
}

// FuturePosition represents a futures position
type FuturePosition struct {
	PositionID          string
	ContractID          string
	Trader              string

	// Position
	Side                string    // "long", "short"
	Quantity            float64   // Number of contracts
	EntryPrice          float64   // Entry price

	// Margin
	InitialMargin       float64   // Initial margin posted
	MaintenanceMargin   float64   // Maintenance margin required
	CurrentMargin       float64   // Current margin balance

	// PnL
	UnrealizedPnL       float64
	RealizedPnL         float64

	// Risk
	LiquidationPrice    float64

	OpenedAt            time.Time
	UpdatedAt           time.Time
}

// CarbonOption represents an option on carbon credits
type CarbonOption struct {
	ContractID          string

	// Option specs
	UnderlyingAsset     string
	OptionType          string    // "call", "put"
	StrikePrice         float64
	ExpiryDate          time.Time
	ContractSize        float64   // tCO2e per contract

	// Pricing (Black-Scholes style)
	Premium             float64   // Option premium
	ImpliedVolatility   float64
	Delta               float64
	Gamma               float64
	Theta               float64
	Vega                float64

	// Market data
	OpenInterest        float64
	Volume              float64

	CreatedAt           time.Time
}

// ============================================================================
// Helper Types
// ============================================================================

// InterestRateModel calculates interest rates based on utilization
type InterestRateModel struct {
	ModelType           string    // "linear", "kinked", "jump"
	BaseRate            float64
	OptimalUtilization  float64
	Slope1              float64
	Slope2              float64
}

// Calculate interest rate based on utilization
func (irm *InterestRateModel) CalculateRate(utilization float64) float64 {
	if utilization <= irm.OptimalUtilization {
		// Before optimal utilization: gentle slope
		return irm.BaseRate + (utilization/irm.OptimalUtilization)*irm.Slope1
	}
	// After optimal utilization: steep slope to discourage over-utilization
	excess := utilization - irm.OptimalUtilization
	maxExcess := 1.0 - irm.OptimalUtilization
	return irm.BaseRate + irm.Slope1 + (excess/maxExcess)*irm.Slope2
}

// RiskParameters defines risk parameters for the protocol
type RiskParameters struct {
	MaxLTV               float64   // Maximum loan-to-value ratio
	LiquidationThreshold float64   // Liquidation at this LTV
	LiquidationPenalty   float64   // Liquidation penalty
	MinHealthFactor      float64   // Minimum health factor
	ReserveFactor        float64   // Reserve factor for protocol
}
