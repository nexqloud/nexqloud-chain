package types

import (
	"github.com/ethereum/go-ethereum/common"
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