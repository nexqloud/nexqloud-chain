# Manual Testing Guide for Halving System

## 🎯 **Overview**

This guide provides step-by-step instructions for manually testing the Bitcoin-style halving system with **2-day halving intervals** (every 2 epochs) in your development environment.

**Testing Scenario:** 7200 NXQ daily emissions with halving every 2 days  
**Environment:** 3-node development setup  
**Duration:** 10 days (5 halving cycles)  
**Multi-sig Address:** `nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5`  

---

## 🏗️ **Pre-Testing Setup**

### **Step 1: Environment Verification**

```bash
# Verify chain is running
nxqd status

# Check node count
ps aux | grep nxqd | grep -v grep

# Verify epoch module
nxqd query epochs epoch-infos
```

### **Step 2: Configure 2-Day Halving**

#### **Option A: Genesis Configuration (Before Chain Start)**
```json
{
  "app_state": {
    "inflation": {
      "params": {
        "enable_inflation": true,
        "daily_emission": "7200000000000000000000",
        "halving_interval_epochs": 2,
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

#### **Option B: Governance Proposal (Live Chain)**
```bash
# Create proposal file
cat > 2day-halving-proposal.json << EOF
{
  "title": "Enable 2-Day Halving for Testing",
  "description": "Set halving interval to 2 epochs for testing cycles",
  "changes": [
    {
      "subspace": "inflation",
      "key": "HalvingIntervalEpochs",
      "value": "2"
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

# Submit proposal
nxqd tx gov submit-proposal param-change 2day-halving-proposal.json \
  --from validator \
  --chain-id nexqloud-dev-1 \
  --gas auto \
  --gas-adjustment 1.3 \
  --yes

# Vote on proposal
nxqd tx gov vote 1 yes --from validator --chain-id nexqloud-dev-1 --yes
```

---

## 📊 **Manual Testing Protocol**

### **Day 1-2 (Period 0): Full Emission Testing**

#### **Expected Behavior:**
- **Daily Emission:** 7200 NXQ
- **Total for Period:** 14,400 NXQ
- **Halving Period:** 0

#### **Test Steps:**

##### **Day 1 - Epoch 1**
```bash
# Record initial state
echo "=== Day 1 - Epoch 1 Testing ==="
date

# Check initial multi-sig balance
INITIAL_BALANCE=$(nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5 --output json | jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"')
echo "Initial Balance: $INITIAL_BALANCE aevmos"

# Check halving parameters
nxqd query inflation params | jq '{
  enable_inflation: .enable_inflation,
  daily_emission: .daily_emission,
  halving_interval_epochs: .halving_interval_epochs,
  multi_sig_address: .multi_sig_address
}'

# Check halving data
nxqd query inflation halving-data

# Wait for epoch end (or trigger manually in dev)
sleep 300  # Wait 5 minutes (adjust for your epoch duration)

# Check post-epoch state
NEW_BALANCE=$(nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5 --output json | jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"')
MINTED=$((NEW_BALANCE - INITIAL_BALANCE))
NXQ_MINTED=$((MINTED / 1000000000000000000000))

echo "New Balance: $NEW_BALANCE aevmos"
echo "Minted: $NXQ_MINTED NXQ"
echo "Expected: 7200 NXQ"

# Validation
if [ "$NXQ_MINTED" -eq 7200 ]; then
    echo "✅ Day 1 PASS: Correct emission"
else
    echo "❌ Day 1 FAIL: Wrong emission"
fi
```

##### **Day 2 - Epoch 2 (First Halving)**
```bash
echo "=== Day 2 - Epoch 2 Testing (First Halving) ==="
date

# Record pre-halving state
PRE_BALANCE=$(nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5 --output json | jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"')
echo "Pre-halving Balance: $PRE_BALANCE aevmos"

# Check halving data before
echo "Pre-halving Data:"
nxqd query inflation halving-data

# Wait for epoch end
sleep 300

# Check post-halving state
POST_BALANCE=$(nxqd query bank balances nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5 --output json | jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"')
MINTED=$((POST_BALANCE - PRE_BALANCE))
NXQ_MINTED=$((MINTED / 1000000000000000000000))

echo "Post-halving Balance: $POST_BALANCE aevmos"
echo "Minted: $NXQ_MINTED NXQ"
echo "Expected: 3600 NXQ (halved)"

# Check halving data after
echo "Post-halving Data:"
nxqd query inflation halving-data

# Validation
if [ "$NXQ_MINTED" -eq 3600 ]; then
    echo "✅ Day 2 PASS: First halving successful"
else
    echo "❌ Day 2 FAIL: Halving did not work"
fi
```

### **Day 3-4 (Period 1): First Halving Period**

#### **Expected Behavior:**
- **Daily Emission:** 3600 NXQ
- **Total for Period:** 7,200 NXQ
- **Halving Period:** 1

#### **Test Steps:**

##### **Day 3 - Epoch 3**
```bash
echo "=== Day 3 - Epoch 3 Testing (Period 1) ==="

# Test same pattern as Day 1, expect 3600 NXQ
# ... (similar testing pattern)

# Expected: 3600 NXQ emission
# Validation: Emission should remain halved
```

##### **Day 4 - Epoch 4 (Second Halving)**
```bash
echo "=== Day 4 - Epoch 4 Testing (Second Halving) ==="

# Expected: 1800 NXQ emission (halved again)
# Validation: Halving period should advance to 2
```

### **Day 5-6 (Period 2): Second Halving Period**

#### **Expected Behavior:**
- **Daily Emission:** 1800 NXQ
- **Total for Period:** 3,600 NXQ
- **Halving Period:** 2

### **Day 7-8 (Period 3): Third Halving Period**

#### **Expected Behavior:**
- **Daily Emission:** 900 NXQ
- **Total for Period:** 1,800 NXQ
- **Halving Period:** 3

### **Day 9-10 (Period 4): Fourth Halving Period**

#### **Expected Behavior:**
- **Daily Emission:** 450 NXQ
- **Total for Period:** 900 NXQ
- **Halving Period:** 4

---

## 📋 **Daily Testing Checklist**

### **Pre-Epoch Checks**
- [ ] Record current multi-sig balance
- [ ] Check current halving period
- [ ] Verify epoch number
- [ ] Note expected emission amount
- [ ] Check total supply

### **Post-Epoch Checks**
- [ ] Verify balance increased correctly
- [ ] Confirm emission amount matches expectation
- [ ] Check halving data updates
- [ ] Validate no tokens went to validators
- [ ] Verify total supply growth

### **Halving Event Checks (Every 2 Days)**
- [ ] Confirm halving period advanced
- [ ] Verify emission amount halved
- [ ] Check last halving epoch updated
- [ ] Validate events emitted
- [ ] Confirm mathematical precision

---

## 🧪 **Automated Monitoring Script**

```bash
#!/bin/bash
# daily-halving-monitor.sh

MULTISIG="nxq12zprcmal9hv52jqf2x4m59ztng0gnh7r96muj5"
LOG_FILE="halving-test-log.txt"

# Function to log with timestamp
log_entry() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a $LOG_FILE
}

# Function to get balance
get_balance() {
    nxqd query bank balances $MULTISIG --output json | jq -r '.balances[] | select(.denom=="aevmos") | .amount // "0"'
}

# Function to get halving data
get_halving_data() {
    nxqd query inflation halving-data --output json
}

# Function to calculate expected emission
calculate_expected_emission() {
    local period=$1
    local base_emission=7200
    local halved_emission=$base_emission
    
    for ((i=0; i<period; i++)); do
        halved_emission=$((halved_emission / 2))
    done
    
    echo $halved_emission
}

# Main monitoring loop
monitor_daily() {
    local day=$1
    local epoch=$2
    
    log_entry "=== Day $day - Epoch $epoch Monitoring ==="
    
    # Pre-epoch state
    local initial_balance=$(get_balance)
    local halving_data=$(get_halving_data)
    local current_period=$(echo $halving_data | jq -r '.current_period')
    local expected_emission=$(calculate_expected_emission $current_period)
    
    log_entry "Initial Balance: $initial_balance aevmos"
    log_entry "Current Period: $current_period"
    log_entry "Expected Emission: $expected_emission NXQ"
    
    # Wait for epoch processing
    log_entry "Waiting for epoch $epoch to complete..."
    sleep 300
    
    # Post-epoch state
    local final_balance=$(get_balance)
    local new_halving_data=$(get_halving_data)
    local new_period=$(echo $new_halving_data | jq -r '.current_period')
    
    # Calculate results
    local minted=$((final_balance - initial_balance))
    local nxq_minted=$((minted / 1000000000000000000000))
    
    log_entry "Final Balance: $final_balance aevmos"
    log_entry "Minted: $nxq_minted NXQ"
    log_entry "New Period: $new_period"
    
    # Validation
    if [ "$nxq_minted" -eq "$expected_emission" ]; then
        log_entry "✅ Day $day PASS: Correct emission"
    else
        log_entry "❌ Day $day FAIL: Expected $expected_emission, got $nxq_minted"
    fi
    
    # Check for halving event
    if [ "$new_period" -gt "$current_period" ]; then
        log_entry "🎉 HALVING EVENT! Advanced from Period $current_period to Period $new_period"
        local next_expected=$(calculate_expected_emission $new_period)
        log_entry "🎉 Next emission: $next_expected NXQ"
    fi
    
    log_entry "---"
}

# Run 10-day test
for day in {1..10}; do
    monitor_daily $day $day
done

log_entry "10-day halving test completed!"
```

---

## 📊 **Expected Results Summary**

### **10-Day Halving Schedule**

| Day | Epoch | Period | Emission (NXQ) | Cumulative (NXQ) | Event |
|-----|-------|--------|----------------|-------------------|-------|
| 1   | 1     | 0      | 7,200          | 7,200             | -     |
| 2   | 2     | 0→1    | 3,600          | 10,800            | 1st Halving |
| 3   | 3     | 1      | 3,600          | 14,400            | -     |
| 4   | 4     | 1→2    | 1,800          | 16,200            | 2nd Halving |
| 5   | 5     | 2      | 1,800          | 18,000            | -     |
| 6   | 6     | 2→3    | 900            | 18,900            | 3rd Halving |
| 7   | 7     | 3      | 900            | 19,800            | -     |
| 8   | 8     | 3→4    | 450            | 20,250            | 4th Halving |
| 9   | 9     | 4      | 450            | 20,700            | -     |
| 10  | 10    | 4→5    | 225            | 20,925            | 5th Halving |

### **Key Validation Points**

✅ **Total Minted:** 20,925 NXQ after 10 days  
✅ **Halving Events:** 5 halvings triggered correctly  
✅ **Mathematical Precision:** Each halving exactly 50% of previous  
✅ **Multi-sig Distribution:** 100% to business address  
✅ **Supply Cap:** Well under 21M limit  
✅ **State Persistence:** Halving data preserved across epochs  

---

## 🚨 **Troubleshooting Guide**

### **Common Issues**

#### **Issue 1: No Minting Occurring**
```bash
# Check inflation enabled
nxqd query inflation params | jq '.enable_inflation'

# Check epoch identifier
nxqd query epochs current-epoch day

# Check multi-sig address set
nxqd query inflation params | jq '.multi_sig_address'
```

#### **Issue 2: Wrong Emission Amounts**
```bash
# Verify halving interval
nxqd query inflation params | jq '.halving_interval_epochs'

# Check current period calculation
nxqd query inflation halving-data

# Verify daily emission base
nxqd query inflation params | jq '.daily_emission'
```

#### **Issue 3: Halving Not Triggering**
```bash
# Check epoch progression
nxqd query epochs epoch-infos | jq '.epochs[] | select(.identifier=="day")'

# Verify halving calculation
nxqd query inflation next-halving

# Check halving data updates
nxqd query inflation halving-data
```

---

## 📈 **Success Criteria**

### **Testing Passes If:**

✅ **Daily Emissions:** Exact amounts minted each day  
✅ **Halving Events:** Emissions halve every 2 days  
✅ **Period Progression:** Halving periods advance correctly  
✅ **Multi-sig Distribution:** 100% tokens to business address  
✅ **State Persistence:** Data survives across epochs  
✅ **Mathematical Accuracy:** No rounding errors  
✅ **Performance:** System handles daily operations smoothly  
✅ **Supply Cap:** Minting respects 21M limit  

### **Testing Fails If:**

❌ **Wrong Amounts:** Emission amounts incorrect  
❌ **No Halving:** Halving events don't trigger  
❌ **Validator Distribution:** Tokens go to validators  
❌ **State Loss:** Halving data resets  
❌ **Math Errors:** Precision problems  
❌ **Performance Issues:** System slows or fails  
❌ **Cap Violation:** Minting exceeds 21M  

---

## 🎯 **Conclusion**

This manual testing guide provides comprehensive validation of the halving system in a realistic development environment. The 2-day halving cycle allows for rapid testing of multiple halving events while maintaining mathematical precision and system reliability.

**Next Step:** Production deployment with 4-year halving intervals after successful manual testing validation.

