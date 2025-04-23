Overview
The NexQloud distribution and rewards system is responsible for generating, allocating, and distributing rewards to network participants. The system consists of two primary reward sources: inflation (newly minted tokens) and transaction fees. These rewards incentivize validators to secure the network and are shared with delegators who stake their tokens with validators.The distribution system is a composite of two Cosmos SDK modules:Inflation Module (x/inflation): Handles token minting and allocation
Distribution Module (x/distribution): Manages reward distribution to validators, delegators and the community pool
2. Module Architecture
The distribution system consists of the following interconnected components:
┌───────────────────┐     ┌─────────────────────┐     ┌────────────────────┐
│   Inflation       │     │  Fee Collection     │     │    Distribution     │
│   Module          │────▶│  Module             │────▶│    Module          │
│ (Token Creation)  │     │ (Fee Aggregation)   │     │ (Reward Allocation) │
└───────────────────┘     └─────────────────────┘     └────────────────────┘
         │                                                      │
         │                                                      │
         ▼                                                      ▼
┌───────────────────┐                              ┌────────────────────────┐
│  Staking Rewards  │                              │     Community Pool     │
│   (53.33% of      │                              │      (46.67% of        │
│    inflation)     │                              │       inflation)       │
└───────────────────┘                              └────────────────────────┘
         │
         │
         ▼
┌───────────────────────────────────────────────────────────────────────────┐
│                            Validator Set                                   │
│  ┌───────────┐    ┌───────────┐    ┌───────────┐         ┌───────────┐    │
│  │ Validator │    │ Validator │    │ Validator │   ...   │ Validator │    │
│  │     1     │    │     2     │    │     3     │         │     n     │    │
│  └───────────┘    └───────────┘    └───────────┘         └───────────┘    │
└───────────────────────────────────────────────────────────────────────────┘
         │                │                │                     │
         │                │                │                     │
         ▼                ▼                ▼                     ▼
┌───────────────────────────────────────────────────────────────────────────┐
│                            Delegators                                      │
└───────────────────────────────────────────────────────────────────────────┘
3. Reward Generation
3.1 Inflation Rewards
Inflation rewards are newly minted tokens created on a per-epoch basis. In NexQloud, epochs are typically aligned with days (i.e., one epoch = one day).Key Implementation Files:x/inflation/v1/keeper/hooks.go: Implements the epoch hooks that trigger inflation
x/inflation/v1/types/inflation_calculation.go: Contains the inflation calculation logic
Minting Process:At the end of each epoch, the AfterEpochEnd hook in the inflation module is triggered
The epoch mint provision is calculated based on the inflation parameters
New tokens are minted by the bank module via MintCoins
Minted tokens are allocated according to the configured distribution parameters
3.2 Transaction Fees
Transaction fees are paid by users to execute transactions on the blockchain. These fees are collected in the fee_collector module account.Key Implementation Files:app/ante/cosmos/fees.go: Implements fee collection logic in the AnteHandler
x/feemarket/keeper/eip1559.go: Implements dynamic fee calculation for EVM transactions
Fee Collection Process:When a transaction is submitted, the DeductFeeDecorator deducts the fee from the transaction sender
Fees are deposited into the fee_collector module account
These fees are then distributed to validators during the next distribution event
4. Distribution Mechanisms
4.1 Validator Rewards
Validators receive both direct rewards and commission from delegator rewards.Sources:A proportional share of the staking rewards allocation (53.33% of inflation)
A proportional share of transaction fees
Commission from delegator rewards (set by each validator)
Distribution Logic:Rewards are distributed proportionally to voting power (stake)
Each validator sets a commission rate when creating their validator
Commission is taken from delegator rewards before distribution
4.2 Delegator Rewards
Delegators receive a portion of the rewards earned by validators they delegated to.Sources:A proportional share of validator rewards, after the validator's commission is deducted
Distribution Logic:Rewards for a delegator are proportional to their delegation amount relative to the total stake of the validator
Rewards accumulate over time and must be explicitly claimed
4.3 Community Pool
The community pool receives a fixed percentage of inflation rewards.Sources:46.67% of newly minted tokens from inflation
Funds can also be directly contributed via FundCommunityPool
Usage:Community pool funds can only be spent through governance proposals
Typically used for ecosystem development, grants, and other community-oriented initiatives
5. Calculation Formulas
5.1 Epoch Mint Provision
The amount of tokens minted per epoch follows an exponential decay function with bonding incentives:
CalculateEpochMintProvision = f(x) / reductionFactor / epochsPerPeriod * 10^18
where:
f(x) = exponentialDecay * bondingIncentive
exponentialDecay = a * (1 - r)^x + c
bondingIncentive = 1 + maxVariance - bondedRatio * (maxVariance / bondingTarget)
Parameters:
- a: Initial value (300,000,000)
- r: Reduction factor (50%)
- x: Period number
- c: Long term inflation (9,375,000)
- bondingTarget: Target bonding ratio (66%)
- maxVariance: Maximum variance (0%)
- reductionFactor: Constant reduction factor (3)
- epochsPerPeriod: Number of epochs in a period (365 by default)
Implementation: x/inflation/v1/types/inflation_calculation.go:CalculateEpochMintProvision
5.2 Inflation Rate
Annual inflation rate is calculated as:
InflationRate = EpochMintProvision * epochsPerPeriod / CirculatingSupply * 100
Parameters:
- EpochMintProvision: Tokens minted per epoch
- epochsPerPeriod: Number of epochs in a period (365 by default)
- CirculatingSupply: Total circulating supply of tokens
Implementation: x/inflation/v1/keeper/inflation.go:GetInflationRate
5.3 Reward Distribution
Distribution of rewards to validators and delegators:
ValidatorReward = (VotingPower / TotalVotingPower) * (StakingRewards + TransactionFees)
ValidatorCommission = ValidatorReward * CommissionRate
ValidatorTakeHome = ValidatorOwnDelegation + ValidatorCommission
DelegatorReward = (DelegationAmount / ValidatorTotalDelegation) * (ValidatorReward - ValidatorCommission)
6. Module Accounts
The distribution system utilizes several module accounts:Inflation Module Account (inflation)
Address: Generated deterministically from the module name
Permissions: Minter
Purpose: Temporary holding account for newly minted tokens before distribution
Fee Collector Account (fee_collector)
Address: Generated deterministically from the module name
Permissions: Burner
Purpose: Collects transaction fees for distribution
Distribution Module Account (distribution)
Address: Generated deterministically from the module name
Permissions: None
Purpose: Manages the community pool and holds unclaimed rewards
These accounts do not have private keys and are controlled programmatically by their respective modules.
7. Claiming Rewards
Rewards must be explicitly claimed by validators and delegators:Validator Commission:
nxqd tx distribution withdraw-validator-commission [validator-address] --from [address] --chain-id [chain-id]
Delegator Rewards:
nxqd tx distribution withdraw-delegator-reward [validator-address] --from [delegator-address] --chain-id [chain-id]
Implementation:x/distribution/keeper/msg_server.go: Implements the message handlers for reward claims
x/distribution/keeper/allocation.go: Manages the allocation of rewards
8. Parameter Configuration
Inflation Module Parameters
// Default parameters in x/inflation/v1/types/params.go
DefaultInflationDistribution = InflationDistribution{
    StakingRewards:  math.LegacyNewDecWithPrec(533333334, 9), // 0.53
    CommunityPool:   math.LegacyNewDecWithPrec(466666666, 9), // 0.47
    UsageIncentives: math.LegacyZeroDec(),                    // Deprecated
}
DefaultExponentialCalculation = ExponentialCalculation{
    A:             math.LegacyNewDec(int64(300_000_000)),
    R:             math.LegacyNewDecWithPrec(50, 2), // 50%
    C:             math.LegacyNewDec(int64(9_375_000)),
    BondingTarget: math.LegacyNewDecWithPrec(66, 2), // 66%
    MaxVariance:   math.LegacyZeroDec(),             // 0%
}
Distribution Module Parameters
The Distribution module has a community_tax parameter, which defines the percentage of transaction fees that go to the community pool. However, in NexQloud this is typically set to 0 since the community pool is funded primarily through inflation.
9. Governance & Parameter Updates
Parameters can be updated through governance proposals:
{
  "title": "Inflation Parameter Change",
  "description": "Update inflation distribution parameters",
  "changes": [
    {
      "subspace": "inflation",
      "key": "InflationDistribution",
      "value": {
        "staking_rewards": "0.700000000",
        "community_pool": "0.300000000",
        "usage_incentives": "0.000000000"
      }
    }
  ],
  "deposit": "10000000unxq"
}
10. Technical Implementation
10.1 Key Files
Inflation Module:x/inflation/v1/keeper/inflation.go: Core implementation of inflation allocation
x/inflation/v1/types/inflation_calculation.go: Implements inflation calculation formulas
x/inflation/v1/keeper/hooks.go: Implements epoch hooks for minting
x/inflation/v1/types/params.go: Defines inflation parameters
Distribution Module:x/distribution/keeper/allocation.go: Implements reward allocation logic
x/distribution/keeper/msg_server.go: Implements message handlers for claiming rewards
x/distribution/keeper/hooks.go: Implements distribution hooks triggered during block processing
10.2 State Management
Inflation Module State:
Store prefixes:
- ParamsKey: Inflation parameters
- PeriodKey: Current period
- EpochIdentifierKey: Identifier for the epochs
- EpochsPerPeriodKey: Number of epochs per period
- SkippedEpochsKey: Number of skipped epochs (when inflation disabled)
Distribution Module State:
Store prefixes:
- FeePoolKey: Community pool funds
- ProposerKey: Current block proposer
- ValidatorOutstandingRewardsPrefix: Outstanding rewards for validators
- DelegatorWithdrawAddrPrefix: Withdraw addresses for delegators
- DelegatorStartingInfoPrefix: Starting info for delegators
- ValidatorHistoricalRewardsPrefix: Historical rewards for validators
- ValidatorCurrentRewardsPrefix: Current rewards for validators
- ValidatorAccumulatedCommissionPrefix: Accumulated commission for validators
- ValidatorSlashEventPrefix: Slash events for validators
10.3 Hooks
Inflation Hooks:BeforeEpochStart: No-op
AfterEpochEnd: Mint tokens and allocate them according to distribution parameters
Distribution Hooks:AfterValidatorCreated: Initialize validator distribution records
BeforeDelegationCreated: No-op
BeforeDelegationSharesModified: Withdraw delegation rewards
AfterDelegationModified: No-op
BeforeValidatorSlashed: Record validator slash events
---This technical document provides a comprehensive overview of the reward generation and distribution mechanisms in the NexQloud blockchain. For specific implementation details, refer to the source code in the respective module directories.
