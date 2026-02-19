Set `EMIT_GUESTBOOK=true` on the deployment and wait for rollout:

```bash
kubectl set env deployment/jerry-exporter EMIT_GUESTBOOK=true
kubectl rollout status deployment/jerry-exporter
```
