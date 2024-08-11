package infrastructure_test

import (
	"context"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/infrastructure"
	"testing"
)

func TestNewSqliteOsmDataRepository(t *testing.T) {
	_, err := infrastructure.NewSqliteOsmDataRepository(":memory:")
	if err != nil {
		t.Fatal(err)
		return
	}
}

func TestSqliteOsmDataRepository_Import(t *testing.T) {
	repo, err := infrastructure.NewSqliteOsmDataRepository(":memory:")
	if err != nil {
		t.Fatal(err)
		return
	}

	err = repo.Import(context.Background(), "/app/data/stachus-latest.osm.pbf")
	if err != nil {
		t.Fatal(err)
		return
	}
}
