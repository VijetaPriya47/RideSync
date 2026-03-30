package types

import pb "ride-sharing/shared/proto/trip"

type OsrmApiResponse struct {
	Waypoints []struct {
		Location []float64 `json:"location"`
	} `json:"waypoints"`
	Routes []struct {
		Distance float64 `json:"distance"`
		Duration float64 `json:"duration"`
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
}

func (o *OsrmApiResponse) ToProto() *pb.Route {
	route := o.Routes[0]
	geometry := route.Geometry.Coordinates

	// If we have waypoints, try to split the geometry into legs (one per segment between waypoints)
	var geometries []*pb.Geometry

	if len(o.Waypoints) >= 2 {
		currentIdx := 0
		for i := 0; i < len(o.Waypoints)-1; i++ {
			// A segment from waypoint i to waypoint i+1
			endWaypoint := o.Waypoints[i+1].Location

			// Find closest indices in geometry to start and end
			startIdx := currentIdx
			endIdx := currentIdx

			minDistEnd := 1e9
			for j := currentIdx; j < len(geometry); j++ {
				d := (geometry[j][0]-endWaypoint[0])*(geometry[j][0]-endWaypoint[0]) +
					(geometry[j][1]-endWaypoint[1])*(geometry[j][1]-endWaypoint[1])
				if d < minDistEnd {
					minDistEnd = d
					endIdx = j
				}
			}

			// extract segment
			var segment []*pb.Coordinate
			for j := startIdx; j <= endIdx && j < len(geometry); j++ {
				segment = append(segment, &pb.Coordinate{
					Latitude:  geometry[j][1], // Note: OSRM is [lon, lat], but the previous code mapped Latitude to [0] which is wrong! Wait!
					Longitude: geometry[j][0],
				})
			}
			geometries = append(geometries, &pb.Geometry{
				Coordinates: segment,
			})
			currentIdx = endIdx
		}
	} else {
		// fallback to single geometry
		coordinates := make([]*pb.Coordinate, len(geometry))
		for i, coord := range geometry {
			coordinates[i] = &pb.Coordinate{
				Latitude:  coord[1],
				Longitude: coord[0],
			}
		}
		geometries = append(geometries, &pb.Geometry{
			Coordinates: coordinates,
		})
	}

	return &pb.Route{
		Geometry: geometries,
		Distance: route.Distance,
		Duration: route.Duration,
	}
}

type PricingConfig struct {
	PricePerUnitOfDistance float64
	PricingPerMinute       float64
}

func DefaultPricingConfig() *PricingConfig {
	return &PricingConfig{
		PricePerUnitOfDistance: 1.5,
		PricingPerMinute:       0.25,
	}
}
