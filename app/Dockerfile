FROM golang:1.24-alpine AS builder
RUN mkdir app
WORKDIR /app
COPY ./app .
RUN go mod download && go mod verify
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
RUN CGO_ENABLED=0 go build -o main .
RUN chmod +x ./main
EXPOSE 8080
CMD ["./main", "serve"]
