
PHONY: setup
setup:
	docker-compose build
	docker-compose up -d

PHONY: test
test: setup
	docker-compose exec toolbox go test ./...

.PHONY: import
import: setup
	docker-compose exec toolbox go run ./cmd/osmintile/ --osm-file data/map.osm --database db.sqlite

.PHONY: run
run: setup
	docker-compose exec toolbox go run ./cmd/osmintile/ --database db.sqlite