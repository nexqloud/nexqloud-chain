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
	"sync"
	"time"

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

// Cache structures to reduce EthCall frequency
type chainStatusCache struct {
	isOpen    bool
	timestamp time.Time
	height    int64
}

type walletLockCache struct {
	isUnlocked bool
	timestamp  time.Time
	height     int64
	amount     *big.Int
}

var (
	whitelist = map[string]bool{
		//staging config
		"0x50823c6fBF2Dd945480951ABBa144b9a1e89dFC3": true,

		//dev config
		// "0xE56A21BB0619225616DE7613937b2b816A14deB1": true,
	}
	
	// Cache with mutex for thread safety
	chainStatusCacheMap = make(map[string]*chainStatusCache)
	walletLockCacheMap  = make(map[string]*walletLockCache)
	cacheMutex          sync.RWMutex
	
	// Cache duration - increased to 30 seconds for better cache hit rate
	cacheDuration = 30 * time.Second
	
	// Emergency mode - disable chain status checks during high load
	emergencyMode = false
	
	// Cache statistics
	cleanupCounter = 0
	cacheHits      = 0
	cacheMisses    = 0
)

// cleanupCache removes expired cache entries to prevent memory leaks
func cleanupCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	
	now := time.Now()
	
	// Cleanup chain status cache
	for key, cache := range chainStatusCacheMap {
		if now.Sub(cache.timestamp) > cacheDuration {
			delete(chainStatusCacheMap, key)
		}
	}
	
	// Cleanup wallet lock cache
	for key, cache := range walletLockCacheMap {
		if now.Sub(cache.timestamp) > cacheDuration {
			delete(walletLockCacheMap, key)
		}
	}
	
	// Log cache statistics every 50th cleanup
	cleanupCounter++
	if cleanupCounter%50 == 0 {
		totalRequests := cacheHits + cacheMisses
		hitRate := 0.0
		if totalRequests > 0 {
			hitRate = float64(cacheHits) / float64(totalRequests) * 100
		}
		log.Printf("📊 Cache Stats - Chain Status: %d entries, Wallet Lock: %d entries, Hit Rate: %.2f%% (%d hits, %d misses)", 
			len(chainStatusCacheMap), len(walletLockCacheMap), hitRate, cacheHits, cacheMisses)
	}
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
	log.Println("🔍 Checking chain status")
	
	currentHeight := ctx.BlockHeight()
	
	// Periodic cache cleanup (every 10th call)
	if currentHeight%10 == 0 {
		cleanupCache()
	}
	
	// Use a simpler cache key - just "chain_status" since we only need one global status
	cacheKey := "chain_status"
	
	// Check cache first
	cacheMutex.RLock()
	if cache, exists := chainStatusCacheMap[cacheKey]; exists {
		// More flexible height check - allow cache hits within 10 blocks
		if time.Since(cache.timestamp) < cacheDuration && 
		   abs(currentHeight - cache.height) <= 10 {
			cacheMutex.RUnlock()
			cacheHits++
			log.Printf("🎯 Cache HIT for key: %s, height: %d, cached_height: %d, time_since: %v", 
				cacheKey, currentHeight, cache.height, time.Since(cache.timestamp))
			if cache.isOpen {
				log.Println("✅ Chain is OPEN (cached)")
			} else {
				log.Println("❌ Chain is CLOSED (cached)")
			}
			return cache.isOpen, nil
		} else {
			log.Printf("⏰ Cache EXPIRED for key: %s, time_since: %v, height_diff: %d", 
				cacheKey, time.Since(cache.timestamp), abs(currentHeight - cache.height))
		}
	} else {
		cacheMisses++
		log.Printf("❌ Cache MISS for key: %s (no entry exists)", cacheKey)
	}
	cacheMutex.RUnlock()
	
	// Debug: Log all existing cache keys
	cacheMutex.RLock()
	log.Printf("🔍 Current cache keys: %v", getCacheKeys())
	cacheMutex.RUnlock()
	
	// Verify cache state before making EthCall
	verifyCacheState()
	
	previousHeight := currentHeight - 1
	
	// Get the previous block's header
	previousHeader := ctx.BlockHeader()
	previousHeader.Height = previousHeight
	
	// Create context for the previous block
	previousCtx := ctx.WithBlockHeader(previousHeader)

	addr := common.HexToAddress(config.OnlineServerCountContract)
	data := hexutil.Bytes(getFunctionSelector("getOnlineServerCount()"))

	// Prepare the EthCall request
	args := types.TransactionArgs{
		From: &from,
		To:   &addr,
		Data: &data,
	}

	argsBytes, err := json.Marshal(args)
	if err != nil {
		return false, err
	}

	req := &types.EthCallRequest{
		Args:            argsBytes,
		GasCap:         uint64(25000000),
		ChainId:        config.ChainID,
		ProposerAddress: previousHeader.ProposerAddress,
	}

	// Call EthCall with previous block's context
	res, err := k.EthCall(previousCtx, req)
	if err != nil {
		return false, err
	}

	// Parse the response to get the online server count
	count := new(big.Int)
	count.SetBytes(res.Ret)
	log.Printf("Online Server Count: %s", count.String())

	// Check if the chain is open based on the count
	threshold := big.NewInt(1000)
	isOpen := count.Cmp(threshold) >= 0

	// Cache the result
	cacheMutex.Lock()
	log.Printf("💾 Caching chain status for key: %s, height: %d, isOpen: %v", cacheKey, currentHeight, isOpen)
	chainStatusCacheMap[cacheKey] = &chainStatusCache{
		isOpen:    isOpen,
		timestamp: time.Now(),
		height:    currentHeight,
	}
	log.Printf("✅ Status cached: %+v", chainStatusCacheMap[cacheKey])
	cacheMutex.Unlock()

	// Verify cache state after caching
	verifyCacheState()

	if isOpen {
		log.Println("✅ Chain is OPEN")
		return true, nil
	}

	log.Println("❌ Chain is CLOSED")
	return false, nil
}

// getCacheKeys returns all current cache keys for debugging
func getCacheKeys() []string {
	keys := make([]string, 0, len(chainStatusCacheMap))
	for key := range chainStatusCacheMap {
		keys = append(keys, key)
	}
	return keys
}

// verifyCacheState logs the current state of the cache for debugging
func verifyCacheState() {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	
	log.Printf("🔍 Cache State Verification:")
	log.Printf("  - Chain Status Cache Entries: %d", len(chainStatusCacheMap))
	log.Printf("  - Wallet Lock Cache Entries: %d", len(walletLockCacheMap))
	
	for key, cache := range chainStatusCacheMap {
		log.Printf("  - Key: %s, Height: %d, IsOpen: %v, Age: %v", 
			key, cache.height, cache.isOpen, time.Since(cache.timestamp))
	}
}

// abs returns the absolute value of an int64
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
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

	currentHeight := ctx.BlockHeight()
	walletKey := from.Hex()
	
	// Periodic cache cleanup (every 10th call)
	if currentHeight%10 == 0 {
		cleanupCache()
	}
	
	// Check cache first with more flexible conditions
	cacheMutex.RLock()
	if cache, exists := walletLockCacheMap[walletKey]; exists {
		// More flexible cache hit conditions:
		// 1. Cache is still valid (within time limit)
		// 2. Height is within 5 blocks
		// 3. Transaction amount is less than or equal to cached amount
		if time.Since(cache.timestamp) < cacheDuration && 
		   abs(currentHeight - cache.height) <= 5 && 
		   txAmount.Cmp(cache.amount) <= 0 {
			cacheMutex.RUnlock()
			log.Printf("🎯 Wallet Cache HIT for: %s, height: %d, cached_height: %d", walletKey, currentHeight, cache.height)
			if cache.isUnlocked {
				log.Println("✅ Wallet is unlocked (cached)")
			} else {
				log.Println("❌ Wallet is locked (cached)")
			}
			return cache.isUnlocked, nil
		} else {
			log.Printf("⏰ Wallet Cache EXPIRED for: %s, time_since: %v, height_diff: %d", 
				walletKey, time.Since(cache.timestamp), abs(currentHeight - cache.height))
		}
	} else {
		log.Printf("❌ Wallet Cache MISS for: %s", walletKey)
	}
	cacheMutex.RUnlock()

	// Define the WalletState contract address
	walletStateContract := common.HexToAddress(config.WalletStateContractAddress)

	// Prepare the function selector for getWalletLock(address)
	functionSelector := getFunctionSelector("getWalletLock(address)")
	paddedAddress := common.LeftPadBytes(from.Bytes(), 32) // 32-byte encoding for address
	data := append(functionSelector, paddedAddress...)

	// Convert data to hexutil.Bytes explicitly
	hexData := hexutil.Bytes(data)

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
	
	var isUnlocked bool
	var resultError error
	
	switch lockStatus {
	case 0: // No_Lock
		log.Println("✅ Wallet is unlocked")
		isUnlocked = true
		resultError = nil

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
				isUnlocked = true
				resultError = nil
			} else {
				log.Printf("❌ Tx %s > Allowed %s (diff: %s)", txAmount.String(), maxAllowed.String(), diff.String())
				isUnlocked = false
				resultError = fmt.Errorf("exceeds limit")
			}
		} else {
			log.Println("✅ Transaction allowed under amount lock")
			isUnlocked = true
			resultError = nil
		}

	case 2: // Absolute_Lock
		log.Println("❌ Wallet is fully locked")
		isUnlocked = false
		resultError = fmt.Errorf("wallet is fully locked")

	default:
		log.Println("❌ Unknown lock status")
		isUnlocked = false
		resultError = fmt.Errorf("unknown lock status")
	}

	// Cache the result only if successful
	if resultError == nil {
		cacheMutex.Lock()
		walletLockCacheMap[walletKey] = &walletLockCache{
			isUnlocked: isUnlocked,
			timestamp:  time.Now(),
			height:     currentHeight,
			amount:     new(big.Int).Set(txAmount),
		}
		cacheMutex.Unlock()
	}

	return isUnlocked, resultError
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
