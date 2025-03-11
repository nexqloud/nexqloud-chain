#!/bin/bash
set -e

# Node configuration
CHAINID="nxqd_6000-1"
MONIKER="${MONIKER:-NexQloudPeer}"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
HOMEDIR="$HOME/.nxqd"
KEYBACKUP_DIR="$HOME/.nxqd_keys_backup"

# Seed node settings
SEED_NODE_IP="${SEED_NODE_IP:-stage-node.nexqloud.net}"

# Binary path
NXQD_BIN="$(which nxqd)"

# Base fee
BASEFEE=1000000000

# to trace evm
TRACE=""

# Security configuration - use file-based keyring for better security
KEYRING="file"

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

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

# Print a section header
print_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Usage instructions
usage() {
    echo "Usage: $0 [init|start]"
    echo
    echo "Commands:"
    echo "  init              Initialize the peer node and download genesis"
    echo "  start             Start the peer node"
    echo
    echo "Environment variables:"
    echo "  MONIKER           Set this to customize your node's name (default: NexQloudPeer)"
    echo "  SEED_NODE_IP      Set this to specify a custom seed node (default: stage-node.nexqloud.net)"
    echo
    echo "Examples:"
    echo "  $0 init           # Initialize peer node"
    echo "  $0 start          # Start the peer node"
    exit 1
}

# Check dependencies
check_dependencies() {
    print_section "Checking Dependencies"
    
    # Check jq
    command -v jq >/dev/null 2>&1 || {
        error_exit "jq not installed. More info: https://stedolan.github.io/jq/download/"
    }
    
    # Check if nxqd binary exists
    if [ -z "$NXQD_BIN" ]; then
        error_exit "nxqd binary not found in PATH. Please install it or add it to your PATH."
    fi
    
    print_success "All dependencies found"
}

# Initialize the peer node
initialize_peer_node() {
    print_section "Initializing Peer Node"
    
    # Remove previous data to start fresh
    if [ -d "$HOMEDIR" ]; then
        print_warning "Removing existing data directory: $HOMEDIR"
        rm -rf "$HOMEDIR"
    fi
    
    # Set client config
    print_info "Configuring client"
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Initialize the node (as non-validator)
    print_section "Node Initialization"
    print_info "Initializing node with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Configure timeout settings if needed
    if [[ $1 == "pending" ]]; then
        print_info "Setting up for pending mode"
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
    
    # Enable prometheus and API services
    print_info "Enabling Prometheus metrics and APIs"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
    else
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
    fi
    
    # Configure RPC access
    print_info "Enabling RPC services"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i '' 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
    else
        sed -i 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
    fi
    
    # Set seed node information
    print_section "Seed Node Configuration"
    print_info "Configuring seed node: $SEED_NODE_IP"
    
    print_info "Fetching seed node ID from: http://$SEED_NODE_IP/node-id"
    SEED_NODE_ID=$(wget -qO- "http://$SEED_NODE_IP/node-id" || curl -s "http://$SEED_NODE_IP/node-id")
    if [ -z "$SEED_NODE_ID" ]; then
        print_warning "Failed to fetch seed node ID, trying to proceed anyway"
        SEED_NODE_ID="UNKNOWN_ID"
    else
        print_info "Seed Node ID: $SEED_NODE_ID"
    fi
    
    SEEDS="$SEED_NODE_ID@$SEED_NODE_IP:26656"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
    else
        sed -i "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
    fi
    
    # Download genesis file
    print_section "Genesis Configuration"
    print_info "Downloading genesis file from: http://$SEED_NODE_IP/genesis.json"
    
    wget -qO- "http://$SEED_NODE_IP/genesis.json" > "$GENESIS" || curl -s "http://$SEED_NODE_IP/genesis.json" > "$GENESIS"
    
    print_info "Validating genesis"
    $NXQD_BIN validate-genesis --home "$HOMEDIR" || print_warning "Genesis validation had issues but proceeding anyway"
    
    # Set up NFT validation bypass for peer nodes
    print_section "Setting Up NFT Validation Configuration"
    
    local nft_config="$HOMEDIR/config/nft_allowlist.json"
    cat > "$nft_config" << EOF
{
  "approved_validators": [],
  "nft_contract_address": "0x816644F8bc4633D268842628EB10ffC0AdcB6099",
  "bypass_validation": true
}
EOF
    
    print_success "Created NFT validation config at $nft_config"
    
    # Update app.toml to point to this file
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/nft_config = ""/nft_config = "config\/nft_allowlist.json"/g' "$APP_TOML" 2>/dev/null || \
        echo 'nft_config = "config/nft_allowlist.json"' >> "$APP_TOML"
    else
        sed -i 's/nft_config = ""/nft_config = "config\/nft_allowlist.json"/g' "$APP_TOML" 2>/dev/null || \
        echo 'nft_config = "config/nft_allowlist.json"' >> "$APP_TOML"
    fi
    
    print_section "Initialization Complete"
    print_success "Peer node has been successfully initialized!"
    print_info "To start the node, run: $0 start"
}

# Start the blockchain node
start_node() {
    print_section "Starting Peer Node"
    print_info "Chain ID: $CHAINID"
    print_info "Home Dir: $HOMEDIR"
    print_info "Log Level: $LOGLEVEL"
    
    $NXQD_BIN start \
        --metrics "$TRACE" \
        --log_level $LOGLEVEL \
        --minimum-gas-prices=0.0001nxq \
        --json-rpc.api eth,txpool,personal,net,debug,web3 \
        --home "$HOMEDIR" \
        --chain-id "$CHAINID"
}

# Main function
main() {
    if [ $# -eq 0 ]; then
        usage
    fi
    
    # Process command
    case "$1" in
        init)
            check_dependencies
            initialize_peer_node "$2"
            ;;
        start)
            check_dependencies
            start_node
            ;;
        *)
            usage
            ;;
    esac
}

# Execute main function
main "$@"