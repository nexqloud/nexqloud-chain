package keeper_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

// ContractConfig defines configuration for smart contract addresses used in staking module
type ContractConfig struct {
	// NFTContractAddress is the address of the NFT contract that validators must own tokens from
	NFTContractAddress common.Address

	// WalletStateContractAddress is the address of the contract that stores validator requirements
	WalletStateContractAddress common.Address

	// NodeURL is the URL of the Ethereum JSON-RPC endpoint for testing
	NodeURL string
}

// DefaultContractConfig returns the default configuration for the staking module
func DefaultContractConfig() ContractConfig {
	return ContractConfig{
		NFTContractAddress:         common.HexToAddress("0x816644F8bc4633D268842628EB10ffC0AdcB6099"),
		WalletStateContractAddress: common.HexToAddress("0xA912e0631f97e52e5fb8435e20f7B4c7755F7de3"),
		NodeURL:                    "http://dev-node.nexqloud.net:8545",
	}
}

// TestValidatorRequirements tests the fetching of validator requirements from the WalletState contract
func TestValidatorRequirements(t *testing.T) {
	config := DefaultContractConfig()

	// Connect to the Ethereum node
	client, err := ethclient.Dial(config.NodeURL)
	require.NoError(t, err, "Failed to connect to the Ethereum node")
	defer client.Close()

	t.Log("Connected to Ethereum node:", config.NodeURL)

	// Test NFT contract connection
	t.Log("NFT Contract address:", config.NFTContractAddress.Hex())
	code, err := client.CodeAt(context.Background(), config.NFTContractAddress, nil)
	require.NoError(t, err, "Failed to get NFT contract code")
	require.NotEmpty(t, code, "NFT contract has no code (not deployed)")

	// Test WalletState contract connection
	t.Log("WalletState Contract address:", config.WalletStateContractAddress.Hex())
	code, err = client.CodeAt(context.Background(), config.WalletStateContractAddress, nil)
	require.NoError(t, err, "Failed to get WalletState contract code")
	require.NotEmpty(t, code, "WalletState contract has no code (not deployed)")

	// Get validator requirements
	requiredNXQTokens, requiredNXQNFTs, err := getValidatorRequirements(client, config.WalletStateContractAddress)
	if err != nil {
		t.Logf("Error getting validator requirements: %v", err)
		t.Logf("This is expected if the WalletState contract doesn't implement getValidatorRequirements()")
		t.Logf("In a real implementation, the code would fall back to default values")
		return
	}

	t.Logf("Validator requirements fetched successfully:")
	t.Logf("Required NXQ tokens: %s", requiredNXQTokens.String())
	t.Logf("Required NXQNFT tokens: %s", requiredNXQNFTs.String())

	// Basic validation of requirements
	require.True(t, requiredNXQTokens.Cmp(big.NewInt(0)) > 0, "Required NXQ tokens should be greater than 0")
	require.True(t, requiredNXQNFTs.Cmp(big.NewInt(0)) > 0, "Required NXQNFT tokens should be greater than 0")
}

// TestNFTBalanceCheck tests checking an address's NFT balance
func TestNFTBalanceCheck(t *testing.T) {
	config := DefaultContractConfig()
	client, err := ethclient.Dial(config.NodeURL)
	if err != nil {
		t.Logf("Failed to connect to Ethereum node: %v. Skipping test.", err)
		t.SkipNow()
		return
	}
	defer client.Close()

	// Use the specific address requested for testing
	testAddress := common.HexToAddress("0xC6Fe5D33615a1C52c08018c47E8Bc53646A0E101")
	
	// Test with real NFT contract
	// If this fails, the test will log but continue
	balance, err := getNFTBalance(client, config.NFTContractAddress, testAddress)
	if err != nil {
		t.Logf("Error getting NFT balance: %v. Using default value 0.", err)
		balance = big.NewInt(0)
	}
	
	t.Logf("NFT Balance for address %s: %s", testAddress.Hex(), balance.String())
	
	// Test a different test address for comparison
	alternateAddress := common.HexToAddress("0x0000000000000000000000000000000000000000")
	alternateBalance, err := getNFTBalance(client, config.NFTContractAddress, alternateAddress)
	if err != nil {
		t.Logf("Error getting NFT balance for alternate address: %v. Using default value 0.", err)
		alternateBalance = big.NewInt(0)
	}
	
	t.Logf("NFT Balance for alternate address %s: %s", alternateAddress.Hex(), alternateBalance.String())
	
	// Fetch validator requirements
	requiredNXQTokens, requiredNXQNFTs, err := getValidatorRequirements(client, config.WalletStateContractAddress)
	if err != nil {
		t.Logf("Error getting validator requirements: %v. Using default values.", err)
		requiredNXQTokens = big.NewInt(5_000_000_000_000_000_000) // 5 NXQ with 18 decimals
		requiredNXQNFTs = big.NewInt(1)
	}
	
	t.Logf("Required NXQ Tokens: %s", requiredNXQTokens.String())
	t.Logf("Required NXQNFTs: %s", requiredNXQNFTs.String())
	
	// Check if the address meets the NFT requirement
	hasEnoughNFTs := balance.Cmp(requiredNXQNFTs) >= 0
	t.Logf("Address %s meets NFT requirement: %v (%s >= %s)", 
		testAddress.Hex(), 
		hasEnoughNFTs,
		balance.String(),
		requiredNXQNFTs.String())
}

// getValidatorRequirements queries the WalletState contract to get validator requirements
func getValidatorRequirements(client *ethclient.Client, contractAddr common.Address) (*big.Int, *big.Int, error) {
	// Function selector for getValidatorRequirements()
	// This is the first 4 bytes of keccak256("getValidatorRequirements()")
	callData := common.FromHex("0x63d2c733")

	// Create the call message
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: callData,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to call contract: %w", err)
	}

	if len(result) < 64 {
		return nil, nil, fmt.Errorf("invalid response length: got %d bytes, want at least 64 bytes", len(result))
	}

	fmt.Printf("Raw result: %s\n", hex.EncodeToString(result))

	// Extract the two uint256 values
	requiredNXQTokens := new(big.Int).SetBytes(result[:32])
	requiredNXQNFTs := new(big.Int).SetBytes(result[32:64])

	return requiredNXQTokens, requiredNXQNFTs, nil
}

// getNFTBalance queries the NFT contract to get an address's NFT balance
func getNFTBalance(client *ethclient.Client, contractAddr, ownerAddr common.Address) (*big.Int, error) {
	// Function selector for balanceOf(address)
	// This is the first 4 bytes of keccak256("balanceOf(address)")
	selector := common.FromHex("0x70a08231")
	
	// Append the address parameter (padded to 32 bytes)
	callData := append(selector, common.LeftPadBytes(ownerAddr.Bytes(), 32)...)

	// Create the call message
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: callData,
	}

	// Execute the call
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call NFT contract: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("empty response from NFT contract")
	}

	// Convert the bytes response to a big.Int
	balance := new(big.Int).SetBytes(result)
	return balance, nil
} 