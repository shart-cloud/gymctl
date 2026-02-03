# GymCTL Docker Usage

## Quick Start

### Using Docker Run

```bash
# Basic interactive mode
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v gymctl-data:/home/gymuser/.gym \
  gymctl:latest

# Run a specific command
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v gymctl-data:/home/gymuser/.gym \
  gymctl:latest list

# Start an exercise
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v gymctl-data:/home/gymuser/.gym \
  gymctl:latest start jerry-root-container
```

### Using Docker Compose

```bash
# Build and start interactive session
docker-compose up -d --build
docker-compose exec gymctl bash

# Inside the container
gymctl list
gymctl start jerry-root-container
gymctl check
```

## Building the Image

```bash
# Build locally
docker build -t gymctl:latest .

# Multi-platform build (for ARM64 and AMD64)
docker buildx build --platform linux/amd64,linux/arm64 -t gymctl:latest .
```

## Volume Mounts

### Required Volumes

| Volume | Purpose | Required |
|--------|---------|----------|
| `/var/run/docker.sock` | Access host Docker daemon | Yes for Docker exercises |
| `/home/gymuser/.gym` | Persist progress and work | Recommended |

### Example with Named Volume

```bash
# Create named volume
docker volume create gymctl-data

# Run with named volume
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v gymctl-data:/home/gymuser/.gym \
  gymctl:latest
```

### Example with Bind Mount

```bash
# Use local directory for persistence
mkdir -p ~/.gym-docker

docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ~/.gym-docker:/home/gymuser/.gym \
  gymctl:latest
```

## Docker-in-Docker Mode

For fully isolated Docker exercises, use the DinD profile:

```bash
# Start with Docker-in-Docker
docker-compose --profile dind up -d

# Connect to DinD from gymctl
docker-compose exec gymctl bash
export DOCKER_HOST=tcp://dind:2375
gymctl start jerry-root-container
```

## Security Considerations

### Running as Non-Root

The container runs as user `gymuser` (UID 1000) by default. To match your host user:

```bash
# Build with your UID/GID
docker build \
  --build-arg UID=$(id -u) \
  --build-arg GID=$(id -g) \
  -t gymctl:custom .

# Run with user mapping
docker run -it --rm \
  --user $(id -u):$(id -g) \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v ~/.gym:/home/gymuser/.gym \
  gymctl:custom
```

### Docker Socket Permissions

If you get permission errors accessing Docker socket:

```bash
# Option 1: Add socket group
docker run -it --rm \
  --group-add $(stat -c '%g' /var/run/docker.sock) \
  -v /var/run/docker.sock:/var/run/docker.sock \
  gymctl:latest

# Option 2: Run as root (not recommended)
docker run -it --rm \
  --user root \
  -v /var/run/docker.sock:/var/run/docker.sock \
  gymctl:latest
```

## Kubernetes Exercises

For Kubernetes exercises, you have several options:

### Option 1: Use Kind (Kubernetes in Docker)

```bash
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v gymctl-data:/home/gymuser/.gym \
  --network host \
  gymctl:latest
```

### Option 2: Mount kubeconfig

```bash
docker run -it --rm \
  -v ~/.kube/config:/home/gymuser/.gym/kubeconfig:ro \
  -e KUBECONFIG=/home/gymuser/.gym/kubeconfig \
  gymctl:latest
```

## Aliases for Convenience

Add to your shell configuration:

```bash
# Bash/Zsh alias
alias gymctl='docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v gymctl-data:/home/gymuser/.gym \
  gymctl:latest'

# Usage
gymctl list
gymctl start jerry-root-container
gymctl check
```

## Troubleshooting

### Permission Denied on Docker Socket

```bash
# Check socket permissions
ls -la /var/run/docker.sock

# Add socket group
docker run --group-add $(stat -c '%g' /var/run/docker.sock) ...
```

### Cannot Start Exercise

```bash
# Ensure Docker socket is mounted
docker run -v /var/run/docker.sock:/var/run/docker.sock ...

# Check Docker access inside container
docker run ... gymctl:latest docker version
```

### Lost Progress

```bash
# Always use a volume for persistence
docker volume create gymctl-data
docker run -v gymctl-data:/home/gymuser/.gym ...

# Backup your progress
docker run -v gymctl-data:/data --rm alpine tar czf /backup.tar.gz /data
```

## Publishing to Registry

```bash
# Tag for Docker Hub
docker tag gymctl:latest yourusername/gymctl:latest
docker push yourusername/gymctl:latest

# Tag for GitHub Container Registry
docker tag gymctl:latest ghcr.io/yourusername/gymctl:latest
docker push ghcr.io/yourusername/gymctl:latest

# Users can then run
docker run -it ghcr.io/yourusername/gymctl:latest
```