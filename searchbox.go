package mapbox

// BoundingBox is a geographic bounding box.
type BoundingBox struct {
	MinLongitude float64
	MinLatitude  float64
	MaxLongitude float64
	MaxLatitude  float64
}

// Suggestion is a single autocomplete suggestion from /suggest.
type Suggestion struct {
	// MapboxID is used as the path parameter for /retrieve.
	// Only valid for the duration of the session (180s); do not store it.
	MapboxID       string          `json:"mapbox_id"`
	Name           string          `json:"name"`
	FeatureType    string          `json:"feature_type"`
	PlaceFormatted string          `json:"place_formatted,omitempty"`
	FullAddress    string          `json:"full_address,omitempty"`
	Address        string          `json:"address,omitempty"`
	Language       string          `json:"language,omitempty"`
	Context        *FeatureContext `json:"context,omitempty"`
	POICategory    []string        `json:"poi_category,omitempty"`
	// Distance is meters from the proximity point, if provided.
	Distance *float64 `json:"distance,omitempty"`
}

// RetrieveFeature is a GeoJSON Feature returned by /retrieve.
type RetrieveFeature struct {
	Type       string                    `json:"type"`
	Geometry   PointGeometry             `json:"geometry"`
	Properties RetrieveFeatureProperties `json:"properties"`
}

// RetrieveFeatureProperties contains the resolved place details from /retrieve.
type RetrieveFeatureProperties struct {
	MapboxID       string              `json:"mapbox_id"`
	FeatureType    string              `json:"feature_type"`
	Name           string              `json:"name"`
	PlaceFormatted string              `json:"place_formatted,omitempty"`
	FullAddress    string              `json:"full_address,omitempty"`
	Coordinates    RetrieveCoordinates `json:"coordinates"`
	Context        FeatureContext      `json:"context"`
}

// RetrieveCoordinates contains the resolved coordinate and accuracy from /retrieve.
type RetrieveCoordinates struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	// Accuracy describes the precision of the coordinate
	// (e.g. "rooftop", "parcel", "interpolated").
	Accuracy string `json:"accuracy,omitempty"`
}
