#!/bin/bash

CHAINID="nxq_7002-2"
MONIKER="NexQCloud"
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
		nxqd keys add "$KEY" --keyring-backend $KEYRING --algo $KEYALGO --home "$HOMEDIR"
	done

	# Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
	nxqd init $MONIKER -o --chain-id $CHAINID --home "$HOMEDIR"

	sed -i 's/127.0.0.1:26657/0.0.0.0:26657/g' "$CONFIG"
	sed -i 's/127.0.0.1:6060/0.0.0.0:6060/g' "$CONFIG"
	sed -i 's/127.0.0.1/0.0.0.0/g' "$APP_TOML"



	# Change parameter token denominations to $TOKEN
	jq '.app_state["staking"]["params"]["bond_denom"]="'$TOKEN'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["crisis"]["constant_fee"]["denom"]="'$TOKEN'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="'$TOKEN'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="'$TOKEN'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["inflation"]["params"]["mint_denom"]="'$TOKEN'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"




	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set claims start time
	current_date=$(date -u +"%Y-%m-%dT%TZ")
	jq -r --arg current_date "$current_date" '.app_state["claims"]["params"]["airdrop_start_time"]=$current_date' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set claims records for validator account
	# amount_to_claim=10000
	# claims_key=${KEYS[0]}
	# node_address=$(nxqd keys show "$claims_key" --keyring-backend $KEYRING --home "$HOMEDIR" | grep "address" | cut -c12-)
	# jq -r --arg node_address "$node_address" --arg amount_to_claim "$amount_to_claim" '.app_state["claims"]["claims_records"]=[{"initial_claimable_amount":$amount_to_claim, "actions_completed":[false, false, false, false],"address":$node_address}]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set claims decay
	jq '.app_state["claims"]["params"]["duration_of_decay"]="1000000s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["claims"]["params"]["duration_until_decay"]="100000s"' >"$TMP_GENESIS" "$GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Claim module account:
	# 0xA61808Fe40fEb8B3433778BBC2ecECCAA47c8c47 || evmos15cvq3ljql6utxseh0zau9m8ve2j8erz89m5wkz
	# jq -r --arg amount_to_claim "$amount_to_claim" '.app_state["bank"]["balances"] += [{"address":"'$PREFIX'15cvq3ljql6utxseh0zau9m8ve2j8erz89m5wkz","coins":[{"denom":"'$TOKEN'", "amount":$amount_to_claim}]}]' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# enable prometheus metrics
	if [[ "$OSTYPE" == "darwin"* ]]; then
		sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"
		sed -i '' 's/prometheus-retention-time = 0/prometheus-retention-time  = 1000000000000/g' "$APP_TOML"
		sed -i '' 's/enabled = false/enabled = true/g' "$APP_TOML"
	else
		sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
		sed -i 's/prometheus-retention-time  = "0"/prometheus-retention-time  = "1000000000000"/g' "$APP_TOML"
		sed -i 's/enabled = false/enabled = true/g' "$APP_TOML"
	fi

	# Change proposal periods to pass within a reasonable time for local testing
	sed -i.bak 's/"max_deposit_period": "172800s"/"max_deposit_period": "30s"/g' "$HOMEDIR"/config/genesis.json
	sed -i.bak 's/"voting_period": "172800s"/"voting_period": "30s"/g' "$HOMEDIR"/config/genesis.json

	# set custom pruning settings
	sed -i.bak 's/pruning = "default"/pruning = "custom"/g' "$APP_TOML"
	sed -i.bak 's/pruning-keep-recent = "0"/pruning-keep-recent = "2"/g' "$APP_TOML"
	sed -i.bak 's/pruning-interval = "0"/pruning-interval = "10"/g' "$APP_TOML"


	# Allocate genesis accounts (cosmos formatted addresses)
    
	for KEY in "${KEYS[@]}"; do
		nxqd add-genesis-account "$KEY" 1000000000000000000000$TOKEN --keyring-backend $KEYRING --home "$HOMEDIR" # 21M
	done


	#VAULTS
	nxqd add-genesis-account nxq1xfm8n072h6antu94tac0skh42u6vj6hrquqznt 2100000000000000000000000$TOKEN --keyring-backend $KEYRING --home $HOMEDIR
	nxqd add-genesis-account nxq1ptds9mnfns4ufesjga0d0c5a392uw85dpe2hd6 2100000000000000000000000$TOKEN --keyring-backend $KEYRING --home $HOMEDIR
	nxqd add-genesis-account nxq1cmys66z7m5pv3ygcpm9y5hsyr8m65at6rqnhtk 2100000000000000000000000$TOKEN --keyring-backend $KEYRING --home $HOMEDIR
	nxqd add-genesis-account nxq1cylpqf2ssa8ajtaledychymmsmc5v8zw3kex26 2100000000000000000000000$TOKEN --keyring-backend $KEYRING --home $HOMEDIR
	nxqd add-genesis-account nxq1dmnc5utac6yz8tcd6r0hjzke3fgsf6t87yljme 2100000000000000000000000$TOKEN --keyring-backend $KEYRING --home $HOMEDIR

	#MAINTANANCE
	nxqd add-genesis-account nxq1mm3avcg58d0ht78syf3uyv30yljmyymy3fugyp 10499000000000000000000000$TOKEN --keyring-backend $KEYRING --home $HOMEDIR


	# for VAULT in "${VAULTS[@]}"; do
	# 	nxqd add-genesis-account "$VAULT" 2100000000000000000000000$TOKEN --keyring-backend $KEYRING --home "$HOMEDIR" # Half of 21M
	# done

	# bc is required to add these big numbers
	# total_supply=$(echo "${#KEYS[@]} * 10500000000000000000000000 + ${#VAULTS[@]} * 2100000000000000000000000" | bc)
	total_supply=$(echo "${#KEYS[@]} * 21000000000000000000000000" | bc)
	echo $total_supply


	jq -r --arg total_supply "$total_supply" '.app_state["bank"]["supply"][0]["amount"]=$total_supply' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Sign genesis transaction
	nxqd gentx "${KEYS[0]}" 100000000000000000000$TOKEN --keyring-backend $KEYRING --chain-id $CHAINID --home "$HOMEDIR" --details "From-GenTx"

	## In case you want to create multiple validators at genesis
	## 1. Back to `nxqd keys add` step, init more keys
	## 2. Back to `nxqd add-genesis-account` step, add balance for those
	## 3. Clone this ~/.nxqd home directory into some others, let's say `~/.clonedconfid`
	## 4. Run `gentx` in each of those folders
	## 5. Copy the `gentx-*` folders under `~/.clonedconfid/config/gentx/` folders into the original `~/.nxqd/config/gentx`

	# Collect genesis tx
	nxqd collect-gentxs --home "$HOMEDIR"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	nxqd validate-genesis --home "$HOMEDIR"

	cp $GENESIS /var/www/html/
	nxqd tendermint show-node-id  --home "$HOMEDIR" > /var/www/html/node-id

	
else	
	# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
	nxqd start --metrics "$TRACE" --log_level $LOGLEVEL --minimum-gas-prices=0.0001$TOKEN --json-rpc.enable  --grpc.enable  --json-rpc.api eth,txpool,personal,net,debug,web3 --api.enable --home "$HOMEDIR" --keyring-backend $KEYRING 
fi
