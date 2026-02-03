# Deployment Guide: Ride-Sharing Microservices

This guide outlines how to deploy the Ride-Sharing application stack using free-tier services.

## Architecture

- **Frontend**: Next.js (Deployed on Vercel or Netlify)
- **Backend**: Go Microservices (Deployed on Render)
    - `api-gateway`
    - `trip-service`
    - `driver-service`
    - `payment-service`
- **Database**: MongoDB (MongoDB Atlas)
- **Message Broker**: RabbitMQ (CloudAMQP)
- **Tracing**: Jaeger (Grafana Cloud or Honeycomb)

---

## 1. Prerequisites (Cloud Services)

Before deploying the code, set up your external resources.

### 1.1 MongoDB Atlas (Database)
1. Create a free account at [MongoDB Atlas](https://www.mongodb.com/cloud/atlas).
2. Create a new Cluster (Shared, Free Tier).
3. Create a Database User (username/password).
4. Allow Access from Anywhere (`0.0.0.0/0`) in IP Whitelist (for Render dynamic IPs).
5. Get the Connection String (driver: Go). It looks like:
   `mongodb+srv://<user>:<password>@cluster0.abcde.mongodb.net/?retryWrites=true&w=majority`
   **Save this as `MONGODB_URI`**.

### 1.2 CloudAMQP (RabbitMQ)
1. Create a free account at [CloudAMQP](https://www.cloudamqp.com/).
2. Create a new Instance (Plan: Little Lemur - Free).
3. Select a region close to your Render region.
4. Copy the "AMQP URL". It looks like:
   `amqp://user:pass@host/vhost`
   **Save this as `RABBITMQ_URI`**.

### 1.3 Grafana Cloud (Tracing/Jaeger)
1. Create a free account at [Grafana Cloud](https://grafana.com/).
2. In the "OTLP/Jaeger" section, find your Endpoint and Credentials.
3. You will need your **OTLP Endpoint**, **Instance ID**, and **Access Token**.
4.  Construct your variables:
    *   **OTEL_EXPORTER_OTLP_ENDPOINT**: `https://<InstanceID>:<Token>@otlp-gateway-prod-us-east-0.grafana.net/otlp/v1/traces`
    *   *Alternatively*, split them:
        *   Endpoint: `https://otlp-gateway-prod-us-east-0.grafana.net/otlp`
        *   Headers: `Authorization=Basic <base64-encoded-creds>`

---

## 2. Backend Deployment (Render)

We will use the included `render.yaml` Blueprint to deploy all 4 services at once.

1. Push your latest code (including `render.yaml`) to GitHub.
2. Sign up/Login to [Render](https://render.com/).
3. Go to **Blueprints** and click **New Blueprint Instance**.
4. Connect your GitHub repository.
5. Render will detect `render.yaml`.
6. You will be prompted to enter the **Environment Variables** (shared secrets):
    - `RABBITMQ_URI`: Paste your CloudAMQP URL.
    - `MONGODB_URI`: Paste your Atlas Connection String.
    - `OTEL_EXPORTER_OTLP_ENDPOINT`: Paste your full Grafana OTLP URL (with user:pass).
    - `OTEL_EXPORTER_OTLP_HEADERS`: (Optional) Paste any required headers.
    - `STRIPE_WEBHOOK_KEY`: Generate a random string or get from Stripe Dashboard later.
    - `STRIPE_SECRET_KEY`: Get from Stripe Dashboard (for Payment Service).
7. Click **Apply**.
8. Render will build and deploy:
    - `api-gateway`
    - `trip-service`
    - `driver-service`
    - `payment-service`

**Note:** The initial build might take a few minutes. Ensure all services show "Live".

### Validating Backend
Once deployed, check the `api-gateway` URL (e.g., `https://api-gateway-xxxx.onrender.com`).
Test the health or WebSocket endpoints.

---

## 3. Frontend Deployment (Vercel)

1. Sign up/Login to [Vercel](https://vercel.com/).
2. Click **Add New...** -> **Project**.
3. Import your GitHub repository.
4. Set the **Root Directory** to `web` (Edit -> Select `web` folder).
5. Open **Environment Variables**:
    - `NEXT_PUBLIC_API_URL`: Set this to your **Render API Gateway URL** (e.g., `https://api-gateway-xxxx.onrender.com`).
      *Note: No trailing slash.*
    - `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`: Your Stripe Public Key.
6. Click **Deploy**.

---

## 4. Connecting the Dots

1. **Stripe Webhook**:
   - In Stripe Dashboard, add a Webhook pointing to `https://api-gateway-xxxx.onrender.com/webhook/stripe`.
   - Events: `checkout.session.completed`.
   - Copy the Signing Secret and update the `STRIPE_WEBHOOK_KEY` env var in Render (Dashboard -> api-gateway -> Environment).

2. **OSRM (Routing)**:
   - The `trip-service` uses `http://router.project-osrm.org` by default. This is a public demo server and may be throttled. For production, consider hosting your own OSRM or using a paid routing API.

## Troubleshooting

- **Service failing to start**: Check the Logs in Render.
- **Connection Refused**: Ensure the `_URL` env vars in `api-gateway` match the internal service names (`trip-service:9093`, etc.) defined in `render.yaml`.
- **Database Error**: Check IP Whitelist in MongoDB Atlas.
