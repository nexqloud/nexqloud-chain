// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package types

import (
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evm "github.com/evmos/evmos/v19/x/evm/types"
)

var ParamsKey = []byte("Params")

var (
	DefaultInflationDenom         = evm.DefaultEVMDenom
	DefaultInflation              = true // ✅ ENABLED for halving system
	DefaultExponentialCalculation = ExponentialCalculation{
		A:             math.LegacyNewDec(int64(300_000_000)),
		R:             math.LegacyNewDecWithPrec(50, 2), // 50%
		C:             math.LegacyNewDec(int64(9_375_000)),
		BondingTarget: math.LegacyNewDecWithPrec(66, 2), // 66%
		MaxVariance:   math.LegacyZeroDec(),             // 0%
	}
	DefaultInflationDistribution = InflationDistribution{
		StakingRewards:  math.LegacyNewDecWithPrec(533333334, 9), // 0.53
		CommunityPool:   math.LegacyNewDecWithPrec(466666666, 9), // 0.47
		UsageIncentives: math.LegacyZeroDec(),                    // Deprecated
	}

	// 🆕 Halving system default parameters
	DefaultDailyEmission         = "7200000000000000000000"                     // 7200 tokens with 18 decimals
	DefaultHalvingIntervalEpochs = uint64(1461)                                 // changing from 2 to 1461 i.e 2 days to 4 years                              // 4 years = 1461 daily epochs
	DefaultMultiSigAddress       = "nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5" // Multi-sig address fallback (primary source is EVM params via governance)
	DefaultMaxSupply             = "21000000000000000000000000"                 // 21M tokens with 18 decimals
)

func NewParams(
	mintDenom string,
	exponentialCalculation ExponentialCalculation,
	inflationDistribution InflationDistribution,
	enableInflation bool,
) Params {
	return Params{
		MintDenom:              mintDenom,
		ExponentialCalculation: exponentialCalculation,
		InflationDistribution:  inflationDistribution,
		EnableInflation:        enableInflation,
	}
}

// default minting module parameters
func DefaultParams() Params {
	dailyEmission, _ := math.NewIntFromString(DefaultDailyEmission)
	maxSupply, _ := math.NewIntFromString(DefaultMaxSupply)

	return Params{
		MintDenom:              DefaultInflationDenom,
		ExponentialCalculation: DefaultExponentialCalculation,
		InflationDistribution:  DefaultInflationDistribution,
		EnableInflation:        DefaultInflation,
		DailyEmission:          dailyEmission,
		HalvingIntervalEpochs:  DefaultHalvingIntervalEpochs,
		MultiSigAddress:        DefaultMultiSigAddress,
		MaxSupply:              maxSupply,
	}
}

func validateMintDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if strings.TrimSpace(v) == "" {
		return errors.New("mint denom cannot be blank")
	}

	return sdk.ValidateDenom(v)
}

func validateExponentialCalculation(i interface{}) error {
	v, ok := i.(ExponentialCalculation)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// validate initial value
	if v.A.IsNegative() {
		return fmt.Errorf("initial value cannot be negative")
	}

	// validate reduction factor
	if v.R.GT(math.LegacyNewDec(1)) {
		return fmt.Errorf("reduction factor cannot be greater than 1")
	}

	if v.R.IsNegative() {
		return fmt.Errorf("reduction factor cannot be negative")
	}

	// validate long term inflation
	if v.C.IsNegative() {
		return fmt.Errorf("long term inflation cannot be negative")
	}

	// validate bonded target
	if v.BondingTarget.GT(math.LegacyNewDec(1)) {
		return fmt.Errorf("bonded target cannot be greater than 1")
	}

	if !v.BondingTarget.IsPositive() {
		return fmt.Errorf("bonded target cannot be zero or negative")
	}

	// validate max variance
	if v.MaxVariance.IsNegative() {
		return fmt.Errorf("max variance cannot be negative")
	}

	return nil
}

func validateInflationDistribution(i interface{}) error {
	v, ok := i.(InflationDistribution)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.StakingRewards.IsNegative() {
		return errors.New("staking distribution ratio must not be negative")
	}

	if !v.UsageIncentives.IsZero() {
		return errors.New("incentives pool distribution is deprecated. UsageIncentives param should be zero")
	}

	if v.CommunityPool.IsNegative() {
		return errors.New("community pool distribution ratio must not be negative")
	}

	totalProportions := v.StakingRewards.Add(v.UsageIncentives).Add(v.CommunityPool)
	if !totalProportions.Equal(math.LegacyNewDec(1)) {
		return errors.New("total distributions ratio should be 1")
	}

	return nil
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

// validateMultiSigAddress validates a bech32 address, allowing empty for fallback to EVM params
func validateMultiSigAddress(i interface{}) error {
	addr, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// Allow empty - will use EVM params as primary source or default fallback
	if addr == "" {
		return nil
	}

	// Validate bech32 address format to prevent panic during minting
	_, err := sdk.AccAddressFromBech32(addr)
	if err != nil {
		return fmt.Errorf("invalid bech32 address: %w", err)
	}

	return nil
}

func (p Params) Validate() error {
	if err := validateMintDenom(p.MintDenom); err != nil {
		return err
	}
	if err := validateExponentialCalculation(p.ExponentialCalculation); err != nil {
		return err
	}
	if err := validateInflationDistribution(p.InflationDistribution); err != nil {
		return err
	}
	if err := validateBool(p.EnableInflation); err != nil {
		return err
	}

	// Validate MultiSigAddress to prevent invalid governance proposals
	if err := validateMultiSigAddress(p.MultiSigAddress); err != nil {
		return err
	}

	// This prevents division-by-zero and other halving-related panics
	if err := ValidateHalvingParams(p); err != nil {
		return err
	}

	return nil
}
