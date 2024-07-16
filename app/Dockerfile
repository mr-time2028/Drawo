FROM golang:1.21-alpine AS builder
RUN mkdir app
WORKDIR /app
COPY ./app .
RUN go mod download && go mod verify
RUN CGO_ENABLED=0 go build -o main .
RUN chmod +x ./main


FROM alpine:latest AS runner
RUN mkdir /app
WORKDIR /app
COPY --from=builder /app .
EXPOSE 8080
ENTRYPOINT ["./main"]
