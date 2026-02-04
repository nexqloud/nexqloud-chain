#!/bin/bash
set -e

# ============================================================================
# NEXQLOUD CHAIN - PRODUCTION PEER NODE (Docker)
# ============================================================================
# This script is for regular peer nodes and persistent peers.
# Optimized for Docker with plug-and-play networking.
# ============================================================================

# ============================================================================
# NODE CONFIGURATION
# ============================================================================
CHAINID="${CHAINID:-nxqd_90009-1}"
MONIKER="${MONIKER:-NexqloudPeer}"
KEYALGO="eth_secp256k1"
LOGLEVEL="${LOGLEVEL:-info}"
HOMEDIR="$HOME/.nxqd"
NXQD_BIN="/usr/local/bin/nxqd"
BASEFEE=1000000000
TRACE=""

# Peers use test keyring (no sensitive keys needed)
KEYRING="${KEYRING:-test}"

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

# 🔌 Plug-and-Play Network Configuration
SEED_SERVICES="${SEED_SERVICES:-seed-node-1 seed-node-2}"
PERSISTENT_PEER_SERVICES="${PERSISTENT_PEER_SERVICES:-}"
OTHER_PERSISTENT_PEERS="${OTHER_PERSISTENT_PEERS:-}"

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
    command -v wget >/dev/null 2>&1 || error_exit "wget not installed"
    
    if [ ! -f "$NXQD_BIN" ]; then
        error_exit "nxqd binary not found at $NXQD_BIN"
    fi
    
    print_success "All dependencies found"
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

# 🔌 Download genesis from available seeds (plug-and-play with fallback)
download_genesis() {
    print_section "Downloading Genesis from Seed Nodes"
    
    # Try each seed service until one succeeds
    for seed_service in $SEED_SERVICES; do
        print_info "Attempting to download genesis from: $seed_service"
        
        # Try HTTP download
        if wget -q --timeout=30 "http://$seed_service:26657/genesis" -O "$GENESIS" 2>/dev/null; then
            if $NXQD_BIN validate-genesis --home "$HOMEDIR" 2>/dev/null; then
                print_success "Successfully downloaded and validated genesis from $seed_service"
                return 0
            else
                print_warning "Genesis validation failed from $seed_service"
            fi
        fi
        
        # Try alternative method
        if genesis_json=$(wget -qO- "http://$seed_service:26657/genesis" 2>/dev/null | jq -r '.result.genesis' 2>/dev/null); then
            if [ -n "$genesis_json" ] && [ "$genesis_json" != "null" ]; then
                echo "$genesis_json" > "$GENESIS"
                if $NXQD_BIN validate-genesis --home "$HOMEDIR" 2>/dev/null; then
                    print_success "Successfully downloaded genesis from $seed_service (alternative method)"
                    return 0
                fi
            fi
        fi
        
        print_warning "Failed to download from $seed_service, trying next..."
    done
    
    error_exit "Failed to download genesis from any seed node"
}

# Initialize the peer node
initialize_peer() {
    print_section "Initializing Peer Node"
    
    # Remove previous data
    if [ -d "$HOMEDIR" ]; then
        print_warning "Removing existing data directory: $HOMEDIR"
        rm -rf "$HOMEDIR" 2>/dev/null || print_warning "Cannot remove $HOMEDIR (likely Docker volume - continuing)"
    fi
    
    # Set client config
    print_info "Configuring client"
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Initialize the chain
    print_section "Chain Initialization"
    print_info "Initializing chain with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Download genesis from available seeds
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
    
    # Add seed nodes as persistent peers (not seeds)
    if [ -n "$SEED_SERVICES" ]; then
        print_info "Discovering seed nodes (adding as persistent peers)..."
        for service in $SEED_SERVICES; do
            if node_id=$(get_node_id_from_service "$service"); then
                if [ -z "$PERSISTENT_PEERS" ]; then
                    PERSISTENT_PEERS="$node_id@$service:26656"
                else
                    PERSISTENT_PEERS="$PERSISTENT_PEERS,$node_id@$service:26656"
                fi
                print_success "Added seed node as persistent peer: $service"
            else
                print_warning "Could not connect to seed $service (may not be running yet)"
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
                print_warning "Could not connect to persistent peer $service"
            fi
        done
    fi
    
    # Add other persistent peers (if this is a persistent peer node)
    if [ -n "$OTHER_PERSISTENT_PEERS" ]; then
        print_info "Discovering other persistent peers..."
        for service in $OTHER_PERSISTENT_PEERS; do
            if node_id=$(get_node_id_from_service "$service"); then
                if [ -z "$PERSISTENT_PEERS" ]; then
                    PERSISTENT_PEERS="$node_id@$service:26656"
                else
                    PERSISTENT_PEERS="$PERSISTENT_PEERS,$node_id@$service:26656"
                fi
                print_success "Added persistent peer: $service"
            else
                print_warning "Could not connect to $service"
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
        print_warning "No persistent peers configured - node may have difficulty connecting"
        # Still clear seeds even if no peers found
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/^seeds = .*/seeds = \"\"/" "$CONFIG"
        else
            sed -i "s/^seeds = .*/seeds = \"\"/" "$CONFIG"
        fi
    fi
    
    # Enable PEX for automatic peer discovery
    print_info "Enabling PEX (Peer Exchange) for automatic peer discovery"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/^pex = false/pex = true/' "$CONFIG"
        sed -i '' 's/^pex = true/pex = true/' "$CONFIG"  # Ensure it's true
    else
        sed -i 's/^pex = false/pex = true/' "$CONFIG"
        sed -i 's/^pex = true/pex = true/' "$CONFIG"  # Ensure it's true
    fi
    
    # Configure peer connection limits
    print_info "Configuring peer connection limits"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/^max_num_inbound_peers = .*/max_num_inbound_peers = 40/' "$CONFIG"
        sed -i '' 's/^max_num_outbound_peers = .*/max_num_outbound_peers = 10/' "$CONFIG"
    else
        sed -i 's/^max_num_inbound_peers = .*/max_num_inbound_peers = 40/' "$CONFIG"
        sed -i 's/^max_num_outbound_peers = .*/max_num_outbound_peers = 10/' "$CONFIG"
    fi
    
    # Enable Prometheus metrics
    print_info "Enabling Prometheus metrics"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
    else
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
    fi
    
    # Enable APIs
    print_info "Enabling APIs"
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/prometheus-retention-time = 0/prometheus-retention-time = 1000/' "$APP_TOML"
        sed -i '' 's/enabled = false/enabled = true/' "$APP_TOML"
    else
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
    print_success "Peer node initialized successfully!"
    print_info "This node will automatically discover and connect to peers via PEX"
}

# Start the node
start_node() {
    print_section "Starting Peer Node"
    print_info "Chain ID: $CHAINID"
    print_info "Moniker: $MONIKER"
    print_info "Home Dir: $HOMEDIR"
    print_info "PEX enabled for automatic peer discovery"

    # If home directory doesn't exist, initialize first
    if [ ! -d "$HOMEDIR" ] || [ ! -f "$GENESIS" ]; then
        print_warning "Home directory or genesis file not found, initializing first..."
        initialize_peer
    fi
    
    $NXQD_BIN start \
        --metrics "$TRACE" \
        --log_level $LOGLEVEL \
        --minimum-gas-prices=0.0001nxq \
        --json-rpc.api eth,txpool,net,debug,web3 \
        --home "$HOMEDIR" \
        --chain-id "$CHAINID"
}

# Main function
main() {
    case "$1" in
        init)
            check_dependencies
            initialize_peer
            ;;
        start)
            check_dependencies
            start_node
            ;;
        *)
            echo "Usage: $0 [init|start]"
            echo "  init  - Initialize the peer node"
            echo "  start - Start the peer node"
            exit 1
            ;;
    esac
}

main "$@"

