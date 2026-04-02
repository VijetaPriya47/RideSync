---
sidebar_position: 2
title: Trip Request TTL Workflow
---

# Trip Request TTL & DLQ Workflow

To ensure riders aren't left waiting indefinitely when no drivers are available, the system implements a robust **Time-To-Live (TTL)** and **Dead Letter Queue (DLQ)** mechanism.

## Backend Architecture

### 1. Request Publication
When a rider requests a trip, the **Trip Service** publishes a `trip.requested` event. This event is routed to the `find_available_drivers` queue.

### 2. Queue Configuration
The `find_available_drivers` queue is configured with specialized RabbitMQ arguments:
- **`x-message-ttl`**: `120000` (120 seconds). Any message sitting in the queue for longer than this is automatically moved.
- **`x-dead-letter-exchange`**: `dlx`. This exchange handles redirected "dead" messages.

### 3. Expiration Logic
If no driver consumes (Accepts) the ride request within the 120-second window:
1. RabbitMQ marks the message as **Expired**.
2. The message is automatically moved to the `dead_letter_queue`.

### 4. API Gateway Notification
The **API Gateway** runs a dedicated `dlq_consumer` that watches the `dead_letter_queue`.
- It identifies messages that expired specifically from the `find_available_drivers` queue.
- It extracts the `OwnerID` (Rider ID) from the message.
- it sends a `TripEventNoDriversFound` WebSocket message to the rider's frontend.

## User Experience
On the frontend, the rider sees a "Searching" timer. If the 120-second window passes without a match:
1. The backend triggers the TTL workflow.
2. The UI switches to the "No drivers found" screen.
3. The rider is prompted to either **Cancel** or **Increase Fare** to attract more drivers.

---
> [!NOTE]
> This ensures high availability and prevents "hanging" requests in the system.
