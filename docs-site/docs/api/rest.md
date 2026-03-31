---
sidebar_position: 3
title: API Design (REST & WebSockets)
---

# API Design (REST & WebSockets)

The API Gateway exposes REST endpoints for client applications (Mobile/Web) and manages real-time communication via WebSockets.

## REST Endpoints

All REST endpoints return a standard JSON response structure.

### Standard Response Structure
```json
{
  "data": { ... },
  "error": {
    "code": "400",
    "message": "Error description"
  }
}
```

### 1. Trip Preview
**Endpoint:** `POST /trip/preview`

Calculates potential routes and fares based on pickup and destination coordinates.

**Request Body:**
```json
{
  "userID": "user_123",
  "pickup": {
    "latitude": 37.7749,
    "longitude": -122.4194
  },
  "destination": {
    "latitude": 34.0522,
    "longitude": -118.2437
  }
}
```

**Success Response (201 Created):**
```json
{
  "data": {
    "tripID": "preview_xyz",
    "route": {
      "geometry": [ ... ],
      "distance": 500.5,
      "duration": 3600
    },
    "rideFares": [
      {
        "id": "fare_abc",
        "userID": "user_123",
        "packageSlug": "standard",
        "totalPriceInCents": 2500
      }
    ]
  }
}
```

### 2. Start Trip
**Endpoint:** `POST /trip/start`

Initializes a trip using a selected fare ID.

**Request Body:**
```json
{
  "rideFareID": "fare_abc",
  "userID": "user_123"
}
```

**Success Response (201 Created):**
```json
{
  "data": {
    "tripID": "trip_789",
    "trip": {
      "id": "trip_789",
      "status": "searching",
      "userID": "user_123",
      "selectedFare": { ... },
      "route": { ... }
    }
  }
}
```

### 3. Stripe Webhook
**Endpoint:** `POST /webhook/stripe`

Handles payment success events from Stripe.

**Headers:**
- `Stripe-Signature`: Required for verification.

**Body:** Raw Stripe event.

---

## WebSocket Endpoints

Real-time communication for drivers and riders.

### 1. Drivers WebSocket
**Endpoint:** `WS /ws/drivers`

Used by driver applications to receive ride requests and send status updates.

**Message Format:**
```json
{
  "type": "event_type",
  "data": { ... }
}
```

### 2. Riders WebSocket
**Endpoint:** `WS /ws/riders`

Used by rider applications to receive real-time updates on trip status and driver location.

**Message Format:**
```json
{
  "type": "event_type",
  "data": { ... }
}
```
