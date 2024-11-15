package model

type GeoData struct {
	ID         ObjectID   `json:"_id"`
	Type       string     `json:"type"`
	Geometry   Geometry   `json:"geometry"`
	Properties Properties `json:"properties"`
}

type ObjectID struct {
	Oid string `json:"$oid"`
}

type Geometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type Properties struct {
	OsmID   int    `json:"osm_id"`
	Name    string `json:"name"`
	Highway string `json:"highway"`
}
