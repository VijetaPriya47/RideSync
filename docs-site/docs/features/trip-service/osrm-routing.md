---
sidebar_position: 1
title: OSRM Routing Engine
---

# OSRM Routing Engine

The Hybrid Logistics Engine calculates dynamic ETAs and trip distances using the Open Source Routing Machine (OSRM) HTTP API. This ensures driver pricing and rider previews are based on real-world road networks rather than simple as-the-crow-flies estimates.

## Fetching the Route

Inside the `trip-service/internal/service/service.go`, the `GetRoute` method builds the coordinate payload and makes the HTTP request to the OSRM project:

```go
	baseURL := env.GetString("OSRM_API", "http://router.project-osrm.org")

	url := fmt.Sprintf(
		"%s/route/v1/driving/%f,%f;%f,%f?overview=full&geometries=geojson",
		baseURL,
		pickup.Longitude, pickup.Latitude,
		destination.Longitude, destination.Latitude,
	)

	log.Printf("Started Fetching from OSRM API: URL: %s", url)

	resp, err := http.Get(url)
```

By passing `overview=full&geometries=geojson`, OSRM returns the exact coordinates needed to draw the polyline route on the frontend React-Leaflet map.

## Parsing the Response

The response is unmarshaled into a native Go struct `tripTypes.OsrmApiResponse`:

```go
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read the response: %v", err)
	}

	var routeResp tripTypes.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &routeResp, nil
```

The underlying struct definition parses out the key properties required for the pricing engine:

```go
type OsrmApiResponse struct {
	Routes []struct {
		Distance float64 `json:"distance"` // distance in meters
		Duration float64 `json:"duration"` // duration in seconds
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	}
}
```

> [!TIP]
> **Graceful Degradation:** The service allows failing over to a simulated mock response if `useOSRMApi` is false, preventing the entire dispatch system from going down if the external routing vendor experiences an outage.
