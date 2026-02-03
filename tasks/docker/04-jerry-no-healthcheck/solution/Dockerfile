# Jerry's Python app - with health monitoring
FROM python:3.11-alpine

WORKDIR /app

# Install curl for health checks
RUN apk add --no-cache curl

COPY server.py .

# Health check configuration
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

EXPOSE 8080
CMD ["python", "server.py"]
