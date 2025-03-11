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
    echo "  init              Initialize the peer node, generate/recover keys, and download genesis"
    echo "  start             Start the peer node"
    echo
    echo "Environment variables:"
    echo "  KEYRING_PASSWORD  Set this to provide the keyring password (optional)"
    echo "  MONIKER           Set this to customize your node's name (default: NexQloudPeer)"
    echo "  SEED_NODE_IP      Set this to specify a custom seed node (default: stage-node.nexqloud.net)"
    echo
    echo "Examples:"
    echo "  $0 init                   # Initialize with interactive password entry"
    echo "  KEYRING_PASSWORD=xyz $0 init  # Initialize with password from environment"
    echo "  $0 start                  # Start the node"
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

# Secure function to generate a key and save mnemonic in a protected file
generate_key() {
    local key_name=$1
    local key_file="${KEYBACKUP_DIR}/${key_name}.info"
    
    print_info "Processing key: $key_name"
    
    # Check if key already exists in backup
    if [ -f "$key_file" ]; then
        print_info "Key $key_name already exists, using existing key"
        
        # Security check for file permissions
        local file_perms=$(stat -c "%a" "$key_file" 2>/dev/null || stat -f "%Lp" "$key_file" 2>/dev/null)
        if [[ "$file_perms" != "600" ]]; then
            print_warning "Key file permissions are not secure! Setting to 600."
            chmod 600 "$key_file"
        fi
        
        # Get the mnemonic and import it - using grep and cut for better compatibility
        local mnemonic=$(cat "$key_file" | grep "mnemonic:" | cut -d':' -f2- | xargs)
        echo "$mnemonic" | $NXQD_BIN keys add "$key_name" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
        return
    fi
    
    # Create secure directory if it doesn't exist
    mkdir -p "$KEYBACKUP_DIR"
    chmod 700 "$KEYBACKUP_DIR"
    
    print_info "Generating new key for $key_name"
    # Instructive message about password
    print_warning "You will be prompted to create a password for your keyring."
    print_warning "This password protects all your keys. Remember it well!"
    
    # Generate key with visible output for user
    $NXQD_BIN keys add "$key_name" --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR" | tee /tmp/key_output.tmp
    
    # Extract mnemonic and address from the output - using grep and sed for better compatibility
    local mnemonic=$(grep -A 1 "mnemonic" /tmp/key_output.tmp | tail -n 1 | xargs)
    local address=$(grep "address:" /tmp/key_output.tmp | cut -d':' -f2 | xargs)
    
    # Create the backup file with secure permissions
    echo "address: $address" > "$key_file"
    echo "mnemonic: $mnemonic" >> "$key_file"
    chmod 600 "$key_file"
    
    # Remove the temporary file securely
    rm -f /tmp/key_output.tmp
    
    print_success "Key $key_name generated and mnemonic saved to $key_file"
    print_warning "IMPORTANT: Securely back up $KEYBACKUP_DIR directory!"
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
    
    # Generate or load validator key
    print_section "Key Management"
    print_info "This process will create secure keys and store mnemonics in $KEYBACKUP_DIR"
    print_info "You will need to enter a password for the keyring"
    
    VAL_KEY="mykey"
    generate_key "$VAL_KEY"
    
    # Initialize the node
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
    
    SEED_NODE_ID_URL="http://$SEED_NODE_IP/node-id"
    print_info "Fetching seed node ID from: $SEED_NODE_ID_URL"
    
    SEED_NODE_ID=$(wget -qO- "$SEED_NODE_ID_URL" || curl -s "$SEED_NODE_ID_URL")
    if [ -z "$SEED_NODE_ID" ]; then
        error_exit "Failed to fetch seed node ID from $SEED_NODE_ID_URL"
    fi
    
    print_info "Seed Node ID: $SEED_NODE_ID"
    SEEDS="$SEED_NODE_ID@$SEED_NODE_IP:26656"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
    else
        sed -i "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
    fi
    
    # Verify seed configuration
    if ! grep -q "seeds = \"$SEEDS\"" "$CONFIG"; then
        error_exit "Failed to set seed node configuration"
    fi
    
    # Download genesis file
    print_section "Genesis Configuration"
    GENESIS_URL="http://$SEED_NODE_IP/genesis.json"
    print_info "Downloading genesis file from: $GENESIS_URL"
    
    if ! wget -qO "$GENESIS" "$GENESIS_URL" && ! curl -s "$GENESIS_URL" > "$GENESIS"; then
        error_exit "Failed to download genesis file from $GENESIS_URL"
    fi
    
    # Validate genesis
    print_info "Validating genesis file"
    if ! $NXQD_BIN validate-genesis --home "$HOMEDIR"; then
        error_exit "Genesis validation failed"
    fi
    
    print_section "Initialization Complete"
    print_success "Peer node has been successfully initialized!"
    print_warning "IMPORTANT: Make sure to securely back up your key mnemonics from: $KEYBACKUP_DIR"
    print_info "To start the node, run: $0 start"
}

# Start the blockchain node
start_node() {
    print_section "Starting Peer Node"
    print_info "Chain ID: $CHAINID"
    print_info "Home Dir: $HOMEDIR"
    print_info "Log Level: $LOGLEVEL"
    
    print_warning "You will need to enter your keyring password"
    
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