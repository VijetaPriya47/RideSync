---
sidebar_position: 10
title: RabbitMQ Queue Registry (11 Queues)
---

# RabbitMQ Queue Registry

The Hybrid Logistics Engine relies on exactly **11 dedicated RabbitMQ queues** to handle all asynchronous events between microservices. Below is an exhaustive list of every queue, its primary function, and the underlying logic connecting it.

## Trip & Driver Search (4 Queues)

These queues manage the core matching algorithm between riders requesting a ride and drivers accepting them.

1. **`find_available_drivers`**
   - **Consumer:** `Driver Service`
   - **Role:** The backbone of the matching engine. It receives `TripEventCreated` and `TripEventDriverNotInterested` events. The driver service processes this queue to locate the next available matching driver.
   - **Key Detail:** It has a global `120s TTL`. Messages here that organically reach the timeout limit are automatically shunted to the `dead_letter_queue`.

2. **`search_retry_queue`**
   - **Consumer:** *None (Headless Wait Queue)*
   - **Role:** Implements the interval driver search. It receives messages from the driver service but has no active consumers. 
   - **Key Detail:** It features a strict `10s TTL`. When a message expires here, it is automatically routed back to the main `TripExchange` to fire another pass through `find_available_drivers`.

3. **`driver_cmd_trip_request`**
   - **Consumer:** `API Gateway`
   - **Role:** Carries direct command payloads addressed to a specific Driver ID to offer them a ride. The API Gateway takes these payloads and forwards them directly to the driver's phone via WebSockets.

4. **`driver_trip_response`**
   - **Consumer:** `Trip Service`
   - **Role:** The inbound pipe from the drivers. When a driver clicks "Accept" or "Decline" on their screen, the API Gateway pushes that response into this queue so the Trip Service can lock the trip or continue the driver search.

---

## API Gateway / WebSocket Notifications (4 Queues)

These queues exclusively handle pushing state-change UI updates down the WebSocket pipes to connected riders and drivers.

5. **`notify_trip_created`**
   - **Consumer:** `API Gateway`
   - **Role:** Signals to the rider UI that the trip has begun the distributed driver search successfully. 

6. **`notify_driver_assign`**
   - **Consumer:** `API Gateway`
   - **Role:** Sends an alert to the rider UI that a driver has successfully accepted their ride request, providing driver details (name, car, ETA).

7. **`notify_driver_no_drivers_found`**
   - **Consumer:** `API Gateway`
   - **Role:** Specifically handles the frontend alert triggered when the matching engine exhausts all active drivers (or the DLQ handles a timeout) and gives up.

8. **`dead_letter_queue`**
   - **Consumer:** `API Gateway` (via `dlq_consumer.go`)
   - **Role:** The ultimate fallback sink for expired or rejected messages. The API Gateway specifically monitors it for expired driver searches or outdated payload rejections, converting them into terminal "No drivers found" signals for the Rider app without relying on the driver-service to manually kill trips.

---

## Payment Workflows (3 Queues)

These isolated queues ensure financial transactions, webhook verifications, and external APIs (Stripe) do not slow down or bottleneck the core routing loops.

9. **`payment_trip_response`**
   - **Consumer:** `Payment Service`
   - **Role:** Informs the payment service that a driver has locked a trip, triggering the initial setup of a Stripe checkout session based on the agreed fare.

10. **`notify_payment_session_created`**
   - **Consumer:** `API Gateway`
   - **Role:** Once the Payment Service generates a Stripe URL, it is pushed into this queue. The API Gateway forwards the URL to the rider's UI, redirecting their device to the payment portal.

11. **`payment_success`**
   - **Consumer:** `Trip Service`
   - **Role:** When Stripe successfully processes a credit card, its webhook hits our API Gateway, which validates the signature. If authentic, the gateway places the payload in this queue. The Trip Service consumes it to change the ride status from "accepted" to "payed".
