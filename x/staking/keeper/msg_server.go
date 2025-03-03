// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkstakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	evmtypes "github.com/evmos/evmos/v19/x/evm/types"
	vestingtypes "github.com/evmos/evmos/v19/x/vesting/types"
)

// msgServer is a wrapper around the Cosmos SDK message server.
type msgServer struct {
	types.MsgServer
	*Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the staking MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	baseMsgServer := sdkstakingkeeper.NewMsgServerImpl(keeper.Keeper)
	return &msgServer{baseMsgServer, keeper}
}

// Delegate defines a method for performing a delegation of coins from a delegator to a validator.
// The method performs some checks if the sender of the tx is a clawback vesting account and then
// relay the message to the Cosmos SDK staking method.
func (k msgServer) Delegate(goCtx context.Context, msg *types.MsgDelegate) (*types.MsgDelegateResponse, error) {
	if err := k.validateDelegationAmountNotUnvested(goCtx, msg.DelegatorAddress, msg.Amount.Amount); err != nil {
		return nil, err
	}

	return k.MsgServer.Delegate(goCtx, msg)
}

// CreateValidator defines a method to create a validator. The method performs some checks if the
// sender of the tx is a clawback vesting account and then relay the message to the Cosmos SDK staking
// method.
func (k msgServer) CreateValidator(goCtx context.Context, msg *types.MsgCreateValidator) (*types.MsgCreateValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	log.Println("========= NFT Validation Start =========")
	log.Printf("Validator address (bech32): %s", msg.ValidatorAddress)
	log.Printf("Delegator address (bech32): %s", msg.DelegatorAddress)

	// Check if EVM Keeper is initialized
	if k.evmKeeper == nil {
		log.Println("FATAL: EVM Keeper not initialized")
		return nil, errorsmod.Wrap(
			errortypes.ErrInvalidRequest,
			"EVM module not configured",
		)
	}

	// NFT Contract Check
	nftContract := common.HexToAddress("0x816644F8bc4633D268842628EB10ffC0AdcB6099")
	log.Printf("NFT Contract address: %s", nftContract.Hex())

	// Convert validator address to correct account format
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		log.Printf("ERROR: Invalid validator operator address: %v", err)
		return nil, errorsmod.Wrap(err, "invalid validator address")
	}
	valAccAddr := sdk.AccAddress(valAddr.Bytes())
	valEvmAddr := common.BytesToAddress(valAccAddr)
	log.Printf("Validator Address conversion: %s (bech32) -> %s (Ethereum)", msg.ValidatorAddress, valEvmAddr.Hex())

	// Also convert delegator address
	delAddr, _ := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	delEvmAddr := common.BytesToAddress(delAddr)
	log.Printf("Delegator Address conversion: %s (bech32) -> %s (Ethereum)", msg.DelegatorAddress, delEvmAddr.Hex())

	// Fetch validator requirements from the WalletState contract
	walletStateContract := common.HexToAddress("0xA912e0631f97e52e5fb8435e20f7B4c7755F7de3")
	requiredNXQTokens, requiredNXQNFTs, err := k.getValidatorRequirements(ctx, walletStateContract)
	if err != nil {
		log.Printf("ERROR: Failed to get validator requirements: %v", err)
		log.Println("Using default values: 5 NXQ tokens, 1 NXQNFT")
		requiredNXQTokens = big.NewInt(5_000_000_000_000_000_000) // Default: 5 NXQ with 18 decimals
		requiredNXQNFTs = big.NewInt(1)                          // Default: 1 NFT
	} else {
		log.Printf("Fetched validator requirements: %s NXQ tokens, %s NXQNFT", 
			requiredNXQTokens.String(), requiredNXQNFTs.String())
	}

	// Get the NFT balance using direct EthCall approach
	nftBalance, err := k.getNFTBalance(ctx, nftContract, valEvmAddr)
	if err != nil {
		// provide a clear error message about the NFT requirement
		log.Printf("ERROR: Failed to query NFT balance: %v", err)
		return nil, errorsmod.Wrap(
			errortypes.ErrUnauthorized,
			fmt.Sprintf("unable to verify NFT ownership: %v - please ensure you own an NFT at contract %s",
				err, nftContract.Hex()),
		)
	}

	log.Printf("NFT Balance for %s: %s", valEvmAddr.Hex(), nftBalance.String())

	// Check if the NFT balance meets the requirement
	if nftBalance.Cmp(requiredNXQNFTs) < 0 {
		log.Printf("ERROR: Validator does not have enough NXQNFT. Required: ≥%s, Found: %s", 
			requiredNXQNFTs.String(), nftBalance.String())
		return nil, errorsmod.Wrap(
			errortypes.ErrUnauthorized, 
			fmt.Sprintf("must own ≥%s NXQNFT, found %s", 
				requiredNXQNFTs.String(), nftBalance.String()),
		)
	}

	log.Println("✅ NFT validation passed successfully")
	log.Println("========= NFT Validation End =========")

	// Convert required NXQ tokens from big.Int to sdk.Int for comparison
	requiredMinSelfDelegation := sdk.NewIntFromBigInt(requiredNXQTokens)
	
	// Ensure minimum self delegation meets the requirement
	if msg.MinSelfDelegation.LT(requiredMinSelfDelegation) {
		log.Printf("ERROR: Minimum self delegation too low. Required: ≥%s NXQ, Found: %s", 
			requiredMinSelfDelegation.String(), msg.MinSelfDelegation.String())
		return nil, errorsmod.Wrap(
			errortypes.ErrInvalidRequest,
			fmt.Sprintf("minimum self delegation must be at least %s NXQ, got %s", 
				requiredMinSelfDelegation.String(), msg.MinSelfDelegation.String()),
		)
	}
	log.Printf("✅ Minimum self delegation requirement met: %s ≥ %s", 
		msg.MinSelfDelegation.String(), requiredMinSelfDelegation.String())

	if err := k.validateDelegationAmountNotUnvested(goCtx, msg.DelegatorAddress, msg.Value.Amount); err != nil {
		log.Printf("ERROR: Delegation validation failed: %v", err)
		return nil, err
	}

	log.Println("========= Create Validator =========")
	log.Printf("Delegator address: %s", msg.DelegatorAddress)
	log.Printf("Value: %s", msg.Value)
	log.Println("====================================")

	return k.MsgServer.CreateValidator(goCtx, msg)
}

// validateDelegationAmountNotUnvested checks if the delegator is a clawback vesting account.
// In such case, checks that the provided delegation amount is available according
// to the current vesting schedule (unvested coins cannot be delegated).
func (k msgServer) validateDelegationAmountNotUnvested(goCtx context.Context, delegatorAddress string, amount math.Int) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(delegatorAddress)
	if err != nil {
		return err
	}

	acc := k.ak.GetAccount(ctx, addr)
	if acc == nil {
		return errorsmod.Wrapf(
			errortypes.ErrUnknownAddress,
			"account %s does not exist", addr,
		)
	}
	// check if delegator address is a clawback vesting account. If not, no check
	// is required.
	clawbackAccount, isClawback := acc.(*vestingtypes.ClawbackVestingAccount)
	if !isClawback {
		return nil
	}

	// vesting account can only delegate
	// if enough free balance (coins not in vesting schedule)
	// plus the vested coins (locked/unlocked)
	bondDenom := k.BondDenom(ctx)
	// GetBalance returns entire account balance
	// balance = free coins + all coins in vesting schedule
	balance := k.bk.GetBalance(ctx, addr, bondDenom)
	unvestedOnly := clawbackAccount.GetVestingCoins(ctx.BlockTime())
	// delegatable coins are going to be all the free coins + vested coins
	// Can only delegate bondable coins
	unvestedBondableAmt := unvestedOnly.AmountOf(bondDenom)
	// A ClawbackVestingAccount can delegate coins from the vesting schedule
	// when having vested locked coins or unlocked vested coins.
	// It CANNOT delegate unvested coins
	delegatableAmt := balance.Amount.Sub(unvestedBondableAmt)
	if delegatableAmt.IsNegative() {
		delegatableAmt = math.ZeroInt()
	}

	if delegatableAmt.LT(amount) {
		return errorsmod.Wrapf(
			vestingtypes.ErrInsufficientVestedCoins,
			"cannot delegate unvested coins. coins available for delegation < delegation amount (%s < %s)",
			delegatableAmt, amount,
		)
	}

	log.Println("========= Delegation custom logic =========")
	log.Println("Delegator address: ", delegatorAddress)
	log.Println("Delegator balance: ", balance)
	log.Println("Unvested only: ", unvestedOnly)
	log.Println("Unvested bondable amount: ", unvestedBondableAmt)
	log.Println("Delegatable amount: ", delegatableAmt)
	log.Println("Delegation amount: ", amount)
	log.Println("===========================================")

	return nil
}

// getValidatorRequirements queries the WalletState contract to get the required number of
// NXQ tokens and NXQNFT's to become a validator
func (k msgServer) getValidatorRequirements(ctx sdk.Context, contractAddr common.Address) (*big.Int, *big.Int, error) {
	log.Printf("Querying validator requirements from contract: %s", contractAddr.Hex())

	// Function selector for getValidatorRequirements()
	// This is the first 4 bytes of keccak256("getValidatorRequirements()")
	callData := []byte{0x63, 0xd2, 0xc7, 0x33}
	hexData := hexutil.Bytes(callData)

	// Construct the TransactionArgs struct
	args := evmtypes.TransactionArgs{
		To:   &contractAddr,
		Data: &hexData,
	}

	// Marshal the TransactionArgs to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal transaction args: %w", err)
	}

	// Create the EthCallRequest
	ethCallRequest := &evmtypes.EthCallRequest{
		Args:   argsJSON,
		GasCap: 25000,
	}

	// Make the EthCall
	log.Printf("Making direct EthCall to WalletState contract %s", contractAddr.Hex())

	// Check if evmKeeper has EthCall method
	ethCaller, ok := k.evmKeeper.(EvmEthCaller)
	if !ok {
		return nil, nil, fmt.Errorf("evmKeeper doesn't implement EthCall")
	}

	// Execute the EthCall
	response, err := ethCaller.EthCall(ctx, ethCallRequest)
	if err != nil {
		log.Printf("ERROR: EthCall to WalletState contract failed: %v", err)
		return nil, nil, err
	}

	// Parse the response
	if response == nil || len(response.Ret) == 0 {
		return nil, nil, fmt.Errorf("empty response from WalletState contract")
	}

	// The response contains two uint256 values packed together (each 32 bytes)
	if len(response.Ret) < 64 {
		return nil, nil, fmt.Errorf("invalid response length from WalletState contract: %d", len(response.Ret))
	}

	// Extract the two uint256 values
	requiredNXQTokens := new(big.Int).SetBytes(response.Ret[:32])
	requiredNXQNFTs := new(big.Int).SetBytes(response.Ret[32:64])

	log.Printf("Retrieved validator requirements: %s NXQ tokens, %s NXQNFT",
		requiredNXQTokens.String(), requiredNXQNFTs.String())

	return requiredNXQTokens, requiredNXQNFTs, nil
}

// getNFTBalance queries the NFT balance using direct EthCall approach
// This is the proper way to query EVM contracts 
func (k msgServer) getNFTBalance(ctx sdk.Context, contractAddr, ownerAddr common.Address) (*big.Int, error) {
	log.Printf("Querying NFT balance using EthCall for address: %s", ownerAddr.Hex())

	// balanceOf function signature (0x70a08231)
	// followed by the address parameter
	callData := append([]byte{0x70, 0xa0, 0x82, 0x31}, common.LeftPadBytes(ownerAddr.Bytes(), 32)...)

	// Convert to hexutil.Bytes for the TransactionArgs
	hexData := hexutil.Bytes(callData)

	// Construct the TransactionArgs struct
	args := evmtypes.TransactionArgs{
		To:   &contractAddr,
		Data: &hexData,
	}

	// Marshal the TransactionArgs to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction args: %w", err)
	}

	// Create the EthCallRequest with the correct args field
	ethCallRequest := &evmtypes.EthCallRequest{
		Args: argsJSON,
		// Set a reasonable gas cap to prevent DoS
		GasCap: 25000,
	}

	// Make the direct call using EthCall
	log.Printf("Making direct EthCall to contract %s", contractAddr.Hex())

	// Check if evmKeeper has EthCall method
	ethCaller, ok := k.evmKeeper.(EvmEthCaller)
	if !ok {
		return nil, fmt.Errorf("evmKeeper doesn't implement EthCall")
	}

	// Make the EthCall
	response, err := ethCaller.EthCall(ctx, ethCallRequest)
	if err != nil {
		// Log the specific error but don't return hardcoded values
		log.Printf("ERROR: EthCall failed: %v", err)
		return nil, err
	}

	// Parse the response
	if response == nil || len(response.Ret) == 0 {
		return nil, fmt.Errorf("empty response from contract")
	}

	// The response.Ret contains the balance as a 32-byte big-endian integer
	balance := new(big.Int).SetBytes(response.Ret)
	log.Printf("NFT balance retrieved successfully: %s", balance.String())

	return balance, nil
}

// EvmEthCaller interface for EthCall
type EvmEthCaller interface {
	EthCall(ctx sdk.Context, req *evmtypes.EthCallRequest) (*evmtypes.MsgEthereumTxResponse, error)
}
