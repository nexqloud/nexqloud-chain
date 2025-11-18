#!/bin/bash
set -e

# ============================================================================
# NETWORK CONFIGURATION - Production Setup with Domain Names
# ============================================================================
FIRST_SEED_DOMAIN="${FIRST_SEED_DOMAIN:-prod-node.nexqloudsite.com}"      # 107.21.198.76
SECOND_SEED_DOMAIN="${SECOND_SEED_DOMAIN:-prod-node-1.nexqloudsite.com}"  # 54.161.133.227
PERSISTENT_PEER_DOMAIN="${PERSISTENT_PEER_DOMAIN:-prod-node-2.nexqloudsite.com}" # 98.86.120.142

# ============================================================================
# NODE CONFIGURATION
# ============================================================================
CHAINID="nxqd_90009-1"
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

# Genesis account balance configuration (in unxq, where 1 NXQ = 10^18 unxq)
# will change on the day of mainnet release with the actual amount of tokens in the snapshot
GENESIS_ACCOUNT_BALANCE="${GENESIS_ACCOUNT_BALANCE:-2100000000000000000000000unxq}"

# Validator stake configuration (in unxq, where 1 NXQ = 10^18 unxq)
# Default: 100 NXQ = 25000000000000000000 unxq
VALIDATOR_STAKE="${VALIDATOR_STAKE:-25000000000000000000unxq}"

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
    
    print_info "Generating new key for $key_name"
    
    # Instructive message about password
    print_warning "You will be prompted to create a password for your keyring."
    print_warning "This password protects all your keys. Remember it well!"
    
    # Generate new key (without --recover flag)
    $NXQD_BIN keys add "$key_name" --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
    
    print_success "Key $key_name generated"
    print_warning "IMPORTANT: Make sure to securely write down the mnemonic phrase shown above!"
    print_warning "The mnemonic phrase is required to recover this key if you lose access!"
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
    jq '.app_state["inflation"]["params"]["enable_inflation"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

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
        
        # Define all known seed nodes (using domain names)
        ALL_SEED_DOMAINS="prod-node.nexqloudsite.com prod-node-1.nexqloudsite.com"
        
        # Build list excluding current domain (resolve to IP for comparison)
        for domain in $ALL_SEED_DOMAINS; do
            # Resolve domain to IP for comparison
            DOMAIN_IP=$(nslookup "$domain" 2>/dev/null | grep -A1 "Name:" | tail -n1 | awk '{print $2}' || echo "unknown")
            if [ "$DOMAIN_IP" != "$CURRENT_IP" ]; then
                if [ -z "$OTHER_SEED_NODES" ]; then
                    OTHER_SEED_NODES="$domain"
                else
                    OTHER_SEED_NODES="$OTHER_SEED_NODES $domain"
                fi
            fi
        done
    fi
    
    # Function to safely get node ID
    get_node_id() {
        local host=$1
        local node_id
        if node_id=$(timeout 5 wget -qO- "http://$host/node-id" 2>/dev/null); then
            echo "$node_id"
        else
            print_warning "Could not get node ID from $host (may not be running yet)"
            return 1
        fi
    }
    
    # Build persistent peers list for seed nodes
    PERSISTENT_PEERS=""
    
    # Add other seed nodes as persistent peers
    for host in $OTHER_SEED_NODES; do
        if get_node_id "$host" >/dev/null 2>&1; then
            NODE_ID=$(get_node_id "$host")
            if [ -n "$NODE_ID" ]; then
                if [ -z "$PERSISTENT_PEERS" ]; then
                    PERSISTENT_PEERS="$NODE_ID@$host:26656"
                else
                    PERSISTENT_PEERS="$PERSISTENT_PEERS,$NODE_ID@$host:26656"
                fi
                print_success "Added seed node peer: $host"
            fi
        fi
    done
    
    # Add dedicated persistent peer
    if get_node_id "$PERSISTENT_PEER_DOMAIN" >/dev/null 2>&1; then
        PERSISTENT_PEER_ID=$(get_node_id "$PERSISTENT_PEER_DOMAIN")
        if [ -n "$PERSISTENT_PEER_ID" ]; then
            if [ -z "$PERSISTENT_PEERS" ]; then
                PERSISTENT_PEERS="$PERSISTENT_PEER_ID@$PERSISTENT_PEER_DOMAIN:26656"
            else
                PERSISTENT_PEERS="$PERSISTENT_PEERS,$PERSISTENT_PEER_ID@$PERSISTENT_PEER_DOMAIN:26656"
            fi
            print_success "Added persistent peer: $PERSISTENT_PEER_DOMAIN"
        fi
    fi
    
    # Apply persistent peers configuration
    if [ -n "$PERSISTENT_PEERS" ]; then
        print_info "Configuring persistent peers: $PERSISTENT_PEERS"
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/^persistent_peers = .*/persistent_peers = \"$PERSISTENT_PEERS\"/" "$CONFIG"
            sed -i '' "s/^seeds = .*/seeds = \"\"/" "$CONFIG"
        else
            sed -i "s/^persistent_peers = .*/persistent_peers = \"$PERSISTENT_PEERS\"/" "$CONFIG"
            sed -i "s/^seeds = .*/seeds = \"\"/" "$CONFIG"
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
    print_info "Setting up proposal periods (5 minutes for voting)"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/"voting_period": "172800s"/"voting_period": "300s"/g' "$GENESIS"
        sed -i '' 's/"max_deposit_period": "172800s"/"max_deposit_period": "86400s"/g' "$GENESIS"

        sed -i '' 's/"veto_threshold": "0.334000000000000000"/"veto_threshold": "1.000000000000000000"/g' "$GENESIS"
    else
        sed -i 's/"voting_period": "172800s"/"voting_period": "86400s"/g' "$GENESIS"
        sed -i 's/"max_deposit_period": "172800s"/"max_deposit_period": "86400s"/g' "$GENESIS"

        sed -i 's/"veto_threshold": "0.334000000000000000"/"veto_threshold": "1.000000000000000000"/g' "$GENESIS"
    fi
    
    print_section "Setting Up Genesis Accounts"
    # Add the primary key as a genesis account with <snapshotted> million tokens (leaving room for halving minting)
    print_info "Adding genesis account with balance: $GENESIS_ACCOUNT_BALANCE"
    local address=$($NXQD_BIN keys show "primary" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
    $NXQD_BIN add-genesis-account "$address" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend "$KEYRING" --home "$HOMEDIR"
    print_success "Added genesis account primary with balance $GENESIS_ACCOUNT_BALANCE"
    
    # Create genesis transaction with validator key
    print_info "Creating genesis transaction with validator key (primary) with stake: $VALIDATOR_STAKE"
    $NXQD_BIN gentx "primary" "$VALIDATOR_STAKE" --gas-prices ${BASEFEE}unxq --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$HOMEDIR"
    
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
