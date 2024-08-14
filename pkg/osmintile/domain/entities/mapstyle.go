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
	Paint       Paint  `json:"paint"`
}

// Paint for reference see: https://docs.mapbox.com/style-spec/reference/layers/#paint
type Paint struct {
	*FillLayer
}

// FillLayer for reference see: https://docs.mapbox.com/style-spec/reference/layers#fill
type FillLayer struct {
	FillColor        *string  `json:"fill-color,omitempty"`
	FillOutlineColor *string  `json:"fill-outline-color,omitempty"`
	FillOpacity      *float64 `json:"fill-opacity,omitempty"`
}

// Source for reference see: https://docs.mapbox.com/style-spec/reference/sources
type Source struct {
	Type      string   `json:"type"`
	TilesURLs []string `json:"tiles"`
}
