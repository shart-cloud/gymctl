# Hint 2: The updateMode You Need Is "Initial"

VPA has four update modes:

| Mode | Behavior |
|------|----------|
| `Off` | Recommendations only — never modifies pods |
| `Initial` | Applies recommendations at pod creation only — never evicts running pods |
| `Recreate` | Evicts running pods when requests drift significantly from recommendation |
| `Auto` | Combines Initial + Recreate — the most aggressive mode |

For a single-replica deployment, `Auto` or `Recreate` will cause downtime on every eviction.

Use `Initial` — VPA sets the right resources when pods are (re)created naturally, without forcing evictions.

Edit the VPA:

```bash
kubectl edit vpa jerry-singleton-vpa
```

Change `updateMode: Auto` to `updateMode: Initial`.
