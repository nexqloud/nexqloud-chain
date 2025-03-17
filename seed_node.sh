#!/bin/bash

CHAINID="nxqd_6000-1"
MONIKER="localtestnet"
# Remember to change to other types of keyring like 'file' in-case exposing to outside world,
# otherwise your balance will be wiped quickly
# The keyring test does not require private key to steal tokens from you
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

	# myKey address 0x7cb61d4117ae31a12e393a1cfa3bac666481d02e | evmos10jmp6sgh4cc6zt3e8gw05wavvejgr5pwjnpcky
	VAL_KEY="mykey"

	# Import keys from mnemonics
	echo "$VAL_MNEMONIC" | nxqd keys add "$VAL_KEY" --recover --keyring-backend "$KEYRING" --algo "$KEYALGO" --home "$HOMEDIR"

	# Set moniker and chain-id for Evmos (Moniker can be anything, chain-id must be an integer)
	nxqd init $MONIKER -o --chain-id "$CHAINID" --home "$HOMEDIR"

	# Change parameter token denominations to nxq
	jq '.app_state["staking"]["params"]["bond_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	# When upgrade to cosmos-sdk v0.47, use gov.params to edit the deposit params
	jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["evm"]["params"]["evm_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"
	jq '.app_state["inflation"]["params"]["mint_denom"]="unxq"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set gas limit in genesis
	jq '.consensus_params["block"]["max_gas"]="10000000"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

	# Set base fee in genesis
	jq '.app_state["feemarket"]["params"]["base_fee"]="'${BASEFEE}'"' "$GENESIS" >"$TMP_GENESIS" && mv "$TMP_GENESIS" "$GENESIS"

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
		# Don't enable memiavl by default
	else
		sed -i 's/prometheus = false/prometheus = true/' "$CONFIG"
	fi

	# Change proposal periods to pass within a reasonable time for local testing
	sed -i.bak 's/"max_deposit_period": "172800s"/"max_deposit_period": "30s"/g' "$GENESIS"
	sed -i.bak 's/"voting_period": "172800s"/"voting_period": "30s"/g' "$GENESIS"

	# Allocate genesis accounts (cosmos formatted addresses)
	nxqd add-genesis-account "$(nxqd keys show "$VAL_KEY" -a --keyring-backend "$KEYRING" --home "$HOMEDIR")" 100000000000000000000000000unxq --keyring-backend "$KEYRING" --home "$HOMEDIR"

	# Sign genesis transaction and collect it
	nxqd gentx "$VAL_KEY" 1000000000000000000000unxq --gas-prices ${BASEFEE}unxq --keyring-backend "$KEYRING" --chain-id "$CHAINID" --home "$HOMEDIR"

	## In case you want to create multiple validators at genesis
	## 1. Back to `nxqd keys add` step, init more keys
	## 2. Back to `nxqd add-genesis-account` step, add balance for those
	## 3. Clone this ~/.nxqd home directory into some others, let's say `~/.clonednxqd`
	## 4. Run `gentx` in each of those folders
	## 5. Copy the `gentx-*` folders under `~/.clonednxqd/config/gentx/` folders into the original `~/.nxqd/config/gentx`

	# Collect genesis tx
	nxqd collect-gentxs --home "$HOMEDIR"

	# Run this to ensure everything worked and that the genesis file is setup correctly
	nxqd validate-genesis --home "$HOMEDIR"

	if [[ $1 == "pending" ]]; then
		echo "pending mode is on, please wait for the first block committed."
	fi

	nxqd tendermint show-node-id  --home "$HOMEDIR" > "$HOMEDIR/node-id"
	sudo cp $GENESIS /usr/share/nginx/html/
	sudo cp "$HOMEDIR/node-id" /usr/share/nginx/html/node-id

	# Enable the RPC
	sed -i '/^\[json-rpc\]/,/^\[/ s|enable = false|enable = true|' "$APP_TOML"
	sed -i 's/address = "127.0.0.1:8545"/address = "0.0.0.0:8545"/g' "$APP_TOML"
	sed -i 's/ws-address = "127.0.0.1:8546"/ws-address = "0.0.0.0:8546"/g' "$APP_TOML"
	
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
