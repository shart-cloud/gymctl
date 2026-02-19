# Hint 3: The Fix

Add either podAntiAffinity OR topologySpreadConstraints to the deployment spec.

**Option A - podAntiAffinity:**
```yaml
spec:
  template:
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - jerry-ha-app
              topologyKey: kubernetes.io/hostname
```

**Option B - topologySpreadConstraints:**
```yaml
spec:
  template:
    spec:
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: kubernetes.io/hostname
        whenUnsatisfiable: ScheduleAnyway
        labelSelector:
          matchLabels:
            app: jerry-ha-app
```