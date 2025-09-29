// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v19/x/inflation/v1/types"
)

// GetHalvingData gets the current halving data from the store
func (k Keeper) GetHalvingData(ctx sdk.Context) types.HalvingData {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixHalvingData)
	if len(bz) == 0 {
		// Return default halving data if not found (first time initialization)
		return types.HalvingData{
			CurrentPeriod:    0,
			LastHalvingEpoch: 0,
			StartEpoch:       1, // Start from epoch 1
		}
	}

	var halvingData types.HalvingData
	k.cdc.MustUnmarshal(bz, &halvingData)
	return halvingData
}

// SetHalvingData stores the halving data in the store
func (k Keeper) SetHalvingData(ctx sdk.Context, halvingData types.HalvingData) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&halvingData)
	store.Set(types.KeyPrefixHalvingData, bz)
}

// InitializeHalvingData initializes halving data for the first time
// This should be called during genesis initialization
func (k Keeper) InitializeHalvingData(ctx sdk.Context, startEpoch uint64) {
	halvingData := types.HalvingData{
		CurrentPeriod:    0,
		LastHalvingEpoch: 0,
		StartEpoch:       startEpoch,
	}
	k.SetHalvingData(ctx, halvingData)
}
