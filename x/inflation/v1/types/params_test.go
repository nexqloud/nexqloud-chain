package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type ParamsTestSuite struct {
	suite.Suite
}

func TestParamsTestSuite(t *testing.T) {
	suite.Run(t, new(ParamsTestSuite))
}

func (suite *ParamsTestSuite) TestParamsValidate() {
	validExponentialCalculation := ExponentialCalculation{
		A:             math.LegacyNewDec(int64(300_000_000)),
		R:             math.LegacyNewDecWithPrec(5, 1),
		C:             math.LegacyNewDec(int64(9_375_000)),
		BondingTarget: math.LegacyNewDecWithPrec(50, 2),
		MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
	}

	validInflationDistribution := InflationDistribution{
		StakingRewards:  math.LegacyNewDecWithPrec(533334, 6),
		UsageIncentives: math.LegacyZeroDec(),
		CommunityPool:   math.LegacyNewDecWithPrec(466666, 6),
	}

	testCases := []struct {
		name     string
		params   Params
		expError bool
	}{
		{
			"default",
			DefaultParams(),
			false,
		},
		{
			"valid",
			NewParams(
				"aevmos",
				validExponentialCalculation,
				validInflationDistribution,
				true,
			),
			false,
		},
		{
			"valid param literal",
			Params{
				MintDenom:              "aevmos",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
			},
			false,
		},
		{
			"invalid - denom",
			NewParams(
				"/aevmos",
				validExponentialCalculation,
				validInflationDistribution,
				true,
			),
			true,
		},
		{
			"invalid - denom",
			Params{
				MintDenom:              "",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
			},
			true,
		},
		{
			"invalid - exponential calculation - negative A",
			Params{
				MintDenom: "aevmos",
				ExponentialCalculation: ExponentialCalculation{
					A:             math.LegacyNewDec(int64(-1)),
					R:             math.LegacyNewDecWithPrec(5, 1),
					C:             math.LegacyNewDec(int64(9_375_000)),
					BondingTarget: math.LegacyNewDecWithPrec(50, 2),
					MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
				},
				InflationDistribution: validInflationDistribution,
				EnableInflation:       true,
			},
			true,
		},
		{
			"invalid - exponential calculation - R greater than 1",
			Params{
				MintDenom: "aevmos",
				ExponentialCalculation: ExponentialCalculation{
					A:             math.LegacyNewDec(int64(300_000_000)),
					R:             math.LegacyNewDecWithPrec(5, 0),
					C:             math.LegacyNewDec(int64(9_375_000)),
					BondingTarget: math.LegacyNewDecWithPrec(50, 2),
					MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
				},
				InflationDistribution: validInflationDistribution,
				EnableInflation:       true,
			},
			true,
		},
		{
			"invalid - exponential calculation - negative R",
			Params{
				MintDenom: "aevmos",
				ExponentialCalculation: ExponentialCalculation{
					A:             math.LegacyNewDec(int64(300_000_000)),
					R:             math.LegacyNewDecWithPrec(-5, 1),
					C:             math.LegacyNewDec(int64(9_375_000)),
					BondingTarget: math.LegacyNewDecWithPrec(50, 2),
					MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
				},
				InflationDistribution: validInflationDistribution,
				EnableInflation:       true,
			},
			true,
		},
		{
			"invalid - exponential calculation - negative C",
			Params{
				MintDenom: "aevmos",
				ExponentialCalculation: ExponentialCalculation{
					A:             math.LegacyNewDec(int64(300_000_000)),
					R:             math.LegacyNewDecWithPrec(5, 1),
					C:             math.LegacyNewDec(int64(-9_375_000)),
					BondingTarget: math.LegacyNewDecWithPrec(50, 2),
					MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
				},
				InflationDistribution: validInflationDistribution,
				EnableInflation:       true,
			},
			true,
		},
		{
			"invalid - exponential calculation - BondingTarget greater than 1",
			Params{
				MintDenom: "aevmos",
				ExponentialCalculation: ExponentialCalculation{
					A:             math.LegacyNewDec(int64(300_000_000)),
					R:             math.LegacyNewDecWithPrec(5, 1),
					C:             math.LegacyNewDec(int64(9_375_000)),
					BondingTarget: math.LegacyNewDecWithPrec(50, 1),
					MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
				},
				InflationDistribution: validInflationDistribution,
				EnableInflation:       true,
			},
			true,
		},
		{
			"invalid - exponential calculation - negative BondingTarget",
			Params{
				MintDenom: "aevmos",
				ExponentialCalculation: ExponentialCalculation{
					A:             math.LegacyNewDec(int64(300_000_000)),
					R:             math.LegacyNewDecWithPrec(5, 1),
					C:             math.LegacyNewDec(int64(9_375_000)),
					BondingTarget: math.LegacyNewDecWithPrec(50, 2).Neg(),
					MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
				},
				InflationDistribution: validInflationDistribution,
				EnableInflation:       true,
			},
			true,
		},
		{
			"invalid - exponential calculation - negative max Variance",
			Params{
				MintDenom: "aevmos",
				ExponentialCalculation: ExponentialCalculation{
					A:             math.LegacyNewDec(int64(300_000_000)),
					R:             math.LegacyNewDecWithPrec(5, 1),
					C:             math.LegacyNewDec(int64(9_375_000)),
					BondingTarget: math.LegacyNewDecWithPrec(50, 2),
					MaxVariance:   math.LegacyNewDecWithPrec(20, 2).Neg(),
				},
				InflationDistribution: validInflationDistribution,
				EnableInflation:       true,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative staking rewards",
			Params{
				MintDenom:              "aevmos",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:  math.LegacyOneDec().Neg(),
					UsageIncentives: math.LegacyNewDecWithPrec(333333, 6),
					CommunityPool:   math.LegacyNewDecWithPrec(133333, 6),
				},
				EnableInflation: true,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative usage incentives",
			Params{
				MintDenom:              "aevmos",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:  math.LegacyNewDecWithPrec(533334, 6),
					UsageIncentives: math.LegacyOneDec().Neg(),
					CommunityPool:   math.LegacyNewDecWithPrec(133333, 6),
				},
				EnableInflation: true,
			},
			true,
		},
		{
			"invalid - inflation distribution - negative community pool rewards",
			Params{
				MintDenom:              "aevmos",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:  math.LegacyNewDecWithPrec(533334, 6),
					UsageIncentives: math.LegacyNewDecWithPrec(333333, 6),
					CommunityPool:   math.LegacyOneDec().Neg(),
				},
				EnableInflation: true,
			},
			true,
		},
		{
			"invalid - inflation distribution - total distribution ratio unequal 1",
			Params{
				MintDenom:              "aevmos",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution: InflationDistribution{
					StakingRewards:  math.LegacyNewDecWithPrec(533333, 6),
					UsageIncentives: math.LegacyNewDecWithPrec(333333, 6),
					CommunityPool:   math.LegacyNewDecWithPrec(133333, 6),
				},
				EnableInflation: true,
			},
			true,
		},

		{
			"invalid - multisig address - invalid bech32",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        "invalid_address",
				MaxSupply:              math.NewInt(21_000_000),
			},
			true,
		},
		{
			"valid - multisig address - empty (fallback to EVM params)",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        "",
				MaxSupply:              math.NewInt(21_000_000),
			},
			false,
		},
		{
			"valid - multisig address - valid bech32",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        sdk.AccAddress([]byte("test_address_12345678")).String(),
				MaxSupply:              math.NewInt(21_000_000),
			},
			false,
		},
		// 🆕 CERTIK ISSUE #8: Division-by-zero and halving params validation
		{
			"invalid - halving interval - zero (division-by-zero)",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  0, // CRITICAL: Would cause panic
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.NewInt(21_000_000),
			},
			true,
		},
		{
			"invalid - daily emission - zero",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.ZeroInt(),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.NewInt(21_000_000),
			},
			true,
		},
		{
			"invalid - daily emission - negative",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(-7200),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.NewInt(21_000_000),
			},
			true,
		},
		{
			"invalid - max supply - zero",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.ZeroInt(),
			},
			true,
		},
		{
			"invalid - max supply - negative",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.NewInt(-21_000_000),
			},
			true,
		},
		{
			"invalid - daily emission exceeds max supply",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(100_000_000),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.NewInt(21_000_000),
			},
			true,
		},
	}

	for _, tc := range testCases {
		err := tc.params.Validate()

		if tc.expError {
			suite.Require().Error(err, tc.name)
		} else {
			suite.Require().NoError(err, tc.name)
		}
	}
}

// 🆕 CERTIK ISSUE #8: Test that SetParams validates params before storing
func (suite *ParamsTestSuite) TestValidateHalvingParams_DivisionByZero() {
	// This test ensures governance cannot set halving_interval_epochs = 0
	// which would cause CalculateHalvingPeriod to panic with division-by-zero

	validExponentialCalculation := ExponentialCalculation{
		A:             math.LegacyNewDec(int64(300_000_000)),
		R:             math.LegacyNewDecWithPrec(5, 1),
		C:             math.LegacyNewDec(int64(9_375_000)),
		BondingTarget: math.LegacyNewDecWithPrec(50, 2),
		MaxVariance:   math.LegacyNewDecWithPrec(20, 2),
	}

	validInflationDistribution := InflationDistribution{
		StakingRewards:  math.LegacyNewDecWithPrec(533334, 6),
		UsageIncentives: math.LegacyZeroDec(),
		CommunityPool:   math.LegacyNewDecWithPrec(466666, 6),
	}

	testCases := []struct {
		name     string
		params   Params
		expError bool
	}{
		{
			"valid halving params",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  1461,
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.NewInt(21_000_000),
			},
			false,
		},
		{
			"CRITICAL: halving_interval_epochs = 0 causes division-by-zero",
			Params{
				MintDenom:              "unxq",
				ExponentialCalculation: validExponentialCalculation,
				InflationDistribution:  validInflationDistribution,
				EnableInflation:        true,
				DailyEmission:          math.NewInt(7200),
				HalvingIntervalEpochs:  0, // Would cause panic without validation
				MultiSigAddress:        sdk.AccAddress([]byte("multisig_address_123")).String(),
				MaxSupply:              math.NewInt(21_000_000),
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := ValidateHalvingParams(tc.params)

			if tc.expError {
				suite.Require().Error(err, "expected error for %s", tc.name)
				suite.Require().Contains(err.Error(), "halving interval epochs must be positive", "error should mention halving interval")
			} else {
				suite.Require().NoError(err, "expected no error for %s", tc.name)
			}
		})
	}
}
