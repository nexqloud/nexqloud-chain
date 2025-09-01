#!/bin/bash
set -e

# ============================================================================
# NETWORK CONFIGURATION - Update these IPs as needed
# ============================================================================
FIRST_SEED_IP="${FIRST_SEED_IP:-98.81.138.222}"
SECOND_SEED_IP="${SECOND_SEED_IP:-98.81.87.61}"
PERSISTENT_PEER_IP="${PERSISTENT_PEER_IP:-155.138.192.236}"

# ============================================================================
# NODE CONFIGURATION
# ============================================================================
CHAINID="nxqd_6000-1"
MONIKER="NexqloudSeedNode1"
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
    echo "Usage: $0 [init|start|test]"
    echo
    echo "Commands:"
    echo "  init              Initialize the node, generate keys, and create genesis"
    echo "                    (Remember to manually comment/uncomment NFT validation in code)"
    echo "  start             Start the node"
    echo "  test              Initialize and start in test mode (bypasses NFT validation)"
    echo
    echo "Environment variables for non-interactive mode:"
    echo "  KEYRING_PASSWORD  Set this to provide the keyring password (optional)"
    echo "                    If not set, you will be prompted interactively"
    echo
    echo "Examples:"
    echo "  $0 init                   # Initialize with interactive password entry"
    echo "  KEYRING_PASSWORD=xyz $0 init  # Initialize with password from environment"
    echo "  $0 start                  # Start the node"
    echo "  $0 test                   # Start in test mode (bypasses NFT validation)"
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

# Initialize the blockchain
initialize_blockchain() {
    print_section "Initializing Blockchain"
    
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
    print_info "This process will create keys for the blockchain"
    print_info "You will need to enter a password for the keyring"
    print_warning "IMPORTANT: Make sure to securely write down all mnemonic phrases when displayed!"

    # Generate a single primary key
    print_info "Processing key: primary"
    generate_key "primary"
    
    # Initialize the chain
    print_section "Chain Initialization"
    print_info "Initializing chain with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Customize genesis settings
    print_info "Customizing genesis parameters"
    print_info "Disabling NFT validation for local testing"

	# Change parameter token denominations to nxq
	jq '.app_state["staking"]["params"]["bond_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["inflation"]["params"]["mint_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
    jq '.app_state["inflation"]["params"]["enable_inflation"]=false' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set base fee in genesis
	jq '.app_state["feemarket"]["params"]["base_fee"]="'${BASEFEE}'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

    # Configure block time (1 block every 8 seconds)
    print_info "Setting block time to 8 seconds (0.125 blocks per second)"
		if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i '' 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    else
        sed -i 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    fi
    
    # Configure seed node network connections
    print_info "Configuring seed node network connections for redundancy"
    
    # Define other seed nodes (exclude current node based on environment or hostname)
    OTHER_SEED_NODES="${OTHER_SEED_NODES:-}"
    # PERSISTENT_PEER_IP is defined at top of script
    
    # If environment variable is not set, auto-detect based on common IPs
    if [ -z "$OTHER_SEED_NODES" ]; then
        # Get current external IP to exclude self
        CURRENT_IP=$(wget -qO- ipinfo.io/ip 2>/dev/null || echo "unknown")
        
        # Define all known seed nodes
        ALL_SEED_IPS="98.81.138.222 98.81.87.61"
        
        # Build list excluding current IP
        for ip in $ALL_SEED_IPS; do
            if [ "$ip" != "$CURRENT_IP" ]; then
                if [ -z "$OTHER_SEED_NODES" ]; then
                    OTHER_SEED_NODES="$ip"
                else
                    OTHER_SEED_NODES="$OTHER_SEED_NODES $ip"
                fi
            fi
        done
    fi
    
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
    
    # Add other seed nodes as persistent peers
    for ip in $OTHER_SEED_NODES; do
        if get_node_id "$ip" >/dev/null 2>&1; then
            NODE_ID=$(get_node_id "$ip")
            if [ -n "$NODE_ID" ]; then
                if [ -z "$PERSISTENT_PEERS" ]; then
                    PERSISTENT_PEERS="$NODE_ID@$ip:26656"
                else
                    PERSISTENT_PEERS="$PERSISTENT_PEERS,$NODE_ID@$ip:26656"
                fi
                print_success "Added seed node peer: $ip"
            fi
        fi
    done
    
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
    
    # Change proposal periods
    print_info "Setting up shortened proposal periods for testing"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/"voting_period": "172800s"/"voting_period": "60s"/g' "$GENESIS"
        sed -i '' 's/"max_deposit_period": "172800s"/"max_deposit_period": "60s"/g' "$GENESIS"
    else
        sed -i 's/"voting_period": "172800s"/"voting_period": "60s"/g' "$GENESIS"
        sed -i 's/"max_deposit_period": "172800s"/"max_deposit_period": "60s"/g' "$GENESIS"
    fi
    
    print_section "Setting Up Genesis Accounts"
    # Add the primary key as a genesis account with the full token supply
    print_info "Adding genesis account with all tokens"
    local address=$($NXQD_BIN keys show "primary" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
    $NXQD_BIN add-genesis-account "$address" "21000000000000000000000000unxq" --keyring-backend "$KEYRING" --home "$HOMEDIR"
    print_success "Added genesis account primary with balance 21000000000000000000000000unxq"
    
    # Create genesis transaction with validator key
    print_info "Creating genesis transaction with validator key (primary)"
    $NXQD_BIN gentx "primary" 100000000000000000000unxq --gas-prices ${BASEFEE}unxq --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Collect and validate genesis transactions
    print_info "Collecting genesis transactions"
    $NXQD_BIN collect-gentxs --home "$HOMEDIR"
    
    print_info "Validating genesis"
    $NXQD_BIN validate-genesis --home "$HOMEDIR"
    
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
    
    # Check file permissions
    print_info "Checking file permissions:"
    ls -la "$CONFIG"
    ls -la "$APP_TOML"
    
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
    print_info "Verifying Tendermint RPC configuration:"
    grep "laddr = \"tcp:" "$CONFIG"
    print_info "Verifying JSON-RPC configuration:"
    grep -A 3 "\[json-rpc\]" "$APP_TOML"
    
    # Configure Tendermint RPC access (needed for validator creation)
    print_info "Enabling Tendermint RPC services"
    print_info "Config file path: $CONFIG"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
    else
        sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
    fi
    
    # Verify the change was applied
    print_info "Verifying Tendermint RPC configuration:"
    grep "laddr = \"tcp:" "$CONFIG"
    
    print_section "Initialization Complete"
    print_success "Node has been successfully initialized!"
    print_info "To start the node, run: $0 start"
    
   
    # Final check to ensure Tendermint RPC is exposed
    print_info "Final check for Tendermint RPC configuration"
    if grep -q 'laddr = "tcp://127.0.0.1:26657"' "$CONFIG"; then
        print_warning "Tendermint RPC still bound to localhost, forcing update"
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        else
            sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        fi
        print_info "Updated Tendermint RPC binding"
    else
        print_success "Tendermint RPC correctly configured to be exposed"
    fi
}

# Start the blockchain node
start_node() {
    print_section "Starting Blockchain Node"
    print_info "Chain ID: $CHAINID"
    print_info "Home Dir: $HOMEDIR"
    print_info "Log Level: $LOGLEVEL"
    
    print_warning "You will need to enter your keyring password"
    
    # Ensure RPC endpoints are exposed before starting
    print_info "Ensuring Ethereum RPC and Tendermint RPC endpoints are exposed"
    print_info "Config file path: $CONFIG"
    print_info "App toml path: $APP_TOML"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        sed -i '' 's|address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
        sed -i '' 's|ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|g' "$APP_TOML"
        sed -i '' 's|enable = false|enable = true|g' "$APP_TOML"
    else
        sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        sed -i 's|address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
        sed -i 's|ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|g' "$APP_TOML"
        sed -i 's|enable = false|enable = true|g' "$APP_TOML"
    fi
    
    # Verify the changes were applied
    print_info "Verifying Tendermint RPC configuration:"
    grep "laddr = \"tcp:" "$CONFIG"
    print_info "Verifying JSON-RPC configuration:"
    grep "enable = true" "$APP_TOML"
    
    print_info "RPC endpoints available at:"
    print_info "- Ethereum JSON-RPC: http://$(hostname -I | awk '{print $1}'):8545"
    print_info "- Tendermint RPC: http://$(hostname -I | awk '{print $1}'):26657"
    
    # Final check to ensure Tendermint RPC is exposed
    print_info "Final check for Tendermint RPC configuration"
    if grep -q 'laddr = "tcp://127.0.0.1:26657"' "$CONFIG"; then
        print_warning "Tendermint RPC still bound to localhost, forcing update"
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        else
            sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        fi
        print_info "Updated Tendermint RPC binding"
    else
        print_success "Tendermint RPC correctly configured to be exposed"
    fi
    
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
            initialize_blockchain "$2"
            ;;
        start)
            check_dependencies
            start_node
            ;;
        test)
            # Test mode for developers - combine init and start with bypass flags
            print_section "Running in Test Mode - NFT Validation Bypassed"
            export BYPASS_NFT_VALIDATION=true
            check_dependencies
            initialize_blockchain
            
            # Additional test mode configuration
            print_info "Applying additional test configuration"
            echo 'NFT_VALIDATION_BYPASS=true' >> "$HOMEDIR/.env"
            
            # Start the node
            print_info "Starting node in test mode"
            $NXQD_BIN start \
                --metrics "$TRACE" \
                --log_level $LOGLEVEL \
                --minimum-gas-prices=0.0001nxq \
                --json-rpc.api eth,txpool,personal,net,debug,web3 \
                --home "$HOMEDIR" \
                --chain-id "$CHAINID"
            ;;
        *)
            usage
            ;;
    esac
}

# Execute main function
main "$@"
