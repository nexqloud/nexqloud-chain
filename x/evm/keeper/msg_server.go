// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"golang.org/x/crypto/sha3"
	// "github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	// "github.com/ethereum/go-ethereum/crypto"
	// "github.com/ethereum/go-ethereum/ethclient"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/evmos/evmos/v19/x/evm/types"
)

// }
var _ types.MsgServer = &Keeper{}
var whitelist = map[string]bool{
	"0x7cB61D4117AE31a12E393a1Cfa3BaC666481D02E": true,
}

func getFunctionSelector(signature string) []byte {
	log.Println("Enter getFunctionSelector()")
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return hash.Sum(nil)[:4] // First 4 bytes of keccak256 hash
}
func (k *Keeper) IsChainOpen(ctx sdk.Context, from common.Address) (bool, error) {
    log.Println("Enter IsChainOpen() and checking for whitelisted addresses")

	if whitelist[from.Hex()] {
        log.Println("Sender is whitelisted, allowing transaction")
        return true, nil
    }
    addr := common.HexToAddress(ContractAddress)
	data := hexutil.Bytes(getFunctionSelector("getOnlineServerCount()"))
	log.Println("data after mod:", data)

    // Prepare the EthCallRequest
    args := types.TransactionArgs{
        From:     &from, // Use the dynamic sender address passed from EthereumTx
        To:       &addr, // Replace with the contract address
        Data:     &data, // Replace with the actual function call data
    }

    argsBytes, err := json.Marshal(args)
    if err != nil {
        log.Println("Failed to marshal args:", err)
        return false, err
    }

    req := &types.EthCallRequest{
        Args:     argsBytes,
        GasCap:   uint64(1000000), // Adjust gas cap as needed
        ChainId:  ChainID,   // Replace with the chain ID
    }

    // Call the EthCall function
    res, err := k.EthCall(ctx, req)
    if err != nil {
        log.Println("Failed to call EthCall:", err)
        return false, err
    }

    // Parse the response to get the online server count
    count := new(big.Int)
    count.SetBytes(res.Ret)

    log.Println("Current Online Server Count:", count)

    // Check if the chain is open based on the count
    if count.Cmp(big.NewInt(1000)) >= 0 {
        log.Println("Chain is open")
        return true, nil
    }

    log.Println("Chain is closed")
    return false, nil
}
// EthereumTx implements the gRPC MsgServer interface. It receives a transaction which is then
// executed (i.e applied) against the go-ethereum EVM. The provided SDK Context is set to the Keeper
// so that it can implements and call the StateDB methods without receiving it as a function
// parameter.
func (k *Keeper) EthereumTx(goCtx context.Context, msg *types.MsgEthereumTx) (*types.MsgEthereumTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := msg.From
	tx := msg.AsTransaction()
	txIndex := k.GetTxIndexTransient(ctx)

	log.Println("=============CUSTOM CODE===============")
	log.Println("Sender:", sender)
	log.Println("Tx Amount:", tx.Value())
	jsonData, err := tx.MarshalJSON()
	if err != nil {
		log.Println("Failed to marshal tx to json:", err)
	}
	log.Println("Tx Data:", string(jsonData))
	log.Println("Tx Index:", string(tx.Data()))
	log.Println("Receiver:", tx.To())

	from := common.HexToAddress(msg.From)
	log.Println("From:", from)
	log.Println("GOING TO CHECK FOR IS CHAIN OPEN OR NOT")
	isOpen, err := k.IsChainOpen(ctx, from)
    if err != nil {
        return nil, errorsmod.Wrap(err, "failed to check if chain is open")
	}
	if !isOpen {
		return nil, errorsmod.Wrap(errors.New("deprecated"), "chain is closed")
	}

	// if msg.From != "" { // TODO: Check if the sender is among the allowed senders
	// 	return nil, errorsmod.Wrap(errors.New("deprecated"), "chain is closed")
	// }

	labels := []metrics.Label{
		telemetry.NewLabel("tx_type", fmt.Sprintf("%d", tx.Type())),
	}
	if tx.To() == nil {
		labels = append(labels, telemetry.NewLabel("execution", "create"))
	} else {
		labels = append(labels, telemetry.NewLabel("execution", "call"))
	}

	response, err := k.ApplyTransaction(ctx, tx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to apply transaction")
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "ethereum_tx", "total"},
			1,
			labels,
		)

		if response.GasUsed != 0 {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "ethereum_tx", "gas_used", "total"},
				float32(response.GasUsed),
				labels,
			)

			// Observe which users define a gas limit >> gas used. Note, that
			// gas_limit and gas_used are always > 0
			gasLimit := math.LegacyNewDec(int64(tx.Gas()))
			gasRatio, err := gasLimit.QuoInt64(int64(response.GasUsed)).Float64()
			if err == nil {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "ethereum_tx", "gas_limit", "per", "gas_used"},
					float32(gasRatio),
					labels,
				)
			}
		}
	}()

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyAmount, tx.Value().String()),
		// add event for ethereum transaction hash format
		sdk.NewAttribute(types.AttributeKeyEthereumTxHash, response.Hash),
		// add event for index of valid ethereum tx
		sdk.NewAttribute(types.AttributeKeyTxIndex, strconv.FormatUint(txIndex, 10)),
		// add event for eth tx gas used, we can't get it from cosmos tx result when it contains multiple eth tx msgs.
		sdk.NewAttribute(types.AttributeKeyTxGasUsed, strconv.FormatUint(response.GasUsed, 10)),
	}

	if len(ctx.TxBytes()) > 0 {
		// add event for tendermint transaction hash format
		hash := tmbytes.HexBytes(tmtypes.Tx(ctx.TxBytes()).Hash())
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyTxHash, hash.String()))
	}

	if to := tx.To(); to != nil {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyRecipient, to.Hex()))
	}

	if response.Failed() {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyEthereumTxFailed, response.VmError))
	}

	txLogAttrs := make([]sdk.Attribute, len(response.Logs))
	for i, log := range response.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to encode log")
		}
		txLogAttrs[i] = sdk.NewAttribute(types.AttributeKeyTxLog, string(value))
	}

	// emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeEthereumTx,
			attrs...,
		),
		sdk.NewEvent(
			types.EventTypeTxLog,
			txLogAttrs...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, sender),
			sdk.NewAttribute(types.AttributeKeyTxType, fmt.Sprintf("%d", tx.Type())),
		),
	})

	return response, nil
}

// UpdateParams implements the gRPC MsgServer interface. When an UpdateParams
// proposal passes, it updates the module parameters. The update can only be
// performed if the requested authority is the Cosmos SDK governance module
// account.
func (k *Keeper) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, expected %s, got %s", k.authority.String(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}
