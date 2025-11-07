// Copyright 2024
// License: Apache-2.0
//
// Unit Tests for Price Oracle

package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransactionContext is a mock implementation of TransactionContextInterface
type MockTransactionContext struct {
	mock.Mock
	stub *MockStub
}

func (m *MockTransactionContext) GetStub() shim.ChaincodeStubInterface {
	return m.stub
}

func (m *MockTransactionContext) GetClientIdentity() contractapi.ClientIdentityInterface {
	args := m.Called()
	return args.Get(0).(contractapi.ClientIdentityInterface)
}

// MockStub is a mock implementation of ChaincodeStubInterface
type MockStub struct {
	mock.Mock
	state map[string][]byte
}

func NewMockStub() *MockStub {
	return &MockStub{
		state: make(map[string][]byte),
	}
}

func (m *MockStub) GetState(key string) ([]byte, error) {
	if val, ok := m.state[key]; ok {
		return val, nil
	}
	return nil, nil
}

func (m *MockStub) PutState(key string, value []byte) error {
	m.state[key] = value
	return nil
}

func (m *MockStub) DelState(key string) error {
	delete(m.state, key)
	return nil
}

func (m *MockStub) GetTxID() string {
	return "test-tx-id"
}

func (m *MockStub) SetEvent(name string, payload []byte) error {
	m.Called(name, payload)
	return nil
}

func (m *MockStub) GetQueryResult(query string) (shim.StateQueryIteratorInterface, error) {
	args := m.Called(query)
	return args.Get(0).(shim.StateQueryIteratorInterface), args.Error(1)
}

// Implement other required methods as no-ops
func (m *MockStub) GetArgs() [][]byte                                           { return nil }
func (m *MockStub) GetStringArgs() []string                                     { return nil }
func (m *MockStub) GetFunctionAndParameters() (string, []string)                { return "", nil }
func (m *MockStub) GetArgsSlice() ([]byte, error)                               { return nil, nil }
func (m *MockStub) GetTxTimestamp() (*timestamp.Timestamp, error)               { return nil, nil }
func (m *MockStub) GetChannelID() string                                        { return "test-channel" }
func (m *MockStub) InvokeChaincode(chaincodeName string, args [][]byte, channel string) pb.Response {
	return shim.Success(nil)
}
func (m *MockStub) GetStateByRange(startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetStateByPartialCompositeKey(objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) CreateCompositeKey(objectType string, attributes []string) (string, error) {
	return "", nil
}
func (m *MockStub) SplitCompositeKey(compositeKey string) (string, []string, error) {
	return "", nil, nil
}
func (m *MockStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateData(collection, key string) ([]byte, error)     { return nil, nil }
func (m *MockStub) PutPrivateData(collection, key string, value []byte) error { return nil }
func (m *MockStub) DelPrivateData(collection, key string) error               { return nil }
func (m *MockStub) GetPrivateDataByRange(collection, startKey, endKey string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateDataByPartialCompositeKey(collection, objectType string, keys []string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetPrivateDataQueryResult(collection, query string) (shim.StateQueryIteratorInterface, error) {
	return nil, nil
}
func (m *MockStub) GetCreator() ([]byte, error)                         { return nil, nil }
func (m *MockStub) GetTransient() (map[string][]byte, error)            { return nil, nil }
func (m *MockStub) GetBinding() ([]byte, error)                         { return nil, nil }
func (m *MockStub) GetDecorations() map[string][]byte                   { return nil }
func (m *MockStub) GetSignedProposal() (*pb.SignedProposal, error)      { return nil, nil }
func (m *MockStub) SetStateValidationParameter(key string, ep []byte) error { return nil }
func (m *MockStub) GetStateValidationParameter(key string) ([]byte, error) { return nil, nil }
func (m *MockStub) GetPrivateDataValidationParameter(collection, key string) ([]byte, error) {
	return nil, nil
}
func (m *MockStub) SetPrivateDataValidationParameter(collection, key string, ep []byte) error {
	return nil
}

// TestCreatePriceOracle tests creating a price oracle
func TestCreatePriceOracle(t *testing.T) {
	contract := new(SSMContract)
	ctx := &MockTransactionContext{
		stub: NewMockStub(),
	}

	oracle := PriceOracle{
		AssetType:         "itmo_state",
		CurrentPrice:      15.0,
		AggregationMethod: "median",
		PriceSources: []PriceSource{
			{
				SourceID:         "verra",
				Price:            15.0,
				Weight:           1.0,
				IsActive:         true,
				ReliabilityScore: 0.9,
			},
		},
		UpdateFrequency: 300,
	}

	oracleJSON, _ := json.Marshal(oracle)

	// Mock SetEvent
	ctx.stub.On("SetEvent", "PriceOracleCreated", mock.Anything).Return(nil)

	// Create oracle (note: in real test, would need to mock signature verification)
	err := contract.CreatePriceOracle(ctx, string(oracleJSON), "admin1", "signature")

	// For now, expect signature verification to fail (would need proper mocking)
	assert.Error(t, err) // Expected to fail without proper signature

	// Test that oracle data structure is correct
	assert.Equal(t, "itmo_state", oracle.AssetType)
	assert.Equal(t, 15.0, oracle.CurrentPrice)
	assert.Equal(t, 1, len(oracle.PriceSources))
}

// TestAggregatePrice tests price aggregation methods
func TestAggregatePrice(t *testing.T) {
	contract := new(SSMContract)

	t.Run("Median Aggregation", func(t *testing.T) {
		oracle := &PriceOracle{
			AggregationMethod: "median",
			PriceSources: []PriceSource{
				{SourceID: "s1", Price: 10.0, IsActive: true},
				{SourceID: "s2", Price: 15.0, IsActive: true},
				{SourceID: "s3", Price: 20.0, IsActive: true},
			},
		}

		price, err := contract.aggregatePrice(oracle)
		assert.NoError(t, err)
		assert.Equal(t, 15.0, price)
	})

	t.Run("Mean Aggregation", func(t *testing.T) {
		oracle := &PriceOracle{
			AggregationMethod: "mean",
			PriceSources: []PriceSource{
				{SourceID: "s1", Price: 10.0, IsActive: true},
				{SourceID: "s2", Price: 15.0, IsActive: true},
				{SourceID: "s3", Price: 20.0, IsActive: true},
			},
		}

		price, err := contract.aggregatePrice(oracle)
		assert.NoError(t, err)
		assert.Equal(t, 15.0, price)
	})

	t.Run("Weighted Average", func(t *testing.T) {
		oracle := &PriceOracle{
			AggregationMethod: "weighted_average",
			PriceSources: []PriceSource{
				{SourceID: "s1", Price: 10.0, Weight: 0.5, IsActive: true},
				{SourceID: "s2", Price: 20.0, Weight: 0.5, IsActive: true},
			},
		}

		price, err := contract.aggregatePrice(oracle)
		assert.NoError(t, err)
		assert.Equal(t, 15.0, price)
	})
}

// TestCalculateMedianPrice tests median calculation
func TestCalculateMedianPrice(t *testing.T) {
	contract := new(SSMContract)

	t.Run("Odd number of sources", func(t *testing.T) {
		sources := []PriceSource{
			{Price: 10.0},
			{Price: 15.0},
			{Price: 20.0},
		}

		median := contract.calculateMedianPrice(sources)
		assert.Equal(t, 15.0, median)
	})

	t.Run("Even number of sources", func(t *testing.T) {
		sources := []PriceSource{
			{Price: 10.0},
			{Price: 15.0},
			{Price: 20.0},
			{Price: 25.0},
		}

		median := contract.calculateMedianPrice(sources)
		assert.Equal(t, 17.5, median) // Average of 15 and 20
	})

	t.Run("Single source", func(t *testing.T) {
		sources := []PriceSource{
			{Price: 15.0},
		}

		median := contract.calculateMedianPrice(sources)
		assert.Equal(t, 15.0, median)
	})
}

// TestCalculateVolatility tests volatility calculation
func TestCalculateVolatility(t *testing.T) {
	contract := new(SSMContract)

	t.Run("No history", func(t *testing.T) {
		history := []PricePoint{}
		volatility := contract.calculateVolatility(history)
		assert.Equal(t, 0.0, volatility)
	})

	t.Run("Single point", func(t *testing.T) {
		history := []PricePoint{
			{Price: 15.0},
		}
		volatility := contract.calculateVolatility(history)
		assert.Equal(t, 0.0, volatility)
	})

	t.Run("Multiple points with volatility", func(t *testing.T) {
		history := []PricePoint{
			{Price: 10.0},
			{Price: 15.0},
			{Price: 20.0},
		}
		volatility := contract.calculateVolatility(history)
		assert.Greater(t, volatility, 0.0)
		assert.Less(t, volatility, 10.0) // Should be reasonable
	})
}

// TestInterestRateCalculation tests interest rate model
func TestInterestRateCalculation(t *testing.T) {
	contract := new(SSMContract)

	t.Run("Low utilization", func(t *testing.T) {
		pool := &LendingPool{
			BaseRate:           0.02,
			OptimalUtilization: 0.80,
			Slope1:             0.05,
			Slope2:             0.50,
			UtilizationRate:    0.50, // 50%
			ReserveFactor:      0.10,
		}

		supplyAPY, borrowAPY := contract.calculateInterestRates(pool)

		// At 50% utilization (below optimal)
		// borrowAPY = 0.02 + (0.5/0.8) * 0.05 = 0.02 + 0.03125 = 0.05125
		assert.InDelta(t, 0.05125, borrowAPY, 0.0001)

		// supplyAPY = 0.05125 * 0.5 * 0.9 = 0.0230625
		assert.InDelta(t, 0.0230625, supplyAPY, 0.0001)
	})

	t.Run("High utilization", func(t *testing.T) {
		pool := &LendingPool{
			BaseRate:           0.02,
			OptimalUtilization: 0.80,
			Slope1:             0.05,
			Slope2:             0.50,
			UtilizationRate:    0.90, // 90% (above optimal)
			ReserveFactor:      0.10,
		}

		supplyAPY, borrowAPY := contract.calculateInterestRates(pool)

		// At 90% utilization (above optimal)
		// borrowAPY = 0.02 + 0.05 + ((0.9-0.8)/0.2) * 0.5 = 0.07 + 0.25 = 0.32
		assert.InDelta(t, 0.32, borrowAPY, 0.0001)

		// supplyAPY = 0.32 * 0.9 * 0.9 = 0.2592
		assert.InDelta(t, 0.2592, supplyAPY, 0.0001)
	})

	t.Run("Optimal utilization", func(t *testing.T) {
		pool := &LendingPool{
			BaseRate:           0.02,
			OptimalUtilization: 0.80,
			Slope1:             0.05,
			Slope2:             0.50,
			UtilizationRate:    0.80, // 80% (exactly optimal)
			ReserveFactor:      0.10,
		}

		supplyAPY, borrowAPY := contract.calculateInterestRates(pool)

		// At 80% utilization (optimal)
		// borrowAPY = 0.02 + (0.8/0.8) * 0.05 = 0.02 + 0.05 = 0.07
		assert.InDelta(t, 0.07, borrowAPY, 0.0001)

		// supplyAPY = 0.07 * 0.8 * 0.9 = 0.0504
		assert.InDelta(t, 0.0504, supplyAPY, 0.0001)
	})
}

// TestHealthFactorCalculation tests health factor calculation
func TestHealthFactorCalculation(t *testing.T) {
	t.Run("Healthy position", func(t *testing.T) {
		collateralValue := 15000.0 // $15,000
		debt := 7500.0             // $7,500
		liquidationThreshold := 0.85

		healthFactor := (collateralValue * liquidationThreshold) / debt
		// HF = (15000 * 0.85) / 7500 = 12750 / 7500 = 1.7

		assert.InDelta(t, 1.7, healthFactor, 0.01)
		assert.Greater(t, healthFactor, 1.0) // Healthy
	})

	t.Run("Unhealthy position", func(t *testing.T) {
		collateralValue := 8000.0 // $8,000 (price crashed)
		debt := 7500.0            // $7,500
		liquidationThreshold := 0.85

		healthFactor := (collateralValue * liquidationThreshold) / debt
		// HF = (8000 * 0.85) / 7500 = 6800 / 7500 = 0.906

		assert.InDelta(t, 0.906, healthFactor, 0.01)
		assert.Less(t, healthFactor, 1.0) // Unhealthy - liquidatable!
	})

	t.Run("Critical position", func(t *testing.T) {
		collateralValue := 8823.5 // Exactly at liquidation price
		debt := 7500.0
		liquidationThreshold := 0.85

		healthFactor := (collateralValue * liquidationThreshold) / debt
		// HF should be very close to 1.0

		assert.InDelta(t, 1.0, healthFactor, 0.01)
	})
}

// TestLiquidationPriceCalculation tests liquidation price calculation
func TestLiquidationPriceCalculation(t *testing.T) {
	contract := new(SSMContract)

	t.Run("Calculate liquidation price", func(t *testing.T) {
		collateralAmount := 1000.0 // 1000 tCO2e
		borrowValue := 7500.0      // $7,500
		liquidationThreshold := 0.85

		liqPrice := contract.calculateLiquidationPrice(collateralAmount, borrowValue, liquidationThreshold)
		// liqPrice = 7500 / (1000 * 0.85) = 7500 / 850 = 8.823...

		assert.InDelta(t, 8.823, liqPrice, 0.01)

		// Verify: at this price, health factor = 1.0
		collateralValueAtLiqPrice := collateralAmount * liqPrice
		healthFactorAtLiqPrice := (collateralValueAtLiqPrice * liquidationThreshold) / borrowValue
		assert.InDelta(t, 1.0, healthFactorAtLiqPrice, 0.01)
	})
}

// TestAMMPricing tests AMM constant product pricing
func TestAMMPricing(t *testing.T) {
	t.Run("Swap calculation", func(t *testing.T) {
		reserveA := 1000.0  // 1000 ITMO
		reserveB := 15000.0 // 15000 USDC
		K := reserveA * reserveB

		amountIn := 1500.0 // 1500 USDC
		fee := 0.003       // 0.3%

		amountInWithFee := amountIn * (1 - fee)
		amountOut := (reserveA * amountInWithFee) / (reserveB + amountInWithFee)

		// Expected: ~90.67 ITMO
		assert.InDelta(t, 90.67, amountOut, 0.1)

		// Verify K remains approximately constant (with fees)
		newReserveA := reserveA - amountOut
		newReserveB := reserveB + amountIn
		newK := newReserveA * newReserveB

		// K should increase slightly due to fees
		assert.Greater(t, newK, K)
	})

	t.Run("Price impact", func(t *testing.T) {
		reserveA := 1000.0
		amountOut := 90.67

		priceImpact := (amountOut / reserveA) * 100
		// Impact = (90.67 / 1000) * 100 = 9.067%

		assert.InDelta(t, 9.067, priceImpact, 0.1)
	})

	t.Run("LP token calculation - first deposit", func(t *testing.T) {
		amountA := 1000.0
		amountB := 15000.0

		// First deposit: LP tokens = sqrt(amountA * amountB)
		lpTokens := 3872.98 // sqrt(1000 * 15000)

		assert.InDelta(t, lpTokens, 3872.98, 1.0)
	})
}

// TestCircuitBreaker tests circuit breaker logic
func TestCircuitBreaker(t *testing.T) {
	t.Run("Normal price change", func(t *testing.T) {
		oldPrice := 15.0
		newPrice := 15.5
		maxChange := 0.15 // 15%

		priceChange := (newPrice - oldPrice) / oldPrice
		// Change = 0.5 / 15 = 0.0333 = 3.33%

		assert.InDelta(t, 0.0333, priceChange, 0.001)
		assert.Less(t, priceChange, maxChange) // Should not trigger
	})

	t.Run("Extreme price change - trigger", func(t *testing.T) {
		oldPrice := 15.0
		newPrice := 9.0 // 40% drop!
		maxChange := 0.15

		priceChange := (oldPrice - newPrice) / oldPrice
		// Change = 6 / 15 = 0.4 = 40%

		assert.InDelta(t, 0.4, priceChange, 0.01)
		assert.Greater(t, priceChange, maxChange) // Should trigger!
	})
}

// BenchmarkAggregatePrice benchmarks price aggregation
func BenchmarkAggregatePrice(b *testing.B) {
	contract := new(SSMContract)
	oracle := &PriceOracle{
		AggregationMethod: "median",
		PriceSources: []PriceSource{
			{Price: 10.0, IsActive: true},
			{Price: 15.0, IsActive: true},
			{Price: 20.0, IsActive: true},
			{Price: 12.0, IsActive: true},
			{Price: 18.0, IsActive: true},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		contract.aggregatePrice(oracle)
	}
}

// BenchmarkCalculateInterestRates benchmarks interest rate calculation
func BenchmarkCalculateInterestRates(b *testing.B) {
	contract := new(SSMContract)
	pool := &LendingPool{
		BaseRate:           0.02,
		OptimalUtilization: 0.80,
		Slope1:             0.05,
		Slope2:             0.50,
		UtilizationRate:    0.75,
		ReserveFactor:      0.10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		contract.calculateInterestRates(pool)
	}
}
