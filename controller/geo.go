package controller

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"net/http"

	"github.com/gocroot/helper/geo"
	"github.com/gocroot/model"
)

func FindNearestRoad(respw http.ResponseWriter, req *http.Request) {
	// Parse request body to get input coordinates
	var requestData struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(respw, "Invalid request", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if err := json.Unmarshal(body, &requestData); err != nil {
		http.Error(respw, "Unable to parse request body", http.StatusBadRequest)
		return
	}

	// Example JSON data (in real case, fetch from a database or service)
	data := `[{"_id": {"$oid": "673681326371846c2e09efe3"}, "type": "Feature", "geometry": {"type": "LineString", "coordinates": [[119.05235, -7.926275], [119.488743, -7.917258]]}, "properties": {"osm_id": -7226110, "name": "Sulawesi", "highway": null}}, {"_id": {"$oid": "673681326371846c2e09efe4"}, "type": "Feature", "geometry": {"type": "LineString", "coordinates": [[119.488743, -7.917258], [118.590623, -7.935815]]}, "properties": {"osm_id": -7226026, "name": "Nusa Tenggara", "highway": null}}]`

	// Parse the data
	var geoData []model.GeoData
	if err := json.Unmarshal([]byte(data), &geoData); err != nil {
		http.Error(respw, "Unable to parse geo data", http.StatusInternalServerError)
		return
	}

	// Find the nearest road
	minDistance := math.MaxFloat64
	var nearestRoad model.GeoData

	for _, road := range geoData {
		for _, coord := range road.Geometry.Coordinates {
			distance := geo.Haversine(requestData.Lat, requestData.Lon, coord[1], coord[0])
			if distance < minDistance {
				minDistance = distance
				nearestRoad = road
			}
		}
	}

	// Prepare response
	response := struct {
		NearestRoad model.GeoData `json:"nearest_road"`
		Distance    float64       `json:"distance"`
	}{
		NearestRoad: nearestRoad,
		Distance:    minDistance,
	}

	respw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(respw).Encode(response); err != nil {
		http.Error(respw, "Unable to encode response", http.StatusInternalServerError)
	}
}
