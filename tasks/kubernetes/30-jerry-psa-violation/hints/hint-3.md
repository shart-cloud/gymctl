# Hint 3: The Fix

Edit the deployment to replace the broken securityContext:

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: web
        image: nginx:1.20
        ports:
        - containerPort: 8080  # nginx needs non-root port
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: var-run
          mountPath: /var/run
      volumes:
      - name: tmp
        emptyDir: {}
      - name: var-run
        emptyDir: {}
```

Apply with `kubectl apply -f deployment.yaml`