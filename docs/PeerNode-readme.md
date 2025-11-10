# NexQloud Blockchain Peer Node

This document explains how to set up and run a NexQloud blockchain peer node using the `peer_node.sh` script.

## Overview

A peer node in the NexQloud blockchain network serves as:

- A non-validating participant in the network
- A block synchronization endpoint
- A provider of blockchain data via RPC endpoints
- A connection point for applications interacting with the blockchain

Unlike seed nodes, peer nodes do not produce blocks or participate in the consensus process.

## Requirements

- Linux or macOS operating system
- `jq` utility installed
- NexQloud binary (`nxqd`) - either built locally or installed globally
- Internet connectivity to reach seed nodes
- Sufficient disk space for blockchain data

## Configuration

The peer node script (`peer_node.sh`) includes several configurable parameters:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `CHAINID` | `nxqd_6000-1` | The chain ID for the blockchain |
| `MONIKER` | `NexQloudPeer` | Node identifier (can be set via `MONIKER` env variable) |
| `SEED_NODE_IP` | `dev-node.nexqloud.net` | IP or hostname of seed node (can be set via env variable) |
| `KEYRING` | `test` | Keyring backend type |
| `HOMEDIR` | `$HOME/.nxqd` | Data directory |
| `NXQD_BIN` | Auto-detected | Path to the nxqd binary |

## RPC Configuration

The peer node script configures the same RPC endpoints as the seed node:

1. **Tendermint RPC**: Exposed on port 26657
   - Bound to `0.0.0.0` (all interfaces)
   - Used for block synchronization

2. **Ethereum JSON-RPC**: Exposed on port 8545
   - Bound to `0.0.0.0` (all interfaces)
   - Used for EVM compatibility
   - Explicitly enabled in configuration

3. **WebSocket RPC**: Exposed on port 8546
   - Bound to `0.0.0.0` (all interfaces)
   - Used for event subscription

## Usage

The peer node script supports the following commands:

### Initialize the Node

```bash
./peer_node.sh init
```

This command:
1. Removes any existing data directory
2. Configures the client with chain ID and keyring settings
3. Initializes the node with the specified moniker
4. Configures RPC endpoints to be accessible from any IP
5. Retrieves the seed node ID and configures it as a seed
6. Downloads the genesis file from the seed node
7. Validates the genesis file

### Start the Node

```bash
./peer_node.sh start
```

This command:
1. Verifies and updates RPC configurations if needed
2. Double-checks that all endpoints are properly exposed
3. Starts the blockchain node with appropriate parameters
4. Provides informative output about available RPC endpoints

## Connection to Seed Nodes

The peer node connects to seed nodes in two ways:

1. **Initial Configuration**: During initialization, it:
   - Fetches the node ID from the seed node
   - Configures the seed node as a peer in `config.toml`
   - Downloads the genesis file from the seed node

2. **Runtime Connection**: When running, it:
   - Connects to the seed node to synchronize blocks
   - Discovers additional peers through the seed node

## Recent Changes and Improvements

Recent updates to the peer node script include:

1. **Simplified Node Role**:
   - Removed validator functionality
   - Simplified to function purely as a peer node
   - Eliminated key generation and management

2. **Improved RPC Configuration**:
   - Changed Tendermint RPC binding from localhost to all interfaces (`0.0.0.0`)
   - Explicitly enabled JSON-RPC server with `enable = true`
   - Verified configurations to ensure proper accessibility

3. **Enhanced Usability**:
   - Added hostname and IP information for RPC endpoints
   - Added verification steps for configurations
   - Improved error handling for seed node connection issues

4. **Better Binary Detection**:
   - Auto-detection of system-wide or local binary
   - Fallback mechanism for finding the nxqd binary

## Troubleshooting

If you encounter issues:

1. **Seed Node Connection**: Verify the seed node is accessible:
   ```bash
   ping dev-node.nexqloud.net
   curl http://dev-node.nexqloud.net/node-id
   ```

2. **RPC Accessibility**: Check configurations with:
   ```bash
   grep "laddr = \"tcp:" $HOME/.nxqd/config/config.toml
   grep "enable = true" $HOME/.nxqd/config/app.toml
   ```

3. **Genesis File**: Ensure the genesis file was properly downloaded:
   ```bash
   ls -la $HOME/.nxqd/config/genesis.json
   $NXQD_BIN validate-genesis --home "$HOME/.nxqd"
   ```

4. **Binary Not Found**: If the nxqd binary is not detected:
   ```bash
   # Build locally
   cd cmd/nxqd && go build
   # Or specify path explicitly
   NXQD_BIN=/path/to/nxqd ./peer_node.sh init
   ```

## Security Considerations

1. Since the peer node exposes RPC endpoints on all interfaces (`0.0.0.0`), consider implementing firewall rules to restrict access in production environments.

2. The keyring is set to `test` mode for simplicity, which is acceptable for peer nodes since they don't hold or stake tokens.

3. Consider monitoring the peer node for performance and connectivity issues.

## Environment Variables

You can customize the peer node by setting these environment variables before running the script:

- `MONIKER`: Set a custom name for your peer node
- `SEED_NODE_IP`: Specify a different seed node IP or hostname
- `NXQD_BIN`: Explicitly set the path to the nxqd binary

Example:
```bash
MONIKER="MyCustomPeer" SEED_NODE_IP="custom-seed.example.com" ./peer_node.sh init
``` 