package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Function to query the rpc endpoint every 10 seconds
// and return the latest block number
func RunLatestBlockNumber() {
	queryTicker := time.NewTicker(10 * time.Second)
	for range queryTicker.C {
		fmt.Println("Querying the rpc endpoint")

		url := "https://eth.llamarpc.com"
		body, _ := json.Marshal(map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_blockNumber",
			"params":  []interface{}{},
			"id":      1,
		})

		// Fetch the latest block number from the rpc endpoint
		// and print it to the console
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Println("Error querying the rpc endpoint:", err)
			continue
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		fmt.Println("Latest block number:", result["result"])
	}
}
