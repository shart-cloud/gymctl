The `app` container expects `APP_MODE=prod`.
Patch the Deployment and let it roll out:

```bash
kubectl set env deployment/jerry-log-mystery APP_MODE=prod
kubectl rollout status deployment/jerry-log-mystery
```

Re-check app logs after the new pod starts.
