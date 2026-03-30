# AI Agent Guidelines for Hybrid Logistics Engine

This document provides essential instructions for AI agents (and human developers) working on the Hybrid Logistics Engine codebase. Compliance with these guidelines ensures that documentation remains synchronized with architectural and code changes.

## 1. Documentation-First Mentality

Whenever you modify the codebase, you **must** assess the impact on the existing documentation. 

### Docusaurus Docs
If your changes affect any of the following, update the corresponding files in `docs-site/docs/`:
- **API endpoints or request/response structures**: Update `docs/api/`.
- **Infrastructure or Deployment**: Update `docs/deployment/`.
- **Core Logic or Architecture**: Update `docs/architecture/`.
- **Database Schema**: Update `docs/database/`.

## 2. Maintaining System Transparency

- **Task Tracking**: Always maintain a `task.md` in the `.brain` or similar directory during active development.
- **Walkthroughs**: After completing a feature or a significant refactor, generate a `walkthrough.md` that visually demonstrates the changes (using screenshots or Mermaid diagrams).
- **Architecture Updates**: If changing service-to-service communication, update the Mermaid diagrams in `docs/architecture/hld.md` or related flow files.

## 3. Communication Standards

- **Low-Level Design (LLD)**: Use LLD documents to detail complex implementations before or after finishing. Refer to `trip_service_lld.md` as a template.
- **Cleanup**: Do not leave "todo" comments or debugging logs in production-ready branches. 

## 4. Verification

Before finishing a task:
- Run `npm run build` in `docs-site` to ensure no broken links were introduced.
- Verify that your changes appear correctly in the sidebar.
- Ensure all Mermaid diagrams render as expected.

## 5. Trip Service → Driver Service (`DRIVER_SERVICE_URL`)

Trip Service calls Driver Service over gRPC to reserve and release passenger seats when a driver accepts a trip and when payment completes (carpooling / multi-seat flows).

- Set **`DRIVER_SERVICE_URL`** to the **Kubernetes Service DNS name and port** where Driver Service listens for gRPC (same port as the driver Service’s `targetPort`).
- Examples:
  - Local Docker Compose (driver defaults to port **8080** inside the container): `driver-service:8080`
  - Development Kubernetes manifest in this repo: `driver-service:8080`
  - Production Kubernetes manifest in this repo uses driver Service port **9092**: `driver-service:9092`
- If the client cannot connect, Trip Service logs a warning and continues; **seat counts on the driver will not stay in sync** until connectivity is fixed.

## 6. RabbitMQ: `find_available_drivers` queue and TTL

The **`find_available_drivers`** queue is declared with **`x-message-ttl`** (120 seconds) so driver-search messages expire if no consumer processes them in time. Expired messages are dead-lettered; the API Gateway notifies riders over WebSocket (`trip.event.no_drivers_found`).

**Important:** RabbitMQ does **not** update arguments on an existing queue. If the queue was created earlier **without** TTL, you must **delete the `find_available_drivers` queue** (or recreate the vhost) before redeploying so it is recreated with the correct TTL. Until then, the 2-minute expiry will not apply.

---
*Following these rules ensures that Hybrid Logistics Engine remains a well-documented, state-of-the-art logistics platform.*
