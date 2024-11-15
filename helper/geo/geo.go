package geo

import (
	"container/heap"
	"fmt"
	"math"
)

func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in kilometers
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1 = lat1 * math.Pi / 180.0
	lat2 = lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func PointToLineDistance(lat1, lon1, lat2, lon2, pLat, pLon float64) float64 {
	lat1Rad, lon1Rad := lat1*math.Pi/180, lon1*math.Pi/180
	lat2Rad, lon2Rad := lat2*math.Pi/180, lon2*math.Pi/180
	pLatRad, pLonRad := pLat*math.Pi/180, pLon*math.Pi/180

	dLon := lon2Rad - lon1Rad
	dLat := lat2Rad - lat1Rad
	u := ((pLatRad-lat1Rad)*dLat + (pLonRad-lon1Rad)*dLon) / (dLat*dLat + dLon*dLon)

	u = math.Max(0, math.Min(1, u))

	closestLat := lat1Rad + u*dLat
	closestLon := lon1Rad + u*dLon

	return Haversine(pLat*180/math.Pi, pLon*180/math.Pi, closestLat*180/math.Pi, closestLon*180/math.Pi)
}

func CoordinateToID(lat, lon float64) string {
	return fmt.Sprintf("%f_%f", lat, lon)
}

const EarthRadiusKm = 6371.0

func PointToPointDistance(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Compute the differences
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	// Apply Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Calculate distance
	return EarthRadiusKm * c
}

type Node struct {
	ID        string
	Latitude  float64
	Longitude float64
	Neighbors []Edge
}

// FindNearestNode finds the closest node in the graph to the specified latitude and longitude.
func FindNearestNode(graph map[string]*Node, targetLat, targetLon float64) string {
	minDistance := math.MaxFloat64
	nearestNodeID := ""

	for _, node := range graph {
		distance := PointToPointDistance(node.Latitude, node.Longitude, targetLat, targetLon)
		if distance < minDistance {
			minDistance = distance
			nearestNodeID = node.ID
		}
	}

	return nearestNodeID
}

type Nodes struct {
	ID        string
	Latitude  float64
	Longitude float64
	Neighbors []Edge // List of edges to neighboring nodes
}

// Edge represents a connection between two nodes with a certain distance (weight).
type Edge struct {
	ToNodeID string
	Distance float64
}

// PriorityQueueItem is a struct to help manage the priority queue in Dijkstra's algorithm.
type PriorityQueueItem struct {
	NodeID   string
	Distance float64
	Index    int
}

// PriorityQueue implements a priority queue for managing nodes by distance.
type PriorityQueue []*PriorityQueueItem

func (pq PriorityQueue) Len() int { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Distance < pq[j].Distance
}
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*PriorityQueueItem)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Update(item *PriorityQueueItem, distance float64) {
	item.Distance = distance
	heap.Fix(pq, item.Index)
}

// Dijkstra finds the shortest path from startID to endID in the graph and returns the distance and path.
func Dijkstra(graph map[string]*Node, startID, endID string) (float64, []string) {
	// Initialize distances and previous node map
	distances := make(map[string]float64)
	previous := make(map[string]string)
	for nodeID := range graph {
		distances[nodeID] = math.Inf(1)
	}
	distances[startID] = 0

	// Priority queue to store nodes by distance
	pq := PriorityQueue{}
	heap.Init(&pq)
	heap.Push(&pq, &PriorityQueueItem{NodeID: startID, Distance: 0})

	for pq.Len() > 0 {
		// Pop the node with the smallest distance
		item := heap.Pop(&pq).(*PriorityQueueItem)
		currentID := item.NodeID

		// Stop if we reached the end node
		if currentID == endID {
			break
		}

		currentNode := graph[currentID]
		for _, edge := range currentNode.Neighbors {
			// Calculate alternative distance
			alt := distances[currentID] + edge.Distance
			if alt < distances[edge.ToNodeID] {
				// Update distance and previous node if shorter path is found
				distances[edge.ToNodeID] = alt
				previous[edge.ToNodeID] = currentID
				heap.Push(&pq, &PriorityQueueItem{NodeID: edge.ToNodeID, Distance: alt})
			}
		}
	}

	// Reconstruct the path from endID to startID
	path := []string{}
	for u := endID; u != ""; u = previous[u] {
		path = append([]string{u}, path...)
	}

	// Return the shortest distance and path
	return distances[endID], path
}
