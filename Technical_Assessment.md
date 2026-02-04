# Technical Assessment & Implementation Details

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

## 🔨 Technical Challenges & Solutions

### Challenge 1: CORS Blocking Frontend-Backend Communication

**The Problem:**  
The frontend (running on port 3000) couldn't talk to the API Gateway (port 8081) because browsers block cross-origin requests by default for security.

**The Solution:**  
Implemented custom Go middleware to handle CORS:
- Responds to preflight OPTIONS requests
- Sets `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, and `Access-Control-Allow-Headers`
- Uses `*` for development, but production should use environment-variable-driven origin whitelisting

### Challenge 2: Graceful Shutdowns

**The Problem:**  
When Tilt auto-reloads or Kubernetes restarts a service, active user connections would get killed mid-request. Users would see hanging connections or lost data—terrible user experience!

**The Solution:**  
Implemented graceful shutdown pattern in Go:
```go
server.Shutdown(context.WithTimeout(context.Background(), 10*time.Second))
```
This gives the server 10 seconds to finish processing in-flight requests before shutting down. No more hanging clients!

### Challenge 3: Bridging Async Backend with Sync Frontend

**The Problem:**  
The backend is event-driven and asynchronous (finding drivers takes time), but users expect immediate feedback. We can't make them wait with a loading spinner for 30 seconds.

**The Solution:**  
- **Immediate Response:** When a user creates a trip, we immediately return "Trip Created" (HTTP 201)
- **Background Processing:** The actual driver search happens asynchronously via RabbitMQ
- **Real-Time Updates:** WebSocket connection pushes updates ("Driver Found!", "Driver Accepted") to the frontend as they happen

This gives users instant feedback while heavy operations run in the background.

### Challenge 4: Maintaining Stateful WebSocket Connections

**The Trade-off:**  
WebSockets require the server to maintain persistent connections, which is more complex than stateless REST. But it prevents "hanging" HTTP requests and timeouts when backend microservices are slow.

**The Implementation:**  
- Separate WebSocket endpoints for riders (`/ws/riders`) and drivers (`/ws/drivers`)
- Connection pooling to manage multiple concurrent connections
- Heartbeat mechanisms to detect and clean up dead connections

## 🚀 Running It Yourself

### Prerequisites
```bash
# Install these first:
- Docker Desktop 4.0+ (for running containers)
- Go 1.23+ (the programming language)
- kubectl 1.28+ (for talking to Kubernetes)
- Tilt 0.33+ (makes development super easy)
- Minikube (creates a local Kubernetes cluster)
```

### Let's Get Started!
```bash
# 1. Clone repository
git clone <repository-url>
cd Ride-Sharing-Microservices-Backend

# 2. Fire up a local Kubernetes cluster
minikube start --driver=docker --memory=6144 --cpus=4

# 3. Generate the gRPC code from proto files
make generate-proto

# 4. Start everything with Tilt (this is the magic command!)
tilt up

# 5. Open these in your browser:
# - Web app: http://localhost:3000
# - API Gateway: http://localhost:8081
# - Jaeger (tracing): http://localhost:16686
# - RabbitMQ dashboard: http://localhost:15672
```

### Development Tips
Tilt watches your files and automatically rebuilds when you make changes. Just edit your code and watch it update! Check the Tilt dashboard at `http://localhost:10350` to see build status and logs.

## 🔧 Common Issues & Solutions

**"Port already in use" errors**  
Make sure nothing else is using ports 3000, 8081, 9092, 9093, 5672, 15672, or 16686. You can check with `lsof -i :<port>` and kill the process if needed.

**Docker running out of memory**  
Go to Docker Desktop settings and bump the memory limit to at least 6GB. Kubernetes needs room to breathe!

**Services crashing or not starting**  
Check the logs to see what's wrong: `kubectl logs -f deployment/<service-name>`  
Common culprits: missing environment variables or database connection issues.

**Tilt acting weird**  
Sometimes it just needs a fresh start: `tilt down` then `tilt up` again.

## 📝 Important Notes

- **OpenStreetMap Routes**: We're using the public OSRM API for route calculations. It's free but rate-limited. For production, you'd want to run your own OSRM server.
  
- **Stripe Payments**: You'll need to set up a Stripe account (free for testing) and add your API keys to the Kubernetes secrets. Don't worry, test mode means no real money changes hands!

- **MongoDB**: Currently using an in-memory implementation for local development. For production, you'd use MongoDB Atlas or run your own MongoDB instance.

## 🧪 Testing & Quality

- Input validation at API boundaries
- Fare ownership verification before trip creation
- Webhook signature validation for security
- Error handling with exponential backoff
- Circuit breaker patterns for external services
