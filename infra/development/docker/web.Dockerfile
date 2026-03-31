# Stage 1: Build
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies based on the preferred package manager
COPY web/package*.json ./
RUN npm install

# Copy source files
COPY web ./

# Ensure the standalone output is configured in next.config.ts (we already did this)
RUN npm run build

# Stage 2: Runner
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV production

# Only copy necessary files from builder
# Next.js standalone mode requires these three things:
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

EXPOSE 3000

ENV PORT 3000
ENV HOSTNAME "0.0.0.0"

CMD ["node", "server.js"]