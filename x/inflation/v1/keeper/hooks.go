// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"fmt"

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
	halvingData := k.GetHalvingData(ctx)
	currentPeriod := types.CalculateHalvingPeriod(epochNumber, int64(halvingData.StartEpoch), params.HalvingIntervalEpochs)

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

	// 🆕 HALVING: Mint and send directly to multi-sig (no staking/community pool distribution)
	if err := k.MintAndSendToMultiSig(ctx, mintedCoin, params); err != nil {
		panic(fmt.Sprintf("failed to mint and send to multi-sig: %v", err))
	}

	// 🆕 HALVING: Update halving data if we crossed into a new period
	if types.ShouldHalve(epochNumber, int64(halvingData.StartEpoch), halvingData.LastHalvingEpoch, params.HalvingIntervalEpochs) {
		halvingData.CurrentPeriod = currentPeriod
		halvingData.LastHalvingEpoch = uint64(epochNumber)
		k.SetHalvingData(ctx, halvingData)

		k.Logger(ctx).Info(
			"HALVING EVENT: entered new period",
			"previous-period", currentPeriod-1,
			"new-period", currentPeriod,
			"epoch", epochNumber,
			"new-daily-emission", dailyEmission.String(),
		)
	}

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
			sdk.NewAttribute("multi_sig_address", params.MultiSigAddress),
		),
	)
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
