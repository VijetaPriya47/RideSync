# Hybrid Logistics Engine

> Setup instructions and full implementation details are available in the [Technical Assessment](Technical_Assessment.md).

A microservices-based backend system simulating a ride-sharing platform. The system matches riders with drivers, computes routes and fares, processes payments, and manages trip lifecycle events.

## Project Demo

Video demo:
[https://drive.google.com/file/d/1jHZYCv_tlQGjw5cV0MShal-_ezUGujMV/view](https://drive.google.com/file/d/1jHZYCv_tlQGjw5cV0MShal-_ezUGujMV/view)

---

# System Overview

The system processes a ride request as follows:

1. Client sends request through the **Next.js frontend**
2. **API Gateway** receives and routes the request
3. **Trip Service** calculates route and estimated fare
4. **Driver Service** finds nearby available drivers
5. **RabbitMQ** coordinates asynchronous events between services
6. **Payment Service** processes payment after trip completion

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

Driver Service uses **geohash indexing** to quickly locate nearby drivers.

---

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
