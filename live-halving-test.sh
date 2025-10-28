#!/bin/bash

# =============================================================================
# Live Halving Test Script (No Restart Required!)
# =============================================================================
# 
# This script tests halving using governance proposals to change parameters
# while your 3-node development environment keeps running.
#
# Benefits:
# - No node restart required
# - Tests governance functionality too
# - More realistic production scenario
# - Faster testing iteration
#
# =============================================================================

set -e

# Configuration
CHAIN_ID="nexqloud-dev-1"
MULTISIG_ADDRESS="nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5"
VALIDATOR_KEY="validator"  # Your validator key name
BINARY="nxqd"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if chain is running
check_chain_status() {
    log_info "Checking if chain is running..."
    if ! $BINARY status &>/dev/null; then
        log_error "Chain is not running! Please start your 3-node dev environment."
        exit 1
    fi
    log_success "Chain is running!"
}

# Submit governance proposal to enable fast halving
submit_fast_halving_proposal() {
    log_info "Creating governance proposal for fast halving..."
    
    cat > fast-halving-proposal.json << EOF
{
  "title": "Enable Fast Halving for Testing",
  "description": "Change halving interval to 4 epochs for rapid testing",
  "changes": [
    {
      "subspace": "inflation",
      "key": "HalvingIntervalEpochs",
      "value": "4"
    },
    {
      "subspace": "inflation",
      "key": "EnableInflation", 
      "value": "true"
    }
  ],
  "deposit": "10000000aevmos"
}
EOF

    log_info "Submitting governance proposal..."
    
    # Submit proposal
    PROPOSAL_TX=$($BINARY tx gov submit-proposal param-change fast-halving-proposal.json \
        --from $VALIDATOR_KEY \
        --chain-id $CHAIN_ID \
        --gas auto \
        --gas-adjustment 1.3 \
        --broadcast-mode block \
        --yes \
        --output json)
    
    # Extract proposal ID
    PROPOSAL_ID=$(echo $PROPOSAL_TX | jq -r '.logs[0].events[] | select(.type=="submit_proposal") | .attributes[] | select(.key=="proposal_id") | .value')
    
    log_success "Proposal submitted! Proposal ID: $PROPOSAL_ID"
    return $PROPOSAL_ID
}

# Vote on proposal
vote_on_proposal() {
    local proposal_id=$1
    
    log_info "Voting YES on proposal $proposal_id..."
    
    $BINARY tx gov vote $proposal_id yes \
        --from $VALIDATOR_KEY \
        --chain-id $CHAIN_ID \
        --gas auto \
        --gas-adjustment 1.3 \
        --broadcast-mode block \
        --yes
    
    log_success "Vote submitted!"
}

# Wait for proposal to pass
wait_for_proposal() {
    local proposal_id=$1
    
    log_info "Waiting for proposal $proposal_id to pass..."
    
    while true; do
        STATUS=$($BINARY query gov proposal $proposal_id --output json | jq -r '.status')
        
        case $STATUS in
            "PROPOSAL_STATUS_VOTING_PERIOD")
                log_info "Proposal still in voting period..."
                sleep 5
                ;;
            "PROPOSAL_STATUS_PASSED")
                log_success "Proposal passed! Parameters updated."
                break
                ;;
            "PROPOSAL_STATUS_REJECTED")
                log_error "Proposal rejected!"
                exit 1
                ;;
            "PROPOSAL_STATUS_FAILED")
                log_error "Proposal failed!"
                exit 1
                ;;
            *)
                log_info "Proposal status: $STATUS"
                sleep 5
                ;;
        esac
    done
}

# Verify parameters updated
verify_parameters() {
    log_info "Verifying parameters were updated..."
    
    PARAMS=$($BINARY query inflation params --output json)
    HALVING_INTERVAL=$(echo $PARAMS | jq -r '.halving_interval_epochs')
    INFLATION_ENABLED=$(echo $PARAMS | jq -r '.enable_inflation')
    
    if [ "$HALVING_INTERVAL" = "4" ]; then
        log_success "✅ Halving interval updated to 4 epochs"
    else
        log_error "❌ Halving interval not updated. Current: $HALVING_INTERVAL"
        exit 1
    fi
    
    if [ "$INFLATION_ENABLED" = "true" ]; then
        log_success "✅ Inflation enabled"
    else
        log_error "❌ Inflation not enabled"
        exit 1
    fi
}

# Monitor balance changes
monitor_halving() {
    log_info "Starting halving monitoring..."
    
    local initial_balance=$($BINARY query bank balances $MULTISIG_ADDRESS --output json | jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"')
    log_info "Initial multi-sig balance: $initial_balance aevmos"
    
    # Monitor for 10 epochs to see multiple halvings
    for epoch in {1..10}; do
        echo ""
        log_info "=== Epoch $epoch Monitoring ==="
        
        # Wait for epoch end
        log_info "Waiting for epoch $epoch to complete..."
        sleep 30  # Adjust based on your epoch duration
        
        # Check new balance
        local new_balance=$($BINARY query bank balances $MULTISIG_ADDRESS --output json | jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"')
        local minted=$((new_balance - initial_balance))
        local nxq_minted=$((minted / 1000000000000000000000))
        
        # Get halving data
        local halving_data=$($BINARY query inflation halving-data --output json)
        local current_period=$(echo $halving_data | jq -r '.current_period')
        
        log_info "New balance: $new_balance aevmos"
        log_info "Minted this epoch: $nxq_minted NXQ"
        log_info "Current halving period: $current_period"
        
        # Check expected amounts
        case $current_period in
            0) log_info "Expected: 7200 NXQ (Period 0 - Full emission)" ;;
            1) log_success "🎉 FIRST HALVING! Expected: 3600 NXQ (Period 1)" ;;
            2) log_success "🎉 SECOND HALVING! Expected: 1800 NXQ (Period 2)" ;;
            3) log_success "🎉 THIRD HALVING! Expected: 900 NXQ (Period 3)" ;;
            *) log_info "Period $current_period" ;;
        esac
        
        initial_balance=$new_balance
    done
}

# Cleanup
cleanup() {
    log_info "Cleaning up..."
    rm -f fast-halving-proposal.json
}

# Main execution
main() {
    echo "=============================================="
    echo "🚀 LIVE HALVING TEST (No Restart Required!)"
    echo "=============================================="
    echo "Using governance to update parameters live"
    echo "Multi-sig Address: $MULTISIG_ADDRESS"
    echo "=============================================="
    
    trap cleanup EXIT
    
    # Pre-flight checks
    check_chain_status
    
    # Submit and execute governance proposal
    submit_fast_halving_proposal
    PROPOSAL_ID=$?
    
    vote_on_proposal $PROPOSAL_ID
    wait_for_proposal $PROPOSAL_ID
    verify_parameters
    
    # Start monitoring
    monitor_halving
    
    log_success "Live halving test completed! 🎉"
}

main "$@"

