#!/bin/bash

CHAINID="nxq_7002-2"
MONIKER="NexQloudPeer"
# Remember to change to other types of keyring like 'file' in-case exposing to outside world,
# otherwise your balance will be wiped quickly
# The keyring test does not require private key to steal tokens from you
KEYRING="test"
KEYALGO="eth_secp256k1"
LOGLEVEL="info"
# Set dedicated home directory for the nxqd instance
HOMEDIR="$HOME/.nxqd"

KEYS[0]="mykey"
# to trace evm
#TRACE="--trace"
TRACE=""
TOKEN="unxq"
PREFIX="nxq"

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

# Parse input flags
install=false
overwrite=""

if [[ "$PATH" == *"root/go/bin"* ]]; then
	echo "WORKING..." > /root/out.log
else
	export PATH=$PATH:/usr/local/go/bin:/root/go/bin
	echo $PATH > /root/out.log
fi


if [[ $install == true ]]; then
	# (Re-)install daemon
	make install
fi

if [[ $1 == "init" ]]; then

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

	# Remove the previous folder
	rm -rf "$HOMEDIR"

	#make install

	# Set client config
	nxqd config keyring-backend $KEYRING --home "$HOMEDIR"
	nxqd config chain-id $CHAINID --home "$HOMEDIR"


	# If keys exist they should be deleted
	for KEY in "${KEYS[@]}"; do
		nxqd keys add "$KEY" --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"  2> /root/certificate
	done

	# Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
	nxqd init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR"

	sed -i 's/127.0.0.1:26657/0.0.0.0:26657/g' "$CONFIG"
	sed -i 's/127.0.0.1:6060/0.0.0.0:6060/g' "$CONFIG"
	sed -i 's/127.0.0.1/0.0.0.0/g' "$APP_TOML"

    # set seed node info
	SEED_NODE_ID="`wget -qO- http://3.20.175.230/node-id`"
	echo "SEED_NODE_ID=$SEED_NODE_ID"
	SEEDS="$SEED_NODE_ID@3.20.175.230:26656"
	#sed -i "s/seeds =.*/seeds = \"$SEEDS\"/g" "$CONFIG"
	sed -i "s/persistent_peers =.*/persistent_peers = \"$SEEDS\"/g" "$CONFIG"

	wget -qO- "http://3.20.175.230/genesis.json" > "$GENESIS"

	# set custom pruning settings
	sed -i.bak 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
	sed -i.bak 's/pruning-keep-recent = "0"/pruning-keep-recent = "2"/g' "$APP_TOML"
	sed -i.bak 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	nxqd validate-genesis --home "$HOMEDIR"
else
	# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
	nxqd start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001$TOKEN --json-rpc.enable  --grpc.enable  --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --home "$HOMEDIR" --keyring-backend $KEYRING 
fi	

