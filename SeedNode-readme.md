# NexQloud Blockchain Seed Node

This document explains how to set up and run a NexQloud blockchain seed node using the `seed_node.sh` script.

## Overview

The seed node is a critical component of the NexQloud blockchain network. It serves as:

- An initial block producer
- A discovery point for peer nodes
- A provider of the genesis file
- A validator for the network

## Requirements

- Linux or macOS operating system
- `jq` utility installed
- NexQloud binary (`nxqd`) - either built locally or installed globally
- Sufficient disk space for blockchain data
- Internet connectivity

## Configuration

The seed node script (`seed_node.sh`) includes several configurable parameters:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `CHAINID` | `nxqd_6000-1` | The chain ID for the blockchain |
| `MONIKER` | `NexqloudSeedNode1` | Node identifier |
| `NXQD_BIN` | Local or system path | Path to the nxqd binary |
| `KEYRING` | `file` | Keyring backend type (secure) |
| `HOMEDIR` | `$HOME/.nxqd` | Data directory |
| `BASEFEE` | `1000000000` | Base fee for transactions |

## RPC Configuration

The script configures multiple RPC endpoints:

1. **Tendermint RPC**: Exposed on port 26657
   - Set to bind to `0.0.0.0` (all interfaces)
   - Used for block synchronization and validator operations

2. **Ethereum JSON-RPC**: Exposed on port 8545
   - Set to bind to `0.0.0.0` (all interfaces)
   - Used for EVM compatibility
   - Explicitly enabled in configuration

3. **WebSocket RPC**: Exposed on port 8546
   - Set to bind to `0.0.0.0` (all interfaces)
   - Used for event subscription

## Usage

The seed node script supports the following commands:

### Initialize the Node

```bash
./seed_node.sh init
```

This command:
1. Removes any existing data directory
2. Sets up client configuration
3. Generates keys (important: write down the mnemonics when displayed!)
4. Initializes the chain with the specified moniker and chain ID
5. Customizes genesis parameters
6. Configures RPC endpoints
7. Creates genesis transactions
8. Validates the genesis file
9. Configures all RPC endpoints to be accessible

### Start the Node

```bash
./seed_node.sh start
```

This command:
1. Verifies the configuration
2. Ensures all RPC endpoints are properly exposed
3. Starts the blockchain node with proper parameters
4. Provides informative output about available endpoints

### Test Mode

```bash
./seed_node.sh test
```

This command combines initialization and starting with additional test configuration:
1. Sets the `BYPASS_NFT_VALIDATION` environment variable
2. Initializes the node
3. Adds test-specific configuration
4. Starts the node

## Key Management

The script generates multiple keys with the following important notes:

1. **IMPORTANT**: When keys are generated, the mnemonic phrases are displayed ONLY ONCE
2. You MUST manually write down and securely store these mnemonics
3. If you lose these mnemonics, you will lose access to the validator and funds
4. Use secure password management for the keyring password

## Genesis Accounts

The seed node initializes with several accounts:

1. `mykey` - Primary validator account
2. `vault1` through `vault5` - Vault accounts with significant token allocation
3. `maintenance` - Maintenance wallet

## Recent Changes and Improvements

Recent updates to the script include:

1. Improved RPC exposure:
   - Changed Tendermint RPC binding from localhost to all interfaces (`0.0.0.0`)
   - Explicitly enabled JSON-RPC server with `enable = true`

2. Enhanced verification:
   - Added configuration verification steps
   - Final checks to ensure proper exposure

3. Better error handling:
   - Fallback mechanisms for RPC configuration
   - Informative error messages

4. Improved debugging:
   - Added hostname and IP information for RPC endpoints
   - Added file permission checks
   
5. Simplified key management:
   - Removed keybackup functionality
   - Clear warnings to manually record mnemonics

## Troubleshooting

If you encounter issues:

1. **RPC Accessibility**: Verify configurations with:
   ```bash
   grep "laddr = \"tcp:" $HOME/.nxqd/config/config.toml
   grep "enable = true" $HOME/.nxqd/config/app.toml
   ```

2. **Binary Not Found**: Ensure the correct path is set in the `NXQD_BIN` variable

3. **Permission Issues**: Check file permissions with:
   ```bash
   ls -la $HOME/.nxqd/config/
   ```

4. **NFT Validation**: For testing, use the `test` command or bypass validation in the code

5. **Lost Keys**: If you've lost your mnemonics:
   - If the validator is already active, you can still access it using the keyring password
   - For recovery without the keyring, you'll need the original mnemonics

## Security Considerations

1. The keyring is set to `file` mode for production security
2. Mnemonics need to be manually recorded and stored securely 
3. Consider using hardware wallets or cold storage solutions for production validators
4. RPC endpoints are exposed on all interfaces - consider firewall rules for production 