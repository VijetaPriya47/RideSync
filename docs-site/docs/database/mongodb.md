---
sidebar_position: 1
title: Database Design (MongoDB)
---

# Database Design (MongoDB)

The RideSync leverages a microservices architecture where the **Trip Service** actively owns the MongoDB database. We use a schema-on-read NoSQL approach, optimized for the high read volume inherent in trip tracking.

## Collections Overview

The database contains two primary collections:
1. `trips`
2. `ride_fares`

### 1. `trips` Collection

Stores complete trip and ride information, including the user, driver details, and embedded fare mapping.

```json
{
  "_id": "ObjectId(\"507f1f77bcf86cd799439011\")",
  "userID": "user_123",
  "status": "driver_assigned",
  "rideFare": {
    "_id": "ObjectId(\"507f1f77bcf86cd799439012\")",
    "userID": "user_123",
    "packageSlug": "sedan",
    "totalPriceInCents": 2850.50,
    "route": {
      "routes": [
        {
          "distance": 5432.8,
          "duration": 720.5,
          "geometry": {
             "coordinates": [[-122.4194, 37.7749], [-122.4195, 37.7750]]
          }
        }
      ]
    }
  },
  "driver": {
    "id": "driver_456",
    "name": "Anya",
    "profilePicture": "https://randomuser.me/api/portraits/lego/1.jpg",
    "carPlate": "ABC-1234"
  }
}
```

The database avoids Joins by embedding driver and rideFare within the document.

### 2. `ride_fares` Collection

Pre-calculated fare estimates for the different vehicle types, calculated through the OSRM spatial API prior to the trip confirmation.

```json
{
  "_id": "ObjectId(\"507f1f77bcf86cd799439013\")",
  "userID": "user_123",
  "packageSlug": "luxury",
  "totalPriceInCents": 3825.75,
  "route": { ... } // Detailed coordinate map
}
```

## Recommended Indexes

We rely on standard B-Tree indexing. Future enhancements will involve geospatial indexes for analytics on historical trips, but the live spatial queries routing occurs in memory over the Driver Service.

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

## Connection Configuration

The Trip service connects to MongoDB via the golang `mongo-driver` configured via `.env` files using connection pooling:

```go
client, err := mongo.Connect(ctx, options.Client().
    ApplyURI(mongoURI).
    SetMaxPoolSize(100).
    SetMinPoolSize(10).
    SetMaxConnIdleTime(30 * time.Second).
    SetRetryWrites(true).
    SetRetryReads(true))
```

## MongoDB Documentation

- [MongoDB TTL Indexes for Auto-Expiration](https://www.mongodb.com/docs/manual/core/index-ttl/)
