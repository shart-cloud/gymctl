# Hint 2: Use controlledResources to Limit VPA Scope

VPA's `resourcePolicy.containerPolicies` has a `controlledResources` field. Set it to `[memory]` only:

```bash
kubectl explain vpa.spec.resourcePolicy.containerPolicies.controlledResources
```

Edit the VPA:

```bash
kubectl edit vpa jerry-app-vpa
```

Find the `containerPolicies` section and add or update `controlledResources`:

```yaml
containerPolicies:
- containerName: app
  controlledResources:
  - memory    # VPA only right-sizes memory — leave CPU alone
```

With this change, VPA will never modify `requests.cpu` — HPA can own that axis cleanly.
