#!/bin/bash

# Governance Proposal Generator
# Generates governance proposals with actual deployed contract addresses

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Function to generate activate security proposal
generate_activate_proposal() {
    local ONLINE_SERVER_CONTRACT=${1:-0x0000000000000000000000000000000000000000}
    local NFT_CONTRACT=${2:-0x0000000000000000000000000000000000000000}
    local WALLET_STATE_CONTRACT=${3:-0x0000000000000000000000000000000000000000}
    local VALIDATOR_APPROVAL_CONTRACT=${4:-0x0000000000000000000000000000000000000000}
    shift 4
    local WHITELIST_ADDRESSES=("$@")
    
    # Build whitelist JSON array
    local WHITELIST_JSON="["
    for i in "${!WHITELIST_ADDRESSES[@]}"; do
        if [ $i -gt 0 ]; then
            WHITELIST_JSON+=","
        fi
        WHITELIST_JSON+="\"${WHITELIST_ADDRESSES[$i]}\""
    done
    WHITELIST_JSON+="]"
    
    cat > activate-security-proposal.json <<EOF
{
  "title": "Activate Chain Security Features",
  "description": "Enable wallet lock and chain status checks with deployed contract addresses. This transitions the chain from bootstrap mode to production mode with full security enforcement.",
  "changes": [
    {
      "subspace": "evm",
      "key": "OnlineServerCountContract",
      "value": "\"$ONLINE_SERVER_CONTRACT\""
    },
    {
      "subspace": "evm",
      "key": "NFTContractAddress",
      "value": "\"$NFT_CONTRACT\""
    },
    {
      "subspace": "evm",
      "key": "WalletStateContractAddress",
      "value": "\"$WALLET_STATE_CONTRACT\""
    },
    {
      "subspace": "evm",
      "key": "ValidatorApprovalContractAddress",
      "value": "\"$VALIDATOR_APPROVAL_CONTRACT\""
    },
    {
      "subspace": "evm",
      "key": "WhitelistedAddresses",
      "value": "$WHITELIST_JSON"
    },
    {
      "subspace": "evm",
      "key": "EnableChainStatusCheck",
      "value": "true"
    },
    {
      "subspace": "evm",
      "key": "EnableWalletLockCheck",
      "value": "true"
    }
  ],
  "deposit": "10000000000000000000unxq"
}
EOF
    
    print_success "Generated: activate-security-proposal.json"
}

# Function to generate emergency disable proposal
generate_disable_proposal() {
    cat > emergency-disable-proposal.json <<EOF
{
  "title": "Emergency: Disable Security Checks",
  "description": "Temporarily disable security checks due to contract bug or emergency situation. Contracts will be fixed and re-enabled via subsequent proposal.",
  "changes": [
    {
      "subspace": "evm",
      "key": "EnableChainStatusCheck",
      "value": "false"
    },
    {
      "subspace": "evm",
      "key": "EnableWalletLockCheck",
      "value": "false"
    }
  ],
  "deposit": "10000000000000000000unxq"
}
EOF
    
    print_success "Generated: emergency-disable-proposal.json"
}

# Function to generate update contract proposal
generate_update_contract_proposal() {
    local CONTRACT_NAME=$1
    local NEW_ADDRESS=$2
    local PARAM_KEY=$3
    
    cat > update-${CONTRACT_NAME,,}-proposal.json <<EOF
{
  "title": "Update $CONTRACT_NAME Contract Address",
  "description": "Deploy and activate new version of $CONTRACT_NAME contract with bug fixes and improvements.",
  "changes": [
    {
      "subspace": "evm",
      "key": "$PARAM_KEY",
      "value": "\"$NEW_ADDRESS\""
    }
  ],
  "deposit": "10000000000000000000unxq"
}
EOF
    
    print_success "Generated: update-${CONTRACT_NAME,,}-proposal.json"
}

# Function to generate add whitelist proposal
generate_add_whitelist_proposal() {
    shift
    local ADDRESSES=("$@")
    
    # Build whitelist JSON array
    local WHITELIST_JSON="["
    for i in "${!ADDRESSES[@]}"; do
        if [ $i -gt 0 ]; then
            WHITELIST_JSON+=","
        fi
        WHITELIST_JSON+="\"${ADDRESSES[$i]}\""
    done
    WHITELIST_JSON+="]"
    
    cat > add-whitelist-proposal.json <<EOF
{
  "title": "Update Whitelist Addresses",
  "description": "Add/update whitelisted addresses that bypass security checks.",
  "changes": [
    {
      "subspace": "evm",
      "key": "WhitelistedAddresses",
      "value": "$WHITELIST_JSON"
    }
  ],
  "deposit": "10000000000000000000unxq"
}
EOF
    
    print_success "Generated: add-whitelist-proposal.json"
}

# Display usage
usage() {
    cat <<EOF
Governance Proposal Generator for NexQloud Chain

Usage:
    $0 <command> [arguments]

Commands:
    activate        Generate proposal to activate security
    disable         Generate emergency disable proposal
    update-contract Generate proposal to update a contract address
    add-whitelist   Generate proposal to update whitelist
    help            Show this help message

Examples:
    # Generate activate security proposal
    $0 activate \\
        0xONLINE_SERVER_CONTRACT \\
        0xNFT_CONTRACT \\
        0xWALLET_STATE_CONTRACT \\
        0xVALIDATOR_APPROVAL_CONTRACT \\
        0xWHITELIST_ADDR_1 \\
        0xWHITELIST_ADDR_2

    # Generate emergency disable proposal
    $0 disable

    # Generate contract update proposal
    $0 update-contract ValidatorApproval 0xNEW_ADDRESS ValidatorApprovalContractAddress

    # Generate whitelist update proposal
    $0 add-whitelist 0xADDR1 0xADDR2 0xADDR3

Environment Variables:
    You can also set these environment variables:
    - ONLINE_SERVER_CONTRACT
    - NFT_CONTRACT
    - WALLET_STATE_CONTRACT
    - VALIDATOR_APPROVAL_CONTRACT
    - WHITELIST_ADDRESSES (comma-separated)

Example with env vars:
    export ONLINE_SERVER_CONTRACT=0x1234...
    export NFT_CONTRACT=0x5678...
    export WALLET_STATE_CONTRACT=0x9abc...
    export VALIDATOR_APPROVAL_CONTRACT=0xdef0...
    export WHITELIST_ADDRESSES=0xaddr1,0xaddr2
    $0 activate

EOF
}

# Main
main() {
    if [ $# -eq 0 ]; then
        usage
        exit 1
    fi

    local COMMAND=$1
    shift

    case "$COMMAND" in
        activate)
            if [ $# -ge 4 ]; then
                # Arguments provided
                generate_activate_proposal "$@"
            elif [ -n "$ONLINE_SERVER_CONTRACT" ] && [ -n "$NFT_CONTRACT" ] && [ -n "$WALLET_STATE_CONTRACT" ] && [ -n "$VALIDATOR_APPROVAL_CONTRACT" ]; then
                # Use environment variables
                IFS=',' read -ra WHITELIST_ARRAY <<< "$WHITELIST_ADDRESSES"
                generate_activate_proposal "$ONLINE_SERVER_CONTRACT" "$NFT_CONTRACT" "$WALLET_STATE_CONTRACT" "$VALIDATOR_APPROVAL_CONTRACT" "${WHITELIST_ARRAY[@]}"
            else
                print_warning "Please provide contract addresses as arguments or set environment variables"
                echo ""
                usage
                exit 1
            fi
            ;;
        disable)
            generate_disable_proposal
            ;;
        update-contract)
            if [ $# -ne 3 ]; then
                print_warning "Usage: $0 update-contract <ContractName> <NewAddress> <ParamKey>"
                exit 1
            fi
            generate_update_contract_proposal "$1" "$2" "$3"
            ;;
        add-whitelist)
            if [ $# -eq 0 ]; then
                if [ -n "$WHITELIST_ADDRESSES" ]; then
                    IFS=',' read -ra WHITELIST_ARRAY <<< "$WHITELIST_ADDRESSES"
                    generate_add_whitelist_proposal "${WHITELIST_ARRAY[@]}"
                else
                    print_warning "Please provide whitelist addresses"
                    exit 1
                fi
            else
                generate_add_whitelist_proposal "$@"
            fi
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            print_warning "Unknown command: $COMMAND"
            echo ""
            usage
            exit 1
            ;;
    esac

    print_info "Proposal generated successfully!"
    print_info "Review the JSON file before submitting"
    echo ""
    print_info "To submit:"
    echo "nxqd tx gov submit-proposal param-change <proposal-file>.json \\"
    echo "  --from=validator1 \\"
    echo "  --chain-id=nxqd_6000-1 \\"
    echo "  --gas=auto \\"
    echo "  --gas-prices=1000000000unxq"
}

main "$@"

