package controller

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/geo"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RouteRequest struct {
	StartLat float64 `json:"start_lat"`
	StartLon float64 `json:"start_lon"`
	EndLat   float64 `json:"end_lat"`
	EndLon   float64 `json:"end_lon"`
}

type Edge struct {
	ToNodeID string
	Distance float64
}

// FindShortestRoute handles finding the shortest route between two points
func FindShortestRoute(respw http.ResponseWriter, req *http.Request) {
	var requestData RouteRequest

	// Parse request body
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

	// Example road data
	data := `[{"_id": {"$oid": "673681326371846c2e09efe3"}, "type": "Feature", "geometry": {"type": "LineString", "coordinates": [[119.05235, -7.926275], [119.488743, -7.917258]]}, "properties": {"osm_id": -7226110, "name": "Sulawesi", "highway": null}}, {"_id": {"$oid": "673681326371846c2e09efe4"}, "type": "Feature", "geometry": {"type": "LineString", "coordinates": [[119.488743, -7.917258], [118.590623, -7.935815]]}, "properties": {"osm_id": -7226026, "name": "Nusa Tenggara", "highway": null}}]`

	var geoData []model.GeoData
	if err := json.Unmarshal([]byte(data), &geoData); err != nil {
		http.Error(respw, "Unable to parse geo data", http.StatusInternalServerError)
		return
	}

	// Build the graph from geoData
	graph := make(map[string]*geo.Node)
	for _, road := range geoData {
		for i := 0; i < len(road.Geometry.Coordinates)-1; i++ {
			start := road.Geometry.Coordinates[i]
			end := road.Geometry.Coordinates[i+1]

			startID := geo.CoordinateToID(start[1], start[0])
			endID := geo.CoordinateToID(end[1], end[0])

			// Create or retrieve start node
			startNode, exists := graph[startID]
			if !exists {
				startNode = &geo.Node{ID: startID, Latitude: start[1], Longitude: start[0]}
				graph[startID] = startNode
			}

			// Create or retrieve end node
			endNode, exists := graph[endID]
			if !exists {
				endNode = &geo.Node{ID: endID, Latitude: end[1], Longitude: end[0]}
				graph[endID] = endNode
			}

			// Calculate distance between nodes and create edges
			distance := geo.PointToPointDistance(start[1], start[0], end[1], end[0])
			startNode.Neighbors = append(startNode.Neighbors, geo.Edge{ToNodeID: endID, Distance: distance})
			endNode.Neighbors = append(endNode.Neighbors, geo.Edge{ToNodeID: startID, Distance: distance})
		}
	}

	// Find the nearest nodes to the start and end coordinates
	startID := geo.FindNearestNode(graph, requestData.StartLat, requestData.StartLon)
	endID := geo.FindNearestNode(graph, requestData.EndLat, requestData.EndLon)

	// Apply Dijkstra to find the shortest path
	distance, path := geo.Dijkstra(graph, startID, endID)

	// Prepare and send response
	response := struct {
		Distance float64  `json:"distance"`
		Path     []string `json:"path"`
	}{
		Distance: distance,
		Path:     path,
	}

	respw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(respw).Encode(response); err != nil {
		http.Error(respw, "Unable to encode response", http.StatusInternalServerError)
	}
}

func TestData(w http.ResponseWriter, req *http.Request) {
	DataLokasi := req.URL.Query().Get("idLocation")
	if DataLokasi == "" {
		var respn *http.Response
		respn.Status = "Error: ID Lokasi Tidak Ditemukan"
		at.WriteJSON(w, http.StatusBadRequest, respn)
		return
	}

	objectIdData, err := primitive.ObjectIDFromHex(DataLokasi)
	if err != nil {
		http.Error(w, "ID Lokasi Tidak Valid", http.StatusBadRequest)
		return
	}

	filter := bson.M{"_id": objectIdData}

	dataLocation, err := atdb.GetOneDoc[model.GeoData](config.MongoconnGeo, "geo", filter)
	if err != nil {
		http.Error(w, "Data Lokasi Tidak Ditemukan", http.StatusNotFound)
		return
	}

	at.WriteJSON(w, http.StatusOK, dataLocation)
}

func GetRoads(respw http.ResponseWriter, req *http.Request) {
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))

	if err != nil {
		_, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))

		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Info = at.GetSecretFromHeader(req)
			respn.Location = "Decode Token Error"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	var longlat model.LongLat
	err = json.NewDecoder(req.Body).Decode(&longlat)
	if err != nil {
		var respn model.Response
		respn.Status = "Error : Body tidak valid"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}
	
	filter := bson.M{
		"geometry": bson.M{
			"$nearSphere": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{longlat.Longitude, longlat.Latitude},
				},
				"$maxDistance": longlat.MaxDistance,
			},
		},
	}

	roads, err := atdb.GetAllDoc[[]model.Roads](config.MongoconnGeo, "jalan", filter)
	if err != nil {
		at.WriteJSON(respw, http.StatusNotFound, roads)
		return
	}
	at.WriteJSON(respw, http.StatusOK, roads)
}

func GetRegion(respw http.ResponseWriter, req *http.Request) {
	// Dekode token untuk autentikasi
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		var respn model.Response
		respn.Status = "Error : Token Tidak Valid"
		respn.Location = "Decode Token Error: " + at.GetLoginFromHeader(req)
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	// Parse koordinat dari body request
	var longlat model.LongLat
	err = json.NewDecoder(req.Body).Decode(&longlat)
	if err != nil {
		var respn model.Response
		respn.Status = "Error : Body tidak valid"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Filter query geospasial
	filter := bson.M{
		"border": bson.M{
			"$geoIntersects": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{longlat.Longitude, longlat.Latitude},
				},
			},
		},
	}

	// Cari region berdasarkan filter
	region, err := atdb.GetOneDoc[model.Region](config.MongoconnGeoVill, "map", filter)
	if err != nil {
		at.WriteJSON(respw, http.StatusNotFound, bson.M{"error": "Region not found"})
		return
	}

	// Format respon sebagai FeatureCollection GeoJSON
	geoJSON := bson.M{
		"type": "FeatureCollection",
		"features": []bson.M{
			{
				"type": "Feature",
				"geometry": bson.M{
					"type":        region.Border.Type,
					"coordinates": region.Border.Coordinates,
				},
				"properties": bson.M{
					"province":     region.Province,
					"district":     region.District,
					"sub_district": region.SubDistrict,
					"village":      region.Village,
				},
			},
		},
	}

	// Kirim respon dalam format GeoJSON
	at.WriteJSON(respw, http.StatusOK, geoJSON)
}
