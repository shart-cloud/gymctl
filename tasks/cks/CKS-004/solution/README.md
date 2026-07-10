# CKS-004 — Solution

A supply-chain check compares the cluster's on-disk binaries against a **trusted
list of hashes**. On `cp-1` that list is `/root/checksums/SHA512SUMS`. One
binary — `kubelet` — was tampered by **appending bytes**, which changes its
SHA-512 while leaving a runnable ELF (the loader ignores trailing bytes).
`kubectl` was left untouched and still matches. Your job: prove which one
drifted, restore it, restart the service, and record the result.

## Why byte-level integrity matters

"It still runs" is not evidence a binary is trustworthy. An attacker who can
append (or patch) bytes into `/usr/bin/kubelet` can ship a backdoored kubelet
that still starts and joins the cluster. The only reliable signal is a
**cryptographic hash compared against a source you trust** — here SHA-512, the
algorithm the Kubernetes release process publishes. A single changed byte
flips the hash completely, so tamper detection is exact, not fuzzy.

## The procedure (what a learner does on the node)

```bash
# reach the control-plane node (kind: docker exec; real: ssh)
docker exec -it jerry-gym-control-plane bash

# 1. compute the live hashes
sha512sum /usr/bin/kubelet /usr/bin/kubectl

# 2. compare against the trusted reference
cat /root/checksums/SHA512SUMS
sha512sum -c /root/checksums/SHA512SUMS      # prints "OK"/"FAILED" per line
#   => /usr/bin/kubelet: FAILED   <- the tampered one
#      /usr/bin/kubectl: OK

# 3. write the report in the required format
#    kubelet: expected=<hash> computed=<hash> status=MISMATCH  (before fix)
#    kubectl: expected=<hash> computed=<hash> status=OK

# 4. restore the tampered binary from a KNOWN-GOOD copy, then
# 5. restart kubelet and confirm the node is Ready
systemctl restart kubelet
```

## Restoring: known-good copy vs. re-download

**In this gym (deterministic, offline):** a pristine copy of each binary was
stashed at `/root/pristine/` *before* the tamper, and `SHA512SUMS` was generated
from those same pristine binaries — so the reference is internally consistent
with this node. Restore is just:

```bash
cp -f /root/pristine/kubelet /usr/bin/kubelet.restore
mv -f /usr/bin/kubelet.restore /usr/bin/kubelet   # swap the inode; don't write a busy file
systemctl restart kubelet                          # re-exec the restored binary
```

We swap the directory entry with `mv` instead of writing `/usr/bin/kubelet` in
place because the running kubelet holds that file open for execution — writing it
directly returns **`ETXTBSY` (text file busy)**. The `mv` leaves the old inode to
the running process; the `systemctl restart` then execs the fresh file.

**On a real node (the production story):** you would not have a local pristine
stash — you would **re-download the exact release** and verify it against the
hashes Kubernetes publishes, e.g.:

```bash
VER=$(/usr/bin/kubelet --version | awk '{print $2}')   # e.g. v1.33.1
ARCH=amd64
curl -LO "https://dl.k8s.io/release/${VER}/bin/linux/${ARCH}/kubelet"
curl -LO "https://dl.k8s.io/release/${VER}/bin/linux/${ARCH}/kubelet.sha512"
echo "$(cat kubelet.sha512)  kubelet" | sha512sum --check   # must say: kubelet: OK
install -m 0755 kubelet /usr/bin/kubelet
systemctl restart kubelet
```

Downloading in a check/solve would be non-deterministic (network, version
drift), so the gym uses the stashed pristine copy — but the *skill* is the
re-download-and-verify flow above.

## Why restart kubelet

Replacing the file on disk does not change the **already-running** process — it
keeps executing the inode it started with (the tampered one, until you swapped
it). `systemctl restart kubelet` makes systemd `exec` the new file, so the node
is genuinely running the restored binary. After the restart the node briefly
goes `NotReady` and then returns to `Ready`.

## What the checks assert

- `/root/binary-verification.txt` exists, contains a `kubelet:` and a `kubectl:`
  line each with `status=OK` and `expected == computed`, and no `status=MISMATCH`
  remains.
- **The real gate:** the *live* `sha512sum /usr/bin/kubelet` (recomputed on the
  node) equals its `SHA512SUMS` entry — same for `kubectl`. A report-only edit
  can never pass this; only a genuine restore does.
- `systemctl is-active kubelet` is `active`, and the node is `Ready`.

`solve.sh` performs the restore + restart + report from the host via
`docker exec` (idempotent, re-runnable). The playtest applies it with
`bash solution/solve.sh`.
