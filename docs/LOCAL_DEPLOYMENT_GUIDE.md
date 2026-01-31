# Local Deployment Guide (Kind + Tilt)

This guide details how to deploy the Ride Sharing application locally using **Kind** (Kubernetes in Docker) and **Tilt**. This setup mirrors the production Kubernetes environment while offering a fast inner development loop.

## Prerequisites

Ensure you have the following installed on your machine:

1.  **Docker**: Container runtime.
2.  **Go**: Programming language (v1.22+).
3.  **Kind**: Tool for running local Kubernetes clusters.
4.  **Tilt**: Tool for microservices development.
5.  **Kubectl**: Kubernetes command-line tool.

## Initial Setup

1.  **Create a Kind Cluster**:
    ```bash
    kind create cluster --name ride-sharing
    ```

2.  **Point Kubectl to Kind**:
    ```bash
    kubectl cluster-info --context kind-ride-sharing
    ```

## Disk Space Management (Critical)

Running a full microservices stack locally consumes significant disk space. If you encounter `no space left on device` errors during image builds or loads:

1.  **Prune Docker System**:
    ```bash
    docker system prune -a -f --volumes
    ```
    *Warning: This deletes all stopped containers, unused networks, and dangling images.*

2.  **Clean Dependencies Caches**:
    ```bash
    go clean -cache
    go clean -modcache
    rm -rf web/node_modules web/.next
    ```


kind delete cluster --name ride-sharing


docker volume prune

## Deployment with Tilt

Tilt manages the entire lifecycle: building Docker images, loading them into Kind, and deploying Kubernetes manifests.

1.  **Start Tilt**:
    ```bash
    tilt up
    ```

2.  **Access the Dashboard**:
    Open [http://localhost:10355](http://localhost:10355) to monitor services and logs in real-time.

## Accessing Services

Once all services are up (green in Tilt), you can access them at these addresses:

| Service | URL | Description |
| :--- | :--- | :--- |
| **Frontend App** | [http://localhost:3000](http://localhost:3000) | The main Ride Sharing interface |
| **Tilt Dashboard** | [http://localhost:10355](http://localhost:10355) | Build logs, status, and control |
| **Jaeger UI** | [http://localhost:16686](http://localhost:16686) | Distributed tracing viewer |
| **RabbitMQ Manager** | [http://localhost:15672](http://localhost:15672) | Queue monitoring (User: `guest`/`guest`) |
| **API Gateway** | [http://localhost:8081](http://localhost:8081) | Direct API access (for debugging) |


## Offline / Restricted Network Mode

If you are working in an environment with restricted internet access (e.g., VPNs, corporate firewalls) where Docker containers cannot reach external APIs:

### 1. OSRM (Routing) Fallback
The `trip-service` attempts to reach `http://router.project-osrm.org`. If this times out:
*   Open `services/trip-service/internal/infrastructure/grpc/grpc_handler.go`.
*   Locate the `PreviewTrip` function.
*   Change the `GetRoute` call's last argument to `false`:
    ```go
    // false = disable external OSRM API and use local mock calculation
    route, err := h.service.GetRoute(ctx, pickupCoord, destinationCoord, false)
    ```

### 2. Stripe (Payment) Fallback
The `payment-service` attempts to reach Stripe APIs. If this fails:
*   The code includes a fallback to return a mock session ID if the Stripe API call returns an error.
*   You will see a log: `Error creating Stripe session... Returning MOCK session.`

## Troubleshooting Common Issues

### "CrashLoopBackOff" on Startup
*   **Cause**: Usually missing environment variables or DB connectivity.
*   **Fix**: Check `infra/development/k8s/secrets.yaml`. Ensure `MONGODB_URI` aligns with the local service (`mongodb://mongodb:27017/ride-sharing`).

### "ImagePullBackOff" or "ErrImageNeverPull"
*   **Cause**: Tilt failed to load the image into Kind, or Kind cannot pull a public image.
*   **Fix**:
    *   Check disk space (see above).
    *   Manually pull/load:
        ```bash
        docker pull mongo:5.0
        kind load docker-image mongo:5.0 --name ride-sharing
        ```

### Frontend Not Updating
*   **Cause**: Next.js cache or build failure.
*   **Fix**: Trigger a rebuild in the Tilt UI, or delete `web/.next` locally and restart Tilt.

## useful Commands

*   **View Pods**: `kubectl get pods`
*   **View Logs**: `kubectl logs -f deployment/<service-name>`
*   **Restart Service**: `kubectl rollout restart deployment/<service-name>`



killall tilt 