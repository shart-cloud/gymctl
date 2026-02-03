# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server main.go

# Stage 2: Production
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .

EXPOSE 8080
CMD ["./server"]
