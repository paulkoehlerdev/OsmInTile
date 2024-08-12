package application

import (
	"context"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/entities"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/service"
	"github.com/paulmach/orb/maptile"
)

type Application interface {
	GetMapStyle(ctx context.Context) (entities.MapStyle, error)
	GetTile(ctx context.Context, x, y, z uint32, acceptGzip bool) ([]byte, error)
}

type application struct {
	styleService service.MapStyleService
	tilesService service.MapTilesService
}

func New(styleService service.MapStyleService, tilesService service.MapTilesService) Application {
	return &application{
		styleService: styleService,
		tilesService: tilesService,
	}
}

func (app *application) GetMapStyle(ctx context.Context) (entities.MapStyle, error) {
	return app.styleService.GetMapStyle(ctx)
}

func (app *application) GetTile(ctx context.Context, x, y, z uint32, acceptGzip bool) ([]byte, error) {
	tile := maptile.Tile{
		X: x,
		Y: y,
		Z: maptile.Zoom(z),
	}
	return app.tilesService.GetMapTile(ctx, tile, acceptGzip)
}
