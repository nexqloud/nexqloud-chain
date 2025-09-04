#!/bin/bash

# ============================================================================
# NETWORK CONFIGURATION - Production Setup with Domain Names
# ============================================================================
SEED_NODE_1_DOMAIN="${SEED_NODE_1_DOMAIN:-prod-node.nexqloudsite.com}"      # 107.21.198.76
SEED_NODE_2_DOMAIN="${SEED_NODE_2_DOMAIN:-prod-node-1.nexqloudsite.com}"    # 54.161.133.227
PERSISTENT_PEER_DOMAIN="${PERSISTENT_PEER_DOMAIN:-prod-node-2.nexqloudsite.com}" # 98.86.120.142

# ============================================================================
# NODE CONFIGURATION
# ============================================================================
CHAINID="nxqd_6000-1"

# Set moniker if environment variable is not set
if [ -z "$MONIKER" ]; then
	MONIKER="NexQloudPeer"
fi

#for local testing
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

# used to exit on first error (any non-zero exit code)
set -e

# Colors for better UX
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Display an informative message with color
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Display error and exit
error_exit() {
    print_message "$RED" "ERROR: $1"
    exit 1
}

# Display a warning
print_warning() {
    print_message "$YELLOW" "WARNING: $1"
}

# Display info
print_info() {
    print_message "$BLUE" "INFO: $1"
}

# Display success
print_success() {
    print_message "$GREEN" "SUCCESS: $1"
}

if [[ $1 == "init" ]]; then

    # Remove the previous folder
    rm -rf "$HOMEDIR"

    # Set client config
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"

    VAL_KEY="mykey"
    VAL_MNEMONIC="copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"

    # Import keys from mnemonics
    echo "$VAL_MNEMONIC" | $NXQD_BIN keys add "$VAL_KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"

    # Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
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

    # Configure block time (1 block every 8 seconds)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i '' 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    else
        sed -i 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    fi

    # Configure RPC endpoints to bind to all interfaces for external access
    print_info "Configuring RPC endpoints for external access"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # Configure Tendermint RPC to bind to all interfaces
        sed -i '' 's|^laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        # Configure Ethereum JSON-RPC to bind to all interfaces
        sed -i '' 's|^address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
    else
        # Configure Tendermint RPC to bind to all interfaces
        sed -i 's|^laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        # Configure Ethereum JSON-RPC to bind to all interfaces
        sed -i 's|^address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
    fi

    # Configure multiple seed nodes for redundancy
    print_info "Configuring multiple seed nodes and persistent peers"
    
    # Using seed nodes and persistent peer from top-level configuration
    
    # Function to safely get node ID with fallback and timeout
    get_node_id() {
        local host=$1
        local node_id
        
        # Try wget first (available on CentOS), then curl as fallback
        if command -v wget >/dev/null 2>&1; then
            if node_id=$(timeout 10 wget -qO- "http://$host/node-id" 2>/dev/null); then
                echo "$node_id"
            else
                print_warning "Could not get node ID from $host (may not be running yet)"
                return 1
            fi
        elif command -v curl >/dev/null 2>&1; then
            if node_id=$(timeout 10 curl -s "http://$host/node-id" 2>/dev/null); then
                echo "$node_id"
            else
                print_warning "Could not get node ID from $host (may not be running yet)"
                return 1
            fi
        else
            print_warning "Neither wget nor curl found for getting node ID from $host"
            return 1
        fi
    }
    
    # Get node IDs from all available seed nodes
    SEED_NODE_1_ID=""
    SEED_NODE_2_ID=""
    PERSISTENT_PEER_ID=""
    
    if get_node_id "$SEED_NODE_1_DOMAIN" >/dev/null 2>&1; then
        SEED_NODE_1_ID=$(get_node_id "$SEED_NODE_1_DOMAIN")
    fi
    
    if get_node_id "$SEED_NODE_2_DOMAIN" >/dev/null 2>&1; then
        SEED_NODE_2_ID=$(get_node_id "$SEED_NODE_2_DOMAIN")
    fi
    
    if get_node_id "$PERSISTENT_PEER_DOMAIN" >/dev/null 2>&1; then
        PERSISTENT_PEER_ID=$(get_node_id "$PERSISTENT_PEER_DOMAIN")
    fi
    
    # Build seeds list (only include available nodes)
    SEEDS=""
    if [ -n "$SEED_NODE_1_ID" ]; then
        SEEDS="$SEED_NODE_1_ID@$SEED_NODE_1_DOMAIN:26656"
        print_success "Added seed node 1: $SEED_NODE_1_DOMAIN"
    fi
    
    if [ -n "$SEED_NODE_2_ID" ]; then
        if [ -n "$SEEDS" ]; then
            SEEDS="$SEEDS,$SEED_NODE_2_ID@$SEED_NODE_2_DOMAIN:26656"
        else
            SEEDS="$SEED_NODE_2_ID@$SEED_NODE_2_DOMAIN:26656"
        fi
        print_success "Added seed node 2: $SEED_NODE_2_DOMAIN"
    fi
    
    # Build persistent peers list
    PERSISTENT_PEERS=""
    if [ -n "$PERSISTENT_PEER_ID" ]; then
        PERSISTENT_PEERS="$PERSISTENT_PEER_ID@$PERSISTENT_PEER_DOMAIN:26656"
        print_success "Added persistent peer: $PERSISTENT_PEER_DOMAIN"
    fi
    
    # Validate we have at least one seed
    if [ -z "$SEEDS" ]; then
        error_exit "No seed nodes available! Check network connectivity."
    fi
    
    # Apply configuration
    print_info "Configuring P2P settings:"
    print_info "Seeds: $SEEDS"
    if [ -n "$PERSISTENT_PEERS" ]; then
        print_info "Persistent Peers: $PERSISTENT_PEERS"
    fi
    
    # Update config.toml with seeds and persistent peers
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^seeds = .*/seeds = \"$SEEDS\"/" "$CONFIG"
        if [ -n "$PERSISTENT_PEERS" ]; then
            sed -i '' "s/^persistent_peers = .*/persistent_peers = \"$PERSISTENT_PEERS\"/" "$CONFIG"
        fi
    else
        sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" "$CONFIG"
        if [ -n "$PERSISTENT_PEERS" ]; then
            sed -i "s/^persistent_peers = .*/persistent_peers = \"$PERSISTENT_PEERS\"/" "$CONFIG"
        fi
    fi

    # Download genesis file with fallback support
    print_info "Downloading genesis file with fallback support"
    GENESIS_DOWNLOADED=false
    
    # Try each seed node for genesis file
    for host in "$SEED_NODE_1_DOMAIN" "$SEED_NODE_2_DOMAIN"; do
        # Try wget first (available on CentOS), then curl as fallback
        if command -v wget >/dev/null 2>&1; then
            if wget -qO "$GENESIS" "http://$host/genesis.json" 2>/dev/null; then
                print_success "Genesis file downloaded from $host (using wget)"
                GENESIS_DOWNLOADED=true
                break
            else
                print_warning "Failed to download genesis from $host using wget"
            fi
        elif command -v curl >/dev/null 2>&1; then
            if curl -s -o "$GENESIS" "http://$host/genesis.json" 2>/dev/null; then
                print_success "Genesis file downloaded from $host (using curl)"
                GENESIS_DOWNLOADED=true
                break
            else
                print_warning "Failed to download genesis from $host using curl"
            fi
        else
            print_warning "Neither wget nor curl available for downloading from $host"
        fi
    done
    
    if [ "$GENESIS_DOWNLOADED" = false ]; then
        error_exit "Could not download genesis file from any seed node!"
    fi

    $NXQD_BIN validate-genesis --home "$HOMEDIR"

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