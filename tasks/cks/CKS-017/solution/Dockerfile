# Multi-stage build: the toolchain stays in the builder, only the binary ships.

# ── Stage 1: builder ─────────────────────────────────────────────────────────
# The full Go toolchain lives here and NEVER reaches the final image.
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
# CGO_ENABLED=0 produces a fully static binary with no libc dependency, so it
# can run on a base image that ships no shared libraries at all.
# -ldflags "-s -w" strips debug info to shave a bit more off.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app .

# ── Stage 2: final ───────────────────────────────────────────────────────────
# distroless/static has NO shell, NO package manager, NO apt, NO libc — just the
# minimum to run a static binary. Nothing to exec into, almost nothing to CVE.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app /app
# nonroot = uid 65532, baked into distroless. The final image never runs as root.
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
