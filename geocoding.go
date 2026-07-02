package mapbox

// FeatureCollection is a GeoJSON FeatureCollection returned by Geocoding v6.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

// PointGeometry is a GeoJSON Point geometry. Coordinates holds [longitude, latitude].
type PointGeometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// Feature is a GeoJSON Feature with Mapbox v6 geocoding properties.
type Feature struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Geometry   PointGeometry     `json:"geometry"`
	Properties FeatureProperties `json:"properties"`
}

// FeatureProperties contains the Mapbox v6 geocoding result properties.
type FeatureProperties struct {
	// MapboxID is the unique Mapbox identifier for this feature.
	MapboxID string `json:"mapbox_id"`
	// FeatureType is the type of feature (e.g. "address", "place", "region").
	FeatureType string `json:"feature_type"`
	// Name is the primary name of the feature.
	Name string `json:"name"`
	// PlaceFormatted is the place portion of the full address.
	PlaceFormatted string `json:"place_formatted,omitempty"`
	// FullAddress is the full formatted address string.
	FullAddress string `json:"full_address,omitempty"`
	// Context contains the hierarchical address components.
	Context FeatureContext `json:"context"`
}

// FeatureContext contains the hierarchical address components from Mapbox v6.
// All fields are optional pointers — nil indicates the component is not present
// for this feature.
type FeatureContext struct {
	Address      *AddressContext `json:"address,omitempty"`
	Street       *ContextItem    `json:"street,omitempty"`
	Neighborhood *ContextItem    `json:"neighborhood,omitempty"`
	Postcode     *ContextItem    `json:"postcode,omitempty"`
	Locality     *ContextItem    `json:"locality,omitempty"`
	Place        *ContextItem    `json:"place,omitempty"`
	District     *ContextItem    `json:"district,omitempty"`
	Region       *RegionContext  `json:"region,omitempty"`
	Country      *CountryContext `json:"country,omitempty"`
}

// AddressContext contains the street-level address components.
type AddressContext struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// AddressNumber is the building/house number.
	AddressNumber string `json:"address_number"`
	// StreetName is the street name.
	StreetName string `json:"street_name"`
}

// ContextItem is a single named level in the address hierarchy.
type ContextItem struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// Name is the display name for this context level.
	Name string `json:"name"`
}

// RegionContext is the region/state/province level in the address hierarchy.
type RegionContext struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// Name is the region name.
	Name string `json:"name"`
	// RegionCode is the ISO 3166-2 region code (e.g. "DE-BY").
	RegionCode string `json:"region_code,omitempty"`
	// RegionCodeFull is the full ISO 3166-2 region code.
	RegionCodeFull string `json:"region_code_full,omitempty"`
}

// CountryContext is the country level in the address hierarchy.
type CountryContext struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// Name is the country name.
	Name string `json:"name"`
	// CountryCode is the ISO 3166-1 alpha-2 country code (e.g. "DE").
	CountryCode string `json:"country_code,omitempty"`
	// CountryCodeAlpha3 is the ISO 3166-1 alpha-3 country code (e.g. "DEU").
	CountryCodeAlpha3 string `json:"country_code_alpha_3,omitempty"`
}
