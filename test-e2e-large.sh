#/bin/bash

go clean -testcache
TEST_COLLECTIBLES=1000 go test -timeout 180m -v -run ^TestE2E$ .
