# Validator Requirements Testing Tools

This directory contains tools for testing and verifying validator requirements in the Nexqloud Chain. The implementation fetches requirements dynamically from the WalletState contract and validates NFT ownership against the NFT contract.

## Configuration

Contract addresses and node URLs are defined within each file for simplicity:

- WalletState Contract: `0xA912e0631f97e52e5fb8435e20f7B4c7755F7de3`
- NFT Contract: `0x816644F8bc4633D268842628EB10ffC0AdcB6099`
- Node URL: `http://dev-node.nexqloud.net:8545`

These values can be modified directly in the code or overridden via command-line flags in the validator-check tool.

## Test Files

### `validator_requirements_test.go`

This test file demonstrates how to:

1. Connect to the Ethereum node
2. Query the WalletState contract for validator requirements
3. Query the NFT contract for a validator's NFT balance
4. Check if the requirements are met

Run the tests with:

```bash
cd x/staking/keeper
go test -v -run TestValidatorRequirements
```

To test a specific address's NFT balance, modify the address in `TestNFTBalanceCheck` and `TestFullValidatorCheck`.

## Command Line Tool

A command-line tool is provided in `cmd/validator-check/main.go` that allows checking any address for validator eligibility.

### Building the Tool

```bash
# Using the Makefile target
make validator-check

# Or build directly
cd cmd/validator-check
go build
```

### Using the Tool

```bash
# Check an address using default settings
./build/validator-check --address=0xYourAddressHere

# Override node URL or contract addresses
./build/validator-check --address=0xYourAddressHere --node=http://your-node:8545 --nft=0xNFTContract --wallet=0xWalletStateContract
```

The tool will output:
1. Current validator requirements (NXQ tokens and NXQNFT tokens)
2. NFT balance for the specified address
3. Whether the address meets the NFT requirement

## Implementation Details

### Fetching Validator Requirements

The validator requirements are fetched using the `getValidatorRequirements()` function on the WalletState contract. For implementation details, see `validator_requirements_test.go` and `cmd/validator-check/main.go`.

### Checking NFT Balance

NFT balances are checked using the ERC-721 standard `balanceOf(address)` function. For implementation details, refer to the source code files.

## Troubleshooting

### Contract Not Deployed

If you see errors like "NFT contract is not deployed" or "WalletState contract is not deployed", verify:
1. The contract addresses are correct
2. The node URL is accessible
3. The contracts have been deployed to the specified addresses

### Invalid Response Length

If you receive "invalid response length" errors when fetching validator requirements, verify:
1. The WalletState contract implements the `getValidatorRequirements()` function correctly
2. The function returns two uint256 values as expected

### Empty Response from NFT Contract

If you get "empty response from NFT contract", verify:
1. The NFT contract implements the ERC-721 standard
2. The `balanceOf(address)` function works as expected

## Integration with Chain Logic

The validator creation process in `x/staking/keeper/msg_server.go` uses similar functions to validate validator requirements. If either check fails, the validator creation is rejected with an appropriate error message.

In case of issues with the WalletState contract, the system falls back to default values:
- 5 NXQ tokens for minimum self-delegation
- 1 NXQNFT for NFT ownership requirement 