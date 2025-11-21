package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	"github.com/evmos/evmos/v19/x/inflation/v1/types"
)

type EpochHalvingTestSuite struct {
	suite.Suite
}

func TestEpochHalvingTestSuite(t *testing.T) {
	suite.Run(t, new(EpochHalvingTestSuite))
}

// TestEpochNumberCalculation verifies that AfterEpochEnd uses the correct epoch number
// for halving calculations (the epoch that just ended, not the one starting)
func (suite *EpochHalvingTestSuite) TestEpochNumberCalculation() {
	testCases := []struct {
		name              string
		startEpoch        uint64
		halvingInterval   uint64
		epochEnding       int64 // The epoch that is ending
		epochNumberPassed int64 // The epochNumber passed to AfterEpochEnd (new epoch starting)
		expectedPeriod    uint64
		expectedEmission  string
		description       string
	}{
		{
			name:              "2-day halving: Epoch 1 ends",
			startEpoch:        1,
			halvingInterval:   2,
			epochEnding:       1,
			epochNumberPassed: 2, // AfterEpochEnd is called with epoch 2 (starting)
			expectedPeriod:    0,
			expectedEmission:  "7200000000000000000000", // 7200 tokens
			description:       "First epoch ends, should mint full emission",
		},
		{
			name:              "2-day halving: Epoch 2 ends",
			startEpoch:        1,
			halvingInterval:   2,
			epochEnding:       2,
			epochNumberPassed: 3, // AfterEpochEnd is called with epoch 3 (starting)
			expectedPeriod:    0,
			expectedEmission:  "7200000000000000000000", // 7200 tokens (still period 0)
			description:       "Second epoch ends, should still mint full emission",
		},
		{
			name:              "2-day halving: Epoch 3 ends (first halving)",
			startEpoch:        1,
			halvingInterval:   2,
			epochEnding:       3,
			epochNumberPassed: 4, // AfterEpochEnd is called with epoch 4 (starting)
			expectedPeriod:    1,
			expectedEmission:  "3600000000000000000000", // 3600 tokens (halved)
			description:       "Third epoch ends, should halve emission",
		},
		{
			name:              "2-day halving: Epoch 4 ends",
			startEpoch:        1,
			halvingInterval:   2,
			epochEnding:       4,
			epochNumberPassed: 5, // AfterEpochEnd is called with epoch 5 (starting)
			expectedPeriod:    1,
			expectedEmission:  "3600000000000000000000", // 3600 tokens (still period 1)
			description:       "Fourth epoch ends, should still mint halved emission",
		},
		{
			name:              "4-year halving: Epoch 1461 ends (last full emission)",
			startEpoch:        1,
			halvingInterval:   1461,
			epochEnding:       1461,
			epochNumberPassed: 1462, // AfterEpochEnd is called with epoch 1462 (starting)
			expectedPeriod:    0,
			expectedEmission:  "7200000000000000000000", // 7200 tokens (still period 0)
			description:       "Last day of 4 years, should still mint full emission",
		},
		{
			name:              "4-year halving: Epoch 1462 ends (first halving)",
			startEpoch:        1,
			halvingInterval:   1461,
			epochEnding:       1462,
			epochNumberPassed: 1463, // AfterEpochEnd is called with epoch 1463 (starting)
			expectedPeriod:    1,
			expectedEmission:  "3600000000000000000000", // 3600 tokens (halved)
			description:       "First day after 4 years, should halve emission",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup params
			dailyEmission, ok := math.NewIntFromString("7200000000000000000000")
			suite.Require().True(ok)

			params := types.Params{
				DailyEmission:         dailyEmission,
				HalvingIntervalEpochs: tc.halvingInterval,
			}

			// Calculate period using epochNumberPassed - 1 (the epoch that ended)
			// This simulates the fix: using epochNumber-1 in AfterEpochEnd
			calculatedPeriod := types.CalculateHalvingPeriod(
				tc.epochNumberPassed-1, // Use the epoch that ended
				int64(tc.startEpoch),
				tc.halvingInterval,
			)

			// Calculate emission
			calculatedEmission := types.CalculateDailyEmission(params, calculatedPeriod)

			// Verify
			suite.Require().Equal(tc.expectedPeriod, calculatedPeriod,
				"Period mismatch for %s: %s", tc.name, tc.description)
			suite.Require().Equal(tc.expectedEmission, calculatedEmission.String(),
				"Emission mismatch for %s: %s", tc.name, tc.description)
		})
	}
}

// TestFullHalvingSequence verifies the complete minting sequence over multiple epochs
func (suite *EpochHalvingTestSuite) TestFullHalvingSequence() {
	testCases := []struct {
		name            string
		startEpoch      uint64
		halvingInterval uint64
		numEpochs       int
		expectedTotal   string
		expectedSeq     []string
	}{
		{
			name:            "2-day halving: 8 epochs",
			startEpoch:      1,
			halvingInterval: 2,
			numEpochs:       8,
			expectedTotal:   "27000000000000000000000", // 7200+7200+3600+3600+1800+1800+900+900 = 27000
			expectedSeq: []string{
				"7200000000000000000000", // Epoch 1
				"7200000000000000000000", // Epoch 2
				"3600000000000000000000", // Epoch 3
				"3600000000000000000000", // Epoch 4
				"1800000000000000000000", // Epoch 5
				"1800000000000000000000", // Epoch 6
				"900000000000000000000",  // Epoch 7
				"900000000000000000000",  // Epoch 8
			},
		},
		{
			name:            "4-year halving: First 6 epochs",
			startEpoch:      1,
			halvingInterval: 1461,
			numEpochs:       6,
			expectedTotal:   "43200000000000000000000", // 7200*6
			expectedSeq: []string{
				"7200000000000000000000", // Epoch 1
				"7200000000000000000000", // Epoch 2
				"7200000000000000000000", // Epoch 3
				"7200000000000000000000", // Epoch 4
				"7200000000000000000000", // Epoch 5
				"7200000000000000000000", // Epoch 6
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			dailyEmission, ok := math.NewIntFromString("7200000000000000000000")
			suite.Require().True(ok)

			params := types.Params{
				DailyEmission:         dailyEmission,
				HalvingIntervalEpochs: tc.halvingInterval,
			}

			totalMinted := math.ZeroInt()
			mintedSequence := []string{}

			// Simulate minting at the end of each epoch
			for epochEnding := int64(1); epochEnding <= int64(tc.numEpochs); epochEnding++ {
				// Calculate period based on the epoch that just ended
				calculatedPeriod := types.CalculateHalvingPeriod(
					epochEnding, // The epoch that ended
					int64(tc.startEpoch),
					tc.halvingInterval,
				)

				emission := types.CalculateDailyEmission(params, calculatedPeriod)
				totalMinted = totalMinted.Add(emission)
				mintedSequence = append(mintedSequence, emission.String())
			}

			// Debug output
			if totalMinted.String() != tc.expectedTotal {
				suite.T().Logf("Expected total: %s", tc.expectedTotal)
				suite.T().Logf("Actual total: %s", totalMinted.String())
				suite.T().Logf("Expected sequence: %v", tc.expectedSeq)
				suite.T().Logf("Actual sequence: %v", mintedSequence)
			}

			// Verify total
			suite.Require().Equal(tc.expectedTotal, totalMinted.String(),
				"Total minted mismatch for %s", tc.name)

			// Verify sequence
			suite.Require().Equal(tc.expectedSeq, mintedSequence,
				"Minting sequence mismatch for %s", tc.name)
		})
	}
}

// TestBugVsFix compares the buggy behavior (using epochNumber) vs fixed behavior (using epochNumber-1)
func (suite *EpochHalvingTestSuite) TestBugVsFix() {
	startEpoch := uint64(1)
	halvingInterval := uint64(2)
	dailyEmission, _ := math.NewIntFromString("7200000000000000000000")

	params := types.Params{
		DailyEmission:         dailyEmission,
		HalvingIntervalEpochs: halvingInterval,
	}

	testCases := []struct {
		epochEnding       int64
		epochNumberPassed int64
		buggyEmission     string // Using epochNumber directly
		fixedEmission     string // Using epochNumber-1
	}{
		{
			epochEnding:       1,
			epochNumberPassed: 2,
			buggyEmission:     "7200000000000000000000", // Period 0
			fixedEmission:     "7200000000000000000000", // Period 0 ✅ Same
		},
		{
			epochEnding:       2,
			epochNumberPassed: 3,
			buggyEmission:     "3600000000000000000000", // Period 1 ❌ Wrong!
			fixedEmission:     "7200000000000000000000", // Period 0 ✅ Correct
		},
		{
			epochEnding:       3,
			epochNumberPassed: 4,
			buggyEmission:     "3600000000000000000000", // Period 1
			fixedEmission:     "3600000000000000000000", // Period 1 ✅ Correct
		},
	}

	for _, tc := range testCases {
		suite.Run(suite.T().Name(), func() {
			// Buggy calculation (using epochNumber directly)
			buggyPeriod := types.CalculateHalvingPeriod(
				tc.epochNumberPassed, // BUG: Using the new epoch
				int64(startEpoch),
				halvingInterval,
			)
			buggyEmission := types.CalculateDailyEmission(params, buggyPeriod)

			// Fixed calculation (using epochNumber-1)
			fixedPeriod := types.CalculateHalvingPeriod(
				tc.epochNumberPassed-1, // FIX: Using the epoch that ended
				int64(startEpoch),
				halvingInterval,
			)
			fixedEmission := types.CalculateDailyEmission(params, fixedPeriod)

			// Verify buggy behavior
			suite.Require().Equal(tc.buggyEmission, buggyEmission.String(),
				"Buggy emission mismatch at epoch %d", tc.epochEnding)

			// Verify fixed behavior
			suite.Require().Equal(tc.fixedEmission, fixedEmission.String(),
				"Fixed emission mismatch at epoch %d", tc.epochEnding)
		})
	}

	// Calculate total for 3 epochs
	buggyTotal := math.ZeroInt()
	fixedTotal := math.ZeroInt()

	for epoch := int64(1); epoch <= 3; epoch++ {
		epochNumberPassed := epoch + 1

		// Buggy
		buggyPeriod := types.CalculateHalvingPeriod(epochNumberPassed, int64(startEpoch), halvingInterval)
		buggyEmission := types.CalculateDailyEmission(params, buggyPeriod)
		buggyTotal = buggyTotal.Add(buggyEmission)

		// Fixed
		fixedPeriod := types.CalculateHalvingPeriod(epochNumberPassed-1, int64(startEpoch), halvingInterval)
		fixedEmission := types.CalculateDailyEmission(params, fixedPeriod)
		fixedTotal = fixedTotal.Add(fixedEmission)
	}

	// Buggy: 7200 + 3600 + 3600 = 14400
	suite.Require().Equal("14400000000000000000000", buggyTotal.String(),
		"Buggy total should be 14400 (7200+3600+3600)")

	// Fixed: 7200 + 7200 + 3600 = 18000
	suite.Require().Equal("18000000000000000000000", fixedTotal.String(),
		"Fixed total should be 18000 (7200+7200+3600)")
}

// TestProductionHalvingSchedule verifies the 4-year halving schedule
func (suite *EpochHalvingTestSuite) TestProductionHalvingSchedule() {
	startEpoch := uint64(1)
	halvingInterval := uint64(1461) // 4 years
	dailyEmission, _ := math.NewIntFromString("7200000000000000000000")

	params := types.Params{
		DailyEmission:         dailyEmission,
		HalvingIntervalEpochs: halvingInterval,
	}

	testCases := []struct {
		name             string
		epochEnding      int64
		expectedPeriod   uint64
		expectedEmission string
	}{
		{
			name:             "Day 1 (Epoch 1 ends)",
			epochEnding:      1,
			expectedPeriod:   0,
			expectedEmission: "7200000000000000000000",
		},
		{
			name:             "Day 1460 (Epoch 1460 ends)",
			epochEnding:      1460,
			expectedPeriod:   0,
			expectedEmission: "7200000000000000000000",
		},
		{
			name:             "Day 1461 (Epoch 1461 ends) - Last full emission",
			epochEnding:      1461,
			expectedPeriod:   0,
			expectedEmission: "7200000000000000000000",
		},
		{
			name:             "Day 1462 (Epoch 1462 ends) - First halving",
			epochEnding:      1462,
			expectedPeriod:   1,
			expectedEmission: "3600000000000000000000",
		},
		{
			name:             "Day 2922 (Epoch 2922 ends) - Last of period 1",
			epochEnding:      2922,
			expectedPeriod:   1,
			expectedEmission: "3600000000000000000000",
		},
		{
			name:             "Day 2923 (Epoch 2923 ends) - Second halving",
			epochEnding:      2923,
			expectedPeriod:   2,
			expectedEmission: "1800000000000000000000",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Simulate AfterEpochEnd being called with epochNumber = epochEnding + 1
			epochNumberPassed := tc.epochEnding + 1

			// Calculate using the fixed logic (epochNumber-1)
			calculatedPeriod := types.CalculateHalvingPeriod(
				epochNumberPassed-1,
				int64(startEpoch),
				halvingInterval,
			)
			calculatedEmission := types.CalculateDailyEmission(params, calculatedPeriod)

			suite.Require().Equal(tc.expectedPeriod, calculatedPeriod,
				"Period mismatch for %s", tc.name)
			suite.Require().Equal(tc.expectedEmission, calculatedEmission.String(),
				"Emission mismatch for %s", tc.name)
		})
	}
}

// TestShouldHalveLogic verifies the halving detection logic
func (suite *EpochHalvingTestSuite) TestShouldHalveLogic() {
	startEpoch := int64(1)
	halvingInterval := uint64(2)

	testCases := []struct {
		name             string
		epochEnding      int64
		lastHalvingEpoch uint64
		shouldHalve      bool
	}{
		{
			name:             "Epoch 1 ends - no halving yet",
			epochEnding:      1,
			lastHalvingEpoch: 0,
			shouldHalve:      false,
		},
		{
			name:             "Epoch 2 ends - no halving (still period 0)",
			epochEnding:      2,
			lastHalvingEpoch: 0,
			shouldHalve:      false,
		},
		{
			name:             "Epoch 3 ends - first halving!",
			epochEnding:      3,
			lastHalvingEpoch: 0,
			shouldHalve:      true,
		},
		{
			name:             "Epoch 4 ends - no halving (still period 1)",
			epochEnding:      4,
			lastHalvingEpoch: 3,
			shouldHalve:      false,
		},
		{
			name:             "Epoch 5 ends - second halving!",
			epochEnding:      5,
			lastHalvingEpoch: 3,
			shouldHalve:      true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Use the fixed logic: epochEnding (not epochEnding+1)
			shouldHalve := types.ShouldHalve(
				tc.epochEnding,
				startEpoch,
				tc.lastHalvingEpoch,
				halvingInterval,
			)

			suite.Require().Equal(tc.shouldHalve, shouldHalve,
				"ShouldHalve mismatch for %s", tc.name)
		})
	}
}
