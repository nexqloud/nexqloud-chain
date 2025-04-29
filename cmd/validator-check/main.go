package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	
	"golang.org/x/crypto/sha3"
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
		WalletStateContractAddress: common.HexToAddress("0x687A737732FFee7b38dF33e91f58723ea19F9145"),
		NodeURL:                    "http://dev-node.nexqloud.net:8545",
	}
}

func main() {
	// Parse command line arguments
	var addressStr string
	flag.StringVar(&addressStr, "address", "", "Ethereum address to check for validator eligibility")
	flag.Parse()

	if addressStr == "" {
		fmt.Println("Please provide an Ethereum address using the --address flag")
		fmt.Println("Example: validator-check --address=0x687A737732FFee7b38dF33e91f58723ea19F9145")
		os.Exit(1)
	}

	// Parse address
	if !strings.HasPrefix(addressStr, "0x") {
		addressStr = "0x" + addressStr
	}
	checkAddr := common.HexToAddress(addressStr)

	// Load config
	config := DefaultContractConfig()
	nftContractAddr := config.NFTContractAddress
	walletStateContractAddr := config.WalletStateContractAddress
	nodeURL := config.NodeURL

	// Connect to Ethereum node
	fmt.Printf("Connecting to Ethereum node at %s...\n", nodeURL)
	client, err := ethclient.Dial(nodeURL)
	if err != nil {
		fmt.Printf("Error connecting to Ethereum node: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()
	fmt.Println("Connected successfully!")

	// Check if contracts are deployed
	var code []byte
	code, err = client.CodeAt(context.Background(), nftContractAddr, nil)
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
		fmt.Printf("Warning: Failed to get validator requirements: %v\n", err)
		fmt.Println("Using default values: 5 NXQ tokens, 5 NXQNFT")
		requiredNXQTokens = big.NewInt(5_000_000_000_000_000_000) // Default: 5 NXQ with 18 decimals
		requiredNXQNFTs = big.NewInt(5)                          // Default: 5 NFT (as per current contract setting)
	}

	fmt.Printf("Current validator requirements:\n")
	fmt.Printf("- Required NXQ tokens: %s\n", requiredNXQTokens.String())
	fmt.Printf("- Required NXQNFT tokens: %s\n", requiredNXQNFTs.String())

	// Check if address is on the approved validators list
	fmt.Printf("\n=== Checking Validator Approval Status ===\n")
	fmt.Printf("Address to check: %s\n", checkAddr.Hex())
	isApproved, err := isApprovedValidator(client, walletStateContractAddr, checkAddr)
	approvalStatus := "UNKNOWN"
	approvalCheck := false
	
	if err != nil {
		if strings.Contains(err.Error(), "execution reverted") || 
		   strings.Contains(err.Error(), "may not be implemented yet") {
			fmt.Printf("Note: Validator approval check is not available on this contract version.\n")
			fmt.Printf("The smart contract function 'isApprovedValidator' may not be implemented yet.\n")
			approvalStatus = "SKIPPED"
		} else {
			fmt.Printf("Warning: Failed to check validator approval status: %v\n", err)
		}
	} else {
		if isApproved {
			fmt.Printf("✅ Address is APPROVED as a validator.\n")
			approvalStatus = "APPROVED"
			approvalCheck = true
		} else {
			fmt.Printf("❌ Address is NOT APPROVED as a validator.\n")
			approvalStatus = "DENIED"
		}
	}

	// Check NFT balance
	fmt.Printf("\n=== Checking NFT Balance ===\n")
	fmt.Printf("Address to check: %s\n", checkAddr.Hex())
	nftBalance, err := getNFTBalance(client, nftContractAddr, checkAddr)
	if err != nil {
		fmt.Printf("Warning: Failed to check NFT balance: %v\n", err)
		fmt.Println("Using default value: 0 NXQNFT")
		nftBalance = big.NewInt(0)
	}

	fmt.Printf("NFT balance: %s\n", nftBalance.String())

	// Check if NFT requirement is met
	nftCheckPassed := nftBalance.Cmp(requiredNXQNFTs) >= 0
	fmt.Printf("\n=== Validator Eligibility Check ===\n")
	if nftCheckPassed {
		fmt.Printf("✅ NFT requirement met: Has %s NXQNFT (required: %s)\n", 
			nftBalance.String(), requiredNXQNFTs.String())
	} else {
		fmt.Printf("❌ NFT requirement NOT met: Has %s NXQNFT (required: %s)\n", 
			nftBalance.String(), requiredNXQNFTs.String())
	}
	
	// Summary of all checks
	fmt.Printf("\n=== Overall Eligibility ===\n")
	
	// If approval check is skipped, only consider NFT balance
	if approvalStatus == "SKIPPED" {
		if nftCheckPassed {
			fmt.Println("✅ Address meets NFT requirements to become a validator.")
			fmt.Println("   Note: Validator approval status could not be checked.")
		} else {
			fmt.Println("❌ Address does NOT meet all requirements to become a validator.")
			fmt.Println("   - Insufficient NXQNFT tokens")
			fmt.Println("   Note: Validator approval status could not be checked.")
		}
	} else {
		// Consider both approval and NFT balance
		if approvalCheck && nftCheckPassed {
			fmt.Println("✅ Address meets all requirements to become a validator.")
		} else {
			fmt.Println("❌ Address does NOT meet all requirements to become a validator.")
			if !approvalCheck && approvalStatus != "UNKNOWN" {
				fmt.Println("   - Not on the approved validators list")
			}
			if !nftCheckPassed {
				fmt.Println("   - Insufficient NXQNFT tokens")
			}
		}
	}

	fmt.Println("\nNote: This tool cannot check minimum self-delegation. Please use chain query commands to verify token balances.")
}

// getFunctionSelector calculates the 4-byte function selector for a Solidity function signature
// This matches Ethereum's implementation exactly
func getFunctionSelector(signature string) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return hash.Sum(nil)[:4] // First 4 bytes of keccak256 hash
}

// getValidatorRequirements queries the WalletState contract
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

// isApprovedValidator checks if an address is on the approved validators list
func isApprovedValidator(client *ethclient.Client, contractAddr, validatorAddr common.Address) (bool, error) {
	// Calculate the function selector for isApprovedValidator(address)
	functionSignature := "isApprovedValidator(address)"
	selector := getFunctionSelector(functionSignature)
	
	// Encode the address parameter
	data := append(selector, common.LeftPadBytes(validatorAddr.Bytes(), 32)...)
	
	// Make the call
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}
	
	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		// If the error contains "execution reverted", it likely means the function doesn't exist
		// or is not accessible (not implemented yet)
		if strings.Contains(err.Error(), "execution reverted") {
			return false, fmt.Errorf("contract function 'isApprovedValidator' may not be implemented yet: %v", err)
		}
		return false, fmt.Errorf("failed to call isApprovedValidator: %v", err)
	}
	
	// Parse boolean result (should be a 32-byte word where any non-zero value is true)
	if len(result) < 32 {
		return false, fmt.Errorf("unexpected result length: got %d bytes, want 32", len(result))
	}
	
	// Check if any byte in the result is non-zero (indicating true)
	for _, b := range result {
		if b != 0 {
			return true, nil
		}
	}
	
	return false, nil
} 