The annotation that marks a StorageClass as default is:
`storageclass.kubernetes.io/is-default-class: "true"`

Remove it from one of the two classes. For this lab, keep `standard-local` as the default
and remove the annotation from `fast-local`.

The trailing `-` removes an annotation:

```bash
kubectl annotate storageclass fast-local \
  storageclass.kubernetes.io/is-default-class-

kubectl get storageclass
```

Only one class should now show `(default)`.
