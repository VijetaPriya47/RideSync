---
sidebar_position: 1
title: API Design (gRPC)
---

# API Design (gRPC)

Communication between the microservices natively leverages gRPC via Protocol Buffers. This ensures robust type safety and lower latency compared to HTTP JSON mapping.

## Trip Service API

The `TripService` manages the core logic of trip creation and routing preview.

```protobuf
syntax = "proto3";

package trip;
option go_package = "shared/proto/trip;trip";

service TripService {
  rpc PreviewTrip(PreviewTripRequest) returns (PreviewTripResponse);
  rpc CreateTrip(CreateTripRequest) returns (CreateTripResponse);
}

message PreviewTripRequest {
  string userID = 1;
  Coordinate startLocation = 2;
  Coordinate endLocation = 3;
}

message CreateTripRequest {
  string rideFareID = 1;
  string userID = 2;
}

message Trip {
  string id = 1;
  RideFare selectedFare = 2;
  Route route = 3;
  string status = 4;
  string userID = 5;
  TripDriver driver = 6;
}
```

The API dictates a strict two-step flow. The client sends a `PreviewTripRequest` to calculate spatial distance and route via OSRM, returning a `RideFare` ID. The client then selects that fare through `CreateTripRequest`.

## Driver Service API

The `DriverService` is exposed internally so the API Gateway can register drivers as they connect on WebSockets. Action-based event streams (e.g. accepting/declining rides) are handled over RabbitMQ instead of unary gRPC methods.

```protobuf
syntax = "proto3";

package driver;
option go_package = "shared/proto/driver;driver";

service DriverService {
  rpc RegisterDriver(RegisterDriverRequest) returns (RegisterDriverResponse);
  rpc UnregisterDriver(RegisterDriverRequest) returns (RegisterDriverResponse);
}

message RegisterDriverRequest {
  string driverID = 1;
  string packageSlug = 2;
}
```
