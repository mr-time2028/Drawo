# Multi-stage Dockerfile for the Drawo Go backend.

FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

# Cache dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o drawo .

# Final minimal image.
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /app/drawo .

EXPOSE 8080

CMD ["./drawo", "serve"]
