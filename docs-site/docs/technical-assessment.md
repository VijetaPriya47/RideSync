---
sidebar_position: 2
id: "technical-assessment"
title: "Technical Assessment"
---

# Technical Assessment & Implementation Details

## Technology Stack

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

## Project Structure

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

## Key Features Implemented

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

## System Characteristics

**Scalability**: Horizontal scaling supported for all services via Kubernetes HPA  
**Latency**: Trip preview &lt;200ms (including OSRM API), trip creation &lt;100ms  
**Reliability**: At-least-once message delivery, dead letter queues, retry mechanisms  
**Observability**: Full request tracing, structured logging, performance metrics

## Technical Challenges & Solutions

### Challenge 1: CORS Blocking Frontend-Backend Communication

**The Problem:**  
The frontend (running on port 3000) was isolated from the API Gateway (port 8081) as browsers block cross-origin requests by default.

**The Solution:**  
Implemented strict Go middleware to handle preflight CORS routines:
- Validates preflight OPTIONS requests dynamically.
- Transmits standardized headers: `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, and `Access-Control-Allow-Headers`.
- For production, utilizes strict environment-variable-driven origin whitelisting protocols explicitly protecting internal networks.

### Challenge 2: Context Cancellation (Graceful Shutdown)

**The Problem:**  
When container orchestration systems pre-empt nodes or Tilt triggers a live-reload cycle, active connections were explicitly terminated mid-request resulting in dropped packages and lost data sequences.

**The Solution:**  
Implemented the standard graceful shutdown pattern parsing native `os.Interrupt` bindings:
```go
server.Shutdown(context.WithTimeout(context.Background(), 10*time.Second))
```
This routine enforces a 10-second contextual bleed allowing in-flight logic boundaries to safely drain prior to container death preventing hanging client requests.

### Challenge 3: Bridging Async Event Models with Synchronous Endpoints

**The Problem:**  
Driver spatial searching algorithms utilize external OpenStreetMap API latency blocks masking synchronous response structures over RabbitMQ bounds. Standard REST structures mandate synchronous returns resulting in indefinite HTTP wait periods.

**The Solution:**  
- **Immediate Response Generation:** Initial POST requests immediately yield standard HTTP 201 (Created) states offloading payload indexing seamlessly into the queue.
- **Background Event Brokerage:** Spatial filtering routes natively dispatch across AMQP background nodes completely asynchronously.
- **Push Telemetry Updating:** Independent WebSocket routines hook into downstream queues emitting updates (Driver Match/Pricing Setup) concurrently pushing data across existing open generic connections dynamically. 

### Challenge 4: Scaling Persistent WebSocket Connections

**The Trade-off:**  
WebSocket architecture inherently bounds server load forcing continuous persistent memory contexts compared to stateless REST. However, this architectural cost definitively removes cyclic cyclic-polling lag structures significantly scaling connection stability thresholds.

**The Implementation:**  
- Separated standard `Upgrade` handling strictly across `/ws/riders` and `/ws/drivers` endpoints isolating connection types.
- Generated connection routing maps holding native reference pointers in-memory spanning concurrent TCP connections.
![Screenshot from 2025-11-12 03-27-53](https://github.com/user-attachments/assets/5d825297-ad2a-400e-8ecf-a9fdc0aa60b6)
- Initialized active "Ping/Pong" routines explicitly sweeping memory maps scrubbing stale container nodes eliminating resource limits.

## Operational Deployment Runbook

### Infrastructure Requirements
```bash
- Docker Desktop 4.0+
- Go 1.23+ 
- kubectl 1.28+ 
- Tilt 0.33+ 
- Minikube
```

### Local Cluster Initialization
```bash
# 1. Clone repository
git clone <repository-url>
cd Ride-Sharing-Microservices-Backend

# 2. Boot Minikube nodes
minikube start --driver=docker --memory=6144 --cpus=4

# 3. Compile local internal gRPC definitions
make generate-proto

# 4. Trigger Tilt macro-build cycles
tilt up
```
![Screenshot from 2025-11-12 03-29-48](https://github.com/user-attachments/assets/5cb3d4a1-8bfa-4156-98cc-42f6d7036928)
![Screenshot from 2025-11-12 03-36-58](https://github.com/user-attachments/assets/331e05cc-d4f3-4436-a237-e3a30423172c)
![Screenshot from 2025-11-12 03-38-35](https://github.com/user-attachments/assets/b97ddf8a-d207-417f-893b-7e3d06499e51)
![Screenshot from 2025-11-12 03-53-28](https://github.com/user-attachments/assets/5a6ed09c-7172-4e19-ba6c-1029fa2cc162)

### Exposed Gateway Mapping
- **Next.js Endpoints**: http://localhost:3000
- **Primary REST Gateway**: http://localhost:8081
- **Jaeger Telemetry Array**: http://localhost:16686
- **AMQP Diagnostics**: http://localhost:15672

### Iterative Build Deployment
Tilt aggressively caches hot-path files executing near real-time injection mappings spanning the `deployment` configurations without necessitating manual image pruning sequences. Local diagnostics are routed universally through http://localhost:10350.

## Troubleshooting Index

- **Port Authority Errors:** Ensure localhost isolation running `lsof -i :<port>`. Key ports: 3000, 8081, 9092, 9093, 5672, 15672, 16686.
- **Node Allocation Warnings (OOM):** Docker Desktop hypervisor settings mandate minimum 6GB dedicated RAM preventing preemption strikes.
- **Pod Container Crashes:** Parse internal logic logs utilizing standard mappings: `kubectl logs -f deployment/<service-name>`.
- **Helm/Tilt Desync:** Halt persistent volume layers running `tilt down` proceeding rapidly alongside `tilt up` completely resetting network variables.

## Technical Caveats

- **OpenStreetMap Throttle Bounds**: Utilizing public non-authenticated REST bindings yields inherent 503 limits triggering exponential backoff delays; prod requires self-hosted instances.
- **Transactional Tokens**: Standard Stripe PCI compliance boundaries require manual environment key mappings located inside explicit Kubernetes config maps isolating sandbox operations from unauthenticated public REST endpoints.
- **MongoDB Node Distribution**: Dev environment builds dynamically spin clustered in-memory configurations dynamically erasing local state maps during standard pod resets avoiding local database persistence artifacts entirely.

## Validation Specifications

- Input validation at API boundaries
- Fare ownership verification before trip creation
- Webhook signature validation for security
- Error handling with exponential backoff
- Circuit breaker patterns for external services
