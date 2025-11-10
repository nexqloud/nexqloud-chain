# Final Implementation Plan: Epoch-Based Halving System

## 🎯 Executive Summary

Daily Epoch Ends → AfterEpochEnd Hook → Our Halving Logic → Mint Tokens → Send to Multi-sig
     ↑                    ↑                    ↑               ↑              ↑
  (Existing)         (Existing Hook)     (Our New Logic)  (Existing)    (Our Addition)

This document provides the **final implementation plan** for Bitcoin-style halving using the existing **epoch infrastructure**. This approach leverages daily epochs (`DayEpochID`) already in the system, making it simpler and safer than block-based implementation.

## ✅ Current State Analysis

### **What We Have:**
- ✅ **Inflation Module**: Currently DISABLED (`DefaultInflation = false`)
- ✅ **Daily Epochs**: `DayEpochID` running every 24 hours automatically
- ✅ **Supply Cap**: 21M token limit already implemented
- ✅ **Hook System**: `AfterEpochEnd` already exists in inflation keeper
- ✅ **Validators**: Already sustainable on transaction fees alone
- ✅ **Clean Architecture**: No conflicts since inflation is disabled

### **What We Need:**
- 🔧 **Halving Logic**: Calculate emission based on epoch count instead of complex math
- 🔧 **Multi-sig Distribution**: Route all minted tokens to business address
- 🔧 **Supply Validation**: Enforce 21M cap before minting

## 🧮 **Mathematical Model (Epoch-Based)**

```
Daily Epochs = 365 per year
Halving Interval = 4 years × 365 days = 1,460 epochs

Epoch-Based Halving Schedule:
- Epochs 1-1,460 (Years 0-4): 3,600 tokens/day
- Epochs 1,461-2,920 (Years 4-8): 1,800 tokens/day  
- Epochs 2,921-4,380 (Years 8-12): 900 tokens/day
- Epochs 4,381-5,840 (Years 12-16): 450 tokens/day
- ...continues until cap reached

Total Supply Convergence:
Total = 7200 × 365 × 4 × (1 + 1/2 + 1/4 + 1/8 + ...)
Total = 5,256,000 × 4 = 21,024,000 tokens
```

## 🛠️ **Implementation Plan**

### **Phase 1: Update Data Structures (Week 1)**

#### 1.1 Update Protocol Buffers
**File**: `proto/evmos/inflation/v1/genesis.proto`

```protobuf
message HalvingData {
  // current_halving_period defines the current halving period (0, 1, 2, ...)
  uint64 current_halving_period = 1;
  
  // last_halving_epoch defines the epoch number when last halving occurred
  int64 last_halving_epoch = 2;
  
  // start_epoch defines the epoch when halving system starts
  int64 start_epoch = 3;
}

message Params {
  // ... existing fields ...
  
  // daily_emission defines tokens minted per day (before halving)
  string daily_emission = 8 [
    (cosmos_proto.scalar) = "cosmos.Int",
    (gogoproto.customtype) = "cosmossdk.io/math.Int",
    (gogoproto.nullable) = false
  ];
  
  // halving_interval_epochs defines epochs between halvings (1460 = 4 years)
  uint64 halving_interval_epochs = 9;
  
  // multi_sig_address for receiving minted tokens (optional)
  string multi_sig_address = 10;
  
  // max_supply defines the maximum token supply (21M)
  string max_supply = 11 [
    (cosmos_proto.scalar) = "cosmos.Int",
    (gogoproto.customtype) = "cosmossdk.io/math.Int",
    (gogoproto.nullable) = false
  ];
}

message GenesisState {
  // ... existing fields ...
  
  // halving_data stores halving-related state
  HalvingData halving_data = 6 [(gogoproto.nullable) = false];
}
```

#### 1.2 Generate Go Code
```bash
cd /Users/saswatapatra/work/nexqloud/nexqloud-chain
make proto-gen
```

### **Phase 2: Implement Core Logic (Week 2)**

#### 2.1 Update Parameters
**File**: `x/inflation/v1/types/params.go`

```go
// Default halving parameters
const (
    // 7200 tokens per day initially
    DefaultDailyEmission = "7200000000000000000000" // 7200 * 10^18
    
    // Halving every 4 years = 1460 daily epochs
    DefaultHalvingIntervalEpochs = uint64(1460)
    
    // Maximum supply of 21 million tokens
    DefaultMaxSupply = "21000000000000000000000000" // 21M * 10^18
)

func DefaultParams() Params {
    return Params{
        MintDenom:                evm.DefaultEVMDenom,
        ExponentialCalculation:   ExponentialCalculation{...}, // Keep existing
        InflationDistribution:    InflationDistribution{...}, // Keep existing
        EnableInflation:          true, // ⚠️ ENABLE for halving
        DailyEmission:           math.NewIntFromString(DefaultDailyEmission),
        HalvingIntervalEpochs:   DefaultHalvingIntervalEpochs,
        MultiSigAddress:         "", // Set via governance
        MaxSupply:              math.NewIntFromString(DefaultMaxSupply),
    }
}
```

#### 2.2 Halving Calculation Logic
**File**: `x/inflation/v1/types/halving_calculation.go` (NEW)

```go
package types

import (
    "cosmossdk.io/math"
    epochstypes "github.com/evmos/evmos/v19/x/epochs/types"
)

// CalculateHalvingPeriod determines current halving period from epoch number
func CalculateHalvingPeriod(currentEpoch, startEpoch int64, halvingInterval uint64) uint64 {
    if currentEpoch < startEpoch {
        return 0
    }
    
    epochsSinceStart := uint64(currentEpoch - startEpoch)
    return epochsSinceStart / halvingInterval
}

// CalculateDailyEmission returns daily emission amount considering halving
func CalculateDailyEmission(params Params, currentHalvingPeriod uint64) math.Int {
    dailyEmission := params.DailyEmission
    
    // Apply halving: divide by 2^period
    if currentHalvingPeriod > 0 {
        // Calculate 2^currentHalvingPeriod
        divisor := math.NewInt(1)
        for i := uint64(0); i < currentHalvingPeriod; i++ {
            divisor = divisor.Mul(math.NewInt(2))
        }
        dailyEmission = dailyEmission.Quo(divisor)
    }
    
    return dailyEmission
}

// ValidateSupplyCap checks if minting would exceed max supply
func ValidateSupplyCap(currentSupply, mintAmount, maxSupply math.Int) error {
    newSupply := currentSupply.Add(mintAmount)
    if newSupply.GT(maxSupply) {
        return fmt.Errorf(
            "minting %s would exceed max supply %s (current: %s)",
            mintAmount.String(),
            maxSupply.String(), 
            currentSupply.String(),
        )
    }
    return nil
}

// ShouldHalvingOccur checks if halving should happen at this epoch
func ShouldHalvingOccur(currentEpoch, startEpoch int64, halvingInterval uint64, lastHalvingEpoch int64) bool {
    if currentEpoch < startEpoch {
        return false
    }
    
    epochsSinceStart := uint64(currentEpoch - startEpoch)
    
    // Check if we've crossed a halving boundary
    currentPeriod := epochsSinceStart / halvingInterval
    expectedHalvingEpoch := startEpoch + int64(currentPeriod * halvingInterval)
    
    return currentEpoch == expectedHalvingEpoch && currentEpoch > lastHalvingEpoch
}
```

### **Phase 3: Modify Epoch Hook (Week 3)**

#### 3.1 Update AfterEpochEnd Logic
**File**: `x/inflation/v1/keeper/hooks.go`

```go
// AfterEpochEnd implements halving-based minting at end of each daily epoch
func (k Keeper) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
    params := k.GetParams(ctx)
    
    // Skip if inflation disabled
    if !params.EnableInflation {
        k.Logger(ctx).Debug("inflation disabled, skipping mint")
        return
    }

    // Only process daily epochs
    if epochIdentifier != epochstypes.DayEpochID {
        return
    }

    halvingData := k.GetHalvingData(ctx)
    
    // Initialize start epoch if this is the first run
    if halvingData.StartEpoch == 0 {
        halvingData.StartEpoch = epochNumber
        k.SetHalvingData(ctx, halvingData)
    }

    // Check for halving event
    if ShouldHalvingOccur(
        epochNumber, 
        halvingData.StartEpoch, 
        params.HalvingIntervalEpochs,
        halvingData.LastHalvingEpoch,
    ) {
        // HALVING EVENT!
        oldPeriod := halvingData.CurrentHalvingPeriod
        halvingData.CurrentHalvingPeriod++
        halvingData.LastHalvingEpoch = epochNumber
        
        k.Logger(ctx).Info(
            "🚨 HALVING EVENT OCCURRED",
            "epoch", epochNumber,
            "old-period", oldPeriod,
            "new-period", halvingData.CurrentHalvingPeriod,
        )
        
        // Emit halving event
        ctx.EventManager().EmitEvent(
            sdk.NewEvent(
                types.EventTypeHalving,
                sdk.NewAttribute("epoch_number", fmt.Sprintf("%d", epochNumber)),
                sdk.NewAttribute("halving_period", fmt.Sprintf("%d", halvingData.CurrentHalvingPeriod)),
                sdk.NewAttribute("previous_period", fmt.Sprintf("%d", oldPeriod)),
            ),
        )
        
        k.SetHalvingData(ctx, halvingData)
    }

    // Calculate current period and daily emission
    currentPeriod := CalculateHalvingPeriod(
        epochNumber, 
        halvingData.StartEpoch, 
        params.HalvingIntervalEpochs,
    )
    
    dailyEmission := CalculateDailyEmission(params, currentPeriod)
    
    if !dailyEmission.IsPositive() {
        k.Logger(ctx).Debug("zero emission calculated, skipping mint")
        return
    }

    // Check supply cap BEFORE minting
    currentSupply := k.bankKeeper.GetSupply(ctx, params.MintDenom)
    if err := ValidateSupplyCap(currentSupply.Amount, dailyEmission, params.MaxSupply); err != nil {
        k.Logger(ctx).Error("supply cap reached, stopping inflation", "error", err)
        
        // Disable inflation permanently when cap reached
        params.EnableInflation = false
        k.SetParams(ctx, params)
        return
    }

    // Mint tokens
    mintedCoin := sdk.Coin{
        Denom:  params.MintDenom,
        Amount: dailyEmission,
    }

    if err := k.MintCoins(ctx, mintedCoin); err != nil {
        k.Logger(ctx).Error("failed to mint coins", "error", err)
        return
    }

    // Send to multi-sig or handle via standard distribution
    if params.MultiSigAddress != "" {
        // Send directly to multi-sig
        if err := k.SendMintedCoinsToMultiSig(ctx, mintedCoin, params.MultiSigAddress); err != nil {
            k.Logger(ctx).Error("failed to send to multi-sig", "error", err)
            return
        }
    } else {
        // Use standard distribution (can be 100% to community pool)
        staking, communityPool, err := k.MintAndAllocateInflation(ctx, mintedCoin, params)
        if err != nil {
            k.Logger(ctx).Error("failed to allocate inflation", "error", err)
            return
        }
        
        k.Logger(ctx).Info("allocated inflation",
            "staking", staking.String(),
            "community_pool", communityPool.String(),
        )
    }

    // Emit mint event
    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            types.EventTypeMint,
            sdk.NewAttribute(types.AttributeEpochNumber, fmt.Sprintf("%d", epochNumber)),
            sdk.NewAttribute(types.AttributeKeyEpochProvisions, dailyEmission.String()),
            sdk.NewAttribute(sdk.AttributeKeyAmount, mintedCoin.Amount.String()),
            sdk.NewAttribute("halving_period", fmt.Sprintf("%d", currentPeriod)),
        ),
    )

    k.Logger(ctx).Info("minted daily tokens",
        "epoch", epochNumber,
        "amount", dailyEmission.String(),
        "halving_period", currentPeriod,
        "recipient", func() string {
            if params.MultiSigAddress != "" {
                return params.MultiSigAddress
            }
            return "standard_distribution"
        }(),
    )
}
```

#### 3.2 Add Helper Functions
**File**: `x/inflation/v1/keeper/halving.go` (NEW)

```go
package keeper

import (
    sdk "github.com/cosmos/cosmos-sdk/types"
    "github.com/evmos/evmos/v19/x/inflation/v1/types"
)

// GetHalvingData returns the halving data from store
func (k Keeper) GetHalvingData(ctx sdk.Context) types.HalvingData {
    store := ctx.KVStore(k.storeKey)
    bz := store.Get(types.KeyPrefixHalvingData)
    if bz == nil {
        return types.HalvingData{
            CurrentHalvingPeriod: 0,
            LastHalvingEpoch:     0,
            StartEpoch:           0,
        }
    }

    var halvingData types.HalvingData
    k.cdc.MustUnmarshal(bz, &halvingData)
    return halvingData
}

// SetHalvingData sets the halving data in store
func (k Keeper) SetHalvingData(ctx sdk.Context, halvingData types.HalvingData) {
    store := ctx.KVStore(k.storeKey)
    bz := k.cdc.MustMarshal(&halvingData)
    store.Set(types.KeyPrefixHalvingData, bz)
}

// SendMintedCoinsToMultiSig sends minted coins directly to multi-sig address
func (k Keeper) SendMintedCoinsToMultiSig(ctx sdk.Context, coin sdk.Coin, multiSigAddr string) error {
    // Convert string to AccAddress
    addr, err := sdk.AccAddressFromBech32(multiSigAddr)
    if err != nil {
        return fmt.Errorf("invalid multi-sig address %s: %w", multiSigAddr, err)
    }

    // Send from module account to multi-sig
    return k.bankKeeper.SendCoinsFromModuleToAccount(
        ctx,
        types.ModuleName,
        addr,
        sdk.NewCoins(coin),
    )
}

// MintCoins mints new coins and adds them to module account
func (k Keeper) MintCoins(ctx sdk.Context, coin sdk.Coin) error {
    return k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(coin))
}
```

### **Phase 4: Update Constants & Keys (Week 4)**

#### 4.1 Add New Keys
**File**: `x/inflation/v1/types/keys.go`

```go
const (
    // ... existing keys ...
    
    // KeyPrefixHalvingData defines the store key for halving data
    KeyPrefixHalvingData = "halving_data"
)

// KeyPrefixHalvingData returns the key for halving data
var (
    KeyPrefixHalvingData = []byte{0x05}
)
```

#### 4.2 Add Event Types
**File**: `x/inflation/v1/types/events.go`

```go
// Event types for inflation module
const (
    // ... existing events ...
    
    EventTypeHalving = "halving"
)

// Event attribute keys
const (
    // ... existing attributes ...
    
    AttributeHalvingPeriod = "halving_period"
    AttributePreviousPeriod = "previous_period"
)
```

### **Phase 5: Testing & Deployment (Week 5)**

#### 5.1 Unit Tests
**File**: `x/inflation/v1/keeper/halving_test.go` (NEW)

```go
func TestHalvingCalculation(t *testing.T) {
    // Test cases for halving logic
    tests := []struct {
        name           string
        epochNumber    int64
        startEpoch     int64
        halvingInterval uint64
        expectedPeriod uint64
    }{
        {"before start", 100, 200, 1460, 0},
        {"period 0", 500, 200, 1460, 0},
        {"first halving", 1660, 200, 1460, 1},
        {"second halving", 3120, 200, 1460, 2},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            period := types.CalculateHalvingPeriod(
                tt.epochNumber, 
                tt.startEpoch, 
                tt.halvingInterval,
            )
            require.Equal(t, tt.expectedPeriod, period)
        })
    }
}

func TestDailyEmissionCalculation(t *testing.T) {
    params := types.DefaultParams()
    
    // Period 0: full emission
    emission0 := types.CalculateDailyEmission(params, 0)
    require.Equal(t, params.DailyEmission, emission0)
    
    // Period 1: half emission
    emission1 := types.CalculateDailyEmission(params, 1)
    expected1 := params.DailyEmission.Quo(math.NewInt(2))
    require.Equal(t, expected1, emission1)
    
    // Period 2: quarter emission
    emission2 := types.CalculateDailyEmission(params, 2)
    expected2 := params.DailyEmission.Quo(math.NewInt(4))
    require.Equal(t, expected2, emission2)
}
```

#### 5.2 Integration Tests
**File**: `x/inflation/v1/keeper/epochs_test.go`

```go
func TestAfterEpochEndHalving(t *testing.T) {
    app := setupTestApp(t)
    ctx := app.BaseApp.NewContext(false, tmproto.Header{})
    
    // Setup initial state
    params := types.DefaultParams()
    params.EnableInflation = true
    params.DailyEmission = math.NewInt(7200000000000000000000) // 7200 tokens
    app.InflationKeeper.SetParams(ctx, params)
    
    // Test epoch 1 (period 0)
    app.InflationKeeper.AfterEpochEnd(ctx, epochstypes.DayEpochID, 1)
    
    // Verify minting occurred
    supply := app.BankKeeper.GetSupply(ctx, params.MintDenom)
    require.Equal(t, params.DailyEmission, supply.Amount)
    
    // ... more test cases for halving scenarios
}
```

#### 5.3 Genesis State Update
**File**: `x/inflation/v1/types/genesis.go`

```go
func DefaultGenesisState() *GenesisState {
    return &GenesisState{
        Params:          DefaultParams(),
        Period:          uint64(0),
        EpochIdentifier: epochstypes.DayEpochID,
        EpochsPerPeriod: 365,
        SkippedEpochs:   0,
        HalvingData: HalvingData{
            CurrentHalvingPeriod: 0,
            LastHalvingEpoch:     0,
            StartEpoch:           0, // Will be set on first epoch
        },
    }
}
```

## 🚀 **Deployment Strategy**

### **Fresh Deployment (Recommended)**
Since you're deploying from scratch:

1. **Genesis Configuration**:
   ```json
   {
     "inflation": {
       "params": {
         "enable_inflation": true,
         "daily_emission": "7200000000000000000000",
         "halving_interval_epochs": 1460,
         "multi_sig_address": "nexqloud1...",
         "max_supply": "21000000000000000000000000"
       },
       "halving_data": {
         "current_halving_period": 0,
         "last_halving_epoch": 0,
         "start_epoch": 0
       }
     }
   }
   ```

2. **Start Network**: Halving begins automatically from epoch 1

### **Governance Upgrade (If Needed)**
If updating existing network:

1. **Submit Software Upgrade Proposal**
2. **Coordinate at Block Height**: All nodes upgrade simultaneously
3. **Initialize Halving State**: Via migration handler

## ✅ **Risk Assessment**

### **LOW RISK** ✅
- ✅ **Uses existing epoch infrastructure**
- ✅ **No BeginBlocker conflicts**
- ✅ **Daily timing already proven**
- ✅ **Supply cap validation**
- ✅ **Validators unaffected** (fee-based)

### **MITIGATED RISKS** ⚠️
- ⚠️ **Multi-sig dependency**: Add fallback to community pool
- ⚠️ **Parameter updates**: Use governance for changes
- ⚠️ **Cap reached**: Automatic inflation disable

## 📋 **Implementation Checklist**

- [ ] **Week 1**: Update protobuf definitions and generate code
- [ ] **Week 2**: Implement halving calculation logic
- [ ] **Week 3**: Modify AfterEpochEnd hook with halving logic  
- [ ] **Week 4**: Add storage keys, events, and constants
- [ ] **Week 5**: Write comprehensive tests
- [ ] **Week 6**: Integration testing and deployment prep

## 🎯 **Success Criteria**

1. **✅ Correct Halving**: Emission halves every 1,460 epochs (4 years)
2. **✅ Supply Cap**: Automatically stops at 21M tokens
3. **✅ Multi-sig Integration**: Tokens sent to business address
4. **✅ Event Emission**: Halving events properly logged
5. **✅ Validator Sustainability**: Network remains secure on transaction fees
6. **✅ Mathematical Accuracy**: Converges to 21,024,000 total supply

## 📊 **Expected Timeline**

- **Simple Daily Minting**: 1-2 weeks
- **Full Halving Implementation**: 6 weeks
- **Testing & Deployment**: 2 weeks
- **Total Project**: 8 weeks

This implementation leverages the robust epoch system already proven in production, minimizing risk while delivering the complete Bitcoin-style halving functionality you require.
