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
	@go test ./go-contracts/...
	@go test ./service/...
	@go test

.PHONY: test-clean
test-clean: clean-testcache test

.PHONY: clean-testcache
clean-testcache:
	@go clean -testcache

.PHONY: bench
bench:
	@go test -bench=. -run=^a
