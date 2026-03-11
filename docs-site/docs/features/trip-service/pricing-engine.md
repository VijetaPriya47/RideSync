---
sidebar_position: 2
title: Pricing Engine
---

# Pricing Engine

The Trip Service automatically calculates upfront fares for riders before they officially hail a vehicle. The fares are tiered across four distinct vehicle classes.

## Base Packages

The system supports four distinct vehicle packages, each associated with a unique base price (stored in cents to prevent floating-point calculation errors):

```go
func getBaseFares() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{PackageSlug: "suv", TotalPriceInCents: 200},
		{PackageSlug: "sedan", TotalPriceInCents: 350},
		{PackageSlug: "van", TotalPriceInCents: 400},
		{PackageSlug: "luxury", TotalPriceInCents: 1000},
	}
}
```

## Dynamic Fare Calculation

When a user requests a preview between a pickup and destination point, the `trip-service` first fetches the OSRM route to determine the exact `distance` and estimated `duration`. It then loops through all base packages to generate specific pricing options:

```go
func (s *service) EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*domain.RideFareModel {
	baseFares := getBaseFares()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, f := range baseFares {
		estimatedFares[i] = estimateFareRoute(f, route)
	}

	return estimatedFares
}
```

The actual mathematical calculation blends the base package price, the distance cost, and the time cost:

```go
func estimateFareRoute(f *domain.RideFareModel, route *tripTypes.OsrmApiResponse) *domain.RideFareModel {
	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := f.TotalPriceInCents

	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	distanceFare := distanceKm * pricingCfg.PricePerUnitOfDistance
	timeFare := durationInMinutes * pricingCfg.PricingPerMinute
	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		TotalPriceInCents: totalPrice,
		PackageSlug:       f.PackageSlug,
	}
}
```

These fare estimations are then firmly locked into the MongoDB `ride_fares` collection so that users cannot spoof API values to get a cheaper ride.
