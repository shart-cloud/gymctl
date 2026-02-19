Patch the Application and rotate the bad token:

```bash
kubectl patch applications.argoproj.io jerry-portfolio -n argocd --type merge -p '{"spec":{"source":{"path":"k8s/overlays/local"},"destination":{"namespace":"portfolio-dev"}}}'

kubectl create namespace portfolio-dev
kubectl patch secret repo-creds -n argocd --type merge -p '{"stringData":{"password":"new-token-value"}}'
```
