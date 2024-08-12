package infrastructure_test

import (
	"context"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/infrastructure"
	"github.com/paulmach/orb"
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

func TestSqliteOsmDataRepository_GetPOIs(t *testing.T) {
	repo := testSetup(t)

	_, err := repo.GetPOIs(context.Background(), orb.Bound{
		Min: orb.Point{-180, -90},
		Max: orb.Point{180, 90},
	})
	if err != nil {
		t.Fatal(err)
		return
	}
}
