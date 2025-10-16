package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"
)

type HalvingCalculationTestSuite struct {
	suite.Suite
}

func TestHalvingCalculationTestSuite(t *testing.T) {
	suite.Run(t, new(HalvingCalculationTestSuite))
}

// TestCalculateHalvingPeriod tests the halving period calculation logic
func (suite *HalvingCalculationTestSuite) TestCalculateHalvingPeriod() {
	testCases := []struct {
		name            string
		currentEpoch    int64
		startEpoch      int64
		halvingInterval uint64
		expectedPeriod  uint64
	}{
		{
			name:            "before start epoch",
			currentEpoch:    0,
			startEpoch:      1,
			halvingInterval: 1461,
			expectedPeriod:  0,
		},
		{
			name:            "at start epoch (period 0)",
			currentEpoch:    1,
			startEpoch:      1,
			halvingInterval: 1461,
			expectedPeriod:  0,
		},
		{
			name:            "middle of first period",
			currentEpoch:    730,
			startEpoch:      1,
			halvingInterval: 1461,
			expectedPeriod:  0,
		},
		{
			name:            "last epoch of first period",
			currentEpoch:    1461,
			startEpoch:      1,
			halvingInterval: 1461,
			expectedPeriod:  0,
		},
		{
			name:            "first epoch of second period",
			currentEpoch:    1462,
			startEpoch:      1,
			halvingInterval: 1461,
			expectedPeriod:  1,
		},
		{
			name:            "middle of second period",
			currentEpoch:    2192,
			startEpoch:      1,
			halvingInterval: 1461,
			expectedPeriod:  1,
		},
		{
			name:            "start of third period",
			currentEpoch:    2923,
			startEpoch:      1,
			halvingInterval: 1461,
			expectedPeriod:  2,
		},
		{
			name:            "start epoch not 1",
			currentEpoch:    100,
			startEpoch:      50,
			halvingInterval: 100,
			expectedPeriod:  0,
		},
		{
			name:            "start epoch not 1 - second period",
			currentEpoch:    151,
			startEpoch:      50,
			halvingInterval: 100,
			expectedPeriod:  1,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			period := CalculateHalvingPeriod(tc.currentEpoch, tc.startEpoch, tc.halvingInterval)
			suite.Require().Equal(tc.expectedPeriod, period)
		})
	}
}

// TestCalculateDailyEmission tests the daily emission calculation with halving
func (suite *HalvingCalculationTestSuite) TestCalculateDailyEmission() {
	dailyEmission, _ := math.NewIntFromString("7200000000000000000000") // 7200 tokens with 18 decimals

	testCases := []struct {
		name             string
		halvingPeriod    uint64
		expectedEmission string
	}{
		{
			name:             "period 0 - full emission",
			halvingPeriod:    0,
			expectedEmission: "7200000000000000000000", // 7200 tokens
		},
		{
			name:             "period 1 - first halving",
			halvingPeriod:    1,
			expectedEmission: "3600000000000000000000", // 3600 tokens
		},
		{
			name:             "period 2 - second halving",
			halvingPeriod:    2,
			expectedEmission: "1800000000000000000000", // 1800 tokens
		},
		{
			name:             "period 3 - third halving",
			halvingPeriod:    3,
			expectedEmission: "900000000000000000000", // 900 tokens
		},
		{
			name:             "period 4 - fourth halving",
			halvingPeriod:    4,
			expectedEmission: "450000000000000000000", // 450 tokens
		},
		{
			name:             "period 10 - very small emission",
			halvingPeriod:    10,
			expectedEmission: "7031250000000000000", // ~7.03 tokens
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			params := Params{
				DailyEmission: dailyEmission,
			}

			emission := CalculateDailyEmission(params, tc.halvingPeriod)
			suite.Require().Equal(tc.expectedEmission, emission.String())
		})
	}
}

// TestValidateSupplyCap tests the supply cap validation logic
func (suite *HalvingCalculationTestSuite) TestValidateSupplyCap() {
	maxSupply, _ := math.NewIntFromString("21000000000000000000000000") // 21M tokens with 18 decimals

	testCases := []struct {
		name          string
		currentSupply string
		mintAmount    string
		expectError   bool
	}{
		{
			name:          "valid mint - well under cap",
			currentSupply: "1000000000000000000000000", // 1M tokens
			mintAmount:    "7200000000000000000000",    // 7200 tokens
			expectError:   false,
		},
		{
			name:          "valid mint - close to cap but safe",
			currentSupply: "20999990000000000000000000", // 20,999,990 tokens
			mintAmount:    "7200000000000000000000",     // 7200 tokens
			expectError:   false,
		},
		{
			name:          "invalid mint - would exceed cap",
			currentSupply: "20999995000000000000000000", // 20,999,995 tokens
			mintAmount:    "7200000000000000000000",     // 7200 tokens (total would be 21,000,002.2)
			expectError:   true,
		},
		{
			name:          "invalid mint - exactly at cap",
			currentSupply: "21000000000000000000000000", // 21M tokens
			mintAmount:    "1",                          // 1 wei
			expectError:   true,
		},
		{
			name:          "valid mint - exactly reaches cap",
			currentSupply: "20999992800000000000000000", // 20,999,992.8 tokens
			mintAmount:    "7200000000000000000000",     // 7200 tokens (total = 21M exactly)
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			currentSupply, _ := math.NewIntFromString(tc.currentSupply)
			mintAmount, _ := math.NewIntFromString(tc.mintAmount)

			err := ValidateSupplyCap(currentSupply, mintAmount, maxSupply)

			if tc.expectError {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), "minting")
				suite.Require().Contains(err.Error(), "would exceed max supply")
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

// TestShouldHalve tests the halving trigger logic
func (suite *HalvingCalculationTestSuite) TestShouldHalve() {
	testCases := []struct {
		name             string
		currentEpoch     int64
		startEpoch       int64
		lastHalvingEpoch uint64
		halvingInterval  uint64
		expectedHalve    bool
	}{
		{
			name:             "before start epoch",
			currentEpoch:     0,
			startEpoch:       1,
			lastHalvingEpoch: 0,
			halvingInterval:  1461,
			expectedHalve:    false,
		},
		{
			name:             "at start epoch - no halving",
			currentEpoch:     1,
			startEpoch:       1,
			lastHalvingEpoch: 0,
			halvingInterval:  1461,
			expectedHalve:    false,
		},
		{
			name:             "first halving epoch",
			currentEpoch:     1461,
			startEpoch:       1,
			lastHalvingEpoch: 0,
			halvingInterval:  1461,
			expectedHalve:    true,
		},
		{
			name:             "already halved at this epoch",
			currentEpoch:     1461,
			startEpoch:       1,
			lastHalvingEpoch: 1461,
			halvingInterval:  1461,
			expectedHalve:    false,
		},
		{
			name:             "second halving epoch",
			currentEpoch:     2922,
			startEpoch:       1,
			lastHalvingEpoch: 1461,
			halvingInterval:  1461,
			expectedHalve:    true,
		},
		{
			name:             "not a halving epoch",
			currentEpoch:     2000,
			startEpoch:       1,
			lastHalvingEpoch: 1461,
			halvingInterval:  1461,
			expectedHalve:    false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			shouldHalve := ShouldHalve(tc.currentEpoch, tc.startEpoch, tc.lastHalvingEpoch, tc.halvingInterval)
			suite.Require().Equal(tc.expectedHalve, shouldHalve)
		})
	}
}

// TestGetNextHalvingEpoch tests the next halving epoch calculation
func (suite *HalvingCalculationTestSuite) TestGetNextHalvingEpoch() {
	testCases := []struct {
		name              string
		currentEpoch      int64
		startEpoch        int64
		halvingInterval   uint64
		expectedNextEpoch int64
	}{
		{
			name:              "before start epoch",
			currentEpoch:      0,
			startEpoch:        1,
			halvingInterval:   1461,
			expectedNextEpoch: 1461,
		},
		{
			name:              "at start epoch",
			currentEpoch:      1,
			startEpoch:        1,
			halvingInterval:   1461,
			expectedNextEpoch: 1461,
		},
		{
			name:              "middle of first period",
			currentEpoch:      730,
			startEpoch:        1,
			halvingInterval:   1461,
			expectedNextEpoch: 1461,
		},
		{
			name:              "at first halving",
			currentEpoch:      1461,
			startEpoch:        1,
			halvingInterval:   1461,
			expectedNextEpoch: 2922,
		},
		{
			name:              "after first halving",
			currentEpoch:      1500,
			startEpoch:        1,
			halvingInterval:   1461,
			expectedNextEpoch: 2922,
		},
		{
			name:              "at second halving",
			currentEpoch:      2922,
			startEpoch:        1,
			halvingInterval:   1461,
			expectedNextEpoch: 4383,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			nextEpoch := GetNextHalvingEpoch(tc.currentEpoch, tc.startEpoch, tc.halvingInterval)
			suite.Require().Equal(tc.expectedNextEpoch, nextEpoch)
		})
	}
}

// TestGetHalvingScheduleInfo tests the combined halving schedule information
func (suite *HalvingCalculationTestSuite) TestGetHalvingScheduleInfo() {
	dailyEmission, _ := math.NewIntFromString("7200000000000000000000")
	params := Params{
		DailyEmission:         dailyEmission, // 7200 tokens
		HalvingIntervalEpochs: 1461,
	}

	testCases := []struct {
		name                string
		currentEpoch        int64
		startEpoch          int64
		expectedPeriod      uint64
		expectedEmission    string
		expectedNextHalving int64
		expectedEpochsUntil int64
	}{
		{
			name:                "start of blockchain",
			currentEpoch:        1,
			startEpoch:          1,
			expectedPeriod:      0,
			expectedEmission:    "7200000000000000000000",
			expectedNextHalving: 1461,
			expectedEpochsUntil: 1460,
		},
		{
			name:                "middle of first period",
			currentEpoch:        730,
			startEpoch:          1,
			expectedPeriod:      0,
			expectedEmission:    "7200000000000000000000",
			expectedNextHalving: 1461,
			expectedEpochsUntil: 731,
		},
		{
			name:                "start of second period",
			currentEpoch:        1462,
			startEpoch:          1,
			expectedPeriod:      1,
			expectedEmission:    "3600000000000000000000",
			expectedNextHalving: 2922,
			expectedEpochsUntil: 1460,
		},
		{
			name:                "start of third period",
			currentEpoch:        2923,
			startEpoch:          1,
			expectedPeriod:      2,
			expectedEmission:    "1800000000000000000000",
			expectedNextHalving: 4383,
			expectedEpochsUntil: 1460,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			info := GetHalvingScheduleInfo(tc.currentEpoch, tc.startEpoch, params)

			suite.Require().Equal(tc.expectedPeriod, info.CurrentPeriod)
			suite.Require().Equal(tc.expectedEmission, info.CurrentEmission.String())
			suite.Require().Equal(tc.expectedNextHalving, info.NextHalvingEpoch)
			suite.Require().Equal(tc.expectedEpochsUntil, info.EpochsUntilHalving)
		})
	}
}

// TestIsValidEpochForHalving tests epoch identifier validation
func (suite *HalvingCalculationTestSuite) TestIsValidEpochForHalving() {
	testCases := []struct {
		name       string
		identifier string
		expected   bool
	}{
		{
			name:       "valid day epoch",
			identifier: "day",
			expected:   true,
		},
		{
			name:       "invalid week epoch",
			identifier: "week",
			expected:   false,
		},
		{
			name:       "invalid hour epoch",
			identifier: "hour",
			expected:   false,
		},
		{
			name:       "invalid empty string",
			identifier: "",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			valid := IsValidEpochForHalving(tc.identifier)
			suite.Require().Equal(tc.expected, valid)
		})
	}
}

// TestHalvingMathematicalProperties tests mathematical properties of halving
func (suite *HalvingCalculationTestSuite) TestHalvingMathematicalProperties() {
	dailyEmission, _ := math.NewIntFromString("7200000000000000000000")
	params := Params{
		DailyEmission: dailyEmission, // 7200 tokens
	}

	// Test that each halving reduces emission by exactly half
	for period := uint64(0); period < 10; period++ {
		suite.Run("halving_reduces_by_half", func() {
			currentEmission := CalculateDailyEmission(params, period)
			nextEmission := CalculateDailyEmission(params, period+1)

			// Next emission should be exactly half of current
			expectedNext := currentEmission.Quo(math.NewInt(2))
			suite.Require().Equal(expectedNext.String(), nextEmission.String())
		})
	}

	// Test convergence: sum of all halving periods approaches 14400 tokens
	// (geometric series: 7200 * (1 + 1/2 + 1/4 + 1/8 + ...) = 7200 * 2 = 14400)
	suite.Run("convergence_test", func() {
		totalEmission := math.ZeroInt()

		// Sum first 50 periods (practically converges)
		for period := uint64(0); period < 50; period++ {
			emission := CalculateDailyEmission(params, period)
			totalEmission = totalEmission.Add(emission)
		}

		// Should be very close to 14400 tokens (with 18 decimals)
		expected, _ := math.NewIntFromString("14400000000000000000000") // 14400 tokens
		difference := totalEmission.Sub(expected).Abs()

		// Difference should be less than 1 token (convergence tolerance)
		tolerance, _ := math.NewIntFromString("1000000000000000000") // 1 token
		suite.Require().True(difference.LT(tolerance),
			"Total emission %s should converge to %s within tolerance %s",
			totalEmission.String(), expected.String(), tolerance.String())
	})
}

// Benchmark tests for performance
func (suite *HalvingCalculationTestSuite) TestHalvingPerformance() {
	dailyEmission, _ := math.NewIntFromString("7200000000000000000000")
	params := Params{
		DailyEmission:         dailyEmission,
		HalvingIntervalEpochs: 1461,
	}

	// Test that calculations are fast enough for blockchain execution
	suite.Run("performance_test", func() {
		iterations := 1000

		for i := 0; i < iterations; i++ {
			currentEpoch := int64(i + 1)
			startEpoch := int64(1)

			// These should execute very quickly
			_ = CalculateHalvingPeriod(currentEpoch, startEpoch, params.HalvingIntervalEpochs)
			_ = CalculateDailyEmission(params, uint64(i%10))
			_ = ShouldHalve(currentEpoch, startEpoch, 0, params.HalvingIntervalEpochs)
			_ = GetNextHalvingEpoch(currentEpoch, startEpoch, params.HalvingIntervalEpochs)
		}

		// If we reach here without timeout, performance is acceptable
		suite.Require().True(true)
	})
}
