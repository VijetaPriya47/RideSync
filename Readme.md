# Hybrid Logistics Engine

A real-world ride-sharing platform built with modern microservices architecture. Think of it as a simplified Uber backend—connecting riders with drivers in real-time, calculating routes, handling payments, and managing the entire trip lifecycle.

## 🎯 What This Project Does

Ever wondered how ride-sharing apps like Uber or Lyft work behind the scenes? This project recreates that magic using modern backend technologies. When a rider requests a trip, the system:
- Finds nearby available drivers using smart location tracking
- Calculates the best route and estimates the fare
- Assigns a driver and keeps everyone updated in real-time
- Processes payments securely when the trip is complete

**What's Inside:**
- 4 independent microservices working together (API Gateway, Trip Service, Driver Service, Payment Service)
- Real-time communication using WebSockets (see driver locations live!)
- Smart message queues that keep everything running smoothly
- Route calculation powered by OpenStreetMap
- Secure payment processing with Stripe

## 🏗️ How It All Works Together

### The Big Picture

Imagine you're ordering a ride. Here's what happens:

1. **You make a request** through the web app (built with Next.js)
2. **API Gateway** receives your request and talks to other services
3. **Trip Service** calculates the route and estimates the fare
4. **Driver Service** finds available drivers nearby
5. **RabbitMQ** makes sure all services stay in sync with message queues
6. **Payment Service** handles the payment when the trip is done

All of this happens in seconds, and you get real-time updates throughout!

### The Services

#### **API Gateway** - The Bouncer (Port 8081)
This is the single entry point for all client requests—like a bouncer at a club checking IDs. It handles authentication, rate limiting (so one user can't spam and crash the system), and routes traffic to the right services.

**Why a gateway?** We don't want to expose internal service IP addresses to the public internet. The frontend only needs to know one address, and we get centralized security and monitoring.

**Built with:** Go, WebSocket for real-time updates  
**What it does:** Routes requests, manages live connections, handles Stripe payment webhooks, CORS handling

#### **Trip Service** - The Brain (gRPC Port 9093)
This is where the magic happens! When you request a ride, this service figures out the best route using OpenStreetMap data and calculates how much it'll cost based on the vehicle type you choose (SUV, Sedan, Van, or Luxury).

**Built with:** gRPC, MongoDB, RabbitMQ  
**What it does:** Calculates routes, estimates fares, manages trip lifecycle, stores trip data

<img width="1912" height="1040" alt="Screenshot from 2025-11-12 03-27-53" src="https://github.com/user-attachments/assets/5d825297-ad2a-400e-8ecf-a9fdc0aa60b6" />

#### **Message Reliability** - Making Sure Nothing Gets Lost
Ever wonder what happens if a message fails? We've got Dead Letter Queues (DLQ) that catch failed messages so we can retry them later. Think of it as a safety net for the system.

<img width="1912" height="1040" alt="Screenshot from 2025-11-12 03-29-48" src="https://github.com/user-attachments/assets/5cb3d4a1-8bfa-4156-98cc-42f6d7036928" />
<img width="719" height="378" alt="Screenshot from 2025-11-12 03-36-58" src="https://github.com/user-attachments/assets/331e05cc-d4f3-4436-a237-e3a30423172c" />
<img width="1920" height="1080" alt="Screenshot from 2025-11-12 03-38-35" src="https://github.com/user-attachments/assets/b97ddf8a-d207-417f-893b-7e3d06499e51" />
<img width="916" height="460" alt="Screenshot from 2025-11-12 03-53-28" src="https://github.com/user-attachments/assets/5a6ed09c-7172-4e19-ba6c-1029fa2cc162" />

#### **Driver Service** - Finding Your Ride (gRPC Port 9092)
This service keeps track of all available drivers and their locations. When you request a ride, it uses smart geohash indexing to quickly find drivers near you and assigns the trip fairly.

**Built with:** gRPC, geohash for location tracking, RabbitMQ  
**What it does:** Tracks driver locations, finds nearby drivers, manages trip assignments, handles driver responses

#### **Web Frontend** - What You See (Port 3000)
A beautiful, responsive web app where you can request rides, see drivers on a map in real-time, and complete payments. Built with modern React and styled with Tailwind CSS.

**Built with:** Next.js 15, React 19, TypeScript, Tailwind CSS, Leaflet maps, Stripe.js

### The Supporting Cast

- **RabbitMQ** - The messenger that keeps all services talking to each other reliably
- **MongoDB** - Where we store all trip data and fare calculations
- **Jaeger** - Helps us trace requests through the system to debug and optimize performance
- **Kubernetes** - Orchestrates all the containers and keeps everything running smoothly
- **Tilt** - Makes local development a breeze with automatic rebuilds when you change code

## 💡 Cool Things This Project Does

### Messages That Never Get Lost
Instead of services calling each other directly, they send messages through RabbitMQ. This means if one service is busy or down, messages wait patiently in queues until they can be processed. It's like leaving a voicemail instead of hanging up when someone doesn't answer!

**Why this matters:** In a synchronous system, if the Driver Service goes down for maintenance, trip requests would fail. With message queues, those requests wait in RabbitMQ and get processed when the service comes back online. We trade instant consistency for **eventual consistency**—and that's what makes the system resilient.

Key message queues:
- `find_available_drivers` - Kicks off the driver search when you request a trip
- `driver_cmd_trip_request` - Sends trip requests to specific drivers
- `driver_trip_response` - Drivers accept or decline through this queue

### Real-Time Updates
WebSocket connections keep you and your driver in sync. See the driver's location update on the map as they approach, get notified when they arrive, and track your trip in real-time. No need to refresh the page!

**The challenge:** We have an event-driven backend (asynchronous) but need to update the frontend in real-time. WebSockets bridge this gap by letting the server push updates to the client instead of the client constantly polling "Are we there yet?"

### Smart Pricing
The system calculates fares based on:
- **Vehicle type** - SUV ($2.00 base), Sedan ($3.50), Van ($4.00), or Luxury ($10.00)
- **Distance** - Uses actual route distance from OpenStreetMap
- **Time** - Factors in estimated trip duration

### Finding Nearby Drivers Fast
Using geohash indexing (a clever way to encode locations), the system can quickly find drivers near you. It's like organizing drivers into geographic buckets, so we don't have to check every single driver in the city!

### Seeing What's Happening Under the Hood
With Jaeger tracing, you can actually see how a request flows through all the services. It's incredibly helpful for debugging and understanding where time is being spent.

### Built to Handle Failures
Things go wrong in distributed systems. That's why we have:
- **Retry logic** - If something fails, try again (with smart backoff)
- **Dead letter queues** - Catch messages that fail repeatedly
- **Graceful shutdowns** - Services clean up properly when restarting (no hanging connections!)

## 🎯 Design Decisions & Trade-offs

### Why Microservices Instead of a Monolith?

**The Problem with Monoliths:**
Imagine 10,000 people leaving a Taylor Swift concert and requesting rides simultaneously. In a monolith, the trip calculation service gets hammered while the payment service sits idle (rides haven't finished yet). To handle the load, you'd have to scale the *entire* application—wasting resources on parts that don't need it.

**The Microservices Solution:**
We broke the system into autonomous services that can scale independently. When trip requests spike, we spin up more Trip Service instances without touching the Payment Service. This is **targeted scaling**.

**The Trade-off:**
- ✅ **Pros:** Independent scaling, easier debugging, isolated failures, independent deployments
- ❌ **Cons:** More complex to deploy and debug, requires orchestration (Kubernetes), harder to trace requests

### Database Strategy: One Database Per Service

**Why not share a database?**
Having all services touch the same database creates **data coupling**—services step on each other's toes. If the Trip Service changes its schema, it could break the Driver Service.

**Our approach:**
Each service owns its data. If the Driver Service needs trip information, it asks the Trip Service via API. This enforces boundaries and respects contracts between services.

**The Trade-off:**
- ✅ **Pros:** True service independence, no accidental coupling, easier to change databases
- ❌ **Cons:** Can't do SQL joins across services, need to handle distributed data consistency

### Communication: Hybrid Approach (gRPC + RabbitMQ)

**Synchronous (gRPC):**
Used for direct service-to-service calls where we need an immediate response (e.g., API Gateway → Trip Service for fare calculation).

**Why gRPC over REST?**
- Type safety with Protocol Buffers (catches bugs at compile time, not in production!)
- Faster than JSON over HTTP
- Strict contracts prevent sending garbage data

**Asynchronous (RabbitMQ):**
Used for operations that don't need immediate responses (e.g., finding available drivers, processing payments).

**Why async messaging?**
Prevents **cascading failures**. If Service A calls Service B synchronously and B goes down, A also fails. With messaging, A publishes to a queue and moves on—B processes when ready.

**The Trade-off:**
- ✅ **Pros:** Services don't need to be awake simultaneously, better resilience, natural buffering
- ❌ **Cons:** Harder to debug, eventual consistency instead of immediate, more moving parts

### API Gateway: The Single Entry Point

**Why not expose all services directly?**
- **Security:** Don't want internal service IPs exposed to the public internet
- **Simplicity:** Frontend only needs to know one address
- **Control:** Can add authentication, rate limiting, and logging in one place

**What we avoid:**
Putting business logic in the API Gateway. It's a router, not a brain. Complex aggregation logic belongs in dedicated services.

### Clean Architecture: Layered Design

We use a 3-layer architecture in each service:

1. **Transport Layer** (Front Door) - Handles HTTP/gRPC, unpacks requests
2. **Service Layer** (Brains) - Business logic lives here, doesn't care how data arrived
3. **Repository Layer** (Storage) - Talks to databases, abstracts storage details

**Why the Repository Pattern?**
Decouples business logic from database technology. Want to switch from MongoDB to PostgreSQL? Just swap the repository implementation—business logic stays untouched. Also makes testing easier with in-memory fakes.

### Development Experience: Tilt.dev

**The Problem:**
Deploying microservices locally is painful—5 Docker images, uploading to registry, restarting Kubernetes cluster. Every. Single. Time.

**The Solution:**
Tilt runs on top of Kubernetes and injects code changes into running containers instantly. It's like hot reload for a complex distributed system, with all service logs in one dashboard.

### A Note on Distributed Systems vs. Microservices

**Are they the same thing?**  
Not quite! While all microservices are distributed systems, not all distributed systems are microservices.

**Distributed System:** A broad architecture where components on networked computers communicate to function as a single unit. This could be databases, web servers, caching layers, etc.

**Microservices:** A specific type of distributed system focused on:
- Loose coupling between services
- Domain-driven design (each service represents a business capability)
- Independent deployability
- Specialized scaling

**Key Differences:**
- **Focus:** Distributed systems focus on running on separate nodes; microservices focus on business-oriented modularity
- **Scope:** Distributed systems are infrastructure (databases, load balancers); microservices are application architecture
- **Goal:** Microservices aim for team agility and independent deployments; distributed systems prioritize performance and reliability
- **Complexity:** Both are complex, but microservices require more DevOps maturity (API management, service discovery, orchestration)

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

## 🚀 Running It Yourself

### What You'll Need
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
# 1. Grab the code
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

## 🧪 Testing & Quality

- Input validation at API boundaries
- Fare ownership verification before trip creation
- Webhook signature validation for security
- Error handling with exponential backoff
- Circuit breaker patterns for external services

## 🎓 What You'll Learn From This Project

Building this project teaches you real-world skills that companies actually use:

- **Microservices** - How to break down a big application into smaller, manageable pieces that work together
- **Event-Driven Architecture** - Using message queues to build resilient systems that can handle failures gracefully
- **Real-Time Communication** - Managing WebSocket connections to keep users updated instantly
- **Payment Processing** - Integrating with Stripe securely, handling webhooks, and managing payment states
- **Kubernetes & Docker** - Deploying and orchestrating containerized applications
- **Distributed Tracing** - Debugging complex systems where a single request touches multiple services
- **Clean Architecture** - Organizing code in a way that's maintainable and testable

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

---

**Built with ❤️ using**: Go · gRPC · RabbitMQ · MongoDB · Kubernetes · Next.js · TypeScript




