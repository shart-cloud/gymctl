Required commands to include in the workflow:

```yaml
ruff check .
bandit -r .
kubeconform -summary k8s/
```

And ensure build has:

```yaml
needs: [quality]
```
