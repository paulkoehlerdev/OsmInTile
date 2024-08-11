package repository

import (
	"context"
	"github.com/paulmach/osm"
)

type OsmDataRepository interface {
	Import(ctx context.Context, path string) error
	GetNode(ctx context.Context, id int64) (*osm.Node, error)
	GetWay(ctx context.Context, id int64) (*osm.Way, error)
	GetRelation(ctx context.Context, id int64) (*osm.Relation, error)
}
