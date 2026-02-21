# gymctl

**Jerry's Gym** — a hands-on training platform for Docker and Kubernetes skills.

Each exercise drops you into a broken environment. Jerry made a mess. You fix it. `gymctl` sets up the environment, validates your solution, and tracks your score across 49 exercises.

---

## Installation

### Option 1: Download binary (recommended)

```bash
# Linux (amd64)
curl -L https://github.com/shart-cloud/gymctl/releases/latest/download/gymctl-linux-amd64 -o gymctl
chmod +x gymctl
sudo mv gymctl /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/shart-cloud/gymctl/releases/latest/download/gymctl-darwin-arm64 -o gymctl
chmod +x gymctl
sudo mv gymctl /usr/local/bin/
```

Binaries available for: `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`, `windows-amd64`.

### Option 2: Run in Docker

```bash
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v gymctl-data:/home/gymuser/.gym \
  ghcr.io/shart-cloud/gymctl:latest
```

### Option 3: Build from source

```bash
git clone https://github.com/shart-cloud/gymctl.git
cd gymctl
go build -o gymctl ./cmd/gymctl
```

Requires Go 1.24+.

---

## Prerequisites

- **Docker track:** Docker Engine running locally
- **Kubernetes track:** Docker + [kind](https://kind.sigs.k8s.io/) (gymctl creates the cluster automatically)
- `kubectl` in your PATH for Kubernetes exercises

---

## Quick Start

![gymctl demo](demo-simple.gif)

```bash
# See all available exercises
gymctl list

# Filter by track
gymctl list --track k8s-fundamentals

# Start an exercise
gymctl start jerry-broken-service

# Check your solution
gymctl check

# Get a hint (costs points)
gymctl hint

# Show your overall progress
gymctl status
```

---

## Commands

| Command | Description |
|---|---|
| `gymctl list` | List all exercises. Filter with `--track`, `--difficulty` |
| `gymctl start <name>` | Set up the exercise environment and display the brief |
| `gymctl check [name]` | Validate your solution against all checks |
| `gymctl hint [name]` | Reveal the next hint (deducts points) |
| `gymctl status` | Show overall progress and scores across all exercises |
| `gymctl stop [name]` | Tear down the exercise environment |
| `gymctl reset [name]` | Reset to the initial broken state |
| `gymctl watch [name]` | Auto-run checks on file save (Docker) or on interval (Kubernetes) |
| `gymctl info [name]` | Show exercise details: description, checks, references |
| `gymctl next` | Show your next recommended exercise from GRIM-9 |
| `gymctl shell` | Open the interactive exercise browser TUI |
| `gymctl diagnose` | Check Docker/Kubernetes/network prerequisites |
| `gymctl recover [name]` | Recover from corrupted state |

---

## Tracks and Exercises

Exercises are grouped into tracks. Progress through a track sequentially, or jump around.

### Docker Fundamentals (`docker-fundamentals`)

Six exercises covering Docker best practices: image size, layer caching, security, healthchecks, and networking.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-root-container` | Jerry's Root Container | intermediate |
| 2 | `jerry-fat-image` | Jerry's Fat Image | intermediate |
| 3 | `jerry-broken-layers` | Jerry's Broken Layer Cache | beginner |
| 4 | `jerry-no-healthcheck` | Jerry's Missing Healthcheck | beginner |
| 5 | `jerry-lost-connection` | Jerry's Lost Connection | intermediate |
| 6 | `jerry-broken-syntax` | Jerry's Broken Dockerfile | beginner |

---

### Kubernetes Fundamentals (`k8s-fundamentals`)

Core Kubernetes concepts: resource limits, Services, ConfigMaps, probes, namespaces, rollouts.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-forgot-resources` | Jerry's Resource Hog | intermediate |
| 2 | `jerry-broken-service` | Jerry's Service Can't Find Pods | beginner |
| 3 | `jerry-missing-configmap` | Jerry's Missing ConfigMap | intermediate |
| 4 | `jerry-probe-failures` | Jerry's Failing Health Checks | intermediate |
| 5 | `jerry-wrong-namespace` | Jerry's Cross-Namespace Confusion | intermediate |
| 6 | `jerry-rollout-stuck` | Jerry's Rollout Is Stuck | intermediate |

---

### Kubernetes Networking (`k8s-networking`)

NodePorts, Ingress, NetworkPolicy, and Gateway API.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-nodeport-mystery` | Jerry's NodePort Mystery | intermediate |
| 2 | `jerry-broken-ingress-host` | Jerry's Broken Ingress Host | beginner |
| 3 | `jerry-networkpolicy-dns` | Jerry Blocked DNS With NetworkPolicy | advanced |
| 4 | `jerry-gateway-route-detached` | Jerry's Gateway Route Detached | advanced |

---

### Kubernetes Troubleshooting (`k8s-troubleshooting`)

Debugging broken clusters, nodes, pods, and workloads.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-pod-unschedulable-taint` | Jerry Tainted Every Node | intermediate |
| 2 | `jerry-coredns-loop` | Jerry Misconfigured CoreDNS | advanced |
| 3 | `jerry-node-notready-kubelet` | Jerry's Node Went NotReady | advanced |
| 4 | `jerry-container-log-mystery` | Jerry's Multi-Container Log Mystery | intermediate |
| 5 | `jerry-resource-hog-hunt` | Jerry's Resource Hog Hunt | advanced |
| 6 | `jerry-hpa-not-scaling` | Jerry's HPA Shows Unknown Metrics | intermediate |
| 7 | `jerry-crd-operator-broken` | Jerry's Application Never Syncs | intermediate |
| 8 | `jerry-node-drain-pdb-blocked` | Jerry's Node Drain Blocked by PDB | advanced |
| 9 | `jerry-wrong-cni-config` | Jerry's Wrong CNI Config | advanced |

---

### Kubernetes Administration (`k8s-admin`)

RBAC, etcd, static pods, PSA, scheduling, kubeconfig.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-rbac-denied` | Jerry's RBAC Access Denied | intermediate |
| 2 | `jerry-etcd-snapshot-missing` | Jerry Forgot The etcd Snapshot | advanced |
| 3 | `jerry-kubeconfig-context-confusion` | Jerry's Kubeconfig Context Confusion | beginner |
| 4 | `jerry-static-pod-misconfigured` | Jerry Broke a Control-Plane Static Pod | advanced |
| 5 | `jerry-psa-violation` | Jerry's Pod Security Admission Violation | intermediate |
| 6 | `jerry-pod-wont-spread` | Jerry's Pods Won't Spread | intermediate |
| 7 | `jerry-affinity-mismatch` | Jerry's Affinity Mismatch | intermediate |

---

### Kubernetes Storage (`k8s-storage`)

PVCs, StorageClasses, StatefulSets, PV lifecycle, and volume types.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-pvc-pending-storageclass` | Jerry's PVC Stuck Pending | intermediate |
| 2 | `jerry-reclaim-policy-surprise` | Jerry's Reclaim Policy Surprise | advanced |
| 3 | `jerry-volume-mount-wrong-path` | Jerry's Volume Mount Goes Nowhere | intermediate |
| 4 | `jerry-access-mode-conflict` | Jerry's PVC Stuck on Access Mode | intermediate |
| 5 | `jerry-statefulset-pvc-orphan` | Jerry's StatefulSet Orphaned Its PVCs | advanced |
| 6 | `jerry-pv-released-not-available` | Jerry's Released PV Won't Rebind | advanced |
| 7 | `jerry-ephemeral-volume-confusion` | Jerry's emptyDir Isn't Persistent | intermediate |
| 8 | `jerry-storageclass-default-conflict` | Jerry's Cluster Has Two Default StorageClasses | intermediate |
| 9 | `jerry-pvc-resize-stuck` | Jerry's PVC Resize Is Rejected | intermediate |
| 10 | `jerry-subpath-mount-breaks-symlink` | Jerry's Config Update Isn't Reaching the Container | advanced |
| 11 | `jerry-projected-volume-misconfigured` | Jerry's Projected Volume Won't Mount | advanced |
| 12 | `jerry-node-local-pv-affinity-broken` | Jerry's Local PV Is Stuck on a Cordoned Node | advanced |

---

### Kubernetes Observability (`k8s-observability`)

Prometheus, metrics exporters, and scrape config.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-exporter-missing-metrics` | Jerry's Exporter Missing Metrics | intermediate |
| 2 | `jerry-prometheus-target-down` | Jerry's Prometheus Target Down | advanced |

---

### Kubernetes Ops (`k8s-ops`)

GitOps drift and configuration management.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-kustomize-drift` | Jerry's Kustomize Drift | intermediate |

---

### GitOps (`k8s-gitops`)

ArgoCD and GitOps workflows.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-argo-out-of-sync` | Jerry's Argo App Out Of Sync | advanced |

---

### DevSecOps (`devsecops`)

CI/CD pipelines and automation.

| # | Exercise | Title | Difficulty |
|---|---|---|---|
| 1 | `jerry-ci-pipeline-fix` | Jerry's CI Pipeline Is Broken | intermediate |

---

## Scoring and Hints

Each exercise starts at **100 points**. You lose points every time you use a hint.

```bash
gymctl hint                # reveal next hint, deducts points
gymctl hint --reveal-all   # show all remaining hints at once
```

Hints are tiered — early hints are subtle nudges, later ones are more direct. Use `gymctl check` as many times as you want; checks don't cost points.

Your score for a completed exercise = 100 − sum of hint costs used.

---

## Workflow Example

```bash
# Start the exercise — gymctl sets up the broken environment
gymctl start jerry-broken-service

# Poke around, try to fix the problem
kubectl get svc
kubectl describe svc jerry-app

# Check your work
gymctl check

# Stuck? Get a hint
gymctl hint

# Auto-check on every change
gymctl watch

# Done — view your score
gymctl status
```

---

## Exercise Structure

Each exercise directory contains:

```
task.yaml          # Exercise definition, checks, hints metadata
setup/             # Kubernetes manifests applied to create the broken state
hints/
  hint-1.md        # Progressive hints
  hint-2.md
  hint-3.md
check/             # Optional check scripts (some exercises)
```

Exercise progress is saved to `~/.gym/progress.yaml`.

---

## Workstation Container

For a full isolated environment with all tooling pre-installed (kubectl, helm, kind, k9s, jq, zsh):

```bash
docker build -f Dockerfile.workstation -t gymctl-workstation .
docker run -it --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  gymctl-workstation
```

Includes: kubectl, helm, kind, kubectx/kubens, docker CLI, jq, yq, vim, zsh with aliases (`k`, `kgp`, `kgs`, `kgd`).

---

## Demo Recordings

This repository includes VHS tape files for creating terminal demonstrations:

- `demo-simple.tape` - Basic help and list commands demo
- `demo-quickstart.tape` - Full quick start workflow
- `demo-tui.tape` - Interactive TUI demonstration
- `demo-workflow.tape` - Complete exercise workflow
- `demo-list.tape` - Exercise browsing and filtering

To regenerate the recordings (requires [VHS](https://github.com/charmbracelet/vhs)):

```bash
vhs demo-simple.tape
```
