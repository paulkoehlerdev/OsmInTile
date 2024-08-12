package service

import (
	"context"
	"fmt"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/entities"
)

const OSMINTILE_VECTOR_SOURCE = "osmintile"

type MapStyleService interface {
	GetMapStyle(ctx context.Context) (entities.MapStyle, error)
}

type mapStyleService struct {
	publicUrl string
}

func NewMapStyleService(publicUrl string) MapStyleService {
	return &mapStyleService{
		publicUrl: publicUrl,
	}
}

func (m *mapStyleService) GetMapStyle(ctx context.Context) (entities.MapStyle, error) {
	return m.defaultMapStyle(), nil
}

func (m *mapStyleService) defaultMapStyle() entities.MapStyle {
	return entities.MapStyle{
		Version: 8,
		Layers: []entities.Layer{
			{
				ID:          "Points",
				Type:        "fill",
				Source:      OSMINTILE_VECTOR_SOURCE,
				SourceLayer: "test-polygons",
			},
		},
		Sources: map[string]entities.Source{
			OSMINTILE_VECTOR_SOURCE: {
				Type: "vector",
				TilesURLs: []string{
					fmt.Sprintf("%s/tiles/{z}/{x}/{y}.pbf", m.publicUrl),
				},
			},
		},
	}
}
