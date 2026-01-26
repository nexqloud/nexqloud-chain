package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/evmos/evmos/v19/x/epochs/types"
	"github.com/stretchr/testify/suite"

	cmdcfg "github.com/evmos/evmos/v19/cmd/config"
)

type HalvingGenesisTestSuite struct {
	suite.Suite
}

func TestHalvingGenesisTestSuite(t *testing.T) {
	// Set up SDK config with nxq prefix before running tests (if not already sealed)
	config := sdk.GetConfig()
	if config.GetBech32AccountAddrPrefix() != cmdcfg.Bech32PrefixAccAddr {
		cmdcfg.SetBech32Prefixes(config)
		config.Seal()
	}
	
	suite.Run(t, new(HalvingGenesisTestSuite))
}

// TestDefaultGenesisState tests that default genesis includes proper halving data
func (suite *HalvingGenesisTestSuite) TestDefaultGenesisState() {
	genesis := DefaultGenesisState()

	// Verify halving data is included
	suite.Require().NotNil(genesis.HalvingData, "HalvingData should be included in default genesis")

	// Check default values
	suite.Require().Equal(uint64(0), genesis.HalvingData.CurrentPeriod, "Should start in period 0")
	suite.Require().Equal(uint64(0), genesis.HalvingData.LastHalvingEpoch, "Should have no previous halving")
	suite.Require().Equal(uint64(1), genesis.HalvingData.StartEpoch, "Should start from epoch 1")

	// Verify other genesis defaults include halving parameters
	params := genesis.Params
	suite.Require().Equal("7200000000000000000000", params.DailyEmission.String(), "Should have default daily emission")
	suite.Require().Equal(uint64(1461), params.HalvingIntervalEpochs, "Should have default halving interval")
	suite.Require().Equal("21000000000000000000000000", params.MaxSupply.String(), "Should have default max supply")
	suite.Require().Equal(DefaultMultiSigAddress, params.MultiSigAddress, "Should have default multi-sig address")
}

// TestNewGenesisState tests custom genesis state creation
func (suite *HalvingGenesisTestSuite) TestNewGenesisState() {
	dailyEmission, _ := math.NewIntFromString("3600000000000000000000")
	maxSupply, _ := math.NewIntFromString("10000000000000000000000000")
	params := Params{
		MintDenom:             "aevmos",
		EnableInflation:       true,
		DailyEmission:         dailyEmission, // Custom emission
		HalvingIntervalEpochs: 730,           // Custom interval
		MultiSigAddress:       "evmos1test",  // Custom address
		MaxSupply:             maxSupply,     // Custom max supply
	}

	halvingData := HalvingData{
		CurrentPeriod:    2,    // Custom period
		LastHalvingEpoch: 1460, // Custom last halving
		StartEpoch:       100,  // Custom start epoch
	}

	genesis := NewGenesisState(
		params,
		5,                      // period
		epochstypes.DayEpochID, // epoch identifier
		365,                    // epochs per period
		10,                     // skipped epochs
		halvingData,            // halving data
	)

	// Verify all fields are set correctly
	suite.Require().Equal(params, genesis.Params)
	suite.Require().Equal(uint64(5), genesis.Period)
	suite.Require().Equal(epochstypes.DayEpochID, genesis.EpochIdentifier)
	suite.Require().Equal(int64(365), genesis.EpochsPerPeriod)
	suite.Require().Equal(uint64(10), genesis.SkippedEpochs)
	suite.Require().Equal(halvingData, genesis.HalvingData)
}

// TestGenesisValidation tests genesis state validation including halving data
func (suite *HalvingGenesisTestSuite) TestGenesisValidation() {
	testCases := []struct {
		name        string
		genesis     GenesisState
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default genesis",
			genesis:     *DefaultGenesisState(),
			expectError: false,
		},
		{
			name: "valid custom genesis",
			genesis: GenesisState{
				Params:          DefaultParams(),
				Period:          0,
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   0,
				HalvingData: HalvingData{
					CurrentPeriod:    1,
					LastHalvingEpoch: 1461,
					StartEpoch:       1,
				},
			},
			expectError: false,
		},
		{
			name: "invalid - zero start epoch",
			genesis: GenesisState{
				Params:          DefaultParams(),
				Period:          0,
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   0,
				HalvingData: HalvingData{
					CurrentPeriod:    0,
					LastHalvingEpoch: 0,
					StartEpoch:       0, // Invalid: should be positive
				},
			},
			expectError: true,
			errorMsg:    "halving start epoch must be positive",
		},
		{
			name: "empty epoch identifier",
			genesis: GenesisState{
				Params:          DefaultParams(),
				Period:          0,
				EpochIdentifier: "",  // Invalid: empty/blank epoch identifier
				EpochsPerPeriod: 365,
				SkippedEpochs:   0,
				HalvingData: HalvingData{
					CurrentPeriod:    0,
					LastHalvingEpoch: 0,
					StartEpoch:       1,
				},
			},
			expectError: true,
			errorMsg:    "epoch identifier",
		},
		{
			name: "invalid parameters",
			genesis: GenesisState{
				Params: func() Params {
					dailyEmission, _ := math.NewIntFromString("7200000000000000000000")
					maxSupply, _ := math.NewIntFromString("21000000000000000000000000")
					return Params{
						MintDenom:             "", // Invalid: empty denom
						EnableInflation:       true,
						DailyEmission:         dailyEmission,
						HalvingIntervalEpochs: 1461,
						MultiSigAddress:       "evmos1test",
						MaxSupply:             maxSupply,
					}
				}(),
				Period:          0,
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   0,
				HalvingData: HalvingData{
					CurrentPeriod:    0,
					LastHalvingEpoch: 0,
					StartEpoch:       1,
				},
			},
			expectError: true,
			errorMsg:    "mint denom",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.genesis.Validate()

			if tc.expectError {
				suite.Require().Error(err, "Expected validation error for case: %s", tc.name)
				if tc.errorMsg != "" {
					suite.Require().Contains(err.Error(), tc.errorMsg, "Error message should contain: %s", tc.errorMsg)
				}
			} else {
				suite.Require().NoError(err, "Expected no validation error for case: %s", tc.name)
			}
		})
	}
}

// TestHalvingDataValidation tests the halving data validation function specifically
func (suite *HalvingGenesisTestSuite) TestHalvingDataValidation() {
	testCases := []struct {
		name        string
		halvingData HalvingData
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid halving data",
			halvingData: HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       1,
			},
			expectError: false,
		},
		{
			name: "valid halving data - in progress",
			halvingData: HalvingData{
				CurrentPeriod:    3,
				LastHalvingEpoch: 4383,
				StartEpoch:       1,
			},
			expectError: false,
		},
		{
			name: "invalid - zero start epoch",
			halvingData: HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       0,
			},
			expectError: true,
			errorMsg:    "halving start epoch must be positive",
		},
		{
			name: "valid - different start epoch",
			halvingData: HalvingData{
				CurrentPeriod:    0,
				LastHalvingEpoch: 0,
				StartEpoch:       100,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := validateHalvingData(tc.halvingData)

			if tc.expectError {
				suite.Require().Error(err, "Expected validation error for case: %s", tc.name)
				if tc.errorMsg != "" {
					suite.Require().Contains(err.Error(), tc.errorMsg, "Error message should contain: %s", tc.errorMsg)
				}
			} else {
				suite.Require().NoError(err, "Expected no validation error for case: %s", tc.name)
			}
		})
	}
}

// TestGenesisConsistency tests that genesis parameters are consistent with halving logic
func (suite *HalvingGenesisTestSuite) TestGenesisConsistency() {
	testCases := []struct {
		name        string
		genesis     GenesisState
		description string
	}{
		{
			name:        "default genesis consistency",
			genesis:     *DefaultGenesisState(),
			description: "Default genesis should have consistent halving parameters",
		},
		{
			name: "custom genesis consistency",
			genesis: GenesisState{
				Params: func() Params {
					dailyEmission, _ := math.NewIntFromString("3600000000000000000000")
					maxSupply, _ := math.NewIntFromString("10000000000000000000000000")
					return Params{
						MintDenom:             "aevmos",
						EnableInflation:       true,
						DailyEmission:         dailyEmission, // 3600 tokens
						HalvingIntervalEpochs: 730,           // 2 years
						MultiSigAddress:       "evmos1test",
						MaxSupply:             maxSupply, // 10M tokens
					}
				}(),
				Period:          0,
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   0,
				HalvingData: HalvingData{
					CurrentPeriod:    0,
					LastHalvingEpoch: 0,
					StartEpoch:       1,
				},
			},
			description: "Custom genesis with smaller supply cap should be consistent",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Validate the genesis state
			err := tc.genesis.Validate()
			suite.Require().NoError(err, tc.description)

			// Test consistency: initial emission should not exceed max supply
			initialEmission := CalculateDailyEmission(tc.genesis.Params, tc.genesis.HalvingData.CurrentPeriod)
			suite.Require().True(initialEmission.LT(tc.genesis.Params.MaxSupply),
				"Initial emission should be less than max supply")

			// Test consistency: start epoch should be positive
			suite.Require().Greater(tc.genesis.HalvingData.StartEpoch, uint64(0),
				"Start epoch should be positive")

			// Test consistency: if current period > 0, last halving epoch should be set
			if tc.genesis.HalvingData.CurrentPeriod > 0 {
				suite.Require().Greater(tc.genesis.HalvingData.LastHalvingEpoch, uint64(0),
					"Last halving epoch should be set if we're in a halving period")
			}
		})
	}
}

// TestGenesisWithDifferentStartEpochs tests genesis validation with various start epochs
func (suite *HalvingGenesisTestSuite) TestGenesisWithDifferentStartEpochs() {
	baseParams := DefaultParams()

	testCases := []struct {
		name       string
		startEpoch uint64
		valid      bool
	}{
		{
			name:       "start epoch 1",
			startEpoch: 1,
			valid:      true,
		},
		{
			name:       "start epoch 100",
			startEpoch: 100,
			valid:      true,
		},
		{
			name:       "start epoch 1000",
			startEpoch: 1000,
			valid:      true,
		},
		{
			name:       "start epoch 0 - invalid",
			startEpoch: 0,
			valid:      false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			genesis := GenesisState{
				Params:          baseParams,
				Period:          0,
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   0,
				HalvingData: HalvingData{
					CurrentPeriod:    0,
					LastHalvingEpoch: 0,
					StartEpoch:       tc.startEpoch,
				},
			}

			err := genesis.Validate()

			if tc.valid {
				suite.Require().NoError(err, "Genesis should be valid for start epoch %d", tc.startEpoch)
			} else {
				suite.Require().Error(err, "Genesis should be invalid for start epoch %d", tc.startEpoch)
			}
		})
	}
}

// TestGenesisUpgradeScenarios tests genesis for chain upgrade scenarios
func (suite *HalvingGenesisTestSuite) TestGenesisUpgradeScenarios() {
	testCases := []struct {
		name        string
		description string
		genesis     GenesisState
	}{
		{
			name:        "new chain deployment",
			description: "Fresh chain with halving from genesis",
			genesis: GenesisState{
				Params:          DefaultParams(),
				Period:          0,
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   0,
				HalvingData: HalvingData{
					CurrentPeriod:    0,
					LastHalvingEpoch: 0,
					StartEpoch:       1,
				},
			},
		},
		{
			name:        "chain upgrade - mid halving period",
			description: "Existing chain upgrading to halving, currently in period 1",
			genesis: GenesisState{
				Params:          DefaultParams(),
				Period:          1461, // Existing period
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   100, // Some epochs skipped
				HalvingData: HalvingData{
					CurrentPeriod:    1,    // Already had one halving
					LastHalvingEpoch: 1461, // Last halving at epoch 1461
					StartEpoch:       1,    // Started from epoch 1
				},
			},
		},
		{
			name:        "chain upgrade - multiple halvings done",
			description: "Chain with multiple halvings already completed",
			genesis: GenesisState{
				Params:          DefaultParams(),
				Period:          5000,
				EpochIdentifier: epochstypes.DayEpochID,
				EpochsPerPeriod: 365,
				SkippedEpochs:   50,
				HalvingData: HalvingData{
					CurrentPeriod:    3,    // Third halving period
					LastHalvingEpoch: 4383, // Last halving at epoch 4383
					StartEpoch:       1,
				},
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Validate the genesis state
			err := tc.genesis.Validate()
			suite.Require().NoError(err, "%s: %s", tc.name, tc.description)

			// For upgrade scenarios, verify the halving data is consistent
			if tc.genesis.HalvingData.CurrentPeriod > 0 {
				// Should have a valid last halving epoch
				suite.Require().Greater(tc.genesis.HalvingData.LastHalvingEpoch, uint64(0),
					"Upgrade scenario should have valid last halving epoch")

				// Current emission should be halved appropriately
				currentEmission := CalculateDailyEmission(tc.genesis.Params, tc.genesis.HalvingData.CurrentPeriod)
				fullEmission := CalculateDailyEmission(tc.genesis.Params, 0)

				// Current emission should be less than full emission for non-zero periods
				suite.Require().True(currentEmission.LT(fullEmission),
					"Current emission should be reduced in halving periods")
			}
		})
	}
}
