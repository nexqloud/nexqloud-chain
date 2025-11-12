#!/bin/bash
set -e

# ============================================================================
# NEXQLOUD CHAIN - PRODUCTION SEED NODE (Docker)
# ============================================================================
# This script is optimized for Docker deployment with:
# - Docker Secrets support for secure credential management
# - Dynamic service discovery using Docker DNS
# - Plug-and-play network configuration
# ============================================================================

# 🔐 Docker Secrets Support
# Secrets are mounted at /run/secrets/<secret_name>
if [ -f "/run/secrets/seed_password" ]; then
    KEYRING_PASSWORD=$(cat /run/secrets/seed_password)
    export KEYRING_PASSWORD
fi

if [ -f "/run/secrets/seed_mnemonic" ]; then
    MNEMONIC=$(cat /run/secrets/seed_mnemonic)
    export MNEMONIC
fi

# ============================================================================
# NODE CONFIGURATION
# ============================================================================
CHAINID="${CHAINID:-nxqd_90000-1}"
MONIKER="${MONIKER:-NexqloudSeedNode}"
KEYALGO="eth_secp256k1"
LOGLEVEL="${LOGLEVEL:-info}"
HOMEDIR="$HOME/.nxqd"
NXQD_BIN="/usr/local/bin/nxqd"
BASEFEE=1000000000
TRACE=""

# Keyring backend - use 'file' for production with Docker secrets
KEYRING="${KEYRING:-file}"

# Genesis account balance configuration (in unxq, where 1 NXQ = 10^18 unxq)
# Default: 3,200,000 NXQ = 3200000000000000000000000 unxq
GENESIS_ACCOUNT_BALANCE="${GENESIS_ACCOUNT_BALANCE:-3200000000000000000000000unxq}"

# Validator stake configuration (in unxq, where 1 NXQ = 10^18 unxq)
# Default: 50 NXQ = 50000000000000000000 unxq
VALIDATOR_STAKE="${VALIDATOR_STAKE:-25000000000000000000unxq}"

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

# 🔌 Plug-and-Play Network Configuration
# These are Docker service names that auto-resolve via Docker DNS
OTHER_SEED_SERVICES="${OTHER_SEED_SERVICES:-}"
PERSISTENT_PEER_SERVICES="${PERSISTENT_PEER_SERVICES:-}"

# Colors for better UX
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Display functions
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

error_exit() {
    print_message "$RED" "ERROR: $1"
    exit 1
}

print_warning() {
    print_message "$YELLOW" "WARNING: $1"
}

print_info() {
    print_message "$BLUE" "INFO: $1"
}

print_success() {
    print_message "$GREEN" "SUCCESS: $1"
}

print_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Check dependencies
check_dependencies() {
    print_section "Checking Dependencies"
    
    command -v jq >/dev/null 2>&1 || error_exit "jq not installed"
    command -v expect >/dev/null 2>&1 || error_exit "expect not installed"
    
    if [ ! -f "$NXQD_BIN" ]; then
        error_exit "nxqd binary not found at $NXQD_BIN"
    fi
    
    print_success "All dependencies found"
}

# 🔐 Generate key with Docker secrets support
generate_key() {
    local key_name=$1
    
    print_info "Processing key: $key_name"
    
    # Check if running in Docker with secrets
    if [ -n "$KEYRING_PASSWORD" ] && [ -n "$MNEMONIC" ]; then
        print_info "Using Docker secrets for key generation"
        
        # Use expect for automated input
        expect << EOF
set timeout 60
log_user 0
spawn $NXQD_BIN keys add "$key_name" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
expect {
    "Enter your bip39 mnemonic" {
        send "$MNEMONIC\r"
        exp_continue
    }
    "Enter keyring passphrase" {
        send "$KEYRING_PASSWORD\r"
        exp_continue
    }
    "Re-enter keyring passphrase" {
        send "$KEYRING_PASSWORD\r"
        exp_continue
    }
    eof
}
EOF
        
        print_success "Key $key_name generated from Docker secrets"
    else
        # Interactive mode (fallback)
        print_warning "No Docker secrets found, using interactive mode"
        print_warning "You will be prompted to create a password for your keyring."
        
        $NXQD_BIN keys add "$key_name" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
        
        print_success "Key $key_name generated"
    fi
}

# 🔌 Get node ID from Docker service (plug-and-play)
get_node_id_from_service() {
    local service=$1
    local node_id
    
    # Try to get node ID via HTTP (Docker DNS resolves service name)
    if node_id=$(timeout 10 wget -qO- "http://$service:26657/status" 2>/dev/null | jq -r '.result.node_info.id' 2>/dev/null); then
        if [ -n "$node_id" ] && [ "$node_id" != "null" ]; then
            echo "$node_id"
            return 0
        fi
    fi
    
    return 1
}

# Initialize the blockchain
initialize_blockchain() {
    print_section "Initializing First Seed Node"
    
    # Remove previous data to start fresh (skip if mounted volume in Docker)
    if [ -d "$HOMEDIR" ]; then
        print_warning "Removing existing data directory: $HOMEDIR"
        rm -rf "$HOMEDIR" 2>/dev/null || print_warning "Cannot remove $HOMEDIR (likely Docker volume - continuing)"
    fi
    
    # Set client config
    print_info "Configuring client"
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Generate or load keys
    print_section "Key Management"
    print_info "Generating validator key using Docker secrets"
    generate_key "primary"
    
    # Initialize the chain
    print_section "Chain Initialization"
    print_info "Initializing chain with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Customize genesis settings
    print_info "Customizing genesis parameters"
    
	# Change parameter token denominations to nxq
	jq '.app_state["staking"]["params"]["bond_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["inflation"]["params"]["mint_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
    jq '.app_state["inflation"]["params"]["enable_inflation"]=true' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set gas limit and base fee
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["feemarket"]["params"]["base_fee"]="'${BASEFEE}'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

    # Configure block time (8 seconds)
    print_info "Setting block time to 8 seconds"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i '' 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    else
        sed -i 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    fi
    
    # 🔌 Configure network connections using Docker service discovery
    print_section "Configuring Network (Plug-and-Play)"
    
    PERSISTENT_PEERS=""
    
    # Add other seed nodes as persistent peers
    if [ -n "$OTHER_SEED_SERVICES" ]; then
        print_info "Discovering other seed nodes..."
        for service in $OTHER_SEED_SERVICES; do
            if node_id=$(get_node_id_from_service "$service"); then
                if [ -z "$PERSISTENT_PEERS" ]; then
                    PERSISTENT_PEERS="$node_id@$service:26656"
                else
                    PERSISTENT_PEERS="$PERSISTENT_PEERS,$node_id@$service:26656"
                fi
                print_success "Added seed peer: $service"
            else
                print_warning "Could not connect to $service (may not be running yet)"
            fi
        done
    fi
    
    # Add persistent peers
    if [ -n "$PERSISTENT_PEER_SERVICES" ]; then
        print_info "Discovering persistent peers..."
        for service in $PERSISTENT_PEER_SERVICES; do
            if node_id=$(get_node_id_from_service "$service"); then
                if [ -z "$PERSISTENT_PEERS" ]; then
                    PERSISTENT_PEERS="$node_id@$service:26656"
                else
                    PERSISTENT_PEERS="$PERSISTENT_PEERS,$node_id@$service:26656"
                fi
                print_success "Added persistent peer: $service"
            else
                print_warning "Could not connect to $service (may not be running yet)"
            fi
        done
    fi
    
    # Apply persistent peers configuration and clear seeds
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
        # Still clear seeds even if no peers found
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/^seeds = .*/seeds = \"\"/" "$CONFIG"
        else
            sed -i "s/^seeds = .*/seeds = \"\"/" "$CONFIG"
        fi
    fi
    
    # Enable Prometheus metrics
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
    
    # Enable gRPC
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' '/^\[grpc\]/,/^\[/ s|^enable = false|enable = true|' "$APP_TOML"
    else
        sed -i '/^\[grpc\]/,/^\[/ s|^enable = false|enable = true|' "$APP_TOML"
    fi
    
    # Change proposal periods
    print_info "Setting up proposal periods (5 minutes for voting)"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/"voting_period": "172800s"/"voting_period": "300s"/g' "$GENESIS"
        sed -i '' 's/"max_deposit_period": "172800s"/"max_deposit_period": "300s"/g' "$GENESIS"
    else
        sed -i 's/"voting_period": "172800s"/"voting_period": "300s"/g' "$GENESIS"
        sed -i 's/"max_deposit_period": "172800s"/"max_deposit_period": "300s"/g' "$GENESIS"
    fi
    
    print_section "Setting Up Genesis Accounts"
    # Add the primary key as a genesis account
    print_info "Adding genesis account with balance: $GENESIS_ACCOUNT_BALANCE"
    
    # Get address - for test keyring, no password needed
    local address
    if [ "$KEYRING" = "test" ]; then
        address=$($NXQD_BIN keys show "primary" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
    elif [ -n "$KEYRING_PASSWORD" ]; then
        # Pipe password for file keyring
        address=$(echo "$KEYRING_PASSWORD" | $NXQD_BIN keys show "primary" -a --keyring-backend "$KEYRING" --home "$HOMEDIR" 2>&1)
    else
        address=$($NXQD_BIN keys show "primary" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
    fi
    
    # Add genesis account - test keyring doesn't need password
    if [ "$KEYRING" = "test" ]; then
        $NXQD_BIN add-genesis-account "$address" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend "$KEYRING" --home "$HOMEDIR"
    elif [ -n "$KEYRING_PASSWORD" ]; then
        echo "$KEYRING_PASSWORD" | $NXQD_BIN add-genesis-account "$address" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend "$KEYRING" --home "$HOMEDIR"
    else
        $NXQD_BIN add-genesis-account "$address" "$GENESIS_ACCOUNT_BALANCE" --keyring-backend "$KEYRING" --home "$HOMEDIR"
    fi
    
    print_success "Added genesis account primary with balance $GENESIS_ACCOUNT_BALANCE"
    
    # Create genesis transaction with validator key
    print_info "Creating genesis transaction with validator key (primary) with stake: $VALIDATOR_STAKE"
    if [ "$KEYRING" = "test" ]; then
        $NXQD_BIN gentx "primary" "$VALIDATOR_STAKE" --gas-prices ${BASEFEE}unxq --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$HOMEDIR"
    elif [ -n "$KEYRING_PASSWORD" ]; then
        echo "$KEYRING_PASSWORD" | $NXQD_BIN gentx "primary" "$VALIDATOR_STAKE" --gas-prices ${BASEFEE}unxq --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$HOMEDIR"
    else
        $NXQD_BIN gentx "primary" "$VALIDATOR_STAKE" --gas-prices ${BASEFEE}unxq --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$HOMEDIR"
    fi
    
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
    
    # Configure RPC access (bind to all interfaces for Docker)
    print_info "Enabling RPC services"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        sed -i '' '/^\[json-rpc\]/,/^\[/ s|enable = false|enable = true|' "$APP_TOML"
        sed -i '' 's|address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
        sed -i '' 's|ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|g' "$APP_TOML"
    else
        sed -i 's|laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|g' "$CONFIG"
        sed -i '/^\[json-rpc\]/,/^\[/ s|enable = false|enable = true|' "$APP_TOML"
        sed -i 's|address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|g' "$APP_TOML"
        sed -i 's|ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|g' "$APP_TOML"
    fi
    
    print_section "Initialization Complete"
    print_success "First seed node initialized successfully!"
}

# Start the node
start_node() {
    print_section "Starting First Seed Node"
    print_info "Chain ID: $CHAINID"
    print_info "Moniker: $MONIKER"
    print_info "Home Dir: $HOMEDIR"

    # If home directory doesn't exist, initialize first
    if [ ! -d "$HOMEDIR" ] || [ ! -f "$GENESIS" ]; then
        print_warning "Home directory or genesis file not found, initializing first..."
        initialize_blockchain
    fi
    
    # Auto-provide password if available
    if [ -n "$KEYRING_PASSWORD" ]; then
        print_info "Using Docker secrets for keyring password"
        echo "$KEYRING_PASSWORD" | $NXQD_BIN start \
            --metrics "$TRACE" \
            --log_level $LOGLEVEL \
            --minimum-gas-prices=0.0001nxq \
            --json-rpc.api eth,txpool,personal,net,debug,web3 \
            --home "$HOMEDIR" \
            --chain-id "$CHAINID"
    else
        print_warning "No Docker secrets found, you may need to enter password"
        $NXQD_BIN start \
            --metrics "$TRACE" \
            --log_level $LOGLEVEL \
            --minimum-gas-prices=0.0001nxq \
            --json-rpc.api eth,txpool,personal,net,debug,web3 \
            --home "$HOMEDIR" \
            --chain-id "$CHAINID"
    fi
}

# Main function
main() {
    case "$1" in
        init)
            check_dependencies
            initialize_blockchain
            ;;
        start)
            check_dependencies
            start_node
            ;;
        *)
            echo "Usage: $0 [init|start]"
            echo "  init  - Initialize the first seed node"
            echo "  start - Start the first seed node"
            exit 1
            ;;
    esac
}

main "$@"

