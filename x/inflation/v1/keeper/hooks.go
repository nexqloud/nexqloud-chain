// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/armon/go-metrics"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstypes "github.com/evmos/evmos/v19/x/epochs/types"
	"github.com/evmos/evmos/v19/x/inflation/v1/types"
)

// BeforeEpochStart: noop, We don't need to do anything here
func (k Keeper) BeforeEpochStart(_ sdk.Context, _ string, _ int64) {
}

// AfterEpochEnd mints and allocates coins at the end of each epoch end
// 🆕 UPDATED: Now implements Bitcoin-style halving with daily emission
func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	params := k.GetParams(ctx)
	skippedEpochs := k.GetSkippedEpochs(ctx)

	// Skip inflation if it is disabled and increment number of skipped epochs
	if !params.EnableInflation {
		// check if the epochIdentifier is "day" before incrementing.
		if epochIdentifier != epochstypes.DayEpochID {
			return
		}
		skippedEpochs++

		k.SetSkippedEpochs(ctx, skippedEpochs)
		k.Logger(ctx).Debug(
			"skipping inflation mint and allocation",
			"height", ctx.BlockHeight(),
			"epoch-id", epochIdentifier,
			"epoch-number", epochNumber,
			"skipped-epochs", skippedEpochs,
		)
		return
	}

	// 🆕 HALVING: Only mint on daily epochs
	if !types.IsValidEpochForHalving(epochIdentifier) {
		k.Logger(ctx).Debug(
			"skipping non-daily epoch for halving",
			"epoch-id", epochIdentifier,
		)
		return
	}

	// 🆕 HALVING: Get halving data and calculate current period
	// Use epochNumber-1 because AfterEpochEnd is called with the NEW epoch number (the one starting)
	// but we need to calculate based on the epoch that just ended
	halvingData := k.GetHalvingData(ctx)
	currentPeriod := types.CalculateHalvingPeriod(epochNumber-1, int64(halvingData.StartEpoch), params.HalvingIntervalEpochs)

	// 🆕 HALVING: Calculate daily emission with halving applied
	dailyEmission := types.CalculateDailyEmission(params, currentPeriod)

	if !dailyEmission.IsPositive() {
		k.Logger(ctx).Error(
			"SKIPPING HALVING MINT: zero or negative daily emission",
			"period", currentPeriod,
			"emission", dailyEmission.String(),
		)
		return
	}

	// 🆕 HALVING: Validate supply cap before minting
	currentSupply := k.bankKeeper.GetSupply(ctx, params.MintDenom).Amount
	if err := types.ValidateSupplyCap(currentSupply, dailyEmission, params.MaxSupply); err != nil {
		k.Logger(ctx).Error(
			"SUPPLY CAP REACHED: halving minting disabled",
			"current-supply", currentSupply.String(),
			"max-supply", params.MaxSupply.String(),
			"error", err.Error(),
		)
		return
	}

	mintedCoin := sdk.Coin{
		Denom:  params.MintDenom,
		Amount: dailyEmission,
	}

	// 🆕 HALVING: Get MultiSigAddress from EVM params (dynamic, not hardcoded)
	// Fallback to inflation params default if EVM params not set
	var multiSigAddress string
	if k.evmKeeper != nil {
		evmParams := k.evmKeeper.GetParams(ctx)
		multiSigAddress = evmParams.GetMultiSigAddress()
	}
	// Fallback to inflation params default if EVM params is empty (backward compatibility)
	if multiSigAddress == "" {
		multiSigAddress = params.MultiSigAddress
		// If inflation params also empty, use the hardcoded default
		if multiSigAddress == "" {
			multiSigAddress = types.DefaultMultiSigAddress
		}
	}

	// 🆕 HALVING: Mint and send directly to multi-sig (no staking/community pool distribution)
	// Use MultiSigAddress from EVM params instead of inflation params
	if multiSigAddress != "" {
		// Create a temporary params struct with the MultiSigAddress from EVM params
		tempParams := params
		tempParams.MultiSigAddress = multiSigAddress
		if err := k.MintAndSendToMultiSig(ctx, mintedCoin, tempParams); err != nil {
			panic(fmt.Sprintf("failed to mint and send to multi-sig: %v", err))
		}
	} else {
		// Fallback to standard distribution if multi-sig not set
		staking, communityPool, err := k.MintAndAllocateInflation(ctx, mintedCoin, params)
		if err != nil {
			panic(fmt.Sprintf("failed to allocate inflation: %v", err))
		}
		k.Logger(ctx).Info(
			"allocated inflation (multi-sig not set)",
			"staking", staking.String(),
			"community_pool", communityPool.String(),
		)
	}

	// 🆕 HALVING: State Reconciliation
	// Always reconcile stored state with calculated state to prevent "split brain"
	// This ensures consistency even if governance changes halving parameters
	stateChanged := false
	wasNaturalHalving := false

	// Check if this is a natural halving event (time-based, not parameter-induced)
	if types.ShouldHalve(epochNumber-1, int64(halvingData.StartEpoch), halvingData.LastHalvingEpoch, params.HalvingIntervalEpochs) {
		wasNaturalHalving = true
		stateChanged = true
		halvingData.LastHalvingEpoch = uint64(epochNumber - 1) // Store the epoch that just ended
	}

	// Always reconcile if calculated period differs from stored period
	// This handles parameter changes that affect period calculation
	if currentPeriod != halvingData.CurrentPeriod {
		stateChanged = true
	}

	// Update state if anything changed
	if stateChanged {
		oldPeriod := halvingData.CurrentPeriod
		halvingData.CurrentPeriod = currentPeriod
		k.SetHalvingData(ctx, halvingData)

		// Emit appropriate event based on the type of change
		if wasNaturalHalving {
			k.Logger(ctx).Info(
				"HALVING EVENT: entered new period",
				"previous-period", oldPeriod,
				"new-period", currentPeriod,
				"epoch-ended", epochNumber-1,
				"new-daily-emission", dailyEmission.String(),
			)
		} else {
			// Period changed due to parameter update, not natural halving
			k.Logger(ctx).Info(
				"PERIOD RECONCILIATION: state updated due to parameter change",
				"old-stored-period", oldPeriod,
				"new-calculated-period", currentPeriod,
				"epoch", epochNumber-1,
				"new-daily-emission", dailyEmission.String(),
			)
		}
	}

	// 🔄 SYNC LEGACY STATE: Update old exponential inflation state to match halving state
	// This ensures queries like GetPeriod(), GetEpochMintProvision(), GetInflationRate()
	// return correct values even though they're designed for the old system
	// NOTE: We always update these, not just when stateChanged, because emission is always happening
	k.SetPeriod(ctx, currentPeriod)                                       // Sync period with halving period
	k.SetEpochMintProvision(ctx, math.LegacyNewDecFromInt(dailyEmission)) // Store actual emission amount

	// 🆕 HALVING: Telemetry for halving system
	defer func() {
		if mintedCoin.Amount.IsInt64() && mintedCoin.Amount.IsPositive() {
			telemetry.IncrCounterWithLabels(
				[]string{types.ModuleName, "halving", "mint", "total"},
				float32(mintedCoin.Amount.Int64()),
				[]metrics.Label{
					telemetry.NewLabel("denom", mintedCoin.Denom),
					telemetry.NewLabel("period", fmt.Sprintf("%d", currentPeriod)),
				},
			)
		}
	}()

	// 🆕 HALVING: Emit halving-specific events
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMint,
			sdk.NewAttribute(types.AttributeEpochNumber, fmt.Sprintf("%d", epochNumber)),
			sdk.NewAttribute(types.AttributeKeyEpochProvisions, dailyEmission.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, mintedCoin.Amount.String()),
			sdk.NewAttribute("halving_period", fmt.Sprintf("%d", currentPeriod)),
		),
	)

	// Add multi-sig address to event if available
	if k.evmKeeper != nil {
		evmParams := k.evmKeeper.GetParams(ctx)
		if multiSigAddr := evmParams.GetMultiSigAddress(); multiSigAddr != "" {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeMint,
					sdk.NewAttribute("multi_sig_address", multiSigAddr),
				),
			)
		}
	}
}

// ___________________________________________________________________________________________________

// Hooks wrapper struct for incentives keeper
type Hooks struct {
	k Keeper
}

var _ epochstypes.EpochHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// epochs hooks
func (h Hooks) BeforeEpochStart(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	h.k.BeforeEpochStart(ctx, epochIdentifier, epochNumber)
}

func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	h.k.AfterEpochEnd(ctx, epochIdentifier, epochNumber)
}
