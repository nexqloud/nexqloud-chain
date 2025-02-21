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

// IsChainOpen checks if the chain is open for new transactions based on the
// online server count from the contract. If the count is greater than or equal
// to 1000, the chain is considered open. Otherwise, it is closed.
//
// The function takes the sender address as an argument and checks if it is
// whitelisted. If it is, the function returns true immediately. Otherwise, it
// calls the getOnlineServerCount() function on the contract and parses the
// response to get the count.
//
// The function returns true if the chain is open and false if it is closed.
func (k *Keeper) IsChainOpen(ctx sdk.Context, from common.Address) (bool, error) {
	log.Println("Enter IsChainOpen() and checking for whitelisted addresses")

	if whitelist[from.Hex()] {
		log.Println("Sender is whitelisted, allowing transaction")
		return true, nil
	}
	addr := common.HexToAddress(OnlineServerCountContract)
	data := hexutil.Bytes(getFunctionSelector("getOnlineServerCount()"))
	log.Println("data after mod:", data)

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
		GasCap:  uint64(1000000), // Adjust gas cap as needed
		ChainId: ChainID,         // Replace with the chain ID
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
// percentage lock, amount lock, and absolute lock (full lock). It also checks
// if the transaction amount is within allowable limits for percentage and
// amount locks. Returns true if the transaction can proceed, otherwise false.

func (k *Keeper) IsWalletUnlocked(ctx sdk.Context, from common.Address, txAmount *big.Int) (bool, error) {
	log.Println("Enter IsWalletUnlocked() - Checking wallet lock status")

	// Define the WalletState contract address
	walletStateContract := common.HexToAddress(WalletStateContract)

	// Prepare the function selector for getWalletLock(address)
	functionSelector := getFunctionSelector("getWalletLock(address)")
	paddedAddress := common.LeftPadBytes(from.Bytes(), 32) // 32-byte encoding for address
	data := append(functionSelector, paddedAddress...)

	log.Println("Calling WalletState with data:", hexutil.Encode(data))

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
		GasCap:  uint64(1000000),
		ChainId: ChainID,
	}

	// Call EthCall function
	res, err := k.EthCall(ctx, req)
	if err != nil {
		log.Println("Failed to call EthCall:", err)
		return false, err
	}

	// Parse the response: (LockStatus, lockValue, lockCode)
	if len(res.Ret) < 96 {
		log.Println("Invalid response length")
		return false, fmt.Errorf("invalid response length")
	}
	log.Println("Raw EthCall Response:", hexutil.Encode(res.Ret))

	// Extract lock status, lock value, and lock code
	lockStatus := new(big.Int).SetBytes(res.Ret[:32]).Uint64() % 256 // Extract only the least significant byte
	lockValue := new(big.Int).SetBytes(res.Ret[32:64])               // Extracting lock value
	lockedAmount := new(big.Int).SetBytes(res.Ret[64:96])            // Extracting lock code

	log.Println("Lock Status Retrieved:", lockStatus)
	log.Println("Lock Value Retrieved:", lockValue.Int64())
	log.Println("Lock Code Retrieved:", lockedAmount.Int64())

	// Fetch the balance using Keeper
	balanceRes, err := k.Balance(ctx, &types.QueryBalanceRequest{
		Address: from.Hex(),
	})
	if err != nil {
		log.Println("Failed to fetch wallet balance:", err)
		return false, err
	}
	log.Println("=============== Wallet Balance:", balanceRes.Balance)
	totalBalance, ok := new(big.Int).SetString(balanceRes.Balance, 10)
	if !ok {
		log.Println("Failed to convert balance to *big.Int")
		return false, fmt.Errorf("failed to convert balance to *big.Int")
	}
	// Calculate locked amount and allowed amount

	// Check lock status and enforce restrictions
	switch lockStatus {
	case 0: // No_Lock
		log.Println("✅ Wallet is unlocked")
		return true, nil

	case 1: // Percentage_Lock
		if totalBalance.Cmp(big.NewInt(0)) == 0 {
			log.Println("❌ Wallet balance is zero, cannot process percentage lock")
			return false, fmt.Errorf("wallet balance is zero")
		}
		// lockedAmount := new(big.Int).Div(new(big.Int).Mul(totalBalance, lockValue), big.NewInt(100))
		maxAllowed := new(big.Int).Sub(totalBalance, lockedAmount) // Amount user can transfer
		log.Printf("✅ Max Allowed Transfer: %s", maxAllowed.String())

		// Check if the transaction amount exceeds the allowed limit
		if txAmount.Cmp(maxAllowed) > 0 {
			log.Println("❌ Transaction exceeds allowed percentage limit")
			return false, fmt.Errorf("transaction exceeds allowed percentage limit")
		}

		log.Println("✅ Transaction allowed under percentage lock")
		return true, nil

	case 2: // Amount_Lock
		// Ensure locked amount is not greater than total balance
		if totalBalance.Cmp(lockValue) < 0 {
			log.Println("❌ Locked amount exceeds wallet balance, blocking transaction")
			return false, fmt.Errorf("locked amount exceeds wallet balance")
		}

		// Convert lockValue to wei (multiply by 10^18)
		lockedAmount = new(big.Int).Mul(lockedAmount, big.NewInt(1e18))
		log.Printf("🔒 Locked Amount (Fixed, in wei): %s", lockedAmount.String())

		// Ensure locked amount is not greater than total balance
		if totalBalance.Cmp(lockedAmount) < 0 {
			log.Println("❌ Locked amount exceeds wallet balance, blocking transaction")
			return false, fmt.Errorf("locked amount exceeds wallet balance")
		}

		// Correctly subtract locked amount
		maxAllowed := new(big.Int).Sub(totalBalance, lockedAmount) // Amount user can transfer
		log.Printf("✅ Max Allowed Transfer: %s", maxAllowed.String())

		// Ensure transaction amount does not exceed max allowed transfer
		if txAmount.Cmp(maxAllowed) > 0 {
			log.Println("❌ Transaction exceeds allowed fixed amount limit")
			return false, fmt.Errorf("transaction exceeds allowed fixed amount limit")
		}
		log.Println("✅ Transaction allowed under amount lock")
		return true, nil

	case 3: // Absolute_Lock (Full lock)
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
	// tx = msg.AsTransaction()
	// txAmount := tx.Value() // Pass this value
	// isUnlocked, err := k.IsWalletUnlocked(ctx, from, txAmount)
	// if err != nil || !isUnlocked {
	// 	return nil, fmt.Errorf("transaction rejected: wallet is locked")
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
