# Halving System Testing Guide

## 🧪 **Phase 1: Local Development Testing**

### **1.1 Unit Test Execution**
```bash
# Test all halving calculation functions
go test ./x/inflation/v1/types -run TestHalvingCalculationTestSuite -v

# Test genesis state handling
go test ./x/inflation/v1/types -run TestHalvingGenesisTestSuite -v

# Test integration scenarios
go test ./x/inflation/v1/keeper -run TestHalvingIntegration -v
```

### **1.2 Build and Compile Test**
```bash
# Ensure everything compiles
make build

# Test specific inflation module
go build ./x/inflation/...
```

---

## 🚀 **Phase 2: Local Chain Deployment Testing**

### **2.1 Initialize Local Testnet**
```bash
# Initialize the chain
nxqd init test-validator --chain-id nexqloud-test-1

# Create genesis with halving parameters
nxqd add-genesis-account nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5 1000000000000000000000000aevmos
nxqd gentx test-validator 1000000000000000000000aevmos --chain-id nexqloud-test-1
nxqd collect-gentxs
```

### **2.2 Configure Genesis with Halving Parameters**
```json
{
  "app_state": {
    "inflation": {
      "params": {
        "mint_denom": "aevmos",
        "enable_inflation": true,
        "daily_emission": "7200000000000000000000",
        "halving_interval_epochs": 4,
        "multi_sig_address": "nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5",
        "max_supply": "21000000000000000000000000"
      },
      "halving_data": {
        "current_period": 0,
        "last_halving_epoch": 0,
        "start_epoch": 1
      }
    }
  }
}
```

### **2.3 Start Local Chain**
```bash
# Start the chain
nxqd start --log_level debug

# In another terminal, check logs for halving events
tail -f ~/.nxqd/logs/nxqd.log | grep -i halving
```

---

## 🔍 **Phase 3: Halving System Verification**

### **3.1 Check Initial State**
```bash
# Query current inflation parameters
nxqd query inflation params

# Query current halving data
nxqd query inflation halving-data

# Check multi-sig balance
nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5
```

### **3.2 Trigger Epoch End (Manual Testing)**
```bash
# Force epoch end to trigger halving minting
nxqd tx epochs end-epoch day --from test-validator --chain-id nexqloud-test-1

# Check if tokens were minted to multi-sig
nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5

# Check total supply
nxqd query bank total
```

### **3.3 Verify Halving Progression**
```bash
# Check halving data after multiple epochs
nxqd query inflation halving-data

# Expected progression:
# Epoch 1-4: Period 0, 7200 tokens/day
# Epoch 5-8: Period 1, 3600 tokens/day  
# Epoch 9-12: Period 2, 1800 tokens/day
```

---

## 🌐 **Phase 4: Testnet Deployment**

### **4.1 Deploy to Testnet**
```bash
# Submit governance proposal to enable halving
nxqd tx gov submit-proposal param-change halving-proposal.json \
  --from validator \
  --chain-id nexqloud-testnet-1

# Vote on proposal
nxqd tx gov vote 1 yes --from validator --chain-id nexqloud-testnet-1
```

### **4.2 Monitor Real-Time Halving**
```bash
# Watch for halving events in real-time
nxqd query inflation events --follow

# Check daily emission amounts
nxqd query inflation current-emission

# Monitor multi-sig balance growth
watch -n 10 'nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5'
```

---

## 📊 **Phase 5: Production Verification**

### **5.1 Key Metrics to Monitor**

#### **Daily Checks:**
```bash
# 1. Verify daily minting amount
nxqd query inflation current-emission
# Expected: 7200 NXQ (period 0), 3600 NXQ (period 1), etc.

# 2. Check multi-sig balance increase
nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5
# Should increase by exactly daily emission amount

# 3. Verify total supply growth
nxqd query bank total
# Should grow by daily emission each day

# 4. Check halving data progression
nxqd query inflation halving-data
# CurrentPeriod should advance every 4 years
```

#### **Weekly Checks:**
```bash
# 1. Verify no tokens going to validators/community pool
nxqd query bank balances $(nxqd query auth module-accounts | jq -r '.accounts[] | select(.name=="fee_collector") | .base_account.address')
# Should be minimal (only transaction fees)

# 2. Check supply cap enforcement
nxqd query inflation params | jq '.max_supply'
# Should be 21000000000000000000000000 (21M tokens)

# 3. Verify halving schedule
nxqd query inflation next-halving
# Should show next halving epoch
```

### **5.2 Automated Monitoring Script**
```bash
#!/bin/bash
# halving-monitor.sh

echo "=== Halving System Health Check ==="
echo "Date: $(date)"
echo ""

# Check current emission
echo "Current Daily Emission:"
nxqd query inflation current-emission

# Check multi-sig balance
echo ""
echo "Multi-Sig Balance:"
nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5

# Check halving data
echo ""
echo "Halving Data:"
nxqd query inflation halving-data

# Check total supply
echo ""
echo "Total Supply:"
nxqd query bank total | grep aevmos

# Check if inflation is enabled
echo ""
echo "Inflation Status:"
nxqd query inflation params | jq '.enable_inflation'
```

---

## 🚨 **Phase 6: Edge Case Testing**

### **6.1 Supply Cap Testing**
```bash
# Test near supply cap (20.9M tokens)
# Should stop minting when approaching 21M limit
```

### **6.2 Network Partition Testing**
```bash
# Test halving continues during validator downtime
# Verify state persistence across restarts
```

### **6.3 Upgrade Testing**
```bash
# Test halving state preservation across chain upgrades
# Verify genesis export/import functionality
```

---

## 📈 **Expected Results**

### **✅ Success Indicators:**
1. **Daily Minting**: Exactly 7200 NXQ minted per day (period 0)
2. **Multi-Sig Growth**: Balance increases by daily emission amount
3. **No Validator Rewards**: Validators only get transaction fees
4. **Halving Progression**: Emission halves every 4 years (1461 epochs)
5. **Supply Cap**: Minting stops at 21M token limit
6. **State Persistence**: Halving data survives restarts

### **❌ Failure Indicators:**
1. **Wrong Amounts**: Not exactly 7200 NXQ per day
2. **Validator Rewards**: Tokens going to validators/community
3. **No Minting**: Multi-sig balance not increasing
4. **State Loss**: Halving data reset after restart
5. **Supply Overflow**: Minting beyond 21M limit

---

## 🔧 **Troubleshooting Commands**

```bash
# Check inflation module logs
nxqd query inflation logs --level debug

# Verify epoch status
nxqd query epochs current-epoch day

# Check for errors
nxqd query inflation errors

# Validate parameters
nxqd query inflation validate-params
```

This comprehensive testing approach ensures your halving system works perfectly in production! 🚀

