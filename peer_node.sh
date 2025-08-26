#!/bin/bash

CHAINID="nxqd_6000-1"

# Set moniker if environment variable is not set
if [ -z "$MONIKER" ]; then
	MONIKER="NexQloudPeer"
fi

KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the nxqd instance
HOMEDIR="$HOME/.nxqd"
# to trace evm
#TRACE="--trace"
TRACE=""

# feemarket params basefee
BASEFEE=1000000000

# Path variables
CONFIG=$HOMEDIR/config/config.toml
APP_TOML=$HOMEDIR/config/app.toml
GENESIS=$HOMEDIR/config/genesis.json
TMP_GENESIS=$HOMEDIR/config/tmp_genesis.json

# validate dependencies are installed
command -v jq >/dev/null 2>&1 || {
	echo >&2 "jq not installed. More info: https://stedolan.github.io/jq/download/"
	exit 1
}

# used to exit on first error (any non-zero exit code)
set -e

if [[ $1 == "init" ]]; then

    # Remove the previous folder
    rm -rf "$HOMEDIR"

    # Set client config
    nxqd config keyring-backend "$KEYRING" --home "$HOMEDIR"
    nxqd config chain-id "$CHAINID" --home "$HOMEDIR"

    VAL_KEY="mykey"
    VAL_MNEMONIC="copper push brief egg scan entry inform record adjust fossil boss egg comic alien upon aspect dry avoid interest fury window hint race symptom"

    # Import keys from mnemonics
    echo "$VAL_MNEMONIC" | nxqd keys add "$VAL_KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"

    # Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
    nxqd init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"

    if [[ $1 == "pending" ]]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
            sed -i '' 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
            sed -i '' 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
            sed -i '' 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
            sed -i '' 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
            sed -i '' 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
            sed -i '' 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
            sed -i '' 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
        else
            sed -i 's/timeout_propose = "3s"/timeout_propose = "30s"/g' "$CONFIG"
            sed -i 's/timeout_propose_delta = "500ms"/timeout_propose_delta = "5s"/g' "$CONFIG"
            sed -i 's/timeout_prevote = "1s"/timeout_prevote = "10s"/g' "$CONFIG"
            sed -i 's/timeout_prevote_delta = "500ms"/timeout_prevote_delta = "5s"/g' "$CONFIG"
            sed -i 's/timeout_precommit = "1s"/timeout_precommit = "10s"/g' "$CONFIG"
            sed -i 's/timeout_precommit_delta = "500ms"/timeout_precommit_delta = "5s"/g' "$CONFIG"
            sed -i 's/timeout_commit = "5s"/timeout_commit = "150s"/g' "$CONFIG"
            sed -i 's/timeout_broadcast_tx_commit = "10s"/timeout_broadcast_tx_commit = "150s"/g' "$CONFIG"
        fi
    fi

    # enable prometheus metrics and all APIs for dev node
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
    else
        sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
    fi

    # Configure block time (1 block every 8 seconds)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i '' 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    else
        sed -i 's/timeout_commit = "5s"/timeout_commit = "8s"/g' "$CONFIG"
        sed -i 's/timeout_commit = "3s"/timeout_commit = "8s"/g' "$CONFIG"
    fi

    # set seed node info
    SEED_NODE_ID="`wget -qO-  http://$SEED_NODE_IP/node-id`"
    echo "SEED_NODE_ID=$SEED_NODE_ID"
    SEEDS="$SEED_NODE_ID@$SEED_NODE_IP:26656"
    sed -i "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"

    wget -qO- "http://$SEED_NODE_IP/genesis.json" > "$GENESIS"

    nxqd validate-genesis --home "$HOMEDIR"

else
    # Start the node
    nxqd start \
        --metrics "$TRACE" \
        --log_level $LOGLEVEL \
        --minimum-gas-prices=0.0001nxq \
        --json-rpc.api eth,txpool,personal,net,debug,web3 \
        --home "$HOMEDIR" \
        --chain-id "$CHAINID"
    
fi