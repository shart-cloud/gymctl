# CKS-004 — Hint 1 (free)

This is a **binary integrity** problem, not a Kubernetes API problem. Everything
happens on the control-plane node's filesystem.

```bash
# cp-1 is the kind control-plane container
docker exec -it jerry-gym-control-plane bash

# the trusted reference (known-good hashes for this node)
cat /root/checksums/SHA512SUMS

# compute the live hashes and let sha512sum compare them for you
sha512sum -c /root/checksums/SHA512SUMS
```

`sha512sum -c` prints `OK` or `FAILED` for each path. **Exactly one** of
`/usr/bin/kubelet` and `/usr/bin/kubectl` will say `FAILED` — that is the
tampered binary (someone appended bytes to it, so it still runs but its hash
changed).

Your job: write a report at `/root/binary-verification.txt`, restore the bad
binary from a known-good copy, restart the service that runs it, and confirm the
node is `Ready`. A pristine copy of each binary is stashed under `/root/pristine/`.
