---
sidebar_position: 1
title: Geohash Spatial Indexing
---

# Geohash Spatial Indexing

To efficiently match dispatch requests with nearby drivers, the Hybrid Logistics Engine heavily relies on spatial indexing via Geohashes before falling back to exact coordinate bounding-box math.

## The Problem

Searching for drivers using raw `Latitude` and `Longitude` float coordinates across thousands of active vehicles requires intense database querying (e.g., "$near" MongoDB operators) which often bottlenecks high-throughput ride-sharing platforms.

## The Geohash Solution

A Geohash combines longitude and latitude into a single alphanumeric string. Locations that are near each other share the same prefix.

In the Driver Service, when a driver registers (`services/driver-service/service.go`), their exact coordinate is immediately encoded using the `github.com/mmcloughlin/geohash` package into a standard hash:

```go
import "github.com/mmcloughlin/geohash"

// ... inside RegisterDriver

	// randomRoute is selected from predefined simulation data
	lat := randomRoute[0][0]
	lon := randomRoute[0][1]

	geohashStr := geohash.Encode(lat, lon)

	driver := &pb.Driver{
		Id:             driverId,
		Geohash:        geohashStr,
		Location:       &pb.Location{Latitude: lat, Longitude: lon},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
	}
```

### Prefix Matching

While the current dispatch algorithm iteratively matches by `PackageSlug`, the encoded `Geohash` is maintained on the `Driver` object specifically to support highly scalable $prefix checks over Redis or MongoDB in production iterations. 

For example, asking for drivers near Geohash `9q8yy` instantly pulls all records starting with `9q8yy`, effectively returning nearby cars in $O(1)$ string-matching time compared to complex spherical geometry calculations.
