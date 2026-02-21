Inspect the current volume configuration:

```bash
kubectl describe pod jerry-uploader | grep -A10 "Volumes:"
kubectl describe pod jerry-uploader | grep -A5 "Mounts:"
```

Look at the volume type. An `emptyDir` volume is created fresh every time the pod
starts — it does not survive pod deletion, even if the pod is on the same node.

Test this yourself:

```bash
kubectl exec jerry-uploader -- sh -c 'echo test > /uploads/test.txt && ls /uploads'
kubectl delete pod jerry-uploader
# The pod is a bare pod with no controller, so it won't auto-restart
# Check what happened to the data (hint: it's gone even if you recreate the pod)
```
