#/bin/bash

source .env.test

OS_NAME=$(uname -s | awk '{print tolower($0)}')
CPU_ARCH=$(uname -m)
PROJECT_ROOT=$(pwd)
EXEC_PATH="$PROJECT_ROOT"/.github/flow-"$OS_NAME"-"$CPU_ARCH"
# EXEC_PATH=/usr/local/bin/flow

shopt -s expand_aliases
alias flow='$EXEC_PATH -f $PROJECT_ROOT/flow.json'

echo "PDS account balance before"
flow accounts get f3fcd2c1a78f5eee | grep Balance

echo "Issuer account balance before"
flow accounts get 01cf0e2f2f715450 | grep Balance

go clean -testcache
TEST_COLLECTIBLES=10000 go test -v -run ^TestE2ELarge$ github.com/flow-hydraulics/flow-pds

echo "PDS account balance after"
flow accounts get f3fcd2c1a78f5eee | grep Balance

echo "Issuer account balance after"
flow accounts get 01cf0e2f2f715450 | grep Balance
