Fix the Service selector to match the deployment label:

```bash
kubectl patch svc metrics-exporter --type merge -p '{"spec":{"selector":{"app":"metrics-target"}}}'
```

Then confirm target health:

```bash
kubectl run prom-api --rm -it --restart=Never --image=busybox:1.36 -- wget -qO- http://prometheus:9090/api/v1/targets
```
