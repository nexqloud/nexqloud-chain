package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

		// Write the total server count to the file
		file_path := os.Getenv("HOME") + "/.nxqd/devices_count"

		// Open the file and write the total server count and it must be editable
		file, err := os.OpenFile(file_path, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			fmt.Println("Error opening the file:", err)
			continue
		}

		_, err = file.WriteString("2000")
		if err != nil {
			fmt.Println("Error writing to the file:", err)
			continue
		}

		file.Close()
		resp.Body.Close()

		fmt.Println("Total server count written to the file")
	}
}
