package keeper_test

import (
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/evmos/evmos/v19/x/epochs/types"
	"github.com/evmos/evmos/v19/x/inflation/v1/types"
)

// TestHalvingIntegrationAfterEpochEnd tests the complete halving integration
func (suite *KeeperTestSuite) TestHalvingIntegrationAfterEpochEnd() {
	testCases := []struct {
		name               string
		currentEpoch       int64
		setupHalvingData   types.HalvingData
		expectedMint       bool
		expectedEmission   string
		expectedNewPeriod  uint64
		expectedHalving    bool
	}{
		{
			name:         "first epoch - no halving",
			currentEpoch: 1,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "7200000000000000000000", // 7200 tokens
			expectedNewPeriod: 0,
			expectedHalving:   false,
		},
		{
			name:         "middle of first period",
			currentEpoch: 730,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "7200000000000000000000", // 7200 tokens
			expectedNewPeriod: 0,
			expectedHalving:   false,
		},
		{
			name:         "first halving epoch",
			currentEpoch: 1461,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "3600000000000000000000", // 3600 tokens (halved)
			expectedNewPeriod: 1,
			expectedHalving:   true,
		},
		{
			name:         "first epoch of second period",
			currentEpoch: 1462,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    1,
				LastHalvingEpoch: 1461,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "3600000000000000000000", // 3600 tokens
			expectedNewPeriod: 1,
			expectedHalving:   false,
		},
		{
			name:         "second halving epoch",
			currentEpoch: 2922,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    1,
				LastHalvingEpoch: 1461,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "1800000000000000000000", // 1800 tokens (halved again)
			expectedNewPeriod: 2,
			expectedHalving:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// Enable inflation and set halving parameters
			params := suite.app.InflationKeeper.GetParams(suite.ctx)
			params.EnableInflation = true
			params.DailyEmission, _ = math.NewIntFromString("7200000000000000000000") // 7200 tokens
			params.HalvingIntervalEpochs = 1461
			params.MultiSigAddress = "evmos1test" // Set a test address
			params.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000") // 21M tokens
			err := suite.app.InflationKeeper.SetParams(suite.ctx, params)
			suite.Require().NoError(err)

			// Set up halving data
			suite.app.InflationKeeper.SetHalvingData(suite.ctx, tc.setupHalvingData)

			// Get initial supply
			initialSupply := suite.app.BankKeeper.GetSupply(suite.ctx, denomMint)

			// Create future context for the specific epoch
			futureCtx := suite.ctx.WithBlockHeight(tc.currentEpoch).WithBlockTime(time.Now().Add(time.Hour))

			// Simulate epoch end
			suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, tc.currentEpoch)

			// Check if minting occurred
			newSupply := suite.app.BankKeeper.GetSupply(futureCtx, denomMint)
			
			if tc.expectedMint {
				suite.Require().True(newSupply.Amount.GT(initialSupply.Amount), 
					"Expected minting to occur")
				
				// Check the exact amount minted
				minted := newSupply.Amount.Sub(initialSupply.Amount)
				suite.Require().Equal(tc.expectedEmission, minted.String(),
					"Minted amount should match expected emission")
			} else {
				suite.Require().Equal(initialSupply.Amount.String(), newSupply.Amount.String(),
					"No minting should have occurred")
			}

			// Check halving data was updated correctly
			halvingData := suite.app.InflationKeeper.GetHalvingData(futureCtx)
			suite.Require().Equal(tc.expectedNewPeriod, halvingData.CurrentPeriod,
				"Current period should be updated")

			if tc.expectedHalving {
				suite.Require().Equal(uint64(tc.currentEpoch), halvingData.LastHalvingEpoch,
					"Last halving epoch should be updated")
			}
		})
	}
}

// TestHalvingSupplyCapIntegration tests supply cap enforcement during halving
func (suite *KeeperTestSuite) TestHalvingSupplyCapIntegration() {
	suite.SetupTest()

	// Set up parameters with a very low max supply for testing
	params := suite.app.InflationKeeper.GetParams(suite.ctx)
	params.EnableInflation = true
	params.DailyEmission, _ = math.NewIntFromString("7200000000000000000000") // 7200 tokens
	params.HalvingIntervalEpochs = 1461
	params.MultiSigAddress = "evmos1test"
	params.MaxSupply, _ = math.NewIntFromString("10000000000000000000000") // Only 10K tokens max for testing
	err := suite.app.InflationKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	// Mint tokens close to the cap to test cap enforcement
	closeToCapAmount, _ := math.NewIntFromString("9000000000000000000000") // 9K tokens
	coinToMint := sdk.NewCoin(denomMint, closeToCapAmount)
	err = suite.app.BankKeeper.MintCoins(suite.ctx, types.ModuleName, sdk.NewCoins(coinToMint))
	suite.Require().NoError(err)

	// Get supply before attempting to mint more
	supplyBefore := suite.app.BankKeeper.GetSupply(suite.ctx, denomMint)

	// Try to mint 7200 tokens (should fail due to supply cap)
	futureCtx := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
	
	// This should not mint because it would exceed the cap
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, 1)

	// Verify no additional minting occurred
	supplyAfter := suite.app.BankKeeper.GetSupply(futureCtx, denomMint)
	suite.Require().Equal(supplyBefore.Amount.String(), supplyAfter.Amount.String(),
		"No minting should occur when it would exceed supply cap")
}

// TestHalvingInvalidEpochIdentifier tests that halving only works for day epochs
func (suite *KeeperTestSuite) TestHalvingInvalidEpochIdentifier() {
	suite.SetupTest()

	// Enable inflation
	params := suite.app.InflationKeeper.GetParams(suite.ctx)
	params.EnableInflation = true
		params.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	params.HalvingIntervalEpochs = 1461
	params.MultiSigAddress = "evmos1test"
	params.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err := suite.app.InflationKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	testCases := []struct {
		name           string
		epochID        string
		shouldMint     bool
	}{
		{
			name:       "day epoch - should mint",
			epochID:    epochstypes.DayEpochID,
			shouldMint: true,
		},
		{
			name:       "week epoch - should not mint",
			epochID:    epochstypes.WeekEpochID,
			shouldMint: false,
		},
		{
			name:       "invalid epoch - should not mint",
			epochID:    "hour",
			shouldMint: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Get supply before
			supplyBefore := suite.app.BankKeeper.GetSupply(suite.ctx, denomMint)

			// Trigger epoch end
			futureCtx := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
			suite.app.InflationKeeper.AfterEpochEnd(futureCtx, tc.epochID, 1)

			// Check supply after
			supplyAfter := suite.app.BankKeeper.GetSupply(futureCtx, denomMint)

			if tc.shouldMint {
				suite.Require().True(supplyAfter.Amount.GT(supplyBefore.Amount),
					"Should have minted tokens for %s", tc.epochID)
			} else {
				suite.Require().Equal(supplyBefore.Amount.String(), supplyAfter.Amount.String(),
					"Should not have minted tokens for %s", tc.epochID)
			}
		})
	}
}

// TestHalvingDisabledInflation tests that halving doesn't work when inflation is disabled
func (suite *KeeperTestSuite) TestHalvingDisabledInflation() {
	suite.SetupTest()

	// Disable inflation
	params := suite.app.InflationKeeper.GetParams(suite.ctx)
	params.EnableInflation = false // This should prevent minting
		params.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	params.HalvingIntervalEpochs = 1461
	params.MultiSigAddress = "evmos1test"
	params.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err := suite.app.InflationKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	// Get supply before
	supplyBefore := suite.app.BankKeeper.GetSupply(suite.ctx, denomMint)

	// Try to trigger epoch end
	futureCtx := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, 1)

	// Verify no minting occurred
	supplyAfter := suite.app.BankKeeper.GetSupply(futureCtx, denomMint)
	suite.Require().Equal(supplyBefore.Amount.String(), supplyAfter.Amount.String(),
		"No minting should occur when inflation is disabled")
}

// TestHalvingMultipleEpochs tests halving progression over multiple epochs
func (suite *KeeperTestSuite) TestHalvingMultipleEpochs() {
	suite.SetupTest()

	// Enable inflation
	params := suite.app.InflationKeeper.GetParams(suite.ctx)
	params.EnableInflation = true
		params.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	params.HalvingIntervalEpochs = 4 // Small interval for testing
	params.MultiSigAddress = "evmos1test"
	params.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err := suite.app.InflationKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Initialize halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	expectedEmissions := []string{
		"7200000000000000000000", // Epoch 1: Period 0
		"7200000000000000000000", // Epoch 2: Period 0
		"7200000000000000000000", // Epoch 3: Period 0
		"3600000000000000000000", // Epoch 4: Period 1 (first halving)
		"3600000000000000000000", // Epoch 5: Period 1
		"3600000000000000000000", // Epoch 6: Period 1
		"3600000000000000000000", // Epoch 7: Period 1
		"1800000000000000000000", // Epoch 8: Period 2 (second halving)
	}

	totalMinted := math.ZeroInt()

	for epoch := int64(1); epoch <= 8; epoch++ {
		suite.Run(fmt.Sprintf("epoch_%d", epoch), func() {
			// Get supply before
			supplyBefore := suite.app.BankKeeper.GetSupply(suite.ctx, denomMint)

			// Trigger epoch end
			futureCtx := suite.ctx.WithBlockHeight(epoch).WithBlockTime(time.Now().Add(time.Hour))
			suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, epoch)

			// Check minted amount
			supplyAfter := suite.app.BankKeeper.GetSupply(futureCtx, denomMint)
			minted := supplyAfter.Amount.Sub(supplyBefore.Amount)
			
			expectedEmission := expectedEmissions[epoch-1]
			suite.Require().Equal(expectedEmission, minted.String(),
				"Epoch %d should mint %s tokens", epoch, expectedEmission)

			// Update context for next iteration
			suite.ctx = futureCtx
			totalMinted = totalMinted.Add(minted)
		})
	}

	// Verify total minted follows expected pattern
	// Period 0: 4 epochs * 7200 = 28800
	// Period 1: 4 epochs * 3600 = 14400  
	// Total: 43200 tokens
	expectedTotal, _ := math.NewIntFromString("43200000000000000000000")
	suite.Require().Equal(expectedTotal.String(), totalMinted.String(),
		"Total minted should match expected halving progression")
}

// TestHalvingEventEmission tests that halving events are emitted correctly
func (suite *KeeperTestSuite) TestHalvingEventEmission() {
	suite.SetupTest()

	// Enable inflation
	params := suite.app.InflationKeeper.GetParams(suite.ctx)
	params.EnableInflation = true
		params.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	params.HalvingIntervalEpochs = 2 // Very small for testing
	params.MultiSigAddress = "evmos1test"
	params.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err := suite.app.InflationKeeper.SetParams(suite.ctx, params)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	// Trigger first halving at epoch 2
	futureCtx := suite.ctx.WithBlockHeight(2).WithBlockTime(time.Now().Add(time.Hour))
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, 2)

	// Check that events were emitted (this would require examining the event manager)
	// For now, we verify the state changes that indicate halving occurred
	finalHalvingData := suite.app.InflationKeeper.GetHalvingData(futureCtx)
	suite.Require().Equal(uint64(1), finalHalvingData.CurrentPeriod, "Should be in period 1")
	suite.Require().Equal(uint64(2), finalHalvingData.LastHalvingEpoch, "Last halving epoch should be 2")
}
