// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper

import (
	"context"
	"log"
	"strings"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkstakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	evmtypes "github.com/evmos/evmos/v19/x/evm/types"
	vestingtypes "github.com/evmos/evmos/v19/x/vesting/types"
)

// Add this struct definition after your EVMKeeper interface
type CallArgs struct {
	To   *common.Address `json:"to"`
	Data []byte          `json:"data"`
}

// Then update your interface
type EVMKeeper interface {
	CallEVM(ctx sdk.Context, args CallArgs, commitment bool) (*evmtypes.MsgEthereumTxResponse, error)
}

// msgServer is a wrapper around the Cosmos SDK message server.
type msgServer struct {
	types.MsgServer
	*Keeper
	evmKeeper EVMKeeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the staking MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper, evmKeeper EVMKeeper) types.MsgServer {
	baseMsgServer := sdkstakingkeeper.NewMsgServerImpl(keeper.Keeper)
	return &msgServer{baseMsgServer, keeper, evmKeeper}
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
	if err := k.validateDelegationAmountNotUnvested(goCtx, msg.DelegatorAddress, msg.Value.Amount); err != nil {
		return nil, err
	}

	// Check if the delegator has an NFT allocated to their wallet
	ctx := sdk.UnwrapSDKContext(goCtx)
	addr, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	nftOwnershipVerified, err := k.checkNFTOwnership(ctx, addr)
	if err != nil {
		return nil, err
	}

	if !nftOwnershipVerified {
		return nil, errorsmod.Wrapf(
			errortypes.ErrUnauthorized,
			"validator creation requires NFT ownership from contract 0x816644F8bc4633D268842628EB10ffC0AdcB6099",
		)
	}

	log.Println("========= Validator NFT Check =========")
	log.Println("Delegator address: ", msg.DelegatorAddress)
	log.Println("NFT check: PASSED")
	log.Println("=======================================")

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

// checkNFTOwnership verifies if the given address owns at least one NFT from the specified contract
func (k msgServer) checkNFTOwnership(ctx sdk.Context, addr sdk.AccAddress) (bool, error) {
	// NFT contract address
	nftContractAddr := common.HexToAddress("0x816644F8bc4633D268842628EB10ffC0AdcB6099")

	// Get the delegator's Ethereum address
	ethAddr := common.BytesToAddress(addr.Bytes())

	// Prepare balanceOf function call
	// Create ABI definition for the function
	parsedABI, err := abi.JSON(strings.NewReader(`[{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`))
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to parse ABI")
	}

	// Pack the function call with the owner address parameter
	data, err := parsedABI.Pack("balanceOf", ethAddr)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to pack balanceOf function call")
	}

	// Create EVM call message
	args := CallArgs{
		To:   &nftContractAddr,
		Data: data,
	}

	// Execute the static call using the EVM keeper
	res, err := k.evmKeeper.CallEVM(ctx, args, true)
	if err != nil {
		return false, errorsmod.Wrapf(err, "failed to call NFT contract")
	}

	// Check if call was successful
	if !res.Failed() {
		// Parse the returned balance
		var balance math.Int
		err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", res.Ret)
		if err != nil {
			return false, errorsmod.Wrapf(err, "failed to unpack balance result")
		}

		// Return true if the address owns at least one NFT
		return !balance.IsZero(), nil
	}

	return false, errorsmod.Wrapf(errortypes.ErrInvalidRequest, "NFT ownership check failed: %v", res.VmError)
}
