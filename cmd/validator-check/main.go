package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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

func main() {
	// Parse command line flags
	nodeURL := flag.String("node", DefaultContractConfig().NodeURL, "Ethereum JSON-RPC URL")
	nftContract := flag.String("nft", DefaultContractConfig().NFTContractAddress.Hex(), "NFT contract address")
	walletStateContract := flag.String("wallet", DefaultContractConfig().WalletStateContractAddress.Hex(), "WalletState contract address")
	address := flag.String("address", "", "Address to check (required)")
	flag.Parse()

	if *address == "" {
		fmt.Println("Error: --address flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Connect to the Ethereum node
	client, err := ethclient.Dial(*nodeURL)
	if err != nil {
		fmt.Printf("Error connecting to node %s: %v\n", *nodeURL, err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Printf("Connected to node: %s\n", *nodeURL)

	// Convert addresses
	nftContractAddr := common.HexToAddress(*nftContract)
	walletStateContractAddr := common.HexToAddress(*walletStateContract)
	checkAddr := common.HexToAddress(*address)

	// Verify contracts have code
	code, err := client.CodeAt(context.Background(), nftContractAddr, nil)
	if err != nil || len(code) == 0 {
		fmt.Printf("Error: NFT contract at %s is not deployed\n", nftContractAddr.Hex())
		os.Exit(1)
	}

	code, err = client.CodeAt(context.Background(), walletStateContractAddr, nil)
	if err != nil || len(code) == 0 {
		fmt.Printf("Error: WalletState contract at %s is not deployed\n", walletStateContractAddr.Hex())
		os.Exit(1)
	}

	// Get validator requirements
	fmt.Printf("\n=== Fetching Validator Requirements ===\n")
	requiredNXQTokens, requiredNXQNFTs, err := getValidatorRequirements(client, walletStateContractAddr)
	if err != nil {
		fmt.Printf("Error getting validator requirements: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Current validator requirements:\n")
	fmt.Printf("- Required NXQ tokens: %s\n", requiredNXQTokens.String())
	fmt.Printf("- Required NXQNFT tokens: %s\n", requiredNXQNFTs.String())

	// Check NFT balance
	fmt.Printf("\n=== Checking NFT Balance ===\n")
	fmt.Printf("Address to check: %s\n", checkAddr.Hex())
	nftBalance, err := getNFTBalance(client, nftContractAddr, checkAddr)
	if err != nil {
		fmt.Printf("Error checking NFT balance: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("NFT balance: %s\n", nftBalance.String())

	// Check if NFT requirement is met
	nftCheckPassed := nftBalance.Cmp(requiredNXQNFTs) >= 0
	fmt.Printf("\n=== Validator Eligibility Check ===\n")
	if nftCheckPassed {
		fmt.Printf("✅ NFT requirement PASSED: Has %s NXQNFT (required: %s)\n", 
			nftBalance.String(), requiredNXQNFTs.String())
	} else {
		fmt.Printf("❌ NFT requirement FAILED: Has %s NXQNFT (required: %s)\n", 
			nftBalance.String(), requiredNXQNFTs.String())
	}

	fmt.Printf("\nNote: Minimum self-delegation check requires access to the chain state and cannot be performed by this tool.\n")
	fmt.Printf("To check if you have enough tokens for the minimum self-delegation (%s NXQ),\n", formatNXQ(requiredNXQTokens))
	fmt.Printf("use the chain's query commands:\n")
	fmt.Printf("    nxqd query bank balances <your-address>\n")
}

// getValidatorRequirements queries the WalletState contract
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

	// Extract the two uint256 values
	requiredNXQTokens := new(big.Int).SetBytes(result[:32])
	requiredNXQNFTs := new(big.Int).SetBytes(result[32:64])

	return requiredNXQTokens, requiredNXQNFTs, nil
}

// getNFTBalance queries the NFT contract
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

// formatNXQ formats a big.Int as NXQ tokens with proper decimals
func formatNXQ(tokens *big.Int) string {
	// NXQ has 18 decimals
	decimals := big.NewInt(1_000_000_000_000_000_000) // 10^18
	
	// If the tokens are less than 1 NXQ, return in smaller units
	if tokens.Cmp(decimals) < 0 {
		return fmt.Sprintf("%s wei", tokens.String())
	}
	
	whole := new(big.Int).Div(tokens, decimals)
	remainder := new(big.Int).Mod(tokens, decimals)
	
	if remainder.Cmp(big.NewInt(0)) == 0 {
		return fmt.Sprintf("%s NXQ", whole.String())
	}
	
	// Show with decimals
	return fmt.Sprintf("%s.%018s NXQ", whole.String(), remainder.String())
} 