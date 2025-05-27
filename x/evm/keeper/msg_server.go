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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	tmtypes "github.com/cometbft/cometbft/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	config "github.com/evmos/evmos/v19/x/config"
	"github.com/evmos/evmos/v19/x/evm/types"
)

// }
var _ types.MsgServer = &Keeper{}
var whitelist = map[string]bool{
	//staging config
	// "0x0845ed4B7CE9c886BC801edaF4f31F5123ffE69A": true,

	//dev config
	"0x9437919dEfb9E20DA4352C9abcA22adD3E473821": true,
}

func getFunctionSelector(signature string) []byte {
	// log.Println("Enter getFunctionSelector()")
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return hash.Sum(nil)[:4] // First 4 bytes of keccak256 hash
}

// IsChainOpen checks if the chain is open for new transactions based on the
// online server count from the contract. If the count is greater than or equal
// to 1000, the chain is considered open. Otherwise, it is closed.
// the function calls the getOnlineServerCount() function on the contract and parses the
// response to get the count.
// The function returns true if the chain is open and false if it is closed.
func (k *Keeper) IsChainOpen(ctx sdk.Context, from common.Address) (bool, error) {

	addr := common.HexToAddress(config.OnlineServerCountContract)
	data := hexutil.Bytes(getFunctionSelector("getOnlineServerCount()"))

	 // Get the current block height
	 currentHeight := ctx.BlockHeight()
	 // Calculate previous block height
	 previousBlock := currentHeight - 1
	// Prepare the EthCallRequest
	args := types.TransactionArgs{
		From: &from, // Use the dynamic sender address passed from EthereumTx
		To:   &addr, // Replace with the contract address
		Data: &data, // Replace with the actual function call data
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		log.Println("Failed to marshal args:", err)
		return false, err
	}

	req := &types.EthCallRequest{
		Args:    argsBytes,
		GasCap:  uint64(25000000), // Set a fixed gas cap
		ChainId: config.ChainID,  // Replace with the chain ID
		BlockNumber: &previousBlock, // Use the previous block height
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

// IsWalletUnlocked checks if a wallet is unlocked and whether a transaction
// can proceed based on the wallet's lock status. It retrieves the lock status,
// lock value, and locked amount from the WalletState contract and evaluates
// the wallet's status. The function considers different lock types: no lock,
// amount lock, and absolute lock (full lock). It also checks
// if the transaction amount is within allowable limits for
// amount locks. Returns true if the transaction can proceed, otherwise false.

func (k *Keeper) IsWalletUnlocked(ctx sdk.Context, from common.Address, txAmount *big.Int) (bool, error) {
	log.Println("Enter IsWalletUnlocked() - Checking wallet lock status")

	// Define the WalletState contract address
	walletStateContract := common.HexToAddress(config.WalletStateContractAddress)

	// Prepare the function selector for getWalletLock(address)
	functionSelector := getFunctionSelector("getWalletLock(address)")
	paddedAddress := common.LeftPadBytes(from.Bytes(), 32) // 32-byte encoding for address
	data := append(functionSelector, paddedAddress...)

	// Convert data to hexutil.Bytes explicitly
	hexData := hexutil.Bytes(data)

	// Prepare EthCall request
	args := types.TransactionArgs{
		From: &from,
		To:   &walletStateContract,
		Data: &hexData, // Corrected type conversion
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		log.Println("Failed to marshal args:", err)
		return false, err
	}

	req := &types.EthCallRequest{
		Args:    argsBytes,
		GasCap:  uint64(25000000), // Set a fixed gas cap
		ChainId: config.ChainID,
	}

	// Call EthCall function
	res, err := k.EthCall(ctx, req)
	if err != nil {
		log.Println("Failed to call EthCall:", err)
		return false, err
	}

	// Parse the response: (LockStatus, lockValue, lockCode)
	if len(res.Ret) < 96 {
		log.Println("Invalid response length", res.Ret)
		return false, fmt.Errorf("invalid response length")
	}
	// log.Println("Raw EthCall Response:", hexutil.Encode(res.Ret))
	lockStatus := new(big.Int).SetBytes(res.Ret[:32]).Uint64() % 256
	lockedAmount := new(big.Int).SetBytes(res.Ret[32:64]) // Correct position
	
	switch lockStatus {
	case 0: // No_Lock
		log.Println("✅ Wallet is unlocked")
		return true, nil

	case 1: // Amount_Lock
		// Ensure locked amount is not greater than total balance
		// Fetch the balance using Keeper
	balanceRes, err := k.Balance(ctx, &types.QueryBalanceRequest{
		Address: from.Hex(),
	})
	if err != nil {
		log.Println("Failed to fetch wallet balance:", err)
		return false, err
	}
	// log.Println("=============== Wallet Balance:", balanceRes.Balance)
	totalBalance, ok := new(big.Int).SetString(balanceRes.Balance, 10)
	if !ok {
		return false, fmt.Errorf("failed to convert balance to *big.Int")
	}
		if totalBalance.Cmp(lockedAmount) < 0 {
			log.Println("❌ Locked amount exceeds wallet balance")
			return false, fmt.Errorf("locked amount exceeds wallet balance")
		}

		// Compute max allowed transfer
		maxAllowed := new(big.Int).Sub(totalBalance, lockedAmount)

		log.Printf("✅ Max Allowed Transfer: %s", maxAllowed.String())

		// Create tolerance of 0.0001 NXQ (10^14 wei)
		tolerance := new(big.Int).Exp(big.NewInt(10), big.NewInt(14), nil)

		// Calculate difference between tx amount and max allowed
		diff := new(big.Int)
		if txAmount.Cmp(maxAllowed) > 0 {
			diff.Sub(txAmount, maxAllowed)

			// If difference is within tolerance, allow the transaction
			if diff.Cmp(tolerance) <= 0 {
				log.Printf("✅ Transaction within tolerance (diff: %s wei)", diff.String())
				return true, nil
			}

			log.Printf("❌ Tx %s > Allowed %s (diff: %s)", txAmount.String(), maxAllowed.String(), diff.String())
			return false, fmt.Errorf("exceeds limit")
		}

		log.Println("✅ Transaction allowed under amount lock")
		return true, nil

	case 2: // Absolute_Lock
		log.Println("❌ Wallet is fully locked")
		return false, fmt.Errorf("wallet is fully locked")

	default:
		log.Println("❌ Unknown lock status")
		return false, fmt.Errorf("unknown lock status")
	}
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

	jsonData, err := tx.MarshalJSON()
	if err != nil {
		log.Println("Failed to marshal tx to json:", err)
	}
	log.Println("Tx Data:", string(jsonData))
	log.Println("Tx Index:", string(tx.Data()))
	log.Println("Receiver:", tx.To())

	from := common.HexToAddress(msg.From)
	log.Println("From:", from)

	// Check whitelist first
	if !whitelist[from.Hex()] {
		// Only check chain status and wallet lock for non-whitelisted addresses
		isOpen, err := k.IsChainOpen(ctx, from)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to check if chain is open")
		}
		if !isOpen {
			return nil, errorsmod.Wrap(errors.New("deprecated"), "chain is closed")
		}

		txAmount := tx.Value()
		isUnlocked, err := k.IsWalletUnlocked(ctx, from, txAmount)
		if err != nil || !isUnlocked {
			return nil, fmt.Errorf("transaction rejected: wallet is locked")
		}
	} else {
		log.Println("Address is whitelisted, skipping chain open and wallet unlock checks")
	}

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
