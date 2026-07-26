# Multi-stage Dockerfile for the Drawo Go backend.

FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o drawo .

FROM alpine:3.20

RUN apk --no-cache add ca-certificates wget tzdata

WORKDIR /app

COPY --from=builder /src/drawo ./drawo
COPY --from=builder /src/migrations ./migrations
COPY --from=builder /src/locales ./locales

EXPOSE 8080

CMD ["./drawo", "serve"]
