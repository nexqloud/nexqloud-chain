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
		name              string
		currentEpoch      int64
		setupHalvingData  types.HalvingData
		expectedMint      bool
		expectedEmission  string
		expectedNewPeriod uint64
		expectedHalving   bool
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
			name:         "epoch 1461 ends - period 0",
			currentEpoch: 1461,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "7200000000000000000000", // 7200 tokens (epoch 1460 ended, period 0)
			expectedNewPeriod: 0,
			expectedHalving:   false,
		},
		{
			name:         "epoch 1462 ends - last period 0",
			currentEpoch: 1462,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "7200000000000000000000", // 7200 tokens (epoch 1461 ended, period 0 - last full emission!)
			expectedNewPeriod: 0,
			expectedHalving:   false,
		},
		{
			name:         "epoch 1463 ends - first halving!",
			currentEpoch: 1463,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "3600000000000000000000", // 3600 tokens (epoch 1462 ended, period 1 - FIRST HALVING!)
			expectedNewPeriod: 1,
			expectedHalving:   true,
		},
		{
			name:         "epoch 2923 ends - period 1",
			currentEpoch: 2923,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    1,
				LastHalvingEpoch: 1463,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "3600000000000000000000", // 3600 tokens (epoch 2922 ended, period 1 - last of period 1)
			expectedNewPeriod: 1,
			expectedHalving:   false,
		},
		{
			name:         "epoch 2924 ends - second halving!",
			currentEpoch: 2924,
			setupHalvingData: types.HalvingData{
				CurrentPeriod:    1,
				LastHalvingEpoch: 1463,
				StartEpoch:       1,
			},
			expectedMint:      true,
			expectedEmission:  "1800000000000000000000", // 1800 tokens (epoch 2923 ended, period 2 - SECOND HALVING!)
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
			// Generate a valid test bech32 address
			testAddr := sdk.AccAddress([]byte("test_halving_addr"))
			params.MultiSigAddress, _ = sdk.Bech32ifyAddressBytes("nxq", testAddr)
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
				// LastHalvingEpoch should be the epoch that ENDED (epochNumber - 1)
				suite.Require().Equal(uint64(tc.currentEpoch-1), halvingData.LastHalvingEpoch,
					"Last halving epoch should be updated to the epoch that ended")
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
	// Generate a valid test bech32 address
	testAddr := sdk.AccAddress([]byte("test_disabled_addr"))
	params.MultiSigAddress, _ = sdk.Bech32ifyAddressBytes("nxq", testAddr)
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
	// Generate a valid test bech32 address
	testAddr := sdk.AccAddress([]byte("test_disabled_addr"))
	params.MultiSigAddress, _ = sdk.Bech32ifyAddressBytes("nxq", testAddr)
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
		name       string
		epochID    string
		shouldMint bool
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
	// Generate a valid test bech32 address
	testAddr := sdk.AccAddress([]byte("test_disabled_addr"))
	params.MultiSigAddress, _ = sdk.Bech32ifyAddressBytes("nxq", testAddr)
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
	// Generate a valid test bech32 address
	testAddr := sdk.AccAddress([]byte("test_disabled_addr"))
	params.MultiSigAddress, _ = sdk.Bech32ifyAddressBytes("nxq", testAddr)
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
	// Generate a valid test bech32 address
	testAddr := sdk.AccAddress([]byte("test_disabled_addr"))
	params.MultiSigAddress, _ = sdk.Bech32ifyAddressBytes("nxq", testAddr)
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

// TestMultiSigAddressFromEVMParams tests that MultiSigAddress is read from EVM params (primary source)
func (suite *KeeperTestSuite) TestMultiSigAddressFromEVMParams() {
	suite.SetupTest()

	// Generate valid bech32 addresses for testing
	primaryAddr := sdk.AccAddress("primary12345678901234") // 20 bytes
	primaryAddrStr, err := sdk.Bech32ifyAddressBytes("nxq", primaryAddr)
	suite.Require().NoError(err)

	fallbackAddr := sdk.AccAddress("fallback12345678901") // 20 bytes
	fallbackAddrStr, err := sdk.Bech32ifyAddressBytes("nxq", fallbackAddr)
	suite.Require().NoError(err)

	// Set up inflation params with a different address (fallback)
	inflationParams := suite.app.InflationKeeper.GetParams(suite.ctx)
	inflationParams.EnableInflation = true
	inflationParams.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	inflationParams.HalvingIntervalEpochs = 1461
	inflationParams.MultiSigAddress = fallbackAddrStr // Fallback address
	inflationParams.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err = suite.app.InflationKeeper.SetParams(suite.ctx, inflationParams)
	suite.Require().NoError(err)

	// Set EVM params with MultiSigAddress (primary source)
	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	evmParams.MultiSigAddress = primaryAddrStr // Primary address
	err = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	// Get initial multi-sig balance
	multiSigAddr, err := sdk.AccAddressFromBech32(primaryAddrStr)
	suite.Require().NoError(err)
	initialBalance := suite.app.BankKeeper.GetBalance(suite.ctx, multiSigAddr, denomMint)

	// Trigger epoch end - should use EVM params address (primary source)
	futureCtx := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, 1)

	// Verify tokens were sent to EVM params address (not fallback)
	finalBalance := suite.app.BankKeeper.GetBalance(futureCtx, multiSigAddr, denomMint)
	minted := finalBalance.Amount.Sub(initialBalance.Amount)
	suite.Require().True(minted.IsPositive(), "Tokens should be minted to EVM params address")
	suite.Require().Equal("7200000000000000000000", minted.String(), "Should mint 7200 tokens")

	// Verify fallback address did NOT receive tokens
	fallbackAddrParsed, err := sdk.AccAddressFromBech32(fallbackAddrStr)
	suite.Require().NoError(err)
	fallbackBalance := suite.app.BankKeeper.GetBalance(futureCtx, fallbackAddrParsed, denomMint)
	suite.Require().True(fallbackBalance.Amount.IsZero(), "Fallback address should not receive tokens")
}

// TestMultiSigAddressFallbackToInflationParams tests fallback to inflation params when EVM params is empty
func (suite *KeeperTestSuite) TestMultiSigAddressFallbackToInflationParams() {
	suite.SetupTest()

	// Generate valid bech32 address for testing
	fallbackAddr := sdk.AccAddress("fallback12345678901") // 20 bytes
	fallbackAddrStr, err := sdk.Bech32ifyAddressBytes("nxq", fallbackAddr)
	suite.Require().NoError(err)

	// Set up inflation params with MultiSigAddress (fallback)
	inflationParams := suite.app.InflationKeeper.GetParams(suite.ctx)
	inflationParams.EnableInflation = true
	inflationParams.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	inflationParams.HalvingIntervalEpochs = 1461
	inflationParams.MultiSigAddress = fallbackAddrStr // Fallback address
	inflationParams.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err = suite.app.InflationKeeper.SetParams(suite.ctx, inflationParams)
	suite.Require().NoError(err)

	// Set EVM params with empty MultiSigAddress (bootstrap mode)
	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	evmParams.MultiSigAddress = "" // Empty (bootstrap mode)
	err = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	// Get initial multi-sig balance
	multiSigAddr, err := sdk.AccAddressFromBech32(fallbackAddrStr)
	suite.Require().NoError(err)
	initialBalance := suite.app.BankKeeper.GetBalance(suite.ctx, multiSigAddr, denomMint)

	// Trigger epoch end - should use inflation params address (fallback)
	futureCtx := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, 1)

	// Verify tokens were sent to inflation params address (fallback)
	finalBalance := suite.app.BankKeeper.GetBalance(futureCtx, multiSigAddr, denomMint)
	minted := finalBalance.Amount.Sub(initialBalance.Amount)
	suite.Require().True(minted.IsPositive(), "Tokens should be minted to inflation params address (fallback)")
	suite.Require().Equal("7200000000000000000000", minted.String(), "Should mint 7200 tokens")
}

// TestMultiSigAddressGovernanceUpdate tests that updating EVM params via governance changes the address
func (suite *KeeperTestSuite) TestMultiSigAddressGovernanceUpdate() {
	suite.SetupTest()

	// Generate valid bech32 addresses for testing
	firstAddr := sdk.AccAddress("first123456789012345") // 20 bytes
	firstAddrStr, err := sdk.Bech32ifyAddressBytes("nxq", firstAddr)
	suite.Require().NoError(err)

	newAddr := sdk.AccAddress("new12345678901234567") // 20 bytes
	newAddrStr, err := sdk.Bech32ifyAddressBytes("nxq", newAddr)
	suite.Require().NoError(err)

	// Set up inflation params
	inflationParams := suite.app.InflationKeeper.GetParams(suite.ctx)
	inflationParams.EnableInflation = true
	inflationParams.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	inflationParams.HalvingIntervalEpochs = 1461
	inflationParams.MultiSigAddress = "nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5" // Old fallback
	inflationParams.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err = suite.app.InflationKeeper.SetParams(suite.ctx, inflationParams)
	suite.Require().NoError(err)

	// Initially set EVM params with first address
	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	evmParams.MultiSigAddress = firstAddrStr // First address
	err = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	// First mint - should go to first address
	addr1, err := sdk.AccAddressFromBech32(firstAddrStr)
	suite.Require().NoError(err)
	initialBalance1 := suite.app.BankKeeper.GetBalance(suite.ctx, addr1, denomMint)

	futureCtx1 := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx1, epochstypes.DayEpochID, 1)

	balance1After := suite.app.BankKeeper.GetBalance(futureCtx1, addr1, denomMint)
	minted1 := balance1After.Amount.Sub(initialBalance1.Amount)
	suite.Require().Equal("7200000000000000000000", minted1.String(), "First mint should go to first address")

	// Simulate governance update - change EVM params MultiSigAddress
	evmParams = suite.app.EvmKeeper.GetParams(futureCtx1)
	evmParams.MultiSigAddress = newAddrStr // New address via governance
	err = suite.app.EvmKeeper.SetParams(futureCtx1, evmParams)
	suite.Require().NoError(err)

	// Second mint - should now go to new address
	addr2, err := sdk.AccAddressFromBech32(newAddrStr)
	suite.Require().NoError(err)
	initialBalance2 := suite.app.BankKeeper.GetBalance(futureCtx1, addr2, denomMint)

	futureCtx2 := futureCtx1.WithBlockHeight(2).WithBlockTime(time.Now().Add(2 * time.Hour))
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx2, epochstypes.DayEpochID, 2)

	balance2After := suite.app.BankKeeper.GetBalance(futureCtx2, addr2, denomMint)
	minted2 := balance2After.Amount.Sub(initialBalance2.Amount)
	suite.Require().Equal("7200000000000000000000", minted2.String(), "Second mint should go to new address")

	// Verify first address did NOT receive second mint
	balance1Final := suite.app.BankKeeper.GetBalance(futureCtx2, addr1, denomMint)
	suite.Require().Equal(balance1After.Amount.String(), balance1Final.Amount.String(),
		"First address should not receive tokens after governance update")
}

// TestMultiSigAddressEmptyBothParams tests behavior when both EVM and inflation params are empty
func (suite *KeeperTestSuite) TestMultiSigAddressEmptyBothParams() {
	suite.SetupTest()

	// Set up inflation params with empty MultiSigAddress
	inflationParams := suite.app.InflationKeeper.GetParams(suite.ctx)
	inflationParams.EnableInflation = true
	inflationParams.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
	inflationParams.HalvingIntervalEpochs = 1461
	inflationParams.MultiSigAddress = "" // Empty
	inflationParams.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
	err := suite.app.InflationKeeper.SetParams(suite.ctx, inflationParams)
	suite.Require().NoError(err)

	// Set EVM params with empty MultiSigAddress (bootstrap mode)
	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	evmParams.MultiSigAddress = "" // Empty (bootstrap mode)
	err = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)
	suite.Require().NoError(err)

	// Set halving data
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       1,
	}
	suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

	// Get initial supply
	initialSupply := suite.app.BankKeeper.GetSupply(suite.ctx, denomMint)

	// Trigger epoch end - should fallback to standard distribution (no multi-sig)
	futureCtx := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
	suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, 1)

	// Verify tokens were still minted (but distributed via standard mechanism, not multi-sig)
	finalSupply := suite.app.BankKeeper.GetSupply(futureCtx, denomMint)
	minted := finalSupply.Amount.Sub(initialSupply.Amount)
	suite.Require().True(minted.IsPositive(), "Tokens should still be minted even without multi-sig address")
	suite.Require().Equal("7200000000000000000000", minted.String(), "Should mint 7200 tokens")
}

// TestMultiSigAddressPriorityOrder tests the priority: EVM params > Inflation params > Standard distribution
func (suite *KeeperTestSuite) TestMultiSigAddressPriorityOrder() {
	testCases := []struct {
		name                     string
		evmMultiSigAddress       string
		inflationMultiSigAddress string
		expectedAddress          string
		description              string
	}{
		{
			name:                     "EVM params set, inflation params set - use EVM",
			evmMultiSigAddress:       "", // Will be set in test
			inflationMultiSigAddress: "", // Will be set in test
			expectedAddress:          "", // Will be set in test
			description:              "EVM params takes priority over inflation params",
		},
		{
			name:                     "EVM params empty, inflation params set - use inflation",
			evmMultiSigAddress:       "",
			inflationMultiSigAddress: "", // Will be set in test
			expectedAddress:          "", // Will be set in test
			description:              "Fallback to inflation params when EVM params empty",
		},
		{
			name:                     "Both empty - use standard distribution",
			evmMultiSigAddress:       "",
			inflationMultiSigAddress: "",
			expectedAddress:          "", // Empty means standard distribution
			description:              "Both empty triggers standard distribution",
		},
	}

	for i, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupTest()

			// Generate valid bech32 addresses for test cases that need them
			var evmAddrStr, inflationAddrStr, expectedAddrStr string

			if i == 0 {
				// First case: EVM params set, inflation params set - use EVM
				evmAddr := sdk.AccAddress(fmt.Sprintf("evmaddr%d123456789012", i))
				evmAddrStr, _ = sdk.Bech32ifyAddressBytes("nxq", evmAddr)
				inflationAddr := sdk.AccAddress(fmt.Sprintf("infaddr%d123456789012", i))
				inflationAddrStr, _ = sdk.Bech32ifyAddressBytes("nxq", inflationAddr)
				expectedAddrStr = evmAddrStr
			} else if i == 1 {
				// Second case: EVM params empty, inflation params set - use inflation
				inflationAddr := sdk.AccAddress(fmt.Sprintf("infaddr%d123456789012", i))
				inflationAddrStr, _ = sdk.Bech32ifyAddressBytes("nxq", inflationAddr)
				expectedAddrStr = inflationAddrStr
			}
			// Third case: Both empty - no addresses needed

			// Set up inflation params
			inflationParams := suite.app.InflationKeeper.GetParams(suite.ctx)
			inflationParams.EnableInflation = true
			inflationParams.DailyEmission, _ = math.NewIntFromString("7200000000000000000000")
			inflationParams.HalvingIntervalEpochs = 1461
			inflationParams.MultiSigAddress = inflationAddrStr
			inflationParams.MaxSupply, _ = math.NewIntFromString("21000000000000000000000000")
			err := suite.app.InflationKeeper.SetParams(suite.ctx, inflationParams)
			suite.Require().NoError(err)

			// Set EVM params
			evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
			evmParams.MultiSigAddress = evmAddrStr
			err = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)
			suite.Require().NoError(err)

			// Set halving data
			halvingData := types.HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			}
			suite.app.InflationKeeper.SetHalvingData(suite.ctx, halvingData)

			// Get initial supply
			initialSupply := suite.app.BankKeeper.GetSupply(suite.ctx, denomMint)

			// Trigger epoch end
			futureCtx := suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now().Add(time.Hour))
			suite.app.InflationKeeper.AfterEpochEnd(futureCtx, epochstypes.DayEpochID, 1)

			// Verify minting occurred
			finalSupply := suite.app.BankKeeper.GetSupply(futureCtx, denomMint)
			minted := finalSupply.Amount.Sub(initialSupply.Amount)
			suite.Require().True(minted.IsPositive(), tc.description)
			suite.Require().Equal("7200000000000000000000", minted.String(), "Should mint 7200 tokens")

			// If expected address is set, verify tokens went there
			if expectedAddrStr != "" {
				expectedAddr, err := sdk.AccAddressFromBech32(expectedAddrStr)
				suite.Require().NoError(err)
				balance := suite.app.BankKeeper.GetBalance(futureCtx, expectedAddr, denomMint)
				suite.Require().Equal(minted.String(), balance.Amount.String(),
					"Tokens should be sent to %s: %s", expectedAddrStr, tc.description)
			}
		})
	}
}
