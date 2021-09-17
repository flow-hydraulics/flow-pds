.PHONY: dev
dev:
	docker-compose up -d db pgadmin emulator
	docker-compose logs -f

.PHONY: stop
stop:
	docker-compose stop

.PHONY: down
down:
	docker-compose down

.PHONY: reset
reset: down dev

.PHONY: test
test:
	@go test $$(go list ./... | grep -v /go-contracts/)

.PHONY: test-clean
test-clean: clean-testcache test

.PHONY: clean-testcache
clean-testcache:
	@go clean -testcache

.PHONY: deploy
deploy:
	flow project deploy --update
