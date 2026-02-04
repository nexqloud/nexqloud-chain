// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/evmos/evmos/v19/x/inflation/v1/types"
)

// SetEpochMintProvision stores the current epoch mint provision
// This stores the ACTUAL amount minted in the halving system,
// not a calculated value from the old exponential formula
func (k Keeper) SetEpochMintProvision(ctx sdk.Context, provision math.LegacyDec) {
	store := ctx.KVStore(k.storeKey)
	bz, err := provision.Marshal()
	if err != nil {
		panic(err)
	}
	store.Set(types.KeyPrefixEpochMintProvision, bz)
}

// GetStoredEpochMintProvision retrieves the stored epoch mint provision
// Returns the actual amount that was minted in the last epoch
func (k Keeper) GetStoredEpochMintProvision(ctx sdk.Context) math.LegacyDec {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrefixEpochMintProvision)
	if len(bz) == 0 {
		return math.LegacyZeroDec()
	}

	var provision math.LegacyDec
	if err := provision.Unmarshal(bz); err != nil {
		panic(err)
	}
	return provision
}
