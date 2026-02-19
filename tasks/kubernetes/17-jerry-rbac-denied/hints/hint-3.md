Update the Role to least-privilege read access:

```yaml
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
```

Edit the role and verify:

```bash
kubectl edit role jerry-readonly -n default
kubectl auth can-i list pods -n default --as=system:serviceaccount:default:jerry-dev
kubectl auth can-i get pods/log -n default --as=system:serviceaccount:default:jerry-dev
kubectl auth can-i delete pods -n default --as=system:serviceaccount:default:jerry-dev
```
