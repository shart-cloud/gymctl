Patch the Service so `targetPort` matches the app's listening port:

```bash
kubectl patch svc jerry-nodeport --type merge -p '{"spec":{"ports":[{"port":80,"targetPort":5678,"nodePort":30080}]}}'
```

Then re-test:

```bash
kubectl run tmp --rm -it --restart=Never --image=busybox:1.36 -- wget -qO- http://jerry-nodeport
```
