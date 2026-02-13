# Hybrid Logistics Engine

> **Note:** For setup instructions, technology stack, and implementation details, please see the [Technical Assessment](Technical_Assessment.md).

A real-world ride-sharing platform built with modern microservices architecture. Think of it as a simplified Uber backend—connecting riders with drivers in real-time, calculating routes, handling payments, and managing the entire trip lifecycle.

## 🎥 Project Demo

[![Project Demo](https://cdn.loom.com/sessions/thumbnails/73bcc4b213bf436091b99df93facfa06-with-play.gif)](https://www.loom.com/share/73bcc4b213bf436091b99df93facfa06)



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

#### **Trip Service** - The Brain (gRPC Port 9093)
This is where the magic happens! When you request a ride, this service figures out the best route using OpenStreetMap data and calculates how much it'll cost based on the vehicle type you choose (SUV, Sedan, Van, or Luxury).

<img width="1912" height="1040" alt="Screenshot from 2025-11-12 03-27-53" src="https://github.com/user-attachments/assets/5d825297-ad2a-400e-8ecf-a9fdc0aa60b6" />

#### **Message Reliability** - Making Sure Nothing Gets Lost
Ever wonder what happens if a message fails? We've got Dead Letter Queues (DLQ) that catch failed messages so we can retry them later. Think of it as a safety net for the system.

<img width="1912" height="1040" alt="Screenshot from 2025-11-12 03-29-48" src="https://github.com/user-attachments/assets/5cb3d4a1-8bfa-4156-98cc-42f6d7036928" />
<img width="719" height="378" alt="Screenshot from 2025-11-12 03-36-58" src="https://github.com/user-attachments/assets/331e05cc-d4f3-4436-a237-e3a30423172c" />
<img width="1920" height="1080" alt="Screenshot from 2025-11-12 03-38-35" src="https://github.com/user-attachments/assets/b97ddf8a-d207-417f-893b-7e3d06499e51" />
<img width="916" height="460" alt="Screenshot from 2025-11-12 03-53-28" src="https://github.com/user-attachments/assets/5a6ed09c-7172-4e19-ba6c-1029fa2cc162" />

#### **Driver Service** - Finding Your Ride (gRPC Port 9092)
This service keeps track of all available drivers and their locations. When you request a ride, it uses smart geohash indexing to quickly find drivers near you and assigns the trip fairly.

#### **Web Frontend** - What You See (Port 3000)
A beautiful, responsive web app where you can request rides, see drivers on a map in real-time, and complete payments. Built with modern React and styled with Tailwind CSS.

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

### Real-Time Updates
WebSocket connections keep you and your driver in sync. See the driver's location update on the map as they approach, get notified when they arrive, and track your trip in real-time. No need to refresh the page!

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

## 🎓 What You'll Learn From This Project

Building this project teaches you real-world skills that companies actually use:

- **Microservices** - How to break down a big application into smaller, manageable pieces that work together
- **Event-Driven Architecture** - Using message queues to build resilient systems that can handle failures gracefully
- **Real-Time Communication** - Managing WebSocket connections to keep users updated instantly
- **Payment Processing** - Integrating with Stripe securely, handling webhooks, and managing payment states
- **Kubernetes & Docker** - Deploying and orchestrating containerized applications
- **Distributed Tracing** - Debugging complex systems where a single request touches multiple services
- **Clean Architecture** - Organizing code in a way that's maintainable and testable

---

**Built with ❤️ using**: Go · gRPC · RabbitMQ · MongoDB · Kubernetes · Next.js · TypeScript



