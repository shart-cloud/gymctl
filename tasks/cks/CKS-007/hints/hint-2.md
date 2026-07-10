# CKS-007 — Hint 2 (cost 25)

Edit `/etc/kubernetes/manifests/kube-apiserver.yaml` on `cp-1` so the container
`command:` list ends up with these five properties. Each flag is its own
`    - --flag=value` line, indented to match its siblings.

| Flag | Target |
| --- | --- |
| `--anonymous-auth` | `false` |
| `--enable-admission-plugins` | `NodeRestriction,PodSecurity` (one flag, comma-separated, both present) |
| `--kubelet-certificate-authority` | `/etc/kubernetes/pki/ca.crt` |
| `--tls-min-version` | `VersionTLS12` |
| `--insecure-port` | leave absent (do **not** add it — a nonzero value makes modern kube-apiserver refuse to start) |

## Traps

- **Back up first:** `cp kube-apiserver.yaml kube-apiserver.yaml.bak`. A malformed
  edit crashes the apiserver.
- **One admission flag, not two.** Put `NodeRestriction` and `PodSecurity` on the
  same `--enable-admission-plugins=` line. A second such line will clobber the
  first.
- **`--anonymous-auth=false` and the health probes.** The apiserver's `/livez`
  and `/readyz` probes hit it anonymously. On a real node this can crash-loop the
  pod if the probes get 401 — know why it might, even though on this kind gym you
  won't see it (the manifest doesn't hot-reload here; see `solution/README.md`).
- **Indentation.** A line indented differently from the other `- --flag` entries
  will fail to parse and the pod won't come up.

Because kind does not hot-reload the apiserver static pod, the checks read the
**manifest content** on the node and confirm the cluster is still reachable.
Verify your own edit the same way:

```bash
docker exec jerry-gym-control-plane grep -E -- \
  '--anonymous-auth=|--enable-admission-plugins=|--kubelet-certificate-authority=|--tls-min-version=' \
  /etc/kubernetes/manifests/kube-apiserver.yaml
```
