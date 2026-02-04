#!/bin/bash

# Test script to verify Personal API is disabled
# This tests if eth_sendTransaction and personal_* methods are accessible

echo "=========================================="
echo "Testing Personal API Access"
echo "=========================================="
echo ""

# JSON-RPC endpoint
RPC_URL="http://localhost:8545"

# Test account (you'll need to use a real account from your keyring)
TEST_ADDRESS="0x1234567890123456789012345678901234567890"
TEST_TO="0x0987654321098765432109876543210987654321"

echo "1. Testing eth_sendTransaction (should fail - requires personal API)"
echo "------------------------------------------------------------------"
curl -X POST $RPC_URL \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_sendTransaction",
    "params": [{
      "from": "'$TEST_ADDRESS'",
      "to": "'$TEST_TO'",
      "value": "0x1",
      "gas": "0x5208"
    }],
    "id": 1
  }' | jq .
echo ""
echo ""

echo "2. Testing personal_unlockAccount (should fail - personal API disabled)"
echo "------------------------------------------------------------------------"
curl -X POST $RPC_URL \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "personal_unlockAccount",
    "params": ["'$TEST_ADDRESS'", "password", 300],
    "id": 2
  }' | jq .
echo ""
echo ""

echo "3. Testing personal_sendTransaction (should fail - personal API disabled)"
echo "--------------------------------------------------------------------------"
curl -X POST $RPC_URL \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "personal_sendTransaction",
    "params": [{
      "from": "'$TEST_ADDRESS'",
      "to": "'$TEST_TO'",
      "value": "0x1"
    }, "password"],
    "id": 3
  }' | jq .
echo ""
echo ""

echo "4. Testing eth_accounts (should work - not part of personal API)"
echo "-----------------------------------------------------------------"
curl -X POST $RPC_URL \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_accounts",
    "params": [],
    "id": 4
  }' | jq .
echo ""
echo ""

echo "5. Testing eth_blockNumber (should work - basic eth API)"
echo "---------------------------------------------------------"
curl -X POST $RPC_URL \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 5
  }' | jq .
echo ""
echo ""

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "✓ Tests 1-3 should return 'method not found' or similar errors"
echo "✓ Tests 4-5 should return valid responses"
echo ""
echo "If personal API is properly disabled:"
echo "  - eth_sendTransaction: Error (requires unlocked account)"
echo "  - personal_unlockAccount: Method not found"
echo "  - personal_sendTransaction: Method not found"
echo "  - eth_accounts: Empty array or valid response"
echo "  - eth_blockNumber: Valid block number"
echo "=========================================="
