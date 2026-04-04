# GCP GKE Deployment Guide

This guide provides step-by-step instructions to deploy the Ride Sharing application to Google Kubernetes Engine (GKE).

## Prerequisites
- Google Cloud Platform (GCP) Account
- `gcloud` CLI installed and authenticated
- `kubectl` CLI installed
- Docker installed

## 1. Environment Setup

Set your project variables for easy copy-pasting:

```bash
export PROJECT_ID=<your-gcp-project-id>
export REGION=europe-west1 # or your preferred region
```

Login to Google Cloud:
```bash
gcloud auth login
gcloud config set project $PROJECT_ID
```

## 2. Infrastructure & Secrets

### 2.1 Edit Secrets
Open `infra/production/k8s/secrets.yaml` and replace the placeholder values:
- `<RABBITMQ_URI>`: URI for your RabbitMQ instance (e.g., from CloudAMQP or self-hosted)
- `<MONGODB_URI>`: URI for your MongoDB (e.g., from MongoDB Atlas)
- `<STRIPE_SECRET_KEY>`: Your Stripe Secret Key
- `<STRIPE_WEBHOOK_KEY>`: Your Stripe Webhook Key

**Note**: The OSRM URL is pre-configured to `http://router.project-osrm.org/route/v1`.

## 3. Build and Push Docker Images

Authenticate Docker with GCP Artifact Registry:
```bash
gcloud auth configure-docker ${REGION}-docker.pkg.dev
```

Create the Artifact Registry repository (if not exists):
```bash
gcloud artifacts repositories create ride-sharing \
    --repository-format=docker \
    --location=$REGION \
    --description="Docker repository for Ride Sharing"
```

Build and Push images:

**API Gateway**
```bash
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/api-gateway:latest --platform linux/amd64 -f infra/production/docker/api-gateway.Dockerfile .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/api-gateway:latest
```

**Driver Service**
```bash
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/driver-service:latest --platform linux/amd64 -f infra/production/docker/driver-service.Dockerfile .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/driver-service:latest
```

**Trip Service**
```bash
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/trip-service:latest --platform linux/amd64 -f infra/production/docker/trip-service.Dockerfile .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/trip-service:latest
```

**Payment Service**
```bash
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/payment-service:latest --platform linux/amd64 -f infra/production/docker/payment-service.Dockerfile .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/payment-service:latest
```

**Finance Service** (ledger gRPC + RabbitMQ consumer; requires PostgreSQL and same `RABBITMQ_URI` as payment)
```bash
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/finance-service:latest --platform linux/amd64 -f infra/production/docker/finance-service.Dockerfile .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/finance-service:latest
```

**User Auth Service** (auth gRPC + audit consumer; requires PostgreSQL and RabbitMQ)
```bash
docker build -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/user-auth-service:latest --platform linux/amd64 -f infra/production/docker/user-auth-service.Dockerfile .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/ride-sharing/user-auth-service:latest
```

Production Kubernetes manifests under `infra/production/k8s/` do not yet include Deployments for finance or user-auth; after pushing these images, add workloads and wire `FINANCE_SERVICE_URL` / `USER_AUTH_SERVICE_URL` on the API gateway. See [Finance & RBAC](../features/finance-rbac.md) for ports and environment variables.

## 4. Deploy to GKE

Create a GKE Cluster (if not exists):
```bash
gcloud container clusters create ride-sharing-cluster \
    --zone $REGION-b \
    --num-nodes 3
```

Get credentials for `kubectl`:
```bash
gcloud container clusters get-credentials ride-sharing-cluster --zone $REGION-b
```

Apply Manifests:
```bash
# 1. Configs and Secrets
kubectl apply -f infra/production/k8s/app-config.yaml
kubectl apply -f infra/production/k8s/secrets.yaml

# 2. Infrastructure Services (RabbitMQ, Jaeger) 
# Note: Skip RabbitMQ if using managed service
kubectl apply -f infra/production/k8s/jaeger-deployment.yaml
kubectl apply -f infra/production/k8s/rabbitmq-deployment.yaml

# 3. Microservices
kubectl apply -f infra/production/k8s/api-gateway-deployment.yaml
kubectl apply -f infra/production/k8s/driver-service-deployment.yaml
kubectl apply -f infra/production/k8s/trip-service-deployment.yaml
kubectl apply -f infra/production/k8s/payment-service-deployment.yaml
```

## 5. Verify Deployment

Check the status of your pods:
```bash
kubectl get pods
```

Get the External IP of the API Gateway:
```bash
kubectl get svc api-gateway
```

## 6. Frontend Deployment (Vercel)

For the frontend, deploy the `web/` directory to Vercel. Ensure you set the environment variables in the Vercel dashboard as identified in `.env.local`:
- `NEXT_PUBLIC_API_URL`: `http://<EXTERNAL-IP-OF-API-GATEWAY>:8081`
- `NEXT_PUBLIC_WEBSOCKET_URL`: `ws://<EXTERNAL-IP-OF-API-GATEWAY>:8081/ws`
- `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`: Your Stripe Public Key

## Kubernetes Ecosystem & Resources

- [Kubernetes Concepts Overview](https://kubernetes.io/docs/concepts/overview/)
- [Kubernetes Components - Deep Dive](https://kubernetes.io/docs/concepts/overview/components/)
- [Understanding Kubernetes Deployment YAML](https://spacelift.io/blog/kubernetes-deployment-yaml)
- [Kubernetes Documentary (YouTube)](https://youtu.be/BE77h7dmoQU)

