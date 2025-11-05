package types_test

import (
	"testing"

	"github.com/evmos/evmos/v19/x/evm/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()

	// Verify bootstrap-safe defaults
	require.Equal(t, types.ZeroAddress, params.OnlineServerCountContract, "OnlineServerCountContract should be zero address")
	require.Equal(t, types.ZeroAddress, params.NFTContractAddress, "NFTContractAddress should be zero address")
	require.Equal(t, types.ZeroAddress, params.WalletStateContractAddress, "WalletStateContractAddress should be zero address")
	require.Equal(t, types.ZeroAddress, params.ValidatorApprovalContractAddress, "ValidatorApprovalContractAddress should be zero address")
	require.Empty(t, params.WhitelistedAddresses, "WhitelistedAddresses should be empty")
	require.False(t, params.EnableChainStatusCheck, "EnableChainStatusCheck should be false")
	require.False(t, params.EnableWalletLockCheck, "EnableWalletLockCheck should be false")

	// Verify default params are in bootstrap mode
	require.True(t, params.IsBootstrapMode(), "Default params should be in bootstrap mode")

	// Verify default params are valid
	require.NoError(t, params.Validate(), "Default params should be valid")
}

func TestIsBootstrapMode(t *testing.T) {
	testCases := []struct {
		name                   string
		enableChainStatusCheck bool
		enableWalletLockCheck  bool
		expectedBootstrap      bool
	}{
		{
			name:                   "Both disabled - Bootstrap mode",
			enableChainStatusCheck: false,
			enableWalletLockCheck:  false,
			expectedBootstrap:      true,
		},
		{
			name:                   "Chain status enabled - Not bootstrap",
			enableChainStatusCheck: true,
			enableWalletLockCheck:  false,
			expectedBootstrap:      false,
		},
		{
			name:                   "Wallet lock enabled - Not bootstrap",
			enableChainStatusCheck: false,
			enableWalletLockCheck:  true,
			expectedBootstrap:      false,
		},
		{
			name:                   "Both enabled - Not bootstrap",
			enableChainStatusCheck: true,
			enableWalletLockCheck:  true,
			expectedBootstrap:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := types.DefaultParams()
			params.EnableChainStatusCheck = tc.enableChainStatusCheck
			params.EnableWalletLockCheck = tc.enableWalletLockCheck

			result := params.IsBootstrapMode()
			require.Equal(t, tc.expectedBootstrap, result)
		})
	}
}

func TestIsContractSet(t *testing.T) {
	testCases := []struct {
		name     string
		address  string
		expected bool
	}{
		{
			name:     "Empty address",
			address:  "",
			expected: false,
		},
		{
			name:     "Zero address",
			address:  types.ZeroAddress,
			expected: false,
		},
		{
			name:     "Valid address",
			address:  "0x1234567890123456789012345678901234567890",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := types.IsContractSet(tc.address)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateParams(t *testing.T) {
	testCases := []struct {
		name        string
		params      types.Params
		expectedErr bool
		errContains string
	}{
		{
			name:        "Default params (bootstrap mode) - valid",
			params:      types.DefaultParams(),
			expectedErr: false,
		},
		{
			name: "Valid production params",
			params: types.Params{
				EvmDenom:                         "unxq",
				ChainConfig:                      types.DefaultChainConfig(),
				ExtraEIPs:                        types.DefaultExtraEIPs,
				AllowUnprotectedTxs:              false,
				ActiveStaticPrecompiles:          types.DefaultStaticPrecompiles,
				EVMChannels:                      types.DefaultEVMChannels,
				AccessControl:                    types.DefaultAccessControl,
				OnlineServerCountContract:        "0x1234567890123456789012345678901234567890",
				NFTContractAddress:               "0x2234567890123456789012345678901234567890",
				WalletStateContractAddress:       "0x3234567890123456789012345678901234567890",
				ValidatorApprovalContractAddress: "0x4234567890123456789012345678901234567890",
				WhitelistedAddresses: []string{
					"0x5234567890123456789012345678901234567890",
				},
				EnableChainStatusCheck: true,
				EnableWalletLockCheck:  true,
			},
			expectedErr: false,
		},
		{
			name: "Invalid contract address",
			params: types.Params{
				EvmDenom:                  "unxq",
				ChainConfig:               types.DefaultChainConfig(),
				ExtraEIPs:                 types.DefaultExtraEIPs,
				AllowUnprotectedTxs:       false,
				ActiveStaticPrecompiles:   types.DefaultStaticPrecompiles,
				EVMChannels:               types.DefaultEVMChannels,
				AccessControl:             types.DefaultAccessControl,
				OnlineServerCountContract: "invalid-address",
				EnableChainStatusCheck:    false,
				EnableWalletLockCheck:     false,
			},
			expectedErr: true,
			errContains: "invalid online server count contract",
		},
		{
			name: "Chain status check enabled but contract not set",
			params: types.Params{
				EvmDenom:                  "unxq",
				ChainConfig:               types.DefaultChainConfig(),
				ExtraEIPs:                 types.DefaultExtraEIPs,
				AllowUnprotectedTxs:       false,
				ActiveStaticPrecompiles:   types.DefaultStaticPrecompiles,
				EVMChannels:               types.DefaultEVMChannels,
				AccessControl:             types.DefaultAccessControl,
				OnlineServerCountContract: types.ZeroAddress,
				EnableChainStatusCheck:    true,
				EnableWalletLockCheck:     false,
			},
			expectedErr: true,
			errContains: "chain status check enabled but contract address not set",
		},
		{
			name: "Wallet lock check enabled but contract not set",
			params: types.Params{
				EvmDenom:                   "unxq",
				ChainConfig:                types.DefaultChainConfig(),
				ExtraEIPs:                  types.DefaultExtraEIPs,
				AllowUnprotectedTxs:        false,
				ActiveStaticPrecompiles:    types.DefaultStaticPrecompiles,
				EVMChannels:                types.DefaultEVMChannels,
				AccessControl:              types.DefaultAccessControl,
				WalletStateContractAddress: types.ZeroAddress,
				EnableChainStatusCheck:     false,
				EnableWalletLockCheck:      true,
			},
			expectedErr: true,
			errContains: "wallet lock check enabled but contract address not set",
		},
		{
			name: "Invalid whitelisted address",
			params: types.Params{
				EvmDenom:                "unxq",
				ChainConfig:             types.DefaultChainConfig(),
				ExtraEIPs:               types.DefaultExtraEIPs,
				AllowUnprotectedTxs:     false,
				ActiveStaticPrecompiles: types.DefaultStaticPrecompiles,
				EVMChannels:             types.DefaultEVMChannels,
				AccessControl:           types.DefaultAccessControl,
				WhitelistedAddresses: []string{
					"invalid-address",
				},
				EnableChainStatusCheck: false,
				EnableWalletLockCheck:  false,
			},
			expectedErr: true,
			errContains: "invalid whitelisted address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()

			if tc.expectedErr {
				require.Error(t, err)
				if tc.errContains != "" {
					require.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateContractAddress(t *testing.T) {
	testCases := []struct {
		name        string
		address     string
		expectedErr bool
	}{
		{
			name:        "Empty address - valid (bootstrap)",
			address:     "",
			expectedErr: false,
		},
		{
			name:        "Zero address - valid (bootstrap)",
			address:     types.ZeroAddress,
			expectedErr: false,
		},
		{
			name:        "Valid hex address",
			address:     "0x1234567890123456789012345678901234567890",
			expectedErr: false,
		},
		{
			name:        "Valid hex address (uppercase)",
			address:     "0x1234567890ABCDEF123456789012345678901234",
			expectedErr: false,
		},
		{
			name:        "Address without 0x prefix (accepted by IsHexAddress)",
			address:     "1234567890123456789012345678901234567890",
			expectedErr: false, // common.IsHexAddress accepts addresses with or without 0x
		},
		{
			name:        "Invalid address - wrong length",
			address:     "0x12345",
			expectedErr: true,
		},
		{
			name:        "Invalid address - non-hex characters",
			address:     "0xGGGG567890123456789012345678901234567890",
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := types.DefaultParams()
			params.OnlineServerCountContract = tc.address

			err := params.Validate()

			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateWhitelistedAddresses(t *testing.T) {
	testCases := []struct {
		name        string
		addresses   []string
		expectedErr bool
	}{
		{
			name:        "Empty list - valid",
			addresses:   []string{},
			expectedErr: false,
		},
		{
			name: "Single valid address",
			addresses: []string{
				"0x1234567890123456789012345678901234567890",
			},
			expectedErr: false,
		},
		{
			name: "Multiple valid addresses",
			addresses: []string{
				"0x1234567890123456789012345678901234567890",
				"0x2234567890123456789012345678901234567890",
				"0x3234567890123456789012345678901234567890",
			},
			expectedErr: false,
		},
		{
			name: "One invalid address in list",
			addresses: []string{
				"0x1234567890123456789012345678901234567890",
				"invalid-address",
			},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := types.DefaultParams()
			params.WhitelistedAddresses = tc.addresses

			err := params.Validate()

			if tc.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBootstrapToProductionTransition(t *testing.T) {
	// Start with bootstrap params
	params := types.DefaultParams()
	require.True(t, params.IsBootstrapMode())
	require.NoError(t, params.Validate())

	// Transition to production (set contracts)
	params.OnlineServerCountContract = "0x1234567890123456789012345678901234567890"
	params.NFTContractAddress = "0x2234567890123456789012345678901234567890"
	params.WalletStateContractAddress = "0x3234567890123456789012345678901234567890"
	params.ValidatorApprovalContractAddress = "0x4234567890123456789012345678901234567890"
	params.WhitelistedAddresses = []string{
		"0x5234567890123456789012345678901234567890",
	}

	// Still in bootstrap mode (checks not enabled yet)
	require.True(t, params.IsBootstrapMode())
	require.NoError(t, params.Validate())

	// Enable security checks
	params.EnableChainStatusCheck = true
	params.EnableWalletLockCheck = true

	// Now in production mode
	require.False(t, params.IsBootstrapMode())
	require.NoError(t, params.Validate())
}

func TestProductionParamsWithZeroAddressShouldFail(t *testing.T) {
	params := types.DefaultParams()

	// Enable checks without setting contracts - should fail validation
	params.EnableChainStatusCheck = true

	err := params.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "chain status check enabled but contract address not set")
}
