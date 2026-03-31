# Hybrid Logistics Engine

> Setup instructions and full implementation details are available in the [Technical Assessment](Technical_Assessment.md).

A microservices-based backend system simulating a ride-sharing platform. The system matches riders with drivers, computes routes and fares, processes payments, and manages trip lifecycle events.

## Project Demo

Video demo:
[https://drive.google.com/file/d/1jHZYCv_tlQGjw5cV0MShal-_ezUGujMV/view](https://drive.google.com/file/d/1jHZYCv_tlQGjw5cV0MShal-_ezUGujMV/view)

---
<img width="768" height="601" alt="image" src="https://github.com/user-attachments/assets/6b0c2716-8af0-4bac-bc5a-87ae4a293a23" />


# System Overview

The system processes a ride request as follows:

1. Client sends request through the **Next.js frontend**
2. **API Gateway** receives and routes the request
3. **Trip Service** calculates route and estimated fare
4. **Driver Service** finds nearby available drivers (considering both idle and carpooling drivers)
5. **RabbitMQ** coordinates asynchronous events between services
6. **Payment Service** processes payment after trip completion
7. **Carpool Engine** manages multi-rider trips and real-time seat availability


---
## Request Lifecycle

Example: Rider requests a trip

1. Client sends ride request to API Gateway.
2. API Gateway forwards request to Trip Service.
3. Trip Service calculates route distance using OSRM.
4. Trip Service publishes `trip.requested` event to RabbitMQ.
5. Driver Service consumes the event and searches nearby drivers.
6. Trip Service updates trip status.
7. WebSocket sends updates to the rider.
8. After completion, Payment Service processes payment through Stripe.
## System Design Considerations

---

### Scalability

Services are independently scalable.

Trip Service handles the highest request volume during ride requests. By isolating it as a separate service, additional instances can be deployed without scaling the entire system.

RabbitMQ queues act as buffers during traffic spikes.

---

### Latency

Low-latency communication between services is achieved using gRPC instead of REST.

Protocol Buffers reduce payload size and provide compile-time schema validation.

---

### Fault Isolation

Failures are isolated through asynchronous messaging.

If a service becomes temporarily unavailable, events remain in RabbitMQ queues until consumers recover.

This prevents cascading service failures.

---

### Data Consistency

The system follows eventual consistency across services.

Each service owns its data store and communicates through events rather than shared databases.

This prevents schema coupling between services.

---

### Horizontal Scaling

Services can be replicated across multiple nodes using Kubernetes.

Stateless services allow load balancing across instances.

---

# Architecture

## Services

### API Gateway (Port 8081)

Single entry point for client requests.

Responsibilities:

* Request routing
* Authentication
* Rate limiting
* API aggregation

The gateway prevents exposure of internal service endpoints.

---

### Trip Service (gRPC : 9093)

Handles trip lifecycle and fare computation.

Responsibilities:

* Route calculation using OpenStreetMap
* Fare estimation
* Trip state management

**Environment:** `DRIVER_SERVICE_URL` must point at the Driver Service gRPC endpoint (host:port as used inside the cluster or Docker network), for example `driver-service:8080` in local Compose / dev Kubernetes, or `driver-service:9092` where production manifests expose that port. If unset, a default may work in some environments but should be set explicitly for carpool seat sync.

---

### Driver Service (gRPC : 9092)

Maintains driver availability and location.

Responsibilities:

* Driver location tracking
* Geospatial search for nearby drivers
* Trip assignment

Uses **geohash indexing** for efficient driver discovery.

---

### Payment Service

Handles trip payment processing.

Responsibilities:

* Stripe payment integration
* Payment verification
* Trip completion handling

---

### Frontend (Port 3000)

Next.js application that provides:

* Trip request interface
* Driver location tracking
* Payment interface

---

# Infrastructure Components

### RabbitMQ

Message broker for asynchronous communication.

Used for:

* Driver matching
* Trip lifecycle events
* Payment processing

Includes:

* Retry logic
* Dead letter queues (DLQ)
* At-least-once delivery handling

The **`find_available_drivers`** queue uses a **message TTL** (120 seconds). Messages that are not consumed in that window expire into the DLQ; the API Gateway turns those into a **no drivers found** signal to the rider. For full details on this mechanism, see [Trip Request TTL & DLQ Workflow](docs/TTL_WORKFLOW.md). **If you upgrade an existing RabbitMQ deployment**, delete the old `find_available_drivers` queue once so it can be recreated with TTL arguments (RabbitMQ does not patch queue options in place).

---

### MongoDB

Stores trip and service state.

Used for:

* Trip records
* Route calculations
* Driver location data

Includes **2dsphere indexing** for geospatial queries.

---

### Jaeger

Distributed tracing system used to track requests across services.

Used for:

* latency debugging
* request tracing
* performance analysis

---

### Kubernetes

Container orchestration platform used for:

* service deployment
* scaling
* service networking

---

### Tilt

Development tool used to run the microservices stack locally with automatic rebuilds.

---

# Key System Features

### Asynchronous Messaging

RabbitMQ is used to decouple services and prevent cascading failures.

Benefits:

* resilience
* buffering during service downtime
* improved reliability

---

### Real-Time Updates

WebSockets enable real-time driver location updates and trip state notifications.

---

### Fare Calculation

Fare is computed based on:

* vehicle type
* route distance
* estimated trip duration

---

### Geospatial Driver Discovery

Driver Service uses **geohash indexing** to quickly locate nearby drivers. For carpooling, the service filters drivers by path overlap and available seats.

---

### Carpool Engine & Multi-Trip Queue

The system supports sophisticated carpooling logic:

*   **Real-time Seat Management**: Drivers track available seats, which update optimistically on acceptance and synchronize with the backend.
*   **Multi-Trip Request Queue**: Drivers can receive and queue multiple ride requests while en route.
*   **Path Overlap Heuristic**: The engine logic ensures riders are only matched with drivers whose current routes significantly overlap, maximizing vehicle utility.
*   **Dynamic Fare Scaling**: Fares are automatically adjusted based on seat count and carpool participation.


### Failure Handling

The system includes:

* retry mechanisms
* dead letter queues
* graceful shutdown handling

---

# Architecture Decisions

## Microservices Architecture

Services are separated into independent components.

Benefits:

* independent scaling
* isolated failures
* independent deployment

Trade-off:

* increased operational complexity

---

## Database Isolation

Each service owns its data.

Benefits:

* reduced service coupling
* independent schema evolution

Trade-off:

* cross-service joins are not possible

---

## Hybrid Communication Model

Two communication patterns are used.

**gRPC**

Used for synchronous service communication requiring immediate responses.

Advantages:

* strongly typed interfaces
* lower latency
* smaller payloads

**RabbitMQ**

Used for asynchronous workflows.

Advantages:

* service decoupling
* fault tolerance
* event buffering

---

## API Gateway Pattern

The API gateway acts as the single public interface.

Benefits:

* centralized authentication
* traffic control
* internal service isolation

---

## Service Layer Architecture

Each service follows a layered structure:

1. Transport Layer – API handlers
2. Service Layer – business logic
3. Repository Layer – database access

This design improves testability and maintainability.

---

# Technologies Used

**Languages**

* Go
* TypeScript

**Communication**

* gRPC
* WebSockets

**Infrastructure**

* Docker
* Kubernetes
* RabbitMQ

**Databases**

* MongoDB

**Observability**

* OpenTelemetry
* Jaeger

**Frontend**

* Next.js
* Tailwind CSS

---

# Skills Demonstrated

This project demonstrates experience with:

* Microservices architecture
* Event-driven systems
* Distributed tracing
* Geospatial querying
* Message queue reliability
* Kubernetes deployment
* Payment system integration

---

