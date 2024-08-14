package repository

import (
	"context"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

type OsmDataRepository interface {
	Import(ctx context.Context, path string) error
	GetBase(ctx context.Context, level int, bound orb.Bound) (*geojson.FeatureCollection, error)
}
