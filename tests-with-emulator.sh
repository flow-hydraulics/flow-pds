#!/bin/bash

if ! command -v flow &> /dev/null
then
    echo "Flow CLI could not be found"
    exit
fi

if ! command -v go &> /dev/null
then
    echo "Go could not be found"
    exit
fi

# Check to see if it's running in the right directory
if [ ! -f "./flow.json" ]; then
  echo "IMPORTANT: This script must be run from the 'flow-pds' root folder, not a subdirectory"
  exit 1
fi

set -a # Mark variables which are modified or created for export to the environment of subsequent commands. Needed for 'go run deploy/main.go' later in the script.
source .env.test
set +a
set -ex
shopt -s expand_aliases
# Run the emulator with the config in ./flow.json
if [ "${NETWORK}" == "emulator" ]; then
  # setting block-time of 1s to emulate testnet + mainnet tempo
  flow emulator -b 1s &
  EMULATOR_PID=$!

  function tearDown {
    kill $EMULATOR_PID
  }

  trap tearDown EXIT
  sleep 1
  SIGNER=emulator-account

  # Create owner, issuer, pds, account, note all accounts have the same keys in this setup
  # addresses are deterministic and listed in flow.json
  PK=46acb0e0918e09a50fc2a6b12f14fc00822ad7dac6c6fd92427ec675b9745cbe5ae93d790e6fdd0683d7dd17b6156cc4201def8d6a992807796a5ce4a789005f

  flow accounts create --network="$NETWORK" --key="$PK" --signer="$SIGNER"
  flow accounts create --network="$NETWORK" --key="$PK" --signer="$SIGNER"
  flow accounts create --network="$NETWORK" --key="$PK" --signer="$SIGNER"

  # Owner
  flow transactions send ./cadence-transactions/flowTokens/transfer_flow_tokens_emulator.cdc \
    100.0 0x01cf0e2f2f715450 --signer="$SIGNER"  --network="$NETWORK"
  # Issuer
  flow transactions send ./cadence-transactions/flowTokens/transfer_flow_tokens_emulator.cdc \
    100.0 0x179b6b1cb6755e31 --signer="$SIGNER"  --network="$NETWORK"
  # PDS account
  flow transactions send ./cadence-transactions/flowTokens/transfer_flow_tokens_emulator.cdc \
    100.0 0xf3fcd2c1a78f5eee --signer="$SIGNER"  --network="$NETWORK"
fi

flow project deploy --network="$NETWORK" --update=true

(cd go-contracts/; go run deploy/main.go)

go clean -testcache

go test ./go-contracts/... -v
go test ./service/... -v
go test -v
