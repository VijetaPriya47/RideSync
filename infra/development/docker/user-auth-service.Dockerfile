FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
WORKDIR /app/services/user-auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -o user-auth-service ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/infra/sql/001_schema.sql ./001_schema.sql
COPY --from=builder /app/services/user-auth-service/user-auth-service .
ENV SQL_SCHEMA_PATH=/root/001_schema.sql
CMD ["./user-auth-service"]
