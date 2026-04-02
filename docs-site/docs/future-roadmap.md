---
sidebar_position: 1
title: Future Roadmap / Capstone
---

# Future Roadmap & Capstone Ideas

The RideSync is currently a highly functional, event-driven microservices architecture. However, there are numerous opportunities to expand its feature set and solve more complex technical challenges.

The following represent **Capstone Ideas** that developers can implement to stretch their knowledge of Go, Kubernetes, and System Design.

## 1. Recommendation Engine (Caching / Redis)

**Goal**: Suggest the fastest or cheapest ride option proactively based on a user's location, rather than making them wait for OSRM calculations on-demand.

**Implementation Concept**:
- Setup a new `Recommendation Service`.
- Connect it to a `Redis` cluster.
- When the `Driver Service` receives background GPS updates from active drivers, it publishes a `DriverLocationUpdate` event to RabbitMQ.
- The `Recommendation Service` consumes these events and updates geospatial indexes in Redis.
- The `API Gateway` can synchronously query this service via gRPC to quickly show riders "Drivers near you" as soon as they open the app.

## 2. Rider-Driver Matching Algorithm enhancements

**Goal**: Implement smarter dispatching rather than a simple random selection of an available driver.

**Implementation Concept**:
- Update the `Driver Service` dispatch logic (`find_available_drivers` queue consumer).
- Instead of picking the first active driver handling the required `PackageSlug`, calculate the ETA of the closest 5 drivers using OSRM.
- Implement an acceptance timeout cascade: Offer it to Driver A. If Driver A does not accept within 10 seconds, send a `driver_cmd_trip_request` to Driver B. 

## 3. Real-Time Driver Map Tracking

**Goal**: Add visual flair to the Rider frontend by moving a car icon on the map continuously.

**Implementation Concept**:
- Use Server-Side **gRPC Streaming** between the gateway and the Driver service.
- Implement a high-performance publish/subscribe mechanism in the API Gateway's WebSocket manager, so the frontend receives coordinate pushes 2-3 times per second.
- Implement smooth interpolation on the frontend map canvas.

## 4. Notification Service Extraction

**Goal**: Separate the responsibility of sending Alerts/WebSockets from the core `API Gateway`.

**Implementation Concept**:
- The API Gateway is currently monolithic in its routing: it handles incoming HTTP, but also manages outgoing WebSockets.
- Spin up a new `Notification Service`.
- Move the Gorilla WebSocket Hub into this new service.
- All services (Trip, Payment) publish "Send User Alert" events to RabbitMQ. ONLY the Notification Service listens to this queue, and handles pushing the data strictly to the client layer. 
