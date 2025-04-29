package keeper

import (
	"fmt"
	"log"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestGetOnlineServerCount(t *testing.T) {
	client, err := ethclient.Dial(NodeURL)
	if err != nil {
		log.Fatal("Failed to connect to Ethereum node:", err)
	}

	contract, err := NewOnlineServerMonitor(common.HexToAddress(OnlineServerCountContract), client)
	if err != nil {
		log.Fatal("Failed to load contract:", err)
	}

	val, err := contract.Reached1000ServerCountValue(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Failed to get online server count:", err)
	}
	fmt.Println("Online Server Count has reached wont change the state:", val)

	count, err := contract.GetOnlineServerCount(&bind.CallOpts{})
	if err != nil {
		log.Fatal("Failed to get online server count:", err)
	}
	fmt.Println("Online Server Count:", count)
}
