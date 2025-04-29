package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	evmtypes "github.com/evmos/evmos/v19/x/evm/types"
)

// CallEVM implements the EVMKeeper interface with the adapter pattern
func (k *Keeper) CallEVM(ctx sdk.Context, abiJSON string, method string, contract common.Address, args ...interface{}) (evmtypes.MsgEthereumTxResponse, error) {
	// We don't need to parse the ABI ourselves, the EVM keeper will do it
	return k.evmKeeper.CallEVM(ctx, abiJSON, method, contract, args...)
}
