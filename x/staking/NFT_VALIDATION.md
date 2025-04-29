# NFT Validation for Validators

## Overview

The Nexqloud Chain implements a novel validator qualification mechanism that requires prospective validators to own a minimum number of NXQNFT tokens and maintain a minimum self-delegation of NXQ tokens. These requirements are dynamically fetched from a smart contract, allowing for updates to the validation criteria without requiring a chain upgrade. This document explains the technical implementation of these features, how they work, and how to troubleshoot common issues.

## How It Works

### Conceptual Flow

1. When a user attempts to create a validator via the `MsgCreateValidator` message, the staking module intercepts this request.
2. The module queries the WalletState contract to get the current validator requirements (NXQ tokens and NFTs).
3. The module converts the validator's address to an Ethereum-compatible address format.
4. The module then makes an EVM call to the NFT contract to check the validator's NFT token balance.
5. If the validator owns at least the required number of NXQNFT tokens, the NFT validation passes.
6. Next, the module checks if the validator's minimum self-delegation is at least the required amount of NXQ tokens.
7. If both checks pass, the validator creation proceeds.
8. If either check fails, the request is rejected with an appropriate error message.

### Technical Components

The validation features consist of these main components:

1. **Interface Implementation**: An interface called `EvmEthCaller` that bridges the staking module to the EVM module.
2. **Requirement Query**: A function that queries the WalletState contract to get the current validator requirements.
3. **NFT Balance Query**: A function that makes a read-only EVM call to check the NFT token balance.
4. **Minimum Self-Delegation Check**: Code that enforces the minimum self-delegation requirement.
5. **Validation Logic**: Code that enforces both the NFT ownership and self-delegation requirements during validator creation.
6. **Error Handling**: Robust error handling to provide clear feedback to users.

## Implementation Details

### Component Structure

The implementation spans across several files:

- `x/staking/keeper/msg_server.go`: Contains the NFT validation and minimum self-delegation validation logic in the `CreateValidator` method
- `x/evm/types/interfaces.go`: Defines the `EthCall` method in the EVMKeeper interface
- `app/app.go`: Implements the `evmKeeperAdapter` that connects the staking module to the EVM functionality

### Smart Contract Addresses

- **WalletState Contract**:  - Used to fetch the current validator requirements
- **NFT Contract**: - Used to check the validator's NFT token balance

### WalletState Contract

The WalletState contract includes a function to get the current validator requirements:

**Function Signature**: `getValidatorRequirements() external view returns (uint256, uint256)`

This function returns two values:
1. The required number of NXQ tokens
2. The required number of NXQNFT tokens

See implementation in the contract source files for details.

### Key Interfaces

The EVM call functionality is implemented through interfaces defined in:
- `x/evm/types/interfaces.go` - Defines the `EthCall` method
- `x/staking/keeper/msg_server.go` - Defines the `EvmEthCaller` interface

For detailed implementation, please refer to the source files directly.

### Core Implementation

The key implementation is in the `CreateValidator` method in `x/staking/keeper/msg_server.go`. This function:

1. Converts validator addresses between Cosmos and Ethereum formats
2. Fetches validator requirements from the WalletState contract
3. Checks the validator's NFT balance
4. Validates that the NFT balance meets requirements
5. Verifies the minimum self-delegation meets requirements
6. Proceeds with validator creation if all requirements are met

For the complete implementation details, please refer to the source file directly.

### Requirement Fetching

The validator requirements are fetched using the `getValidatorRequirements` method in `x/staking/keeper/msg_server.go`. This function:

1. Constructs the function selector for `getValidatorRequirements()`
2. Makes an EVM call to the WalletState contract
3. Parses the response to extract the required token amounts

See the source file for implementation details.

## Fallback Mechanism

If the call to fetch validator requirements fails, the system falls back to default values:
- 5 NXQ tokens for minimum self-delegation
- 1 NXQNFT for NFT ownership requirement

This ensures that validator creation can proceed even if there are temporary issues with the WalletState contract.

## Troubleshooting

### Common Issues and Solutions

#### "evmKeeper doesn't implement EthCall"

**Cause**: The `EvmEthCaller` interface is properly defined in the staking module, but the adapter in `app/app.go` doesn't implement the required `EthCall` method.

**Solution**: Update the `evmKeeperAdapter` in `app/app.go` to include the `EthCall` method that delegates to the EVM keeper's method.

#### "empty response from contract"

**Cause**: The NFT contract or WalletState contract address is not deployed, or doesn't implement the required functions correctly.

**Solution**: 
1. Verify the contract addresses are correct
2. Check that the contracts are deployed on the chain
3. Verify the contracts implement the required functions correctly

#### "minimum self delegation must be at least X NXQ"

**Cause**: The validator is trying to set a minimum self-delegation amount that is less than the required amount fetched from the WalletState contract.

**Solution**: Increase the `--min-self-delegation` parameter to match or exceed the requirement from the WalletState contract.

#### "must own ≥X NXQNFT"

**Cause**: The validator does not own enough NXQNFT tokens as required by the WalletState contract.

**Solution**: Obtain the required number of NXQNFT tokens before attempting to create a validator.

#### "insufficient funds" when creating validator

**Cause**: While the validation checks passed, the account doesn't have enough tokens to stake.

**Solution**: Fund the validator account with enough tokens before attempting to create a validator.

## Testing

To test the validation:

1. Deploy the NFT contract with standard ERC-721 interface
2. Deploy the WalletState contract and set the validator requirements
3. Mint the required NFTs to the validator's address
4. Set `--min-self-delegation` to at least the required amount
5. Verify the logs show both "NFT validation passed successfully" and "Minimum self delegation requirement met"

For more detailed testing guides and tools, see `VALIDATOR_TESTING.md`.

## Configuration Updates

The validator requirements can be updated by calling the `setValidatorRequirements` function on the WalletState contract. This allows for updating the requirements without needing to upgrade the chain.

## Deployment Requirements

For validators to successfully create their nodes, they need:

1. At least the required number of NXQNFT tokens in their validator address
2. A minimum self-delegation of at least the required number of NXQ tokens
3. Sufficient tokens for staking (at least the amount specified in the self-delegation)
4. A properly configured validator node

## Security Considerations

The dynamic validation mechanism provides several security benefits:
1. The NFT ownership requirement ensures only those who own the required NFTs can become validators
2. The minimum self-delegation requirement ensures validators have sufficient economic stake in the network
3. The ability to update requirements via the WalletState contract allows for governance to adjust validation criteria as needed

Together, these enhance security and provide stake-like commitments from validators beyond just the token stake. 