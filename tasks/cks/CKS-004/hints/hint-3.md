# CKS-004 — Hint 3 (cost 50)

Full walkthrough on `cp-1`. This is what `solution/solve.sh` runs (from the host
via `docker exec`); here it is as you would type it on the node.

```bash
docker exec -it jerry-gym-control-plane bash   # you are now root on cp-1

REF=/root/checksums/SHA512SUMS
REPORT=/root/binary-verification.txt

# --- restore the tampered kubelet from the stashed known-good copy ---
# swap the inode (don't write the busy, running binary in place -> ETXTBSY)
cp -f /root/pristine/kubelet /usr/bin/kubelet.restore
mv -f /usr/bin/kubelet.restore /usr/bin/kubelet
# kubectl was never tampered; restoring it too is harmless/idempotent
cp -f /root/pristine/kubectl /usr/bin/kubectl.restore
mv -f /usr/bin/kubectl.restore /usr/bin/kubectl

# --- restart the affected service so it execs the restored binary ---
systemctl restart kubelet
for i in $(seq 1 30); do
  [ "$(systemctl is-active kubelet)" = active ] && break
  sleep 1
done

# --- write the report in the required format (now both should be OK) ---
: > "$REPORT"
for b in kubelet kubectl; do
  path="/usr/bin/$b"
  exp=$(grep " $path\$" "$REF" | awk '{print $1}')
  got=$(sha512sum "$path" | awk '{print $1}')
  if [ "$exp" = "$got" ]; then st=OK; else st=MISMATCH; fi
  printf '%s: expected=%s computed=%s status=%s\n' "$b" "$exp" "$got" "$st" >> "$REPORT"
done
cat "$REPORT"
```

Confirm:

```bash
sha512sum -c /root/checksums/SHA512SUMS     # /usr/bin/kubelet: OK  /usr/bin/kubectl: OK
systemctl is-active kubelet                  # active
kubectl get nodes                            # Ready
```

The full, idempotent script is `solution/solve.sh`, and `solution/README.md`
explains why byte-level integrity matters and the real-node re-download flow
(`dl.k8s.io` + published `.sha512`) that this offline gym stands in for.
