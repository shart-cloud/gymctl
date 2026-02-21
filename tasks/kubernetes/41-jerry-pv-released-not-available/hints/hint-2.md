A Released PV has a stale `claimRef` at `.spec.claimRef`. Patch it to null to clear it:

```bash
PV=$(kubectl get pv -l jerry.gym/state=released -o jsonpath='{.items[0].metadata.name}')
kubectl patch pv "$PV" --type=merge -p '{"spec":{"claimRef": null}}'
```

After this, watch the PV phase change:

```bash
kubectl get pv "$PV" -w
```

It should transition from Released → Available within a few seconds.
The waiting PVC should then bind to it automatically.
