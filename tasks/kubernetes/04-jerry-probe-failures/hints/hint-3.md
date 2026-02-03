# Hint 3: Complete solution

Update the deployment with proper probe configuration:

```yaml
livenessProbe:
  exec:
    command:
    - cat
    - /tmp/healthy
  initialDelaySeconds: 70  # Wait for startup
  periodSeconds: 10
  timeoutSeconds: 1
  failureThreshold: 3      # Allow some failures

readinessProbe:
  exec:
    command:
    - cat
    - /tmp/healthy
  initialDelaySeconds: 65  # Slightly before liveness
  periodSeconds: 5
  timeoutSeconds: 1
  failureThreshold: 1
```

Apply the changes:
```bash
kubectl apply -f deployment.yaml
```