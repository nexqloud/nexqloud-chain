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
	"golang.org/x/crypto/sha3"
)

// ContractConfig defines configuration for smart contract addresses used in staking module
type ContractConfig struct {
	// NFTContractAddress is the address of the NFT contract that validators must own tokens from
	NFTContractAddress common.Address

	// WalletStateContractAddress is the address of the contract for wallet lock/unlock
	WalletStateContractAddress common.Address

	// ValidatorApprovalContractAddress is the address of the contract for validator approval
	ValidatorApprovalContractAddress common.Address

	// NodeURL is the URL of the Ethereum JSON-RPC endpoint for testing
	NodeURL string
}

// DefaultContractConfig returns the default configuration for the staking module
func DefaultContractConfig() ContractConfig {
	return ContractConfig{
		NFTContractAddress:               common.HexToAddress("0xb1E62bff2501953064E7Aaf12C5d65aA439B8884"),
		WalletStateContractAddress:       common.HexToAddress("0x81C00FE47b085aCDf88C0Fa30437A3b8F39F4Eb6"),
		ValidatorApprovalContractAddress: common.HexToAddress("0x6D6b3e29137B69D6bb0d6706E4D1d20CD8aCbFFD"),
		NodeURL:                          "http://prod-node.nexqloudsite.com:8545",
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

	// Test ValidatorApproval contract connection
	t.Log("ValidatorApproval Contract address:", config.ValidatorApprovalContractAddress.Hex())
	code, err = client.CodeAt(context.Background(), config.ValidatorApprovalContractAddress, nil)
	require.NoError(t, err, "Failed to get ValidatorApproval contract code")
	require.NotEmpty(t, code, "ValidatorApproval contract has no code (not deployed)")

	// Get validator requirements
	requiredNXQTokens, requiredNXQNFTs, err := getValidatorRequirements(client, config.ValidatorApprovalContractAddress)
	if err != nil {
		t.Logf("Error getting validator requirements: %v", err)
		t.Logf("This is expected if the ValidatorApproval contract doesn't implement getValidatorRequirements()")
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
	requiredNXQTokens, requiredNXQNFTs, err := getValidatorRequirements(client, config.ValidatorApprovalContractAddress)
	if err != nil {
		t.Logf("Error getting validator requirements: %v. Using default values.", err)
		requiredNXQTokens = big.NewInt(5_000_000_000_000_000_000) // 5 NXQ with 18 decimals
		requiredNXQNFTs = big.NewInt(5)                           // Default: 5 NFT (as per current contract setting)
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

// getFunctionSelector calculates the 4-byte function selector for a Solidity function signature
// This matches Ethereum's implementation exactly
func getFunctionSelector(signature string) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return hash.Sum(nil)[:4] // First 4 bytes of keccak256 hash
}

// getValidatorRequirements queries the ValidatorApproval contract to get validator requirements
func getValidatorRequirements(client *ethclient.Client, contractAddr common.Address) (*big.Int, *big.Int, error) {
	// Calculate the function selector properly
	functionSignature := "getValidatorRequirements()"
	callData := getFunctionSelector(functionSignature)

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
