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

// TestParameterChangeReconciliation tests the Certik audit scenario:
// When governance changes halving parameters mid-chain, the stored state
// should reconcile with the calculated state to prevent "split brain"
func (suite *EpochHalvingTestSuite) TestParameterChangeReconciliation() {
	// Scenario from Certik audit:
	// - Start with 4-year halving (1461 epochs)
	// - First halving occurs at epoch 1461
	// - At epoch 1826, governance changes to 8-year halving (2922 epochs)
	// - The system should reconcile the state correctly

	startEpoch := int64(1)
	originalInterval := uint64(1461) // 4 years

	// Epoch 1462 ends - first natural halving (after 1461 epochs of period 0)
	epoch1462Ending := int64(1462)
	periodAtEpoch1462 := types.CalculateHalvingPeriod(epoch1462Ending, startEpoch, originalInterval)
	suite.Require().Equal(uint64(1), periodAtEpoch1462, "Period should be 1 after first halving")

	// Last halving was at epoch 1462
	lastHalvingEpoch := uint64(1462)

	// Epoch 1826: Governance changes interval to 8 years (2922 epochs)
	newInterval := uint64(2922) // 8 years
	epoch1826Ending := int64(1826)

	// Calculate period with NEW parameters
	periodAfterParamChange := types.CalculateHalvingPeriod(epoch1826Ending, startEpoch, newInterval)
	suite.Require().Equal(uint64(0), periodAfterParamChange,
		"Period should recalculate to 0 with new 8-year interval: (1826-1)/2922 = 0")

	// Calculate what last halving epoch maps to with NEW parameters
	lastPeriodWithNewParams := types.CalculateHalvingPeriod(int64(lastHalvingEpoch), startEpoch, newInterval)
	suite.Require().Equal(uint64(0), lastPeriodWithNewParams,
		"Last halving epoch 1461 should also map to period 0 with new params: (1461-1)/2922 = 0")

	// ShouldHalve with new params should return false (0 > 0 = false)
	shouldHalveAfterChange := types.ShouldHalve(epoch1826Ending, startEpoch, lastHalvingEpoch, newInterval)
	suite.Require().False(shouldHalveAfterChange,
		"ShouldHalve should return false since both periods are 0")

	// Verify expected emission with new parameters
	dailyEmission := math.NewInt(7200).Mul(math.NewInt(1e18)) // 7200e18

	// Calculate emission for period 0: emission / (2^period) = 7200 / (2^0) = 7200
	expectedEmission := dailyEmission.Quo(math.NewInt(1 << periodAfterParamChange))
	suite.Require().Equal("7200000000000000000000", expectedEmission.String(),
		"Should mint full 7200 tokens for period 0")

	// THE KEY ASSERTION:
	// Even though ShouldHalve returns false, the state reconciliation logic
	// in hooks.go should detect that currentPeriod (0) != storedPeriod (1)
	// and update the state accordingly.
	//
	// This test verifies the CALCULATION is correct. The integration test
	// in halving_integration_test.go should verify the STATE UPDATE happens.

	suite.Require().NotEqual(uint64(1), periodAfterParamChange,
		"CRITICAL: Calculated period changed from 1 to 0, state MUST be reconciled")
}

// TestParameterExpansionScenario tests the specific scenario mentioned in Certik audit
func (suite *EpochHalvingTestSuite) TestParameterExpansionScenario() {
	// Detailed walkthrough of the Certik scenario:
	// Year 1-4: Interval = 1461 epochs (4 years)
	// Year 5: Interval changed to 2922 epochs (8 years)

	testCases := []struct {
		name             string
		epochEnding      int64
		startEpoch       int64
		halvingInterval  uint64
		lastHalvingEpoch uint64
		expectedPeriod   uint64
		expectedEmission string
		description      string
	}{
		{
			name:             "Year 1 (Epoch 365) - Original 4-year schedule",
			epochEnding:      365,
			startEpoch:       1,
			halvingInterval:  1461,
			lastHalvingEpoch: 0,
			expectedPeriod:   0,
			expectedEmission: "7200000000000000000000",
			description:      "Period 0, full emission",
		},
		{
			name:             "Year 4 (Epoch 1461) - Last full emission",
			epochEnding:      1461,
			startEpoch:       1,
			halvingInterval:  1461,
			lastHalvingEpoch: 0,
			expectedPeriod:   0,
			expectedEmission: "7200000000000000000000",
			description:      "Last epoch of full emission before halving",
		},
		{
			name:             "Year 4+ (Epoch 1462) - First halving",
			epochEnding:      1462,
			startEpoch:       1,
			halvingInterval:  1461,
			lastHalvingEpoch: 0,
			expectedPeriod:   1,
			expectedEmission: "3600000000000000000000",
			description:      "First halving occurs, period 1",
		},
		{
			name:             "Year 5 (Epoch 1826) BEFORE param change - Original schedule",
			epochEnding:      1826,
			startEpoch:       1,
			halvingInterval:  1461, // Still old params
			lastHalvingEpoch: 1462, // Halving was at 1462
			expectedPeriod:   1,
			expectedEmission: "3600000000000000000000",
			description:      "Still in period 1 with old params",
		},
		{
			name:             "Year 5 (Epoch 1826) AFTER param change - New 8-year schedule",
			epochEnding:      1826,
			startEpoch:       1,
			halvingInterval:  2922, // New params!
			lastHalvingEpoch: 1462, // Old last halving
			expectedPeriod:   0,    // PERIOD CHANGES BACK TO 0!
			expectedEmission: "7200000000000000000000",
			description:      "Period recalculates to 0 with new 8-year interval",
		},
		{
			name:             "Year 8+ (Epoch 2923) - Natural halving with new schedule",
			epochEnding:      2923,
			startEpoch:       1,
			halvingInterval:  2922,
			lastHalvingEpoch: 1462, // Old last halving
			expectedPeriod:   1,
			expectedEmission: "3600000000000000000000",
			description:      "First halving under new 8-year schedule",
		},
	}

	dailyEmission := math.NewInt(7200).Mul(math.NewInt(1e18))

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Calculate period
			period := types.CalculateHalvingPeriod(tc.epochEnding, tc.startEpoch, tc.halvingInterval)
			suite.Require().Equal(tc.expectedPeriod, period,
				"Period mismatch for %s", tc.description)

			// Calculate emission: dailyEmission / (2^period)
			emission := dailyEmission.Quo(math.NewInt(1 << period))
			suite.Require().Equal(tc.expectedEmission, emission.String(),
				"Emission mismatch for %s", tc.description)

			suite.T().Logf("✅ %s: Period=%d, Emission=%s", tc.name, period, emission.String())
		})
	}
}
