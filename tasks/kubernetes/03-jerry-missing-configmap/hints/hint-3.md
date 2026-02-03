# Hint 3: Complete solution

Create the ConfigMap with this command:

```bash
kubectl create configmap app-config \
  --from-literal=DATABASE_HOST=postgres.database.svc.cluster.local \
  --from-literal=DATABASE_PORT=5432 \
  --from-literal=APP_MODE=production
```

Or create a YAML file:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  DATABASE_HOST: postgres.database.svc.cluster.local
  DATABASE_PORT: "5432"
  APP_MODE: production
```

Then apply it:
```bash
kubectl apply -f configmap.yaml
```