Start with an authorization check using impersonation:

```bash
kubectl auth can-i list pods -n default --as=system:serviceaccount:default:jerry-dev
kubectl auth can-i get pods/log -n default --as=system:serviceaccount:default:jerry-dev
```

Then inspect the current role and binding:

```bash
kubectl get role jerry-readonly -n default -o yaml
kubectl get rolebinding jerry-readonly-binding -n default -o yaml
```
