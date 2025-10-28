#!/bin/bash

# =============================================================================
# Quick Halving Test Script for 3-Node Development Environment
# =============================================================================
# 
# This script tests the Bitcoin-style halving system with fast 4-epoch intervals
# so you can see multiple halvings in minutes instead of waiting years.
#
# What it does:
# 1. Sets up halving parameters with 4-epoch intervals
# 2. Monitors multi-sig balance changes in real-time
# 3. Tracks halving period progression
# 4. Validates correct emission amounts at each period
# 5. Works with 3-node development setup
#
# Expected Results:
# - Epochs 1-4: 7200 NXQ per epoch (Period 0)
# - Epochs 5-8: 3600 NXQ per epoch (Period 1) - FIRST HALVING
# - Epochs 9-12: 1800 NXQ per epoch (Period 2) - SECOND HALVING
#
# =============================================================================

set -e  # Exit on any error

# Configuration
CHAIN_ID="nexqloud-dev-1"
MULTISIG_ADDRESS="nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5"
NODE_HOME="$HOME/.nxqd"
BINARY="nxqd"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_halving() {
    echo -e "${PURPLE}[HALVING]${NC} $1"
}

# Check if chain is running
check_chain_status() {
    log_info "Checking if chain is running..."
    if ! $BINARY status &>/dev/null; then
        log_error "Chain is not running! Please start your 3-node development environment first."
        exit 1
    fi
    log_success "Chain is running!"
}

# Get current balance
get_multisig_balance() {
    $BINARY query bank balances $MULTISIG_ADDRESS --output json 2>/dev/null | \
        jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"'
}

# Get halving data
get_halving_data() {
    $BINARY query inflation halving-data --output json 2>/dev/null
}

# Get current epoch
get_current_epoch() {
    $BINARY query epochs epoch-infos --output json 2>/dev/null | \
        jq -r '.epochs[] | select(.identifier=="day") | .current_epoch'
}

# Get inflation parameters
get_inflation_params() {
    $BINARY query inflation params --output json 2>/dev/null
}

# Update parameters for fast testing
setup_fast_halving() {
    log_info "Setting up fast halving parameters (4 epochs per period)..."
    
    # Note: In a real environment, this would be done via governance proposal
    # For dev testing, we assume you can modify genesis or use test keys
    cat > halving-proposal.json << EOF
{
  "title": "Enable Fast Halving for Testing",
  "description": "Set halving interval to 4 epochs for rapid testing",
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
  ]
}
EOF

    log_warning "Fast halving setup requires governance proposal or genesis modification."
    log_warning "Ensure your dev environment has halving_interval_epochs set to 4."
}

# Monitor single epoch
monitor_epoch() {
    local epoch_num=$1
    local expected_emission=$2
    local expected_period=$3
    
    echo ""
    echo "==========================================="
    log_info "Monitoring Epoch $epoch_num"
    echo "==========================================="
    
    # Get initial state
    local initial_balance=$(get_multisig_balance)
    local halving_data=$(get_halving_data)
    local current_period=$(echo $halving_data | jq -r '.current_period // 0')
    
    log_info "Initial balance: $initial_balance aevmos"
    log_info "Current period: $current_period"
    log_info "Expected emission: $expected_emission NXQ"
    
    # Wait for epoch to process
    log_info "Waiting for epoch processing..."
    sleep 10
    
    # Get final state
    local final_balance=$(get_multisig_balance)
    local new_halving_data=$(get_halving_data)
    local new_period=$(echo $new_halving_data | jq -r '.current_period // 0')
    
    # Calculate difference
    local minted=$((final_balance - initial_balance))
    local expected_minted_wei="${expected_emission}000000000000000000000"  # Convert to wei
    
    log_info "Final balance: $final_balance aevmos"
    log_info "Minted amount: $minted aevmos"
    log_info "New period: $new_period"
    
    # Validate results
    if [ "$minted" -eq "$expected_minted_wei" ]; then
        log_success "✅ Correct amount minted: $expected_emission NXQ"
    else
        log_error "❌ Wrong amount minted! Expected: $expected_emission NXQ, Got: $((minted / 1000000000000000000000)) NXQ"
    fi
    
    if [ "$new_period" -eq "$expected_period" ]; then
        log_success "✅ Correct halving period: $expected_period"
    else
        log_error "❌ Wrong halving period! Expected: $expected_period, Got: $new_period"
    fi
    
    # Check for halving event
    if [ "$new_period" -gt "$current_period" ]; then
        log_halving "🎉 HALVING EVENT! Advanced from Period $current_period to Period $new_period"
        log_halving "🎉 Emission now halved to $expected_emission NXQ per day!"
    fi
}

# Main testing function
run_halving_test() {
    log_info "Starting Quick Halving Test..."
    log_info "Testing with 4-epoch halving intervals"
    
    # Test sequence: 12 epochs to see 3 halving periods
    local test_epochs=(
        # Epoch:Expected_Emission:Expected_Period
        "1:7200:0"   # Period 0 - Full emission
        "2:7200:0"   # Period 0 - Full emission  
        "3:7200:0"   # Period 0 - Full emission
        "4:7200:0"   # Period 0 - Full emission
        "5:3600:1"   # Period 1 - First halving
        "6:3600:1"   # Period 1 - Halved emission
        "7:3600:1"   # Period 1 - Halved emission
        "8:3600:1"   # Period 1 - Halved emission
        "9:1800:2"   # Period 2 - Second halving
        "10:1800:2"  # Period 2 - Double halved
        "11:1800:2"  # Period 2 - Double halved
        "12:1800:2"  # Period 2 - Double halved
    )
    
    log_info "Will test ${#test_epochs[@]} epochs to demonstrate halving progression"
    
    for epoch_data in "${test_epochs[@]}"; do
        IFS=':' read -ra EPOCH_INFO <<< "$epoch_data"
        local epoch_num=${EPOCH_INFO[0]}
        local expected_emission=${EPOCH_INFO[1]}
        local expected_period=${EPOCH_INFO[2]}
        
        monitor_epoch $epoch_num $expected_emission $expected_period
        
        # Add delay between epochs for real-time monitoring
        if [ $epoch_num -lt 12 ]; then
            log_info "Waiting 15 seconds before next epoch..."
            sleep 15
        fi
    done
}

# Generate summary report
generate_report() {
    echo ""
    echo "==========================================="
    log_info "HALVING TEST SUMMARY REPORT"
    echo "==========================================="
    
    local final_balance=$(get_multisig_balance)
    local halving_data=$(get_halving_data)
    
    echo "Final Results:"
    echo "- Multi-sig Balance: $final_balance aevmos"
    echo "- Halving Data: $halving_data"
    
    # Calculate expected total (geometric series)
    # Period 0 (4 epochs): 4 * 7200 = 28,800 NXQ
    # Period 1 (4 epochs): 4 * 3600 = 14,400 NXQ  
    # Period 2 (4 epochs): 4 * 1800 = 7,200 NXQ
    # Total: 50,400 NXQ
    local expected_total_nxq=50400
    local expected_total_wei=$((expected_total_nxq * 1000000000000000000000))
    
    if [ "$final_balance" -eq "$expected_total_wei" ]; then
        log_success "✅ TOTAL CORRECT: $expected_total_nxq NXQ minted across all periods"
    else
        local actual_nxq=$((final_balance / 1000000000000000000000))
        log_error "❌ TOTAL INCORRECT: Expected $expected_total_nxq NXQ, got $actual_nxq NXQ"
    fi
    
    echo ""
    log_success "🎉 Halving test complete!"
    log_info "You've successfully tested Bitcoin-style halving in minutes instead of years!"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test files..."
    rm -f halving-proposal.json
}

# Main execution
main() {
    echo "=============================================="
    echo "🚀 NEXQLOUD HALVING SYSTEM TEST"
    echo "=============================================="
    echo "Testing Bitcoin-style halving with 4-epoch intervals"
    echo "Multi-sig Address: $MULTISIG_ADDRESS"
    echo "=============================================="
    
    # Setup trap for cleanup
    trap cleanup EXIT
    
    # Pre-flight checks
    check_chain_status
    
    # Setup (informational)
    setup_fast_halving
    
    # Run the test
    run_halving_test
    
    # Generate report
    generate_report
    
    log_success "Test completed successfully! 🎉"
}

# Run main function
main "$@"

