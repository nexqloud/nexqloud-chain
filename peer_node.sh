#!/bin/bash

CHAINID="nxqd_6000-1"

# Set moniker if environment variable is not set
if [ -z "$MONIKER" ]; then
	MONIKER="NexQloudPeer"
fi

#dev
if [ -z "$SEED_NODE_IP" ]; then
    SEED_NODE_IP="13.203.229.219"
fi

# #staging
# if [ -z "$SEED_NODE_IP" ]; then
#     SEED_NODE_IP="stage-node.nexqloud.net"
# fi

# Set the path to the nxqd binary
# NXQD_BIN="$(pwd)/cmd/nxqd/nxqd"

#for remote testing
NXQD_BIN="/usr/local/bin/nxqd"

KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the nxqd instance
HOMEDIR="$HOME/.nxqd"
# to trace evm
#TRACE="--trace"
TRACE=""

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

# Function to download file based on OS
download_file() {
    local url=$1
    local output=$2
    if [[ "$OSTYPE" == "darwin"* ]]; then
        curl -s "$url" > "$output"
    else
        wget -qO- "$url" > "$output"
    fi
    return $?
}

# used to exit on first error (any non-zero exit code)
set -e

# Initialize the node
init() {
    echo "Initializing node..."
    
    # Remove the previous folder
    rm -rf "$HOMEDIR"
    
    # Set client config
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"
    $NXQD_BIN config node tcp://$SEED_NODE_IP:26657 --home "$HOMEDIR"

    # Generate a new key
    echo "$VAL_MNEMONIC" | $NXQD_BIN keys add "$MONIKER" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
    VAL_ADDRESS=$($NXQD_BIN keys show "$MONIKER" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
    echo "Validator address: $VAL_ADDRESS"

    # Initialize the node
    $NXQD_BIN init "$MONIKER" -o --chain-id "$CHAINID" --home "$HOMEDIR"

    # Configure timeouts and settings based on OS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # Create temporary files for macOS sed
        TMP_CONFIG=$(mktemp)
        TMP_APP_TOML=$(mktemp)
        
        # Modify config.toml
        sed 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        sed 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        sed 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        sed 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        sed 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        sed 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        sed 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        sed 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        
        # Enable prometheus metrics
        sed 's/prometheus = false/prometheus = true/' "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        
        # Enable RPC
        sed '/^\[json-rpc\]/,/^\[/ s|enable = false|enable = true|' "$APP_TOML" > "$TMP_APP_TOML" && mv "$TMP_APP_TOML" "$APP_TOML"
        sed 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML" > "$TMP_APP_TOML" && mv "$TMP_APP_TOML" "$APP_TOML"
        sed 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML" > "$TMP_APP_TOML" && mv "$TMP_APP_TOML" "$APP_TOML"
        
        # Clean up temporary files
        rm -f "$TMP_CONFIG" "$TMP_APP_TOML"
    else
        # Linux sed commands
        sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
        sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
        sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
        sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
        sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
        sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
        sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
        sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
        
        # Enable prometheus metrics
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
        
        # Enable RPC
        sed -i '/^\[json-rpc\]/,/^\[/ s|enable = false|enable = true|' "$APP_TOML"
        sed -i 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
    fi

    # Download genesis file from seed node
    echo "Downloading genesis file from seed node..."
    if ! download_file "http://$SEED_NODE_IP:26657/genesis" "$GENESIS"; then
        echo "Failed to download genesis file from seed node"
        exit 1
    fi

    # Extract the genesis data from the JSON-RPC response
    jq '.result.genesis' "$GENESIS" > "$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

    # Verify the genesis file has gentxs
    if ! jq -e '.app_state.genutil.gen_txs | length > 0' "$GENESIS" > /dev/null; then
        echo "Warning: Genesis file does not contain any gentxs"
    fi

    # Verify the genesis file has the correct bond denomination
    if ! jq -e '.app_state.staking.params.bond_denom == "unxq"' "$GENESIS" > /dev/null; then
        echo "Error: Genesis file has incorrect bond denomination"
        exit 1
    fi

    # Validate the genesis file
    echo "Validating genesis file..."
    $NXQD_BIN validate-genesis --home "$HOMEDIR"

    # Set up node ID for sharing
    echo "Setting up node ID..."
    $NXQD_BIN tendermint show-node-id --home "$HOMEDIR" > "$HOMEDIR/node-id"

    # Set seed node info
    echo "Getting seed node ID..."
    SEED_NODE_ID=$(curl -s http://$SEED_NODE_IP:26657/status | jq -r '.result.node_info.id')
    if [ -z "$SEED_NODE_ID" ]; then
        echo "Failed to get seed node ID"
        exit 1
    fi
    echo "Using seed node ID: $SEED_NODE_ID"
    SEEDS="$SEED_NODE_ID@$SEED_NODE_IP:26656"

    # Update seeds in config
    if [[ "$OSTYPE" == "darwin"* ]]; then
        TMP_CONFIG=$(mktemp)
        sed "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG" > "$TMP_CONFIG" && mv "$TMP_CONFIG" "$CONFIG"
        rm -f "$TMP_CONFIG"
    else
        sed -i "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
    fi

    echo "Node initialized successfully!"
}

if [[ $1 == "init" ]]; then
    init
else
    # Start the node
    $NXQD_BIN start \
        --metrics "$TRACE" \
        --log_level $LOGLEVEL \
        --minimum-gas-prices=0.0001nxq \
        --json-rpc.api eth,txpool,personal,net,debug,web3 \
        --home "$HOMEDIR" \
        --chain-id "$CHAINID"
fi
