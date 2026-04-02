---
sidebar_position: 4
title: Driver Dispatch Case Study
---

# Case Study: Driver Dispatch & 120s TTL Optimization

**Role:** Principal Platform Engineer & Technical Writer  
**Feature:** Driver Dispatch & 120s TTL (Time-To-Live)  
**System context:** Go (Golang), Kubernetes, RabbitMQ, MongoDB

---

## Executive Summary (Non-Tech)
**Success:** A rider requests a trip, and the closest available driver sees the request and accepts it within seconds, confirming the ride.  
**Failure (Timeout):** If no driver is found within 2 minutes, the system automatically stops searching and notifies the rider so they can try again or change their fare.  

---

## Scenario 1: The Success Path (Match < 10s)
*A rider requests a trip, and a driver accepts within the golden window.*

### Technical Workflow
1. **Producer:** `Trip Service` publishes `trip.requested` to the `find_available_drivers` queue in RabbitMQ.
2. **Topic/Queue:** `find_available_drivers` stores the message with an `x-message-ttl` of 120,000ms.
3. **Consumer:** `Driver Service` consumes the message and pushes it to eligible drivers via WebSockets.
4. **State Change:** When a driver accepts, `Trip Service` updates the MongoDB record from `status: "pending"` to `status: "accepted"`.
5. **Acknowledge:** The `Driver Service` acknowledges the RabbitMQ message, removing it from the queue.

### The "10x" Observer
- **OTel Spans:** `trip_request_latency`, `driver_acceptance_lag`.
- **Prometheus Metric:** `match_latency_seconds` (Histogram).
- **SQL/Log Query:**
  ```sql
  -- Find the duration between request and acceptance
  SELECT trip_id, (updated_at - created_at) as match_time 
  FROM trips 
  WHERE status = 'accepted' AND trip_id = 'TRIP_ID_123';
  ```

---

## Scenario 2: The Failure Path (Timeout @ 120s)
*No driver is found within the 120-second timeout.*

### Technical Workflow
1. **RabbitMQ TTL:** The message sits in `find_available_drivers` for 120s with no `ACK`.
2. **Dead Letter:** RabbitMQ automatically moves the message to the `dlx` exchange, then to `dead_letter_queue`.
3. **DLQ Consumer:** `API Gateway` consumes the expired message from the DLQ.
4. **State Change:** No change in `trips` collection (remains `pending` unless explicitly canceled later).
5. **Notification:** API Gateway identifies the `OwnerID` and sends a `TripEventNoDriversFound` WebSocket to the rider.

### The "10x" Observer
- **OTel Spans:** `queue_ttl_expiry`, `dlq_redirection`.
- **Prometheus Metric:** `driver_search_timeout_total` (Counter).
- **SQL/Log Query:**
  ```bash
  # Check logs for expired messages in the DLQ
  kubectl logs api-gateway | grep "dlq consumer: expired message for user_id: USER_123"
  ```

---

## Scenario 3: The Edge Case (Race Condition - Double Accept)
*Two drivers accept the same ride at the exact same millisecond.*

### Can it happen?
**No.** We have implemented a **Compare-and-Swap (CAS)** strategy at the database level.

### Technical Prevention
1. **Atomic Update:** The `UpdateTrip` repository logic uses a combined filter: `_id: TRIP_ID` AND `status: "pending"`.
2. **First Actor Wins:** The first driver's request updates the status to `accepted`.
3. **Second Actor Fails:** The second request tries to update where `status: "pending"`, but since the status is now `accepted`, the `ModifiedCount` returns `0`.
4. **Graceful Rejection:** The service detects `ModifiedCount == 0` and returns an error to the second driver: *"Trip already assigned."*

### The "10x" Observer
- **OTel Spans:** `atomic_update_check`, `concurrency_conflict`.
- **Prometheus Metric:** `conflicting_acceptance_attempts_total` (Counter).
- **SQL/Log Query:**
  ```mongodb
  // Verify that only one driver is assigned to the trip
  db.trips.find({ "_id": ObjectId("TRIP_ID_123") }).pretty()
  ```

---

**Future Backend Upgrades Required:**
To tackle overlapping efficiently at scale, the backend must implement **partial path overlapping**. This could involve:
- Passing the active Tripline (Polyline) to PostGIS or Redis spatial operations.
- Intersecting Trip A's remaining untouched geometry with Trip B's start/end coordinates.
- Only dispatching the AMQP `DriverCmdTripRequest` if the backend confidently verifies a partial overlap, keeping irrelevant requests off the driver's WebSocket connection entirely.


## Scenario 4: Carpool Overlap Dispatching
*A second rider requests a carpool while a driver is already on an active carpool trip.*

### Technical Workflow
1. **Producer:** `Trip Service` publishes `trip.requested` for Trip B (carpool).
2. **Consumer:** `Driver Service` consumes the request.
3. **Capacity Check:** Filter out drivers who have `AvailableSeats < requested_seats`.
4. **Geospatial Check:** For each eligible carpool driver, the service fetches their active trips via the Trip HTTP API. A **bounding box heuristic** (with a ±0.005 degree / ~0.5 km tolerance) is calculated natively in Go over the driver's active route geometry.
5. **Dispatch:** Only drivers whose active routes successfully overlap with Trip B's requested route receive the AMQP WebSocket `DriverCmdTripRequest`. Non-overlapping requests are silently discarded from the broadcast.

### The "10x" Observer
- **OTel Spans:** `fetch_active_trips`, `calculate_boundingBox_overlap`.
- **Prometheus Metric:** `carpool_overlap_filter_droprate` (Counter).
- **SQL/Log Query:**
  ```bash
  # Check logs for driver suitability evaluation
  kubectl logs driver-service | grep "Found suitable drivers: current="
  ```

---

## Infrastructure Impact
- **HPA Scaling:** `Driver Service` scales based on the `rabbitmq_queue_depth` of `find_available_drivers`. If the queue grows, K8s spins up more pods to handle the load.
- **Database Load:** Atomic updates ensure we don't need complex distributed locks (Redis Redlock), keeping the MongoDB write-load predictable and lightweight.

> [!IMPORTANT]
> The Distributed CAS pattern is the backbone of our 99.9% consistency in the dispatch loop, while the native Go geospatial clipping preserves driver WebSocket bandwidth during high-concurrency carpool searches.
