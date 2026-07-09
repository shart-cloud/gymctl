# CKS-017 — Hint 3 (cost 50)

Near-complete `app/Dockerfile`:

```dockerfile
# Stage 1: build a static binary with the full toolchain.
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
# CGO_ENABLED=0 => static binary with no libc, runnable on distroless/static.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app .

# Stage 2: distroless — no shell, no package manager, no toolchain.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app /app
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
```

Build and sanity-check:

```bash
docker build -t app:secure app
docker run -d -p 8080:8080 app:secure
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/   # -> 200
```

The full manifest is in `solution/Dockerfile`.
