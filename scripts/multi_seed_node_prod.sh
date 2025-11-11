#!/bin/bash
set -e

# ============================================================================
# NEXQLOUD CHAIN - PRODUCTION MULTI-SEED NODE (Docker)
# ============================================================================
# This script is for additional seed nodes that download genesis from the
# first seed node. Optimized for Docker with plug-and-play networking.
# ============================================================================

# 🔐 Docker Secrets Support
if [ -f "/run/secrets/seed1_password" ]; then
    KEYRING_PASSWORD=$(cat /run/secrets/seed1_password)
    export KEYRING_PASSWORD
elif [ -f "/run/secrets/seed2_password" ]; then
    KEYRING_PASSWORD=$(cat /run/secrets/seed2_password)
    export KEYRING_PASSWORD
elif [ -f "/run/secrets/seed3_password" ]; then
    KEYRING_PASSWORD=$(cat /run/secrets/seed3_password)
    export KEYRING_PASSWORD
fi

if [ -f "/run/secrets/seed1_mnemonic" ]; then
    MNEMONIC=$(cat /run/secrets/seed1_mnemonic)
    export MNEMONIC
elif [ -f "/run/secrets/seed2_mnemonic" ]; then
    MNEMONIC=$(cat /run/secrets/seed2_mnemonic)
    export MNEMONIC
elif [ -f "/run/secrets/seed3_mnemonic" ]; then
    MNEMONIC=$(cat /run/secrets/seed3_mnemonic)
    export MNEMONIC
fi

# ============================================================================
# NODE CONFIGURATION
# ============================================================================
CHAINID="${CHAINID:-nxqd_6000-1}"
MONIKER="${MONIKER:-NexqloudMultiSeed}"
KEYALGO="eth_secp256k1"
LOGLEVEL="${LOGLEVEL:-info}"
HOMEDIR="$HOME/.nxqd"
NXQD_BIN="/usr/local/bin/nxqd"
BASEFEE=1000000000
TRACE=""

KEYRING="${KEYRING:-file}"

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

# 🔌 Plug-and-Play Network Configuration
FIRST_SEED_SERVICE="${FIRST_SEED_SERVICE:-seed-node-1}"
OTHER_SEED_SERVICES="${OTHER_SEED_SERVICES:-}"
PERSISTENT_PEER_SERVICES="${PERSISTENT_PEER_SERVICES:-}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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
    command -v wget >/dev/null 2>&1 || error_exit "wget not installed"
    
    if [ ! -f "$NXQD_BIN" ]; then
        error_exit "nxqd binary not found at $NXQD_BIN"
    fi
    
    print_success "All dependencies found"
}

# 🔐 Generate key with Docker secrets support
generate_key() {
    local key_name=$1
    
    print_info "Processing key: $key_name"
    
    # Check if running with test keyring (auto-generate without prompts)
    if [ "$KEYRING" = "test" ] && [ -n "$MNEMONIC" ]; then
        print_info "Using test keyring with mnemonic"
        echo "$MNEMONIC" | $NXQD_BIN keys add "$key_name" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR" 2>&1 | grep -v "override"
        print_success "Key $key_name generated"
        return 0
    fi
    
    if [ -n "$KEYRING_PASSWORD" ] && [ -n "$MNEMONIC" ]; then
        print_info "Using Docker secrets for key generation"
        
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
        print_warning "No Docker secrets found, using interactive mode"
        $NXQD_BIN keys add "$key_name" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"
        print_success "Key $key_name generated"
    fi
}

# 🔌 Get node ID from Docker service
get_node_id_from_service() {
    local service=$1
    local node_id
    
    if node_id=$(timeout 10 wget -qO- "http://$service:26657/status" 2>/dev/null | jq -r '.result.node_info.id' 2>/dev/null); then
        if [ -n "$node_id" ] && [ "$node_id" != "null" ]; then
            echo "$node_id"
            return 0
        fi
    fi
    
    return 1
}

# 🔌 Download genesis from first seed (plug-and-play)
download_genesis() {
    print_section "Downloading Genesis from First Seed"
    
    print_info "Attempting to download genesis from: $FIRST_SEED_SERVICE"
    
    # Try HTTP download using Docker service name
    if wget -q --timeout=30 "http://$FIRST_SEED_SERVICE:26657/genesis" -O "$GENESIS"; then
        print_success "Successfully downloaded genesis from $FIRST_SEED_SERVICE"
        
        # Validate genesis
        if $NXQD_BIN validate-genesis --home "$HOMEDIR" 2>/dev/null; then
            print_success "Genesis validation successful"
            return 0
        else
            print_warning "Genesis validation failed, trying alternative method..."
        fi
    fi
    
    # Alternative: Try to get genesis from status endpoint
    print_info "Trying alternative download method..."
    if genesis_json=$(wget -qO- "http://$FIRST_SEED_SERVICE:26657/genesis" 2>/dev/null | jq -r '.result.genesis' 2>/dev/null); then
        if [ -n "$genesis_json" ] && [ "$genesis_json" != "null" ]; then
            echo "$genesis_json" > "$GENESIS"
            print_success "Successfully downloaded genesis (alternative method)"
            return 0
        fi
    fi
    
    error_exit "Failed to download genesis from $FIRST_SEED_SERVICE"
}

# Initialize the multi-seed node
initialize_multi_seed() {
    print_section "Initializing Multi-Seed Node"
    
    # Remove previous data
    if [ -d "$HOMEDIR" ]; then
        print_warning "Removing existing data directory: $HOMEDIR"
        rm -rf "$HOMEDIR" 2>/dev/null || print_warning "Cannot remove $HOMEDIR (likely Docker volume - continuing)"
    fi
    
    # Set client config
    print_info "Configuring client"
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Generate key
    print_section "Key Management"
    print_info "Generating validator key using Docker secrets"
    generate_key "primary"
    
    # Initialize the chain
    print_section "Chain Initialization"
    print_info "Initializing chain with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Download genesis from first seed
    download_genesis
    
    # Configure block time
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
    
    # Add first seed node as persistent peer
    print_info "Discovering first seed node..."
    if node_id=$(get_node_id_from_service "$FIRST_SEED_SERVICE"); then
        PERSISTENT_PEERS="$node_id@$FIRST_SEED_SERVICE:26656"
        print_success "Added first seed as persistent peer: $FIRST_SEED_SERVICE"
    else
        print_warning "Could not get node ID from $FIRST_SEED_SERVICE"
    fi
    
    # Add other seed nodes as persistent peers
    if [ -n "$OTHER_SEED_SERVICES" ]; then
        print_info "Discovering other seed nodes..."
        for service in $OTHER_SEED_SERVICES; do
            # Skip if it's the current node or the first seed
            if [ "$service" = "$FIRST_SEED_SERVICE" ]; then
                continue
            fi
            
            if node_id=$(get_node_id_from_service "$service"); then
                if [ -z "$PERSISTENT_PEERS" ]; then
                    PERSISTENT_PEERS="$node_id@$service:26656"
                else
                    PERSISTENT_PEERS="$PERSISTENT_PEERS,$node_id@$service:26656"
                fi
                print_success "Added seed node as persistent peer: $service"
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
    
    # Apply network configuration - use persistent_peers only, clear seeds
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
    
    # Enable PEX for automatic peer discovery
    print_info "Enabling PEX (Peer Exchange)"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/^pex = false/pex = true/' "$CONFIG"
        sed -i '' 's/^pex = true/pex = true/' "$CONFIG"  # Ensure it's true
    else
        sed -i 's/^pex = false/pex = true/' "$CONFIG"
        sed -i 's/^pex = true/pex = true/' "$CONFIG"  # Ensure it's true
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
    print_success "Multi-seed node initialized successfully!"
}

# Start the node
start_node() {
    print_section "Starting Multi-Seed Node"
    print_info "Chain ID: $CHAINID"
    print_info "Moniker: $MONIKER"
    print_info "Home Dir: $HOMEDIR"

    # If home directory doesn't exist, initialize first
    if [ ! -d "$HOMEDIR" ] || [ ! -f "$GENESIS" ]; then
        print_warning "Home directory or genesis file not found, initializing first..."
        initialize_multi_seed
    fi
    
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
            initialize_multi_seed
            ;;
        start)
            check_dependencies
            start_node
            ;;
        *)
            echo "Usage: $0 [init|start]"
            echo "  init  - Initialize the multi-seed node"
            echo "  start - Start the multi-seed node"
            exit 1
            ;;
    esac
}

main "$@"

