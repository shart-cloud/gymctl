# Hint 2: Probe configuration issues

The app takes 60 seconds to start, but:
- `initialDelaySeconds: 5` - probe starts after only 5 seconds
- `failureThreshold: 1` - fails after just 1 failed check
- No readiness probe to control traffic

Adjust these values to match the app's behavior.