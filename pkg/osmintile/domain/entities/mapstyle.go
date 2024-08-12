package entities

// MapStyle for reference see: https://docs.mapbox.com/style-spec/reference/root
type MapStyle struct {
	Version int               `json:"version"` // must be 8
	Layers  []Layer           `json:"layers"`
	Sources map[string]Source `json:"sources"`
}

// Layer for reference see: https://docs.mapbox.com/style-spec/reference/layers
type Layer struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Source      string `json:"source"`
	SourceLayer string `json:"source-layer"`
	*FillLayer
}

// FillLayer for reference see: https://docs.mapbox.com/style-spec/reference/layers#fill
type FillLayer struct {
	FillColor string `json:"fill-color"`
}

// Source for reference see: https://docs.mapbox.com/style-spec/reference/sources
type Source struct {
	Type      string   `json:"type"`
	TilesURLs []string `json:"tiles"`
}
