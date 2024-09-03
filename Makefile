ARGS=--osm-file data/stachus-latest.osm.pbf

PHONY: setup
setup:
	docker-compose build
	docker-compose up -d

PHONY: test
test: setup
	docker-compose exec toolbox go test ./...

PHONY: help
help: setup
	docker-compose exec toolbox go run ./cmd/osmintile/ -h

.PHONY: run
run: setup
	docker-compose exec toolbox go run ./cmd/osmintile/ $(ARGS)