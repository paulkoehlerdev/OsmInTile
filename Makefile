
PHONY: setup
setup:
	docker-compose build
	docker-compose up -d

PHONY: test
test: setup
	docker-compose exec toolbox go test ./...