package repository

import (
	"context"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

type OsmDataRepository interface {
	Import(ctx context.Context, path string) error
	GetBuildings(ctx context.Context, bound orb.Bound) (*geojson.FeatureCollection, error)
}
