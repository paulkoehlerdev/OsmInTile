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

func (m *mapTilesService) GetMapTile(ctx context.Context, tile maptile.Tile, acceptGzip bool) ([]byte, error) {
	bounds := tile.Bound(1)
	collections, err := m.getFeaturesFor(ctx, bounds)
	if err != nil {
		return nil, fmt.Errorf("error getting features: %w", err)
	}

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

func (m *mapTilesService) getFeaturesFor(ctx context.Context, bounds orb.Bound) (map[string]*geojson.FeatureCollection, error) {
	pois, err := m.dataRepository.GetBuildings(ctx, bounds)
	if err != nil {
		return nil, fmt.Errorf("get pois failed: %w", err)
	}

	return map[string]*geojson.FeatureCollection{
		"osm-indoor-buildings": pois,
	}, nil
}

func (m *mapTilesService) cleanLayers(layers mvt.Layers) mvt.Layers {
	layers.Clip(mvt.MapboxGLDefaultExtentBound)
	layers.Simplify(simplify.DouglasPeucker(1.0))
	layers.RemoveEmpty(1.0, 2.0)
	return layers
}
