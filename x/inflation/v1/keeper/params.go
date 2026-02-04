// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v19/x/inflation/v1/types"
)

// GetParams returns the total set of inflation parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if len(bz) == 0 {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the inflation params in a single key
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {

	// This prevents chain halts from invalid params like halving_interval_epochs = 0
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Validate halving-specific params (division-by-zero protection)
	if err := types.ValidateHalvingParams(params); err != nil {
		return fmt.Errorf("invalid halving params: %w", err)
	}

	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(types.ParamsKey, bz)

	return nil
}
