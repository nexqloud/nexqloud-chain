#!/bin/bash
set -e

# Second Seed Node Configuration
CHAINID="nxqd_6000-1"
MONIKER="NexqloudSeedNode2"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
HOMEDIR="$HOME/.nxqd"

#for local testing
# NXQD_BIN="$(pwd)/cmd/nxqd/nxqd"

#for remote testing
NXQD_BIN="/usr/local/bin/nxqd"

BASEFEE=1000000000
# to trace evm
TRACE=""

# Security configuration - use file-based keyring for better security
KEYRING="file"

# First seed node IP (where to get genesis from)
FIRST_SEED_IP="${FIRST_SEED_IP:-98.81.138.222}"

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
    echo "  init              Initialize the second seed node and sync with first seed"
    echo "  start             Start the second seed node"
    echo
    echo "Environment variables:"
    echo "  KEYRING_PASSWORD  Set this to provide the keyring password (optional)"
    echo "  FIRST_SEED_IP     IP of the first seed node (default: 98.81.138.222)"
    echo
    echo "Examples:"
    echo "  $0 init                          # Initialize second seed node"
    echo "  FIRST_SEED_IP=custom-ip $0 init  # Initialize with custom first seed IP"
    echo "  $0 start                         # Start the second seed node"
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
    if [ ! -f "$NXQD_BIN" ]; then
        error_exit "nxqd binary not found at $NXQD_BIN\nBuild it with: cd cmd/nxqd && go build"
    fi
    
    print_success "All dependencies found"
}

# Simplified function to generate a key
generate_key() {
    local key_name=$1
    
    print_info "Processing key: $key_name"
    
    print_info "Generating key for $key_name"
    
    # Instructive message about password
    print_warning "You will be prompted to create a password for your keyring."
    print_warning "This password protects all your keys. Remember it well!"
    
    # Generate key
    $NXQD_BIN keys add "$key_name" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
    
    print_success "Key $key_name generated"
    print_warning "IMPORTANT: Make sure to securely write down the mnemonic phrase shown above!"
}

# Initialize the second seed node
initialize_second_seed() {
    print_section "Initializing Second Seed Node"
    
    # Remove previous data to start fresh
    if [ -d "$HOMEDIR" ]; then
        print_warning "Removing existing data directory: $HOMEDIR"
        rm -rf "$HOMEDIR"
    fi
    
    # Set client config
    print_info "Configuring client"
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Generate or load keys
    print_section "Key Management"
    print_info "This process will create keys for the second seed node"
    print_info "You will need to enter a password for the keyring"
    print_warning "IMPORTANT: Make sure to securely write down all mnemonic phrases when displayed!"

    # Generate a single primary key for this seed node
    print_info "Processing key: primary"
    generate_key "primary"
    
    # Initialize the chain
    print_section "Chain Initialization"
    print_info "Initializing second seed node with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Download genesis from first seed node
    print_section "Downloading Genesis from First Seed"
    print_info "Downloading genesis file from first seed node: $FIRST_SEED_IP"
    
    if wget -qO "$GENESIS" "http://$FIRST_SEED_IP/genesis.json"; then
        print_success "Downloaded genesis file from first seed node: $FIRST_SEED_IP"
    else
        error_exit "Failed to download genesis file from first seed node: $FIRST_SEED_IP. Make sure the first seed node is running and accessible."
    fi
    
    # Validate the downloaded genesis
    print_info "Validating downloaded genesis file"
    if ! $NXQD_BIN validate-genesis --home "$HOMEDIR"; then
        error_exit "Downloaded genesis file is invalid"
    fi
    
    print_success "Genesis file validated successfully"
    
    # Configure block time (1 block every 8 seconds)
    print_info "Setting block time to 8 seconds (0.125 blocks per second)"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i '' 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    else
        sed -i 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    fi
    
    # Configure second seed node network connections
    print_info "Configuring second seed node network connections"
    
    # Define other seed nodes and persistent peer
    OTHER_SEED_NODES="${OTHER_SEED_NODES:-$FIRST_SEED_IP}"
    PERSISTENT_PEER_IP="${PERSISTENT_PEER_IP:-96.30.197.66}"
    
    # Function to safely get node ID
    get_node_id() {
        local ip=$1
        local node_id
        if node_id=$(wget -qO- "http://$ip/node-id" 2>/dev/null); then
            echo "$node_id"
        else
            print_warning "Could not get node ID from $ip (may not be running yet)"
            return 1
        fi
    }
    
    # Build persistent peers list for seed nodes
    PERSISTENT_PEERS=""
    
    # Add first seed node as persistent peer
    if get_node_id "$FIRST_SEED_IP" >/dev/null 2>&1; then
        FIRST_SEED_ID=$(get_node_id "$FIRST_SEED_IP")
        if [ -n "$FIRST_SEED_ID" ]; then
            PERSISTENT_PEERS="$FIRST_SEED_ID@$FIRST_SEED_IP:26656"
            print_success "Added first seed node as peer: $FIRST_SEED_IP"
        fi
    fi
    
    # Add dedicated persistent peer
    if get_node_id "$PERSISTENT_PEER_IP" >/dev/null 2>&1; then
        PERSISTENT_PEER_ID=$(get_node_id "$PERSISTENT_PEER_IP")
        if [ -n "$PERSISTENT_PEER_ID" ]; then
            if [ -z "$PERSISTENT_PEERS" ]; then
                PERSISTENT_PEERS="$PERSISTENT_PEER_ID@$PERSISTENT_PEER_IP:26656"
            else
                PERSISTENT_PEERS="$PERSISTENT_PEERS,$PERSISTENT_PEER_ID@$PERSISTENT_PEER_IP:26656"
            fi
            print_success "Added persistent peer: $PERSISTENT_PEER_IP"
        fi
    fi
    
    # Apply persistent peers configuration
    if [ -n "$PERSISTENT_PEERS" ]; then
        print_info "Configuring persistent peers: $PERSISTENT_PEERS"
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/persistent_peers =.*/persistent_peers = \"$PERSISTENT_PEERS\"/g" "$CONFIG"
        else
            sed -i "s/persistent_peers =.*/persistent_peers = \"$PERSISTENT_PEERS\"/g" "$CONFIG"
        fi
    else
        print_warning "No persistent peers configured (other nodes may not be running yet)"
    fi
    
    # Prometheus
    print_info "Enabling Prometheus metrics and APIs"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
        sed -i '' 's/prometheus-retention-time = 0/prometheus-retention-time = 1000/' "$APP_TOML"
        sed -i '' 's/enabled = false/enabled = true/' "$APP_TOML"
    else
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
        sed -i 's/prometheus-retention-time = 0/prometheus-retention-time = 1000/' "$APP_TOML"
        sed -i 's/enabled = false/enabled = true/' "$APP_TOML"
    fi
    
    # Set up node ID for sharing
    print_info "Setting up node ID"
    $NXQD_BIN tendermint show-node-id --home "$HOMEDIR" > "$HOMEDIR/node-id"
    
    # Copy files for sharing (if nginx is available)
    if [ -d "/usr/share/nginx/html" ]; then
        print_info "Copying genesis and node-id for sharing via nginx"
	sudo cp $GENESIS /usr/share/nginx/html/
	sudo cp "$HOMEDIR/node-id" /usr/share/nginx/html/node-id
    else
        print_warning "Nginx directory not found, skipping file copying"
    fi
    
    # Configure RPC access
    print_info "Enabling RPC services"
    print_info "Config file path: $CONFIG"
    print_info "App toml path: $APP_TOML"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # Update Tendermint RPC binding
        sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        # Update JSON-RPC settings
        sed -i '' '/^\[json-rpc\]/,/^\[/ s|enable = false|enable = true|' "$APP_TOML"
        sed -i '' 's|address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
        sed -i '' 's|ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|g' "$APP_TOML"
    else
        # Update Tendermint RPC binding
        sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        # Update JSON-RPC settings
        sed -i '/^\[json-rpc\]/,/^\[/ s|enable = false|enable = true|' "$APP_TOML"
        sed -i 's|address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
        sed -i 's|ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|g' "$APP_TOML"
    fi
    
    # Verify the changes were applied
    print_info "Verifying RPC configuration:"
    grep "laddr = \"tcp:" "$CONFIG"
    
    print_section "Initialization Complete"
    print_success "Second seed node has been successfully initialized!"
    print_info "Genesis file downloaded from: $FIRST_SEED_IP"
    print_info "To start the node, run: $0 start"
}

# Start the blockchain node
start_node() {
    print_section "Starting Second Seed Node"
    print_info "Chain ID: $CHAINID"
    print_info "Home Dir: $HOMEDIR"
    print_info "Log Level: $LOGLEVEL"
    
    print_warning "You will need to enter your keyring password"
    
    # Ensure RPC endpoints are exposed before starting
    print_info "Ensuring RPC endpoints are properly configured"
    
    print_info "RPC endpoints will be available at:"
    print_info "- Ethereum JSON-RPC: http://$(hostname -I | awk '{print $1}'):8545"
    print_info "- Tendermint RPC: http://$(hostname -I | awk '{print $1}'):26657"
    
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
            initialize_second_seed "$2"
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
