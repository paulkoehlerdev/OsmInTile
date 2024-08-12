package application

import (
	"context"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/entities"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/service"
)

type Application interface {
	GetMapStyle(ctx context.Context) (entities.MapStyle, error)
}

type application struct {
	styleService service.MapStyleService
}

func New(styleService service.MapStyleService) Application {
	return &application{
		styleService: styleService,
	}
}

func (app *application) GetMapStyle(ctx context.Context) (entities.MapStyle, error) {
	return app.styleService.GetMapStyle(ctx)
}
