// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"context"
	"log"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkstakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"math/big"

	"github.com/ethereum/go-ethereum/common"
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
	log.Println("Validator address (bech32): ", msg.ValidatorAddress)
	log.Println("Delegator address (bech32): ", msg.DelegatorAddress)

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
	log.Println("NFT Contract address: ", nftContract.Hex())

	// Convert VALIDATOR OPERATOR address to Ethereum address
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		log.Println("ERROR: Invalid validator operator address:", err)
		return nil, errorsmod.Wrap(err, "invalid validator address")
	}

	valAccAddr := sdk.AccAddress(valAddr)
	valEvmAddr := common.BytesToAddress(valAccAddr)
	log.Println("Validator Ethereum address: ", valEvmAddr.Hex())

	// Standard ERC721/ERC1155 balanceOf ABI
	abiJSON := `[{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"}]`

	log.Println("Calling NFT contract balanceOf method...")

	// Try to call the contract with VALIDATOR address
	res, err := k.evmKeeper.CallEVM(
		ctx,
		abiJSON,
		"balanceOf",
		nftContract,
		valEvmAddr,
	)

	if err != nil {
		log.Printf("ERROR: NFT balance check failed: %v", err)
		log.Printf("Parameters: contract=%s, method=balanceOf, owner=%s",
			nftContract.Hex(), valEvmAddr.Hex())
		
		return nil, errorsmod.Wrapf(
			errortypes.ErrInvalidRequest,
			"failed NFT check for validator %s: %v", 
			msg.ValidatorAddress, err,
		)
	}

	// Process the balance result
	nftBalance := new(big.Int).SetBytes(res.Ret)
	log.Println("NFT Balance: ", nftBalance.String())

	if nftBalance.Cmp(big.NewInt(1)) < 0 {
		log.Println("ERROR: Validator does not own any NXQNFT")
		return nil, errorsmod.Wrap(errortypes.ErrUnauthorized, "must own ≥1 NXQNFT")
	}

	log.Println("✅ NFT validation passed successfully")
	log.Println("========= NFT Validation End =========")

	if err := k.validateDelegationAmountNotUnvested(goCtx, msg.DelegatorAddress, msg.Value.Amount); err != nil {
		log.Println("ERROR: Delegation validation failed:", err)
		return nil, err
	}

	log.Println("========= Create Validator =========")
	log.Println("Delegator address: ", msg.DelegatorAddress)
	log.Println("Value: ", msg.Value)
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
