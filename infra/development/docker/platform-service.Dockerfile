FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/platform-service
RUN CGO_ENABLED=0 GOOS=linux go build -o platform-service ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/infra/sql/001_schema.sql ./001_schema.sql
COPY --from=builder /app/services/platform-service/platform-service .
# Do not override in PaaS with infra/sql/... — that path is not in the image.
ENV SQL_SCHEMA_PATH=/root/001_schema.sql
CMD ["./platform-service"]
