# Hybrid Logistics Engine

A production-oriented microservices architecture for a ride-sharing platform, implementing real-time driver matching, route optimization, and payment processing. Built with Go, gRPC, RabbitMQ, and deployed on Kubernetes.

## 🎯 Project Overview

This platform demonstrates modern backend engineering practices through a real-world use case: connecting riders with drivers in real-time. The system handles trip requests, calculates dynamic pricing, manages driver assignments, and processes payments—all while maintaining service isolation and scalability.

**Key Metrics:**
- 4 independent microservices communicating via gRPC
- 8 specialized message queues for event-driven workflows
- Real-time updates via WebSocket connections
- Integration with OSRM API for route calculation
- Stripe payment processing with webhook handling

## 🏗️ Architecture

### Service Overview

```
┌─────────────┐      ┌──────────────┐      ┌──────────────┐
│  Web Client │─────▶│ API Gateway  │◀────▶│ Trip Service │
│  (Next.js)  │      │   (HTTP/WS)  │      │    (gRPC)    │
└─────────────┘      └──────┬───────┘      └──────┬───────┘
                            │                      │
                            │    ┌─────────────────┤
                            │    │                 │
                     ┌──────▼────▼──┐      ┌──────▼────────┐
                     │   RabbitMQ   │◀────▶│ Driver Service│
                     │ (Message Bus)│      │    (gRPC)     │
                     └──────────────┘      └───────────────┘
```

### Core Services

#### **API Gateway** (Port 8081)
Entry point for all client requests. Handles HTTP routing, WebSocket connections for real-time updates, and coordinates between frontend and backend services.<img width="1282" height="833" alt="Screenshot from 2025-11-12 03-56-33" src="https://github.com/user-attachments/assets/9297db26-ef40-4929-ae8e-ba5592b8e0bf" />


**Technology**: Go HTTP server, Gorilla WebSocket  
**Responsibilities**: Request routing, CORS handling, WebSocket management, Stripe webhooks

#### **Trip Service** (gRPC Port 9093)
Core business logic for trip lifecycle management. Calculates routes using OSRM API, estimates fares with multiple package types (SUV, Sedan, Van, Luxury), and persists trip data.

**Technology**: gRPC, MongoDB, RabbitMQ publisher  
**Responsibilities**: Route calculation, fare estimation, trip state management, event publishing
<img width="1912" height="1040" alt="Screenshot from 2025-11-12 03-27-53" src="https://github.com/user-attachments/assets/5d825297-ad2a-400e-8ecf-a9fdc0aa60b6" />


#### **DLQ/DLX Queries**
**Technology**: gRPC, RabbitMQ publisher  
**Responsibilities**: Dead Letter Queues and Dead Letter Exhange
<img width="1912" height="1040" alt="Screenshot from 2025-11-12 03-29-48" src="https://github.com/user-attachments/assets/5cb3d4a1-8bfa-4156-98cc-42f6d7036928" />
<img width="719" height="378" alt="Screenshot from 2025-11-12 03-36-58" src="https://github.com/user-attachments/assets/331e05cc-d4f3-4436-a237-e3a30423172c" /><img width="1920" height="1080" alt="Screenshot from 2025-11-12 03-38-35" src="https://github.com/user-attachments/assets/b97ddf8a-d207-417f-893b-7e3d06499e51" />

<img width="916" height="460" alt="Screenshot from 2025-11-12 03-53-28" src="https://github.com/user-attachments/assets/5a6ed09c-7172-4e19-ba6c-1029fa2cc162" />



#### **Driver Service** (gRPC Port 9092)
Manages driver operations including real-time location tracking, trip assignment logic, and driver availability. Uses geohash-based spatial indexing for efficient proximity searches.

**Technology**: gRPC, geohash indexing, RabbitMQ consumer/publisher  
**Responsibilities**: Driver registration, location tracking, trip dispatch, acceptance workflow

#### **Web Frontend** (Port 3000)
Modern React application with interactive maps, real-time trip tracking, and payment integration.

**Technology**: Next.js 15, React 19, TypeScript, Tailwind CSS, Leaflet maps, Stripe.js

### Infrastructure Components

- **RabbitMQ**: Message broker with topic exchange, durable queues, and dead letter exchange (DLX) for failed messages
- **MongoDB**: Document store for trips and fare calculations with geospatial indexing
- **Jaeger**: Distributed tracing for request flow visualization and performance monitoring
- **Kubernetes**: Container orchestration with health checks, config management, and service discovery
- **Tilt**: Local development environment with hot reloading and automated builds

## 💡 Technical Highlights

### Event-Driven Architecture
Implemented asynchronous message passing with RabbitMQ to decouple services and enable independent scaling. Events flow through specialized queues:
- `find_available_drivers` - Trip creation triggers driver search
- `driver_cmd_trip_request` - Commands sent to specific drivers
- `driver_trip_response` - Driver acceptance/decline responses
- Message durability ensures reliability during service restarts

### Real-Time Communication
Dual WebSocket endpoints (`/ws/drivers`, `/ws/riders`) provide bidirectional communication for live location updates and trip status changes. Connection pooling and heartbeat mechanisms maintain stable connections.

### Dynamic Pricing Engine
Multi-tier fare calculation considering:
- Base fare by vehicle type (SUV: $2.00, Sedan: $3.50, Van: $4.00, Luxury: $10.00)
- Distance-based pricing using OSRM route calculations
- Time-based components for duration estimates
- Extensible pricing configuration for surge pricing

### Geospatial Driver Matching
Efficient driver discovery using geohash spatial indexing. Proximity-based searches find available drivers within configurable radius, with fair dispatch preventing driver starvation.

### Observability & Monitoring
OpenTelemetry instrumentation across HTTP, gRPC, and database operations. Jaeger traces visualize request flows through microservices, helping identify bottlenecks and debug issues in distributed transactions.

### Graceful Degradation
Circuit breaker patterns, retry mechanisms with exponential backoff, and fallback responses ensure system resilience. Dead letter queues capture failed messages for manual recovery.

## 🛠️ Technology Stack

### Backend
- **Language**: Go 1.23 (goroutines for concurrency, static binary compilation)
- **Communication**: gRPC with Protocol Buffers, HTTP/REST, WebSocket
- **Message Broker**: RabbitMQ with AMQP 0.9.1
- **Database**: MongoDB 7.x with geospatial indexing
- **Tracing**: OpenTelemetry + Jaeger
- **Payments**: Stripe API with webhook signature verification

### Frontend
- **Framework**: Next.js 15 with App Router, React 19
- **Styling**: Tailwind CSS 3.4, Radix UI components
- **Maps**: Leaflet 1.9 with React bindings
- **Geolocation**: Geohash libraries for spatial encoding

### Infrastructure
- **Containers**: Docker with multi-stage builds
- **Orchestration**: Kubernetes (deployments, services, configmaps, secrets)
- **Development**: Tilt for hot reloading and local K8s workflow
- **Build**: Go modules with vendoring support

## 🚀 Getting Started

### Prerequisites
```bash
# Required tools
- Docker Desktop 4.0+
- Go 1.23+
- kubectl 1.28+
- Tilt 0.33+
- Minikube (for local Kubernetes)
```

### Quick Start
```bash
# 1. Clone repository
git clone <repository-url>
cd Ride-Sharing-Microservices-Backend

# 2. Start local Kubernetes cluster
minikube start --driver=docker --memory=6144 --cpus=4

# 3. Generate Protocol Buffer files
make generate-proto

# 4. Start development environment
tilt up

# 5. Access services
# - Web UI: http://localhost:3000
# - API Gateway: http://localhost:8081
# - Jaeger UI: http://localhost:16686
# - RabbitMQ Management: http://localhost:15672
```

### Development Workflow
Tilt monitors file changes and automatically rebuilds/redeploys affected services. View build status and logs in the Tilt UI at `http://localhost:10350`.

## 📁 Project Structure

```
.
├── services/
│   ├── api-gateway/          # HTTP/WebSocket gateway
│   ├── trip-service/         # Trip management (clean architecture)
│   │   ├── cmd/              # Application entrypoint
│   │   ├── internal/
│   │   │   ├── domain/       # Business logic
│   │   │   ├── infrastructure/ # External integrations
│   │   │   └── service/      # Application services
│   │   └── pkg/types/        # Public types
│   └── driver-service/       # Driver operations
├── shared/
│   ├── contracts/            # Shared contracts (AMQP, HTTP, WS)
│   ├── messaging/            # RabbitMQ client abstraction
│   ├── proto/                # Generated gRPC code
│   └── types/                # Common type definitions
├── web/                      # Next.js frontend
├── proto/                    # Protocol Buffer definitions
├── infra/
│   ├── development/k8s/      # Local K8s manifests
│   └── production/k8s/       # Production configurations
└── Tiltfile                  # Development automation
```

## 🔑 Key Features Implemented

### Trip Management
- ✅ Route calculation with OSRM API integration
- ✅ Multi-tier pricing (4 vehicle categories)
- ✅ Trip state machine (Pending → Driver Assigned → In Progress → Completed)
- ✅ Real-time trip updates via WebSocket
- ✅ Fare validation and user ownership checks

### Driver Operations
- ✅ Geohash-based location indexing
- ✅ Real-time location updates
- ✅ Fair dispatch algorithm
- ✅ Trip acceptance/decline workflow
- ✅ Driver availability management

### Payment Processing
- ✅ Stripe Checkout session creation
- ✅ Webhook signature verification
- ✅ Payment state tracking
- ✅ Idempotency handling

### Infrastructure
- ✅ Distributed tracing with OpenTelemetry
- ✅ Message durability and reliability
- ✅ Graceful shutdown handling
- ✅ Health checks and readiness probes
- ✅ Hot reloading in development

## 📊 System Characteristics

**Scalability**: Horizontal scaling supported for all services via Kubernetes HPA  
**Latency**: Trip preview <200ms (including OSRM API), trip creation <100ms  
**Reliability**: At-least-once message delivery, dead letter queues, retry mechanisms  
**Observability**: Full request tracing, structured logging, performance metrics

## 🧪 Testing & Quality

- Input validation at API boundaries
- Fare ownership verification before trip creation
- Webhook signature validation for security
- Error handling with exponential backoff
- Circuit breaker patterns for external services

## 📈 Future Enhancements

- [ ] JWT authentication and authorization
- [ ] Redis caching for routes and driver locations
- [ ] PostgreSQL for relational data (users, driver profiles)
- [ ] Prometheus metrics and Grafana dashboards
- [ ] CI/CD pipeline with GitHub Actions
- [ ] End-to-end integration tests
- [ ] Load testing with k6
- [ ] Rate limiting and API quotas

## 🎓 Learning Outcomes

This project demonstrates practical experience with:
- **Microservices Architecture**: Service decomposition, communication patterns, and orchestration
- **Event-Driven Systems**: Asynchronous messaging, event sourcing, and eventual consistency
- **Cloud-Native Development**: Containerization, Kubernetes, and cloud-ready design
- **Real-Time Systems**: WebSocket management, connection pooling, and message broadcasting
- **Payment Integration**: Secure payment processing, webhook handling, and PCI compliance
- **Observability**: Distributed tracing, monitoring, and debugging complex systems
- **Domain-Driven Design**: Clean architecture, bounded contexts, and business logic isolation

## 🔧 Troubleshooting

**Port conflicts**: Ensure ports 3000, 8081, 9092, 9093, 5672, 15672, 16686 are available  
**Memory issues**: Increase Docker memory to 6GB+ in Docker Desktop settings  
**Service failures**: Check logs with `kubectl logs -f deployment/<service-name>`  
**Build issues**: Run `tilt down` then `tilt up` to reset the environment

## 📝 Notes
- **OSRM API**: Uses public OSRM instance; for production, deploy your own OSRM server
- **Stripe**: Requires Stripe API keys (test mode) configured in secrets.

  
- **MongoDB**: Currently uses in-memory implementation for development; production requires MongoDB Atlas or self-hosted instance

---

**Built with**: Go · gRPC · RabbitMQ · MongoDB · Kubernetes · Next.js · TypeScript




