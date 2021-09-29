#/bin/bash

source .env.test

OS_NAME=$(uname -s | awk '{print tolower($0)}')
CPU_ARCH=$(uname -m)
PROJECT_ROOT=$(pwd)
EXEC_PATH="$PROJECT_ROOT"/.github/flow-"$OS_NAME"-"$CPU_ARCH"
# EXEC_PATH=/usr/local/bin/flow

shopt -s expand_aliases
alias flow='$EXEC_PATH -f $PROJECT_ROOT/flow.json'

go clean -testcache
TEST_COLLECTIBLES=1000 go test -timeout 180m -v -run ^TestE2ELarge$ .
