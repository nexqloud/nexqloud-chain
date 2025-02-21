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

// Helper function to make EthCall
func (k *Keeper) makeEthCall(ctx sdk.Context, from common.Address, to common.Address, data hexutil.Bytes) (*types.EthCallResponse, error) {
	args := types.TransactionArgs{
		From: &from,
		To:   &to,
		Data: &data,
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		log.Println("Failed to marshal args:", err)
		return nil, err
	}

	req := &types.EthCallRequest{
		Args:    argsBytes,
		GasCap:  uint64(1000000),
		ChainId: ChainID,
	}

	return k.EthCall(ctx, req)
}

// IsChainOpen checks if the chain is open for new transactions
func (k *Keeper) IsChainOpen(ctx sdk.Context, from common.Address) (bool, error) {
	log.Println("Enter IsChainOpen() and checking for whitelisted addresses")

	if whitelist[from.Hex()] {
		log.Println("Sender is whitelisted, allowing transaction")
		return true, nil
	}

	return k.checkOnlineServerCount(ctx, from)
}

// Helper function to check online server count
func (k *Keeper) checkOnlineServerCount(ctx sdk.Context, from common.Address) (bool, error) {
	contractAddr := common.HexToAddress(OnlineServerCountContract)
	data := hexutil.Bytes(getFunctionSelector("getOnlineServerCount()"))

	res, err := k.makeEthCall(ctx, from, contractAddr, data)
	if err != nil {
		log.Println("Failed to call EthCall:", err)
		return false, err
	}

	count := new(big.Int)
	count.SetBytes(res.Ret)
	log.Println("Current Online Server Count:", count)

	isOpen := count.Cmp(big.NewInt(1000)) >= 0
	if isOpen {
		log.Println("Chain is open")
	} else {
		log.Println("Chain is closed")
	}
	return isOpen, nil
}

// IsWalletUnlocked checks if a wallet is unlocked for transactions
func (k *Keeper) IsWalletUnlocked(ctx sdk.Context, from common.Address, txAmount *big.Int) (bool, error) {
	log.Println("Enter IsWalletUnlocked() - Checking wallet lock status")

	lockStatus, lockValue, lockedAmount, err := k.getWalletLockInfo(ctx, from)
	if err != nil {
		return false, err
	}

	totalBalance, err := k.getWalletBalance(ctx, from)
	if err != nil {
		return false, err
	}

	return k.checkLockStatus(lockStatus, lockValue, lockedAmount, totalBalance, txAmount)
}

// Helper function to get wallet lock information
func (k *Keeper) getWalletLockInfo(ctx sdk.Context, from common.Address) (uint64, *big.Int, *big.Int, error) {
	walletStateContract := common.HexToAddress(WalletStateContract)
	functionSelector := getFunctionSelector("getWalletLock(address)")
	paddedAddress := common.LeftPadBytes(from.Bytes(), 32)
	data := append(functionSelector, paddedAddress...)

	res, err := k.makeEthCall(ctx, from, walletStateContract, hexutil.Bytes(data))
	if err != nil {
		return 0, nil, nil, err
	}

	if len(res.Ret) < 96 {
		return 0, nil, nil, fmt.Errorf("invalid response length")
	}

	lockStatus := new(big.Int).SetBytes(res.Ret[:32]).Uint64() % 256
	lockValue := new(big.Int).SetBytes(res.Ret[32:64])
	lockedAmount := new(big.Int).SetBytes(res.Ret[64:96])

	return lockStatus, lockValue, lockedAmount, nil
}

// Helper function to get wallet balance
func (k *Keeper) getWalletBalance(ctx sdk.Context, from common.Address) (*big.Int, error) {
	balanceRes, err := k.Balance(ctx, &types.QueryBalanceRequest{
		Address: from.Hex(),
	})
	if err != nil {
		return nil, err
	}

	totalBalance, ok := new(big.Int).SetString(balanceRes.Balance, 10)
	if !ok {
		return nil, fmt.Errorf("failed to convert balance to *big.Int")
	}
	return totalBalance, nil
}

// Helper function to check lock status and apply rules
func (k *Keeper) checkLockStatus(lockStatus uint64, lockValue, lockedAmount, totalBalance, txAmount *big.Int) (bool, error) {
	switch lockStatus {
	case 0: // No_Lock
		log.Println("✅ Wallet is unlocked")
		return true, nil

	case 1: // Percentage_Lock
		return k.handlePercentageLock(totalBalance, lockValue, lockedAmount, txAmount)

	case 2: // Amount_Lock
		return k.handleAmountLock(totalBalance, lockValue, lockedAmount, txAmount)

	case 3: // Absolute_Lock
		log.Println("❌ Wallet is fully locked")
		return false, fmt.Errorf("wallet is fully locked")

	default:
		log.Println("❌ Unknown lock status")
		return false, fmt.Errorf("unknown lock status")
	}
}

// Helper function to handle percentage lock
func (k *Keeper) handlePercentageLock(totalBalance, lockValue, lockedAmount, txAmount *big.Int) (bool, error) {
	if totalBalance.Cmp(big.NewInt(0)) == 0 {
		return false, fmt.Errorf("wallet balance is zero")
	}

	lockedAmount = new(big.Int).Mul(totalBalance, lockValue)
	lockedAmount = new(big.Int).Div(lockedAmount, big.NewInt(100))
	maxAllowed := new(big.Int).Sub(totalBalance, lockedAmount)

	if txAmount.Cmp(maxAllowed) > 0 {
		return false, fmt.Errorf("transaction exceeds allowed percentage limit")
	}
	return true, nil
}

// Helper function to handle amount lock
func (k *Keeper) handleAmountLock(totalBalance, lockValue, lockedAmount, txAmount *big.Int) (bool, error) {
	if totalBalance.Cmp(lockValue) < 0 {
		return false, fmt.Errorf("locked amount exceeds wallet balance")
	}

	lockedAmount = new(big.Int).Mul(lockedAmount, big.NewInt(1e18))
	if totalBalance.Cmp(lockedAmount) < 0 {
		return false, fmt.Errorf("locked amount exceeds wallet balance")
	}

	maxAllowed := new(big.Int).Sub(totalBalance, lockedAmount)
	if txAmount.Cmp(maxAllowed) > 0 {
		return false, fmt.Errorf("transaction exceeds allowed fixed amount limit")
	}
	return true, nil
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
	txAmount := tx.Value() // Pass this value
	isUnlocked, err := k.IsWalletUnlocked(ctx, from, txAmount)
	if err != nil || !isUnlocked {
		return nil, fmt.Errorf("transaction rejected: wallet is locked")
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
