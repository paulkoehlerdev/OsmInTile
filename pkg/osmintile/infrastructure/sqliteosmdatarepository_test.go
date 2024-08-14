package infrastructure_test

import (
	"context"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/infrastructure"
	"testing"
)

func testSetup(t *testing.T) *infrastructure.SqliteOsmDataRepository {
	repo, err := infrastructure.NewSqliteOsmDataRepository(":memory:")
	if err != nil {
		t.Fatal(err)
		return nil
	}

	err = repo.Import(context.Background(), "/app/data/stachus-latest.osm.pbf")
	if err != nil {
		t.Fatal(err)
		return nil
	}

	return repo
}

func TestSqliteOsmDataRepository_Import(t *testing.T) {
	testSetup(t)
}
