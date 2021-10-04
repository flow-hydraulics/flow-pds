# export vars from root env file
ifneq (,$(wildcard ./.env.test))
	include .env.test
	export
endif

.PHONY: dev
dev: up deploy

.PHONY: deploy
deploy:
	bash ./deploy.sh

.PHONY: stop
stop:
	docker-compose stop

.PHONY: up
up:
	docker-compose up -d db pgadmin emulator

.PHONY: down
down:
	docker-compose down

.PHONY: reset
reset: down dev

.PHONY: test
test:
	@go test ./go-contracts/... -v
	@go test ./service/... -v
	@go test -v

.PHONY: test-contracts
test-contracts:
	@go test ./go-contracts/contracts_test.go -v

.PHONY: tests-with-emulator
tests-with-emulator:
	./tests-with-emulator.sh

.PHONY: test-clean
test-clean: clean-testcache test

.PHONY: clean-testcache
clean-testcache:
	@go clean -testcache

.PHONY: bench
bench:
	@go test -bench=. -run=^a
