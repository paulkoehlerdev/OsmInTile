package repository

import (
	"context"
)

type OsmDataRepository interface {
	Import(ctx context.Context, path string) error
	// Getter function for each Layer
}
