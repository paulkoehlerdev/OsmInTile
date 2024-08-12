
PHONY: setup
setup:
	docker-compose build
	docker-compose up -d

PHONY: test
test: setup
	docker-compose exec toolbox go test ./...

.PHONY: run
run: setup
	docker-compose exec toolbox go run ./cmd/osmintile/
