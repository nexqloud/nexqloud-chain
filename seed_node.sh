#!/bin/bash
set -e

# Node configuration
CHAINID="nxqd_6000-1"
MONIKER="NexqloudSeedNode1"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
HOMEDIR="$HOME/.nxqd"
KEYBACKUP_DIR="$HOME/.nxqd_keys_backup"

#for local testing
NXQD_BIN="$(pwd)/cmd/nxqd/nxqd"

#for remote testing
# NXQD_BIN="/usr/local/bin/nxqd"

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

# Setup genesis accounts with proper balances
setup_genesis_accounts() {
    print_section "Setting Up Genesis Accounts"
    
    # Define key names and their initial balances
    # We use simple arrays for better compatibility
    KEY_NAMES=("mykey")
    KEY_BALANCES=("1000000000000000000000unxq")
    KEY_ROLES=("Validator")
    
    # Add genesis accounts with balances
    for i in "${!KEY_NAMES[@]}"; do
        local key=${KEY_NAMES[$i]}
        local balance=${KEY_BALANCES[$i]}
        local role=${KEY_ROLES[$i]}
        
        print_info "Adding $role: $key with balance $balance"
        local address=$($NXQD_BIN keys show "$key" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
        $NXQD_BIN add-genesis-account "$address" "$balance" --keyring-backend "$KEYRING" --home "$HOMEDIR"
        print_success "Added genesis account $key with balance $balance"
    done
    
    # Add vault and maintenance keys
    print_info "Generating vault and maintenance keys"
    
    # Generate vault keys
    for i in {1..5}; do
        print_info "Generating vault key $i"
        local vault_key="vault$i"
        generate_key "$vault_key"
        local address=$($NXQD_BIN keys show "$vault_key" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
        $NXQD_BIN add-genesis-account "$address" "2100000000000000000000000unxq" --keyring-backend "$KEYRING" --home "$HOMEDIR"
        print_success "Added vault $i with address $address"
    done
    
    # Generate maintenance wallet key
    print_info "Generating maintenance wallet key"
    generate_key "maintenance"
    local maint_address=$($NXQD_BIN keys show "maintenance" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")
    $NXQD_BIN add-genesis-account "$maint_address" "10499000000000000000000000unxq" --keyring-backend "$KEYRING" --home "$HOMEDIR"
    print_success "Added maintenance wallet with address $maint_address"
    
    print_info "Creating genesis transaction with validator key (mykey)"
    # Sign genesis transaction with the validator key
    $NXQD_BIN gentx "mykey" 100000000000000000000unxq --gas-prices ${BASEFEE}unxq --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$HOMEDIR"
}

# Function to set up NFT validation bypass
setup_nft_validation_bypass() {
    print_section "Setting Up NFT Validation Bypass"
    
    # Get validator addresses for mykey and dev0
    local val1_addr=$(jq -r '.app_state.genutil.gen_txs[0].body.messages[0].validator_address' "$GENESIS")
    local val1_eth_addr=$(jq -r '.app_state.genutil.gen_txs[0].body.messages[0].delegator_address' "$GENESIS" | $NXQD_BIN debug addr --home "$HOMEDIR" | grep "eth" | cut -d' ' -f3)
    
    print_info "Validator Address: $val1_addr"
    print_info "Validator ETH Address: $val1_eth_addr"
    
    # Create NFT allowlist file
    local nft_config="$HOMEDIR/config/nft_allowlist.json"
    cat > "$nft_config" << EOF
{
  "approved_validators": [
    "$val1_eth_addr"
  ],
  "nft_contract_address": "0x816644F8bc4633D268842628EB10ffC0AdcB6099",
  "bypass_validation": true
}
EOF
    
    print_success "Created NFT validation bypass config at $nft_config"
    
    # Update app.toml to point to this file
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/nft_config = ""/nft_config = "config\/nft_allowlist.json"/g' "$APP_TOML" 2>/dev/null || \
        echo 'nft_config = "config/nft_allowlist.json"' >> "$APP_TOML"
    else
        sed -i 's/nft_config = ""/nft_config = "config\/nft_allowlist.json"/g' "$APP_TOML" 2>/dev/null || \
        echo 'nft_config = "config/nft_allowlist.json"' >> "$APP_TOML"
    fi
    
    print_success "Updated app.toml to use NFT allowlist"
}

# Initialize the blockchain
initialize_blockchain() {
    print_section "Initializing Blockchain"
    
    # Remove previous data to start fresh
    if [ -d "$HOMEDIR" ]; then
        print_warning "Removing existing data directory: $HOMEDIR"
        rm -rf "$HOMEDIR"
    fi
    
    # Also remove key backup directory to prevent invalid mnemonic errors
    if [ -d "$KEYBACKUP_DIR" ]; then
        print_warning "Removing existing key backup directory: $KEYBACKUP_DIR"
        rm -rf "$KEYBACKUP_DIR"
    fi
    
    # Set client config
    print_info "Configuring client"
    $NXQD_BIN config keyring-backend "$KEYRING" --home "$HOMEDIR"
    $NXQD_BIN config chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Generate or load keys
    print_section "Key Management"
    print_info "This process will create secure keys and store mnemonics in $KEYBACKUP_DIR"
    print_info "You will need to enter a password for the keyring"
    
    generate_key "mykey"
    
    # Initialize the chain
    print_section "Chain Initialization"
    print_info "Initializing chain with moniker: $MONIKER and chain-id: $CHAINID"
    $NXQD_BIN init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"
    
    # Customize genesis settings
    print_info "Customizing genesis parameters"
	jq '.app_state["staking"]["params"]["bond_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["inflation"]["params"]["mint_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
    
    # Disable NFT validation for validator approval (for testing only)
    print_info "Disabling NFT validation for local testing"

	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set base fee in genesis
	jq '.app_state["feemarket"]["params"]["base_fee"]="'${BASEFEE}'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

    # Adjust timeout settings for pending mode
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

    # Adjust governance parameters for faster testing
    print_info "Setting up shortened proposal periods for testing"
	sed -i.bak 's/"max_deposit_period": "172800s"/"max_deposit_period": "30s"/g' "$GENESIS"
	sed -i.bak 's/"voting_period": "172800s"/"voting_period": "30s"/g' "$GENESIS"

    # Setup genesis accounts
    setup_genesis_accounts
    
    # Setup NFT validation bypass
    setup_nft_validation_bypass
    
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
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
        sed -i '' 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
    else
	sed -i 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
	sed -i 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
    fi
    
    print_section "Initialization Complete"
    print_success "Node has been successfully initialized!"
    print_warning "IMPORTANT: Make sure to securely back up your key mnemonics from: $KEYBACKUP_DIR"
    print_info "To start the node, run: $0 start"
    
    # Add reminder about manual modification
    print_warning "IMPORTANT: Remember to manually comment/uncomment NFT validation in the code as needed."
    print_info "NFT validation is controlled in x/staking/keeper/msg_server.go in the CreateValidator function."
}

# Start the blockchain node
start_node() {
    print_section "Starting Blockchain Node"
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
