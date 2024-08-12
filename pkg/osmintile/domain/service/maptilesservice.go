package service

import (
	"context"
	"fmt"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/repository"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"github.com/paulmach/orb/simplify"
	"math"
)

type MapTilesService interface {
	GetMapTile(ctx context.Context, tile maptile.Tile, acceptGzip bool) ([]byte, error)
}

type mapTilesService struct {
	dataRepository repository.OsmDataRepository
}

func NewMapTilesService(dataRepository repository.OsmDataRepository) MapTilesService {
	return &mapTilesService{
		dataRepository: dataRepository,
	}
}

func (m mapTilesService) GetMapTile(ctx context.Context, tile maptile.Tile, acceptGzip bool) ([]byte, error) {
	bounds := tile.Bound(1)
	collections, err := m.getFeaturesFor(ctx, bounds)

	layers := mvt.NewLayers(collections)
	layers.ProjectToTile(tile)

	layers = m.cleanLayers(layers)

	var data []byte
	if acceptGzip {
		data, err = mvt.MarshalGzipped(layers)
	} else {
		data, err = mvt.Marshal(layers)
	}
	if err != nil {
		return nil, fmt.Errorf("marshal layers failed: %w", err)
	}

	return data, nil
}

func (m mapTilesService) getFeaturesFor(ctx context.Context, bounds orb.Bound) (map[string]*geojson.FeatureCollection, error) {
	return map[string]*geojson.FeatureCollection{
		"test-pattern": m.generateTestPattern(bounds),
	}, nil
}

func (m mapTilesService) generateTestPattern(bound orb.Bound) *geojson.FeatureCollection {
	const TEST_PATTERN_RESOLUTION = 20

	center := bound.Center()
	radius := (bound.Left() - center.Lat()) * 0.125

	geometry := orb.LineString{}

	for i := 0; i <= TEST_PATTERN_RESOLUTION; i++ {
		angle := (float64(i)/float64(TEST_PATTERN_RESOLUTION))*(math.Pi*2) + math.Pi/4
		lat := math.Cos(angle)*radius + center.Lat()
		lon := math.Sin(angle)*radius + center.Lon()
		point := orb.Point{
			lon, lat,
		}
		geometry = append(geometry, point)
	}

	fc := geojson.NewFeatureCollection()
	fc.Append(geojson.NewFeature(geometry))
	return fc
}

func (m mapTilesService) cleanLayers(layers mvt.Layers) mvt.Layers {
	layers.Clip(mvt.MapboxGLDefaultExtentBound)
	layers.Simplify(simplify.DouglasPeucker(1.0))
	layers.RemoveEmpty(1.0, 2.0)
	return layers
}
