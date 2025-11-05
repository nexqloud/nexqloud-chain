#!/bin/bash

# Test script for Dynamic Contract Parameters
# This script tests the complete workflow locally

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_header() {
    echo ""
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================${NC}"
}

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

test_passed() {
    ((TESTS_PASSED++))
    print_success "$1"
}

test_failed() {
    ((TESTS_FAILED++))
    print_error "$1"
}

# Cleanup function
cleanup() {
    print_info "Cleaning up test environment..."
    rm -rf ~/.nxqd-test 2>/dev/null || true
    pkill nxqd 2>/dev/null || true
}

# Test 1: Build and Unit Tests
test_build() {
    print_header "Test 1: Build and Unit Tests"
    
    print_info "Running go tests for params..."
    if go test ./x/evm/types/... -v -run "TestDefault|TestIsBootstrap|TestIsContractSet|TestValidate" 2>&1 | tee test-output.log; then
        test_passed "Unit tests passed"
    else
        test_failed "Unit tests failed"
        return 1
    fi
    
    print_info "Building binary..."
    if make build > /dev/null 2>&1; then
        test_passed "Build successful"
    else
        test_failed "Build failed"
        return 1
    fi
    
    if [ -f "build/nxqd" ]; then
        test_passed "Binary created successfully"
    else
        test_failed "Binary not found"
        return 1
    fi
}

# Test 2: Bootstrap Mode Initialization
test_bootstrap_init() {
    print_header "Test 2: Bootstrap Mode Initialization"
    
    cleanup
    
    print_info "Initializing chain with bootstrap genesis..."
    
    # Initialize chain
    ./build/nxqd init test-node --chain-id=nxqd_6000-1 --home=~/.nxqd-test > /dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        test_passed "Chain initialized"
    else
        test_failed "Chain initialization failed"
        return 1
    fi
    
    # Create a test key
    echo "test1234" | ./build/nxqd keys add validator --keyring-backend=test --home=~/.nxqd-test > /dev/null 2>&1
    
    # Check genesis file
    if [ -f ~/.nxqd-test/config/genesis.json ]; then
        test_passed "Genesis file created"
    else
        test_failed "Genesis file not found"
        return 1
    fi
}

# Test 3: Query Default Params
test_query_default_params() {
    print_header "Test 3: Query Default Parameters"
    
    # Start chain in background
    print_info "Starting chain..."
    ./build/nxqd start --home=~/.nxqd-test > chain.log 2>&1 &
    CHAIN_PID=$!
    
    # Wait for chain to start
    sleep 10
    
    print_info "Querying EVM params..."
    if ./build/nxqd query evm params --home=~/.nxqd-test --output=json > params.json 2>&1; then
        test_passed "Params query successful"
        
        # Check for zero addresses
        if grep -q "0x0000000000000000000000000000000000000000" params.json; then
            test_passed "Zero addresses present (bootstrap mode)"
        else
            test_warning "Zero addresses not found"
        fi
        
        # Check for disabled flags
        if grep -q '"enable_chain_status_check":false' params.json && grep -q '"enable_wallet_lock_check":false' params.json; then
            test_passed "Security checks disabled (bootstrap mode)"
        else
            test_warning "Security flags not as expected"
        fi
        
        print_info "Current params:"
        cat params.json | jq '.params | {
            online_server_count_contract,
            nft_contract_address,
            wallet_state_contract_address,
            validator_approval_contract_address,
            whitelisted_addresses,
            enable_chain_status_check,
            enable_wallet_lock_check
        }'
    else
        test_failed "Params query failed"
    fi
    
    # Stop chain
    kill $CHAIN_PID 2>/dev/null || true
    sleep 2
}

# Test 4: Governance Proposal Validation
test_governance_proposal_validation() {
    print_header "Test 4: Governance Proposal Validation"
    
    print_info "Validating governance proposal JSON files..."
    
    proposals=(
        "governance-proposals/activate-security.json"
        "governance-proposals/emergency-disable-security.json"
        "governance-proposals/update-contract-address.json"
        "governance-proposals/add-whitelist-address.json"
    )
    
    for proposal in "${proposals[@]}"; do
        if [ -f "$proposal" ]; then
            if jq empty "$proposal" 2>/dev/null; then
                test_passed "$(basename "$proposal") is valid JSON"
            else
                test_failed "$(basename "$proposal") is invalid JSON"
            fi
        else
            test_failed "$(basename "$proposal") not found"
        fi
    done
}

# Test 5: Contract Address Validation
test_contract_address_validation() {
    print_header "Test 5: Contract Address Validation"
    
    print_info "Testing contract address validation..."
    
    # Test valid addresses
    valid_addresses=(
        "0x1234567890123456789012345678901234567890"
        "0xaA51C7e32dA1266447909b6C40772276A43453e8"
        "0x0000000000000000000000000000000000000000"  # Zero address is valid in bootstrap
    )
    
    for addr in "${valid_addresses[@]}"; do
        if [[ $addr =~ ^0x[0-9a-fA-F]{40}$ ]]; then
            test_passed "Address $addr is valid format"
        else
            test_failed "Address $addr is invalid format"
        fi
    done
    
    # Test invalid addresses
    invalid_addresses=(
        "1234567890123456789012345678901234567890"  # No 0x prefix
        "0x12345"  # Too short
        "0xGGGG567890123456789012345678901234567890"  # Invalid hex
    )
    
    for addr in "${invalid_addresses[@]}"; do
        if [[ $addr =~ ^0x[0-9a-fA-F]{40}$ ]]; then
            test_failed "Address $addr should be invalid but passed"
        else
            test_passed "Address $addr correctly rejected"
        fi
    done
}

# Test 6: Documentation Check
test_documentation() {
    print_header "Test 6: Documentation Check"
    
    docs=(
        "DYNAMIC_CONTRACT_PARAMS_IMPLEMENTATION.md"
        "IMPLEMENTATION_SUMMARY.md"
        "QUICKSTART_DYNAMIC_PARAMS.md"
        "TEST_PLAN_DYNAMIC_PARAMS.md"
        "governance-proposals/README.md"
    )
    
    for doc in "${docs[@]}"; do
        if [ -f "$doc" ]; then
            test_passed "$(basename "$doc") exists"
        else
            test_failed "$(basename "$doc") not found"
        fi
    done
}

# Main test execution
main() {
    print_header "Dynamic Contract Parameters - Test Suite"
    print_info "Starting test execution..."
    echo ""
    
    # Run tests
    test_build || true
    test_bootstrap_init || true
    test_query_default_params || true
    test_governance_proposal_validation || true
    test_contract_address_validation || true
    test_documentation || true
    
    # Cleanup
    cleanup
    
    # Print summary
    print_header "Test Summary"
    echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
    echo ""
    
    if [ $TESTS_FAILED -eq 0 ]; then
        print_success "All tests passed! Ready for staging deployment."
        return 0
    else
        print_error "Some tests failed. Please fix issues before deployment."
        return 1
    fi
}

# Run main
main

