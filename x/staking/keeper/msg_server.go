// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkstakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	config "github.com/evmos/evmos/v19/x/config"
	evmtypes "github.com/evmos/evmos/v19/x/evm/types"
	vestingtypes "github.com/evmos/evmos/v19/x/vesting/types"
	"golang.org/x/crypto/sha3"
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

// customValidatorChecks performs custom validation checks for validators
// This includes NFT ownership validation and minimum self-delegation requirements
func (k msgServer) customValidatorChecks(ctx sdk.Context, msg *types.MsgCreateValidator) error {
	
	// Check if we're in a genesis block (height <= 1)
	// During initialization, bypass all validation
	if ctx.BlockHeight() <= 1 {
		return nil
	}

	// Check if EVM Keeper is initialized
	if k.evmKeeper == nil {

		return errorsmod.Wrap(
			errortypes.ErrInvalidRequest,
			"EVM module not configured",
		)
	}

	// NFT Contract Check
	nftContract := common.HexToAddress(config.NFTContractAddress)
	log.Printf("NFT Contract address: %s", nftContract.Hex())

	// Convert validator address to correct account format
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		log.Printf("ERROR: Invalid validator operator address: %v", err)
		return errorsmod.Wrap(err, "invalid validator address")
	}

	// Convert validator address to EVM address format for validation checks
	valAccAddr := sdk.AccAddress(valAddr.Bytes())
	valEvmAddr := common.BytesToAddress(valAccAddr)

	// Check if the validator is approved (with simplified error handling)
	isApproved, err := k.isApprovedValidator(ctx, valEvmAddr)
	if err != nil {
		log.Printf("WARNING: Failed to check if validator is approved: %v, assuming not approved", err)
		return errorsmod.Wrap(err, "failed to check if validator is approved")
		// Continue with validation instead of returning error
	} else if isApproved {
		log.Printf("Validator %s is approved via contract call", valEvmAddr.Hex())
	} else {
		return errorsmod.Wrap(err, "validator is not approved")
	}

	// Get validator requirements (with fallback)
	walletStateContract := common.HexToAddress(config.WalletStateContractAddress)
	requiredNXQTokens, requiredNXQNFTs, err := k.getValidatorRequirements(ctx, walletStateContract)
	if err != nil {
	
		requiredNXQTokens = big.NewInt(5_000_000_000_000_000_000) // Default: 5 NXQ with 18 decimals
		requiredNXQNFTs = big.NewInt(5)                           // Default: 5 NFT
	}

	// Get NFT balance
	nftBalance, err := k.getNFTBalance(ctx, nftContract, valEvmAddr)
	if err != nil {

		return errorsmod.Wrap(
			errortypes.ErrUnauthorized,
			fmt.Sprintf("unable to verify NFT ownership: %v - please ensure you own an NFT at contract %s",
				err, nftContract.Hex()),
		)
	}

	// Check if the NFT balance meets the requirement
	if nftBalance.Cmp(requiredNXQNFTs) < 0 {
		log.Printf("NFT balance check failed: required %s, found %s",
			requiredNXQNFTs.String(), nftBalance.String())
		return errorsmod.Wrap(
			errortypes.ErrUnauthorized,
			fmt.Sprintf("must own ≥%s NXQNFT, found %s",
				requiredNXQNFTs.String(), nftBalance.String()),
		)
	}

	log.Println("✅ NFT validation passed successfully")

	// Convert required NXQ tokens from big.Int to sdk.Int for comparison
	requiredMinSelfDelegation := sdk.NewIntFromBigInt(requiredNXQTokens)

	// Ensure minimum self delegation meets the requirement
	if msg.MinSelfDelegation.LT(requiredMinSelfDelegation) {
		log.Printf("Minimum self delegation check failed: required %s, found %s",
			requiredMinSelfDelegation.String(), msg.MinSelfDelegation.String())
		return errorsmod.Wrap(
			errortypes.ErrInvalidRequest,
			fmt.Sprintf("minimum self delegation must be at least %s NXQ, got %s",
				requiredMinSelfDelegation.String(), msg.MinSelfDelegation.String()),
		)
	}
	log.Printf("✅ Minimum self delegation requirement met: %s ≥ %s",
		msg.MinSelfDelegation.String(), requiredMinSelfDelegation.String())

	return nil
}

// CreateValidator defines a method to create a validator. The method performs some checks if the
// sender of the tx is a clawback vesting account and then relay the message to the Cosmos SDK staking
// method.
func (k msgServer) CreateValidator(goCtx context.Context, msg *types.MsgCreateValidator) (*types.MsgCreateValidatorResponse, error) {
	// Unwrap the SDK context
	ctx := sdk.UnwrapSDKContext(goCtx)

	// NFT validation is now enabled
	err := k.customValidatorChecks(ctx, msg)
	if err != nil {
		return nil, err
	}

	// This check is always performed, regardless of node type
	if err := k.validateDelegationAmountNotUnvested(goCtx, msg.DelegatorAddress, msg.Value.Amount); err != nil {
		log.Printf("ERROR: Delegation validation failed: %v", err)
		return nil, err
	}


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

// getFunctionSelector calculates the 4-byte function selector for a Solidity function signature
// This matches Ethereum's implementation exactly
func getFunctionSelector(signature string) []byte {
	log.Printf("Calculating function selector for: %s", signature)
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	selector := hash.Sum(nil)[:4] // First 4 bytes of keccak256 hash
	log.Printf("Calculated selector: 0x%x", selector)
	return selector
}

// getValidatorRequirements queries the WalletState contract to get the required number of
// NXQ tokens and NXQNFT's to become a validator
func (k msgServer) getValidatorRequirements(ctx sdk.Context, contractAddr common.Address) (*big.Int, *big.Int, error) {

	// Calculate the function selector using the getFunctionSelector function
	functionSignature := "getValidatorRequirements()"
	callData := getFunctionSelector(functionSignature)
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
		GasCap: uint64(25000000), // Use default gas cap from config
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

	// balanceOf function signature
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
		GasCap: uint64(25000000), // Use default gas cap from config
	}

	// Make the direct call using EthCall

	// Check if evmKeeper has EthCall method
	ethCaller, ok := k.evmKeeper.(EvmEthCaller)
	if !ok {
		return nil, fmt.Errorf("evmKeeper doesn't implement EthCall")
	}

	// Make the EthCall
	response, err := ethCaller.EthCall(ctx, ethCallRequest)
	if err != nil {
		log.Printf("ERROR: EthCall failed: %v", err)
		return nil, err
	}

	// Parse the response
	if response == nil || len(response.Ret) == 0 {
		return nil, fmt.Errorf("empty response from contract")
	}

	// The response.Ret contains the balance as a 32-byte big-endian integer
	balance := new(big.Int).SetBytes(response.Ret)

	return balance, nil
}

// EvmEthCaller interface for EthCall
type EvmEthCaller interface {
	EthCall(ctx sdk.Context, req *evmtypes.EthCallRequest) (*evmtypes.MsgEthereumTxResponse, error)
}

// isApprovedValidator checks if an address is on the approved validators list
func (k msgServer) isApprovedValidator(ctx sdk.Context, validatorAddr common.Address) (bool, error) {

	// Get WalletState contract address
	walletStateContract := common.HexToAddress(config.WalletStateContractAddress)

	// Calculate the function selector for isApprovedValidator(address)
	functionSignature := "isApprovedValidator(address)"
	callData := getFunctionSelector(functionSignature)

	// Encode the address parameter (padded to 32 bytes)
	addressParam := common.LeftPadBytes(validatorAddr.Bytes(), 32)
	callData = append(callData, addressParam...)

	// Prepare the transaction arguments
	hexData := hexutil.Bytes(callData)
	args := evmtypes.TransactionArgs{
		To:   &walletStateContract,
		Data: &hexData,
	}

	// Marshal the transaction arguments to JSON
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return false, fmt.Errorf("failed to marshal transaction args: %w", err)
	}

	// Prepare the EthCall request
	req := &evmtypes.EthCallRequest{Args: argsJSON}

	// Make the call to the contract
	resp, err := k.evmKeeper.EthCall(ctx, req)
	if err != nil {
		return false, fmt.Errorf("failed to call isApprovedValidator: %w", err)
	}

	// Parse the result
	if len(resp.Ret) < 32 {
		return false, fmt.Errorf("unexpected result length: got %d bytes, want 32", len(resp.Ret))
	}

	// Check if the result is true (any non-zero byte means true for Solidity bool)
	for _, b := range resp.Ret {
		if b != 0 {
			log.Printf("Validator %s is approved", validatorAddr.Hex())
			return true, nil
		}
	}

	log.Printf("Validator %s is NOT approved", validatorAddr.Hex())
	return false, nil
}
