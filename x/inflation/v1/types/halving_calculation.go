// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package types

import (
	"fmt"

	"cosmossdk.io/math"
	epochstypes "github.com/evmos/evmos/v19/x/epochs/types"
)

// CalculateHalvingPeriod determines the current halving period from epoch number
// Each halving period lasts for halvingIntervalEpochs (default: 1461 epochs = 4 years)
// Returns:
//   - Period 0: Epochs 1-1461 (full emission)
//   - Period 1: Epochs 1462-2922 (50% emission)
//   - Period 2: Epochs 2923-4383 (25% emission)
//   - etc.
func CalculateHalvingPeriod(currentEpoch, startEpoch int64, halvingInterval uint64) uint64 {
	if currentEpoch < startEpoch {
		return 0
	}

	epochsSinceStart := uint64(currentEpoch - startEpoch + 1)
	return (epochsSinceStart - 1) / halvingInterval
}

// CalculateDailyEmission returns the daily emission amount considering the current halving period
// The emission starts at dailyEmission and halves each period:
// - Period 0: 7200 tokens/day
// - Period 1: 3600 tokens/day
// - Period 2: 1800 tokens/day
// - Period 3: 900 tokens/day
// - etc.
func CalculateDailyEmission(params Params, currentHalvingPeriod uint64) math.Int {
	dailyEmission := params.DailyEmission

	// Apply halving: divide by 2^period
	if currentHalvingPeriod > 0 {
		// Calculate 2^period efficiently
		divisor := math.NewInt(1 << currentHalvingPeriod)
		dailyEmission = dailyEmission.Quo(divisor)
	}

	return dailyEmission
}

// ValidateSupplyCap checks if minting the specified amount would exceed the maximum supply
// Returns an error if the mint would cause the total supply to exceed maxSupply
func ValidateSupplyCap(currentSupply, mintAmount, maxSupply math.Int) error {
	newSupply := currentSupply.Add(mintAmount)

	if newSupply.GT(maxSupply) {
		return fmt.Errorf(
			"minting %s would exceed max supply: current=%s, max=%s, would_be=%s",
			mintAmount.String(),
			currentSupply.String(),
			maxSupply.String(),
			newSupply.String(),
		)
	}

	return nil
}

// ShouldHalve checks if a halving event should occur at the current epoch
// Returns true if we've crossed into a new halving period
func ShouldHalve(currentEpoch, startEpoch int64, lastHalvingEpoch uint64, halvingInterval uint64) bool {
	currentPeriod := CalculateHalvingPeriod(currentEpoch, startEpoch, halvingInterval)
	lastPeriod := CalculateHalvingPeriod(int64(lastHalvingEpoch), startEpoch, halvingInterval)

	return currentPeriod > lastPeriod
}

// GetNextHalvingEpoch calculates when the next halving will occur
func GetNextHalvingEpoch(currentEpoch, startEpoch int64, halvingInterval uint64) int64 {
	currentPeriod := CalculateHalvingPeriod(currentEpoch, startEpoch, halvingInterval)
	nextHalvingEpoch := startEpoch + int64((currentPeriod+1)*halvingInterval)
	return nextHalvingEpoch
}

// EstimateRemainingSupply calculates approximately how many tokens can still be minted
// before reaching the supply cap, considering the halving schedule
func EstimateRemainingSupply(currentSupply, maxSupply math.Int) math.Int {
	if currentSupply.GTE(maxSupply) {
		return math.ZeroInt()
	}
	return maxSupply.Sub(currentSupply)
}

// ValidateHalvingParams performs basic validation on halving parameters
func ValidateHalvingParams(params Params) error {
	if params.DailyEmission.IsZero() || params.DailyEmission.IsNegative() {
		return fmt.Errorf("daily emission must be positive, got: %s", params.DailyEmission.String())
	}

	if params.HalvingIntervalEpochs == 0 {
		return fmt.Errorf("halving interval epochs must be positive, got: %d", params.HalvingIntervalEpochs)
	}

	if params.MaxSupply.IsZero() || params.MaxSupply.IsNegative() {
		return fmt.Errorf("max supply must be positive, got: %s", params.MaxSupply.String())
	}

	// Validate that daily emission doesn't immediately exceed max supply
	if params.DailyEmission.GT(params.MaxSupply) {
		return fmt.Errorf("daily emission (%s) cannot exceed max supply (%s)",
			params.DailyEmission.String(), params.MaxSupply.String())
	}

	// 🆕 CERTIK ISSUE #7: Validate MultiSigAddress to prevent chain panics
	// This validation is critical - an invalid address will cause AfterEpochEnd() to panic
	if err := validateMultiSigAddress(params.MultiSigAddress); err != nil {
		return fmt.Errorf("invalid multi-sig address in halving params: %w", err)
	}

	return nil
}

// CalculateHalvingSchedule returns information about the halving schedule
type HalvingScheduleInfo struct {
	CurrentPeriod      uint64
	CurrentEmission    math.Int
	NextHalvingEpoch   int64
	EpochsUntilHalving int64
}

// GetHalvingScheduleInfo returns comprehensive information about the current halving state
func GetHalvingScheduleInfo(currentEpoch, startEpoch int64, params Params) HalvingScheduleInfo {
	currentPeriod := CalculateHalvingPeriod(currentEpoch, startEpoch, params.HalvingIntervalEpochs)
	currentEmission := CalculateDailyEmission(params, currentPeriod)
	nextHalvingEpoch := GetNextHalvingEpoch(currentEpoch, startEpoch, params.HalvingIntervalEpochs)
	epochsUntilHalving := nextHalvingEpoch - currentEpoch

	return HalvingScheduleInfo{
		CurrentPeriod:      currentPeriod,
		CurrentEmission:    currentEmission,
		NextHalvingEpoch:   nextHalvingEpoch,
		EpochsUntilHalving: epochsUntilHalving,
	}
}

// IsValidEpochForHalving checks if the current epoch identifier is the daily epoch
// We only want to mint on daily epochs, not weekly or other epoch types
func IsValidEpochForHalving(epochIdentifier string) bool {
	return epochIdentifier == epochstypes.DayEpochID
}
