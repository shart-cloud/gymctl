# Hint 3: The solution

Edit the Service to use the correct selector:

```yaml
spec:
  selector:
    app: jerry-web
```

Remove the `environment: production` selector since pods don't have that label.

Apply the changes:
```bash
kubectl apply -f service.yaml
```