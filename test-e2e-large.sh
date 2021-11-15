#/bin/bash

go clean -testcache
TEST_PACK_COUNT=100 go test -timeout 24h -v -run ^TestE2E$ .
