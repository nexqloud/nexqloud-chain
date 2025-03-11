#!/bin/bash

CHAINID="nxqd_6000-1"

# Set moniker if environment variable is not set
if [ -z "$MONIKER" ]; then
	MONIKER="NexQloudPeer"
fi

if [ -z "$SEED_NODE_IP" ]; then
    SEED_NODE_IP="dev-node.nexqloud.net"
fi

KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the nxqd instance
HOMEDIR="$HOME/.nxqd"
# to trace evm
#TRACE="--trace"
TRACE=""

# Use system-wide nxqd binary 
NXQD_BIN="$(which nxqd)"
# If not found, try using local binary
if [ -z "$NXQD_BIN" ]; then
    NXQD_BIN="$(pwd)/cmd/nxqd/nxqd"
    if [ ! -f "$NXQD_BIN" ]; then
        echo "Error: nxqd binary not found. Please build it first with: cd cmd/nxqd && go build"
        exit 1
    fi
fi

# feemarket params basefee
BASEFEE=1000000000

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
	echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
	exit 1
}

# used to exit on first error (any non-zero exit code)
set -e

if [[ $1 == "init" ]]; then

    # Remove the previous folder
    rm -rf "$HOMEDIR"

    # Set client config
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"

    # Initialize the node (as non-validator)
    echo "Initializing peer node with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"

    if [[ $1 == "pending" ]]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
            sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
            sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
            sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
            sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
            sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
            sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
            sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
        else
            sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
            sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
            sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
            sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
            sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
            sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
            sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
            sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
        fi
    fi

    # enable prometheus metrics and all APIs for dev node
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
    else
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
    fi

    # Enable the RPC
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i '' 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
        sed -i '' 's|enable = false|enable = true|g' "$APP_TOML"
        sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
    else
        sed -i 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
        sed -i 's|enable = false|enable = true|g' "$APP_TOML"
        sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
    fi

    # Verify the changes were applied
    echo "Verifying Tendermint RPC configuration:"
    grep "laddr = \"tcp:" "$CONFIG"
    echo "Verifying JSON-RPC configuration:"
    grep "enable = true" "$APP_TOML"

    # set seed node info
    echo "Configuring seed node: $SEED_NODE_IP"
    SEED_NODE_ID="`wget -qO-  http://$SEED_NODE_IP/node-id 2>/dev/null || curl -s http://$SEED_NODE_IP/node-id`"
    echo "SEED_NODE_ID=$SEED_NODE_ID"
    
    if [ -z "$SEED_NODE_ID" ]; then
        echo "WARNING: Failed to fetch seed node ID, trying to proceed anyway"
        SEED_NODE_ID="UNKNOWN_ID"
    fi
    
    SEEDS="$SEED_NODE_ID@$SEED_NODE_IP:26656"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
    else
        sed -i "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
    fi

    # Download genesis file
    echo "Downloading genesis file from: http://$SEED_NODE_IP/genesis.json"
    wget -qO- "http://$SEED_NODE_IP/genesis.json" > "$GENESIS" 2>/dev/null || \
    curl -s "http://$SEED_NODE_IP/genesis.json" > "$GENESIS"
    
    # Validate genesis
    echo "Validating genesis file"
    $NXQD_BIN validate-genesis --home "$HOMEDIR" || echo "WARNING: Genesis validation had issues but proceeding anyway"
    
    echo "Initialization complete! To start the node, run: $0 start"

else
    # Start the node
    echo "Starting peer node with chain-id: $CHAINID"
    
    # Make sure RPC endpoints are properly configured before starting
    echo "Ensuring Ethereum RPC and Tendermint RPC endpoints are exposed"
    echo "Config file path: $CONFIG"
    echo "App toml path: $APP_TOML"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i '' 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
        sed -i '' 's|enable = false|enable = true|g' "$APP_TOML"
        sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
    else
        sed -i 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
        sed -i 's|enable = false|enable = true|g' "$APP_TOML"
        sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
    fi
    
    # Verify the changes were applied
    echo "Verifying Tendermint RPC configuration:"
    grep "laddr = \"tcp:" "$CONFIG"
    echo "Verifying JSON-RPC configuration:"
    grep "enable = true" "$APP_TOML"
    
    echo "RPC endpoints available at:"
    echo "- Ethereum JSON-RPC: http://$(hostname -I | awk '{print $1}'):8545"
    echo "- Tendermint RPC: http://$(hostname -I | awk '{print $1}'):26657"
    
    # Final check for Tendermint RPC configuration
    echo "Final check for Tendermint RPC configuration"
    if grep -q 'laddr = "tcp://127.0.0.1:26657"' "$CONFIG"; then
        echo "WARNING: Tendermint RPC still bound to localhost, forcing update"
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        else
            sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        fi
        echo "Updated Tendermint RPC binding"
    else
        echo "SUCCESS: Tendermint RPC correctly configured to be exposed"
    fi
    
    $NXQD_BIN start \
        --metrics "$TRACE" \
        --log_level $LOGLEVEL \
        --minimum-gas-prices=0.0001nxq \
        --json-rpc.api eth,txpool,personal,net,debug,web3 \
        --home "$HOMEDIR" \
        --chain-id "$CHAINID"
    
fi