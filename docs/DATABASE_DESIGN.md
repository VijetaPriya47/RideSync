# Database Design - Ride Sharing Platform

## Overview

This document provides a comprehensive overview of the database schema used in the ride-sharing platform. The system uses **MongoDB** as its primary datastore for trip and fare management, following a microservices architecture pattern.

---

## Database System

- **Type**: MongoDB (NoSQL Document Database)
- **Version**: MongoDB 5.0+
- **Database Name**: `ride-sharing`
- **Connection**: Configured via `MONGODB_URI` environment variable
- **Architecture**: Owned and managed by the Trip Service microservice

---

## Collections Overview

The database contains **2 main collections**:

| Collection | Purpose | Owner Service |
|------------|---------|---------------|
| `trips` | Stores ride/trip information with user, driver, status, and fare details | Trip Service |
| `ride_fares` | Stores pre-calculated fare estimates for different vehicle types | Trip Service |

---

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          RIDE-SHARING DATABASE                          │
│                         (MongoDB: ride-sharing)                         │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                            TRIPS COLLECTION                              │
├──────────────────────────────────────────────────────────────────────────┤
│  _id              : ObjectId [PK]                                        │
│  userID           : String                                               │
│  status           : String                                               │
│                     ("pending" | "driver_assigned" |                     │
│                      "in_progress" | "completed")                        │
│                                                                          │
│  ┌──────────────────────────────────────────────────────┐               │
│  │ rideFare (Embedded)                                  │               │
│  ├──────────────────────────────────────────────────────┤               │
│  │  _id                : ObjectId                       │               │
│  │  userID             : String                         │               │
│  │  packageSlug        : String                         │               │
│  │  totalPriceInCents  : Float64                        │               │
│  │  ┌──────────────────────────────────────────────┐   │               │
│  │  │ route (OSRM API Response)                    │   │               │
│  │  ├──────────────────────────────────────────────┤   │               │
│  │  │  routes: [                                   │   │               │
│  │  │    {                                         │   │               │
│  │  │      distance: Float64                       │   │               │
│  │  │      duration: Float64                       │   │               │
│  │  │      geometry: {                             │   │               │
│  │  │        coordinates: [[Float64, Float64]]     │   │               │
│  │  │      }                                        │   │               │
│  │  │    }                                         │   │               │
│  │  │  ]                                           │   │               │
│  │  └──────────────────────────────────────────────┘   │               │
│  └──────────────────────────────────────────────────────┘               │
│                                                                          │
│  ┌──────────────────────────────────────────────────────┐               │
│  │ driver (Embedded - nullable)                         │               │
│  ├──────────────────────────────────────────────────────┤               │
│  │  id              : String                            │               │
│  │  name            : String                            │               │
│  │  profilePicture  : String (URL)                      │               │
│  │  carPlate        : String                            │               │
│  └──────────────────────────────────────────────────────┘               │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ references (by _id)
                                    ↓
┌──────────────────────────────────────────────────────────────────────────┐
│                         RIDE_FARES COLLECTION                            │
├──────────────────────────────────────────────────────────────────────────┤
│  _id                : ObjectId [PK]                                      │
│  userID             : String                                             │
│  packageSlug        : String                                             │
│                       ("sedan" | "luxury" | "van" | "economy")           │
│  totalPriceInCents  : Float64                                            │
│                                                                          │
│  ┌──────────────────────────────────────────────────────┐               │
│  │ route (OSRM API Response)                            │               │
│  ├──────────────────────────────────────────────────────┤               │
│  │  routes: [                                           │               │
│  │    {                                                 │               │
│  │      distance: Float64 (meters)                      │               │
│  │      duration: Float64 (seconds)                     │               │
│  │      geometry: {                                     │               │
│  │        coordinates: [[Float64, Float64]]             │               │
│  │      }                                                │               │
│  │    }                                                 │               │
│  │  ]                                                   │               │
│  └──────────────────────────────────────────────────────┘               │
└──────────────────────────────────────────────────────────────────────────┘


┌─────────────────────────────────────────────────────────────────────────┐
│                    IN-MEMORY DATA STRUCTURES                            │
│                        (NOT PERSISTED)                                  │
└─────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                    DRIVER (In-Memory - Driver Service)                   │
├──────────────────────────────────────────────────────────────────────────┤
│  id              : String                                                │
│  name            : String                                                │
│  profilePicture  : String (URL)                                          │
│  carPlate        : String                                                │
│  geohash         : String (for spatial indexing)                         │
│  packageSlug     : String                                                │
│  location        : { latitude: Float64, longitude: Float64 }             │
└──────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│              PAYMENT INTENT (In-Memory - Payment Service)                │
├──────────────────────────────────────────────────────────────────────────┤
│  ID              : String                                                │
│  TripID          : String                                                │
│  UserID          : String                                                │
│  DriverID        : String                                                │
│  Amount          : Int64 (cents)                                         │
│  Currency        : String ("usd")                                        │
│  StripeSessionID : String                                                │
│  CreatedAt       : Timestamp                                             │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Detailed Schema Documentation

### Collection 1: `trips`

**Purpose**: Stores complete trip/ride information including user details, assigned driver, trip status, and embedded fare/route data.

#### Schema Structure

```json
{
  "_id": ObjectId("507f1f77bcf86cd799439011"),
  "userID": "user_123",
  "status": "driver_assigned",
  "rideFare": {
    "_id": ObjectId("507f1f77bcf86cd799439012"),
    "userID": "user_123",
    "packageSlug": "sedan",
    "totalPriceInCents": 2850.50,
    "route": {
      "routes": [
        {
          "distance": 5432.8,
          "duration": 720.5,
          "geometry": {
            "coordinates": [
              [-122.4194, 37.7749],
              [-122.4195, 37.7750],
              [-122.4196, 37.7751]
            ]
          }
        }
      ]
    }
  },
  "driver": {
    "id": "driver_456",
    "name": "Lando Norris",
    "profilePicture": "https://randomuser.me/api/portraits/lego/1.jpg",
    "carPlate": "ABC-1234"
  }
}
```

#### Field Specifications

| Field | Type | Required | Indexed | Description |
|-------|------|----------|---------|-------------|
| `_id` | ObjectId | ✓ | ✓ (default) | MongoDB unique identifier |
| `userID` | String | ✓ | ✗ | ID of the user/rider who requested the trip |
| `status` | String | ✓ | ✗ | Current trip status (see status values below) |
| `rideFare` | Object | ✓ | ✗ | Embedded fare calculation with complete route |
| `rideFare._id` | ObjectId | ✓ | ✗ | Fare record identifier |
| `rideFare.userID` | String | ✓ | ✗ | User ID (matches trip.userID) |
| `rideFare.packageSlug` | String | ✓ | ✗ | Vehicle category selected |
| `rideFare.totalPriceInCents` | Float64 | ✓ | ✗ | Total fare amount in cents |
| `rideFare.route` | Object | ✓ | ✗ | Complete OSRM API response |
| `driver` | Object | ✗ | ✗ | Driver details (null until assigned) |
| `driver.id` | String | ✗ | ✗ | Driver unique ID |
| `driver.name` | String | ✗ | ✗ | Driver full name |
| `driver.profilePicture` | String | ✗ | ✗ | URL to driver's avatar |
| `driver.carPlate` | String | ✗ | ✗ | Vehicle license plate |

#### Trip Status Values

| Status | Description |
|--------|-------------|
| `pending` | Trip created, awaiting driver acceptance |
| `driver_assigned` | Driver accepted the trip request |
| `in_progress` | Trip is currently ongoing |
| `completed` | Trip finished successfully |

#### Operations

- **Create Trip**: Insert new trip document with embedded rideFare
- **Get Trip by ID**: Find trip by `_id`
- **Update Trip**: Update `status` and/or `driver` fields
- **Query by User**: Find all trips for a specific `userID`

---

### Collection 2: `ride_fares`

**Purpose**: Stores pre-calculated fare estimates for different vehicle types. Generated during trip preview phase before actual trip creation.

#### Schema Structure

```json
{
  "_id": ObjectId("507f1f77bcf86cd799439013"),
  "userID": "user_123",
  "packageSlug": "luxury",
  "totalPriceInCents": 3825.75,
  "route": {
    "routes": [
      {
        "distance": 5432.8,
        "duration": 720.5,
        "geometry": {
          "coordinates": [
            [-122.4194, 37.7749],
            [-122.4195, 37.7750],
            [-122.4196, 37.7751]
          ]
        }
      }
    ]
  }
}
```

#### Field Specifications

| Field | Type | Required | Indexed | Description |
|-------|------|----------|---------|-------------|
| `_id` | ObjectId | ✓ | ✓ (default) | MongoDB unique identifier |
| `userID` | String | ✓ | ✗ | User who requested the fare estimate |
| `packageSlug` | String | ✓ | ✗ | Vehicle category for pricing |
| `totalPriceInCents` | Float64 | ✓ | ✗ | Calculated fare in cents |
| `route` | Object | ✓ | ✗ | Complete OSRM route response |
| `route.routes[0].distance` | Float64 | ✓ | ✗ | Total distance in meters |
| `route.routes[0].duration` | Float64 | ✓ | ✗ | Estimated duration in seconds |
| `route.routes[0].geometry.coordinates` | Array | ✓ | ✗ | Route polyline as coordinate pairs |

#### Package Types (Vehicle Categories)

| Package Slug | Description | Multiplier |
|--------------|-------------|------------|
| `economy` | Budget-friendly option | 1.0x |
| `sedan` | Standard sedan vehicle | 1.2x |
| `luxury` | Premium/luxury vehicle | 1.8x |
| `van` | Large van/SUV | 1.5x |

#### Pricing Formula

```
totalPriceInCents = (
  (distance * pricePerUnitDistance) + 
  (duration * pricePerMinute)
) * packageMultiplier * 100
```

**Default Pricing Configuration**:
- Base price per unit distance: $1.50
- Price per minute: $0.25
- All prices stored in cents for precision

#### Operations

- **Save RideFare**: Insert fare calculation for a vehicle type
- **Get RideFare by ID**: Find fare by `_id` for trip creation
- **Validate Fare**: Ensure fare belongs to requesting user

---

## Data Flow & Lifecycle

### 1. Trip Preview Flow

```
User Request → Trip Service
                    ↓
            OSRM API (Route Calculation)
                    ↓
        Calculate Fares (All Vehicle Types)
                    ↓
        INSERT → ride_fares Collection (Multiple Documents)
                    ↓
        Return Fare Options to User
```

**Example**: User requests route from Point A to Point B. System creates 4 fare documents (economy, sedan, luxury, van) in `ride_fares` collection.

### 2. Trip Creation Flow

```
User Selects Fare → Validate Fare (ride_fares)
                            ↓
                    CREATE Trip Document
                            ↓
            INSERT → trips Collection (status: "pending")
                            ↓
                Publish Event → RabbitMQ
                            ↓
                    Driver Service Notified
```

### 3. Trip Assignment Flow

```
Driver Accepts → Driver Service
                        ↓
                RabbitMQ Event
                        ↓
                Trip Service
                        ↓
        UPDATE trips (status: "driver_assigned", driver: {...})
                        ↓
                WebSocket → User Notified
```

### 4. Trip Completion Flow

```
Trip Ends → UPDATE trips (status: "completed")
                    ↓
            Trigger Payment Flow
                    ↓
            Archive/Cleanup
```

---

## In-Memory Data Structures

### Driver Data (Driver Service)

**Storage**: In-memory (not persisted to MongoDB)  
**Reason**: High-frequency updates for real-time location tracking

```go
type Driver struct {
    ID              string
    Name            string
    ProfilePicture  string
    CarPlate        string
    Geohash         string    // For spatial indexing
    PackageSlug     string    // Vehicle type offered
    Location        struct {
        Latitude  float64
        Longitude float64
    }
}
```

**Spatial Indexing**: Uses geohash for efficient proximity searches and driver discovery.

### Payment Data (Payment Service)

**Storage**: Managed by Stripe + In-memory state  
**Reason**: PCI compliance and security

```go
type PaymentIntent struct {
    ID              string
    TripID          string
    UserID          string
    DriverID        string
    Amount          int64     // In cents
    Currency        string    // "usd"
    StripeSessionID string
    CreatedAt       time.Time
}
```

---

## Indexes & Performance

### Current Indexes

| Collection | Index | Type | Purpose |
|------------|-------|------|---------|
| `trips` | `_id` | Default | Primary key lookup |
| `ride_fares` | `_id` | Default | Primary key lookup |

### Recommended Indexes for Production

```javascript
// trips collection
db.trips.createIndex({ "userID": 1 })
db.trips.createIndex({ "status": 1 })
db.trips.createIndex({ "userID": 1, "status": 1 })
db.trips.createIndex({ "driver.id": 1 })

// ride_fares collection
db.ride_fares.createIndex({ "userID": 1 })
db.ride_fares.createIndex({ "userID": 1, "packageSlug": 1 })
```

### Performance Considerations

1. **Embedded Documents**: The schema uses embedded documents (rideFare, driver) rather than references, optimizing for read performance
2. **No Joins**: MongoDB's document model eliminates the need for expensive joins
3. **Read-Heavy**: Schema optimized for read operations common in trip tracking
4. **Write Patterns**: Sequential writes during trip lifecycle stages

---

## Design Decisions

### Why Embedded Documents?

**Chosen**: Embedded `rideFare` and `driver` within `trips`

**Rationale**:
- ✅ Data always accessed together (single query)
- ✅ Atomicity for updates
- ✅ Better read performance
- ✅ Matches access patterns

**Alternative**: Separate collections with references
- ❌ Would require multiple queries or $lookup
- ❌ No transactional guarantees across collections
- ❌ Increased latency

### Why Separate `ride_fares` Collection?

**Rationale**:
- Multiple fare options generated per route request
- Fares exist before trip creation
- User may not create trip (just browsing prices)
- Allows fare validation during trip creation

### Why No User/Driver Collections?

**Current State**: Users and drivers not persisted in MongoDB

**Rationale**:
- Authentication handled externally (JWT tokens)
- Driver locations change frequently (in-memory for performance)
- Future enhancement: PostgreSQL for user profiles

---

## Database Operations by Service

### Trip Service

**Collections**: `trips`, `ride_fares`

**Operations**:
```go
// Trip operations
CreateTrip(trip) → trips.InsertOne()
GetTripByID(id) → trips.FindOne({_id: id})
UpdateTrip(id, status, driver) → trips.UpdateOne()

// Fare operations
SaveRideFare(fare) → ride_fares.InsertOne()
GetRideFareByID(id) → ride_fares.FindOne({_id: id})
```

### Driver Service

**Collections**: None (in-memory only)

**In-Memory Operations**:
- Register/unregister drivers
- Update driver locations
- Find available drivers by package type
- Geohash-based proximity search

### Payment Service

**Collections**: None (uses Stripe)

**External Operations**:
- Create Stripe checkout session
- Handle webhook events
- Validate payment completion

---

## Backup & Recovery

### Backup Strategy

```bash
# Backup entire database
mongodump --uri="mongodb://localhost:27017/ride-sharing" --out=/backup/$(date +%Y%m%d)

# Backup specific collection
mongodump --uri="mongodb://localhost:27017/ride-sharing" --collection=trips --out=/backup/trips
```

### Restore Strategy

```bash
# Restore entire database
mongorestore --uri="mongodb://localhost:27017/ride-sharing" /backup/20260114

# Restore specific collection
mongorestore --uri="mongodb://localhost:27017/ride-sharing" --collection=trips /backup/trips
```

---

## Migration Strategy

### Current State

- **No migration files**: Schema-on-read approach
- **Version control**: Managed through application code
- **Schema evolution**: Handled via backward-compatible changes

### Adding New Fields

```go
// New field with default value
type TripModel struct {
    // ... existing fields
    Rating    *float64 `bson:"rating,omitempty"`  // New optional field
    CreatedAt time.Time `bson:"createdAt,omitempty"` // Auto-populate
}
```

### Breaking Changes

If schema changes break compatibility:
1. Add new field with default values
2. Background job to populate existing documents
3. Update application code to use new field
4. Remove old field after transition period

---

## Security Considerations

### Data Protection

1. **Connection Security**: TLS/SSL for MongoDB connections in production
2. **Authentication**: MongoDB user authentication enabled
3. **Authorization**: Role-based access control (RBAC)
4. **Encryption**: Encryption at rest for sensitive data

### PII Data

| Field | Type | Handling |
|-------|------|----------|
| `userID` | Identifier | Anonymized/hashed |
| `driver.name` | PII | Encrypted at rest |
| `driver.carPlate` | PII | Encrypted at rest |
| Location coordinates | Sensitive | Time-limited retention |

---

## Monitoring & Observability

### Key Metrics

1. **Collection Size**: Monitor document count and storage size
2. **Query Performance**: Track slow queries (>100ms)
3. **Index Usage**: Ensure indexes are being utilized
4. **Connection Pool**: Monitor active connections

### Monitoring Commands

```javascript
// Collection statistics
db.trips.stats()
db.ride_fares.stats()

// Query performance
db.trips.find({userID: "user_123"}).explain("executionStats")

// Index usage
db.trips.aggregate([{$indexStats: {}}])
```

---

## Future Enhancements

### Planned Improvements

1. **Add Indexes**: Implement recommended production indexes
2. **User Collection**: PostgreSQL for user profiles and authentication
3. **Driver Collection**: PostgreSQL for driver profiles and documents
4. **Payment Collection**: Track payment history in database
5. **Trip History**: Archival strategy for completed trips
6. **Geospatial Indexes**: MongoDB geospatial queries for location-based features
7. **Audit Trail**: Track all changes to trip documents
8. **Data Retention**: Automatic cleanup of old fare estimates

### Scalability Roadmap

1. **Sharding**: Shard trips collection by `userID` for horizontal scaling
2. **Read Replicas**: Add read replicas for analytics queries
3. **Caching Layer**: Redis for frequently accessed data
4. **Time-Series**: Specialized storage for location tracking history

---

## Connection Configuration

### Environment Variables

```bash
MONGODB_URI=mongodb://localhost:27017/ride-sharing
```

### Connection Options

```go
// Production connection with options
client, err := mongo.Connect(ctx, options.Client().
    ApplyURI(mongoURI).
    SetMaxPoolSize(100).
    SetMinPoolSize(10).
    SetMaxConnIdleTime(30 * time.Second).
    SetRetryWrites(true).
    SetRetryReads(true))
```

---

## Related Documentation

- [Local Deployment Guide](./LOCAL_DEPLOYMENT_GUIDE.md)
- [GCP Deployment Guide](./GCP_DEPLOYMENT_GUIDE.md)
- [Architecture Overview](./architecture/)
- [API Documentation](../services/api-gateway/)

---

## Appendix: Sample Queries

### Query Examples

```javascript
// Find all pending trips
db.trips.find({ status: "pending" })

// Find all trips for a user
db.trips.find({ userID: "user_123" })

// Find trip with driver info
db.trips.find({ "driver.id": "driver_456" })

// Find fares for user by package type
db.ride_fares.find({ 
    userID: "user_123", 
    packageSlug: "sedan" 
})

// Count trips by status
db.trips.aggregate([
    { $group: { 
        _id: "$status", 
        count: { $sum: 1 } 
    }}
])

// Average fare by package type
db.ride_fares.aggregate([
    { $group: {
        _id: "$packageSlug",
        avgPrice: { $avg: "$totalPriceInCents" }
    }}
])
```

---

**Last Updated**: January 14, 2026  
**Version**: 1.0  
**Maintained by**: Ride-Sharing Platform Team

