package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/entities"
	"github.com/paulkoehlerdev/OsmInTile/pkg/osmintile/domain/repository"
	"github.com/paulkoehlerdev/OsmInTile/styles"
	"text/template"
)

type MapStyleService interface {
	GetMapStyle(ctx context.Context) (entities.MapStyle, error)
}

type mapStyleService struct {
	publicUrl      string
	templates      *template.Template
	dataRepository repository.OsmDataRepository
}

type mapStyleInfo struct {
	PublicURL string
	Bounds    string
	Center    string
}

func NewMapStyleService(publicUrl string, dataRepository repository.OsmDataRepository) (MapStyleService, error) {
	templates, err := template.ParseFS(styles.FS, "*.json")
	if err != nil {
		return nil, fmt.Errorf("error loading template dir: %w", err)
	}

	return &mapStyleService{
		publicUrl:      publicUrl,
		templates:      templates,
		dataRepository: dataRepository,
	}, nil
}

func (m *mapStyleService) GetMapStyle(ctx context.Context) (entities.MapStyle, error) {
	return m.getMapStyle(ctx, "default.json")
}

func (m *mapStyleService) getMapStyle(ctx context.Context, name string) (entities.MapStyle, error) {
	styleInfo, err := m.getMapStyleInfo(ctx)
	if err != nil {
		return entities.MapStyle{}, fmt.Errorf("error getting map style info: %w", err)
	}

	writer := bytes.Buffer{}
	err = m.templates.ExecuteTemplate(&writer, name, styleInfo)
	if err != nil {
		return entities.MapStyle{}, fmt.Errorf("error executing template: %w", err)
	}

	return writer.Bytes(), nil
}

func (m *mapStyleService) getMapStyleInfo(ctx context.Context) (mapStyleInfo, error) {
	bound, err := m.getMapBounds(ctx)
	if err != nil {
		return mapStyleInfo{}, fmt.Errorf("error getting map bounds: %w", err)
	}

	boundJson, err := json.Marshal(bound)
	if err != nil {
		return mapStyleInfo{}, fmt.Errorf("error marshalling bounds: %w", err)
	}

	center, err := m.getMapCenter(ctx)
	if err != nil {
		return mapStyleInfo{}, fmt.Errorf("error getting map center: %w", err)
	}

	centerJson, err := json.Marshal(center)
	if err != nil {
		return mapStyleInfo{}, fmt.Errorf("error marshalling center: %w", err)
	}

	return mapStyleInfo{
		PublicURL: m.publicUrl,
		Bounds:    string(boundJson),
		Center:    string(centerJson),
	}, nil
}

func (m *mapStyleService) getMapBounds(ctx context.Context) ([4]float64, error) {
	bound, err := m.dataRepository.GetMapBounds(ctx)
	if err != nil {
		return [4]float64{}, fmt.Errorf("error getting map bounds: %w", err)
	}

	out := [4]float64{}
	out[0] = bound.Min.Lon()
	out[1] = bound.Min.Lat()
	out[2] = bound.Max.Lon()
	out[3] = bound.Max.Lat()

	return out, nil
}

func (m *mapStyleService) getMapCenter(ctx context.Context) ([2]float64, error) {
	center, err := m.dataRepository.GetMapCenter(ctx)
	if err != nil {
		return [2]float64{}, fmt.Errorf("error getting map center: %w", err)
	}

	out := [2]float64{}
	out[0] = center.Lon()
	out[1] = center.Lat()

	return out, nil
}
