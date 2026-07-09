# CKS-013 — Hint 1 (free)

Get the lay of the land:

```bash
kubectl get pods -n gatekeeper-system            # the webhook is already running
kubectl get constrainttemplates                  # empty — nothing is defined
kubectl get constraints                           # empty — nothing is enforced
kubectl -n app get pod legacy-app -o yaml | grep -A3 securityContext
```

Gatekeeper enforces nothing until you give it policy. Two objects per rule:

- A **ConstraintTemplate** holds the Rego and *defines a new kind*.
- A **Constraint** (a resource of that new kind) *turns the rule on* and says
  **where** it applies.

You need two rules: deny privileged containers (everywhere but `kube-system`),
and require an `owner` label (only in `app` and `staging`). Keep the Rego
generic — do the "where" with the Constraint's `match` block.

Docs: <https://open-policy-agent.github.io/gatekeeper/website/docs/howto/>
