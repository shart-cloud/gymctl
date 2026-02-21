Start by inspecting the current state:

```bash
kubectl get pv
kubectl get pvc
kubectl describe pvc jerry-new-claim
```

Find the PV in Released state:

```bash
kubectl get pv -l jerry.gym/state=released
kubectl describe pv <pv-name>
```

Notice the `Claim` field in the describe output still references `jerry-original-claim`
even though that claim no longer exists. This is the stale `claimRef` that is blocking
the new claim from binding.
