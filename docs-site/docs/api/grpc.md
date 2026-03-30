---
sidebar_position: 1
title: API Design (gRPC)
---

# API Design (gRPC)

Communication between the microservices natively leverages gRPC via Protocol Buffers. This ensures robust type safety and lower latency compared to HTTP JSON mapping.

## Trip Service API

The `TripService` manages the core logic of trip creation and routing preview.

### Service Definition

```protobuf
syntax = "proto3";

package trip;

service TripService {
  rpc PreviewTrip(PreviewTripRequest) returns (PreviewTripResponse);
  rpc CreateTrip(CreateTripRequest) returns (CreateTripResponse);
}
```

### Methods and Messages

#### PreviewTrip
Calculates spatial distance and route via OSRM, returning a list of available ride fares.

**Request:** `PreviewTripRequest`
```protobuf
message PreviewTripRequest {
  string userID = 1;
  Coordinate startLocation = 2;
  Coordinate endLocation = 3;
}

message Coordinate {
  double latitude = 1;
  double longitude = 2;
}
```

**Response:** `PreviewTripResponse`
```protobuf
message PreviewTripResponse {
  string tripID = 1;
  Route route = 2;
  repeated RideFare rideFares = 3;
}

message Route {
  repeated Geometry geometry = 1;
  double distance = 2;
  double duration = 3;
}

message Geometry {
  repeated Coordinate coordinates = 1;
}

message RideFare {
  string id = 1;
  string userID = 2;
  string packageSlug = 3;
  double totalPriceInCents = 4;
}
```

#### CreateTrip
Initializes a trip based on a selected fare.

**Request:** `CreateTripRequest`
```protobuf
message CreateTripRequest {
  string rideFareID = 1;
  string userID = 2;
}
```

**Response:** `CreateTripResponse`
```protobuf
message CreateTripResponse {
  string tripID = 1;
  Trip trip = 2;
}

message Trip {
  string id = 1;
  RideFare selectedFare = 2;
  Route route = 3;
  string status = 4;
  string userID = 5;
  TripDriver driver = 6;
}

message TripDriver {
  string id = 1;
  string name = 2;
  string profilePicture = 3;
  string carPlate = 4;
}
```

---

## Driver Service API

The `DriverService` manages driver registration and availability.

### Service Definition

```protobuf
syntax = "proto3";

package driver;

service DriverService {
  rpc RegisterDriver(RegisterDriverRequest) returns (RegisterDriverResponse);
  rpc UnregisterDriver(RegisterDriverRequest) returns (RegisterDriverResponse);
}
```

### Methods and Messages

#### RegisterDriver
Registers a driver for a specific category (package slug).

**Request:** `RegisterDriverRequest`
```protobuf
message RegisterDriverRequest {
  string driverID = 1;
  string packageSlug = 2;
}
```

**Response:** `RegisterDriverResponse`
```protobuf
message RegisterDriverResponse {
  Driver driver = 1;
}

message Driver {
  string id = 1;
  string name = 2;
  string profilePicture = 3;
  string carPlate = 4;
  string geohash = 5;
  string packageSlug = 6;
  Location location = 7;
}

message Location {
  double latitude = 1;
  double longitude = 2;
}
```

#### UnregisterDriver
Unregisters a driver. Uses the same request/response structure as `RegisterDriver`.
