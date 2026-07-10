#!/usr/bin/env bash
#
# CKS-004 solution — verify /usr/bin/kubelet and /usr/bin/kubectl against the
# trusted reference on cp-1, restore the tampered binary, restart kubelet, and
# write the verification report.
#
# This is the tier's analogue of "kubectl apply -f solution/": the work is on a
# node's filesystem, so we drive it from the HOST via `docker exec` into the
# kind control-plane node, mirroring how the harness resolves `node: cp-1`
# (internal/environment/kind_provider.go: cp-1 -> jerry-gym-control-plane).
#
# A learner on a real cluster would SSH to the control-plane node, run the same
# sha512sum comparison, RE-DOWNLOAD the affected release from dl.k8s.io and
# verify it against the published hashes, then swap it in and restart kubelet.
# Here the pristine binary was stashed at /root/pristine before the tamper, so
# we restore from there for a fully deterministic, offline playtest (see README).
#
# Idempotent: safe to run repeatedly (re-restores, re-restarts, rewrites report).
set -euo pipefail

NODE="${CKS004_NODE:-jerry-gym-control-plane}"
REF=/root/checksums/SHA512SUMS

if ! docker exec "$NODE" test -f "$REF"; then
  echo "reference $REF not found on $NODE" >&2
  exit 1
fi

echo "Restoring binaries and writing verification report on $NODE ..."

docker exec -i "$NODE" sh -s <<'REMOTE'
set -e
REF=/root/checksums/SHA512SUMS
REPORT=/root/binary-verification.txt
PRISTINE=/root/pristine

# 1. Restore both binaries from the stashed known-good copies. We swap the
#    directory entry (mv over the path) rather than writing the file in place:
#    the running kubelet holds the old (tampered) inode, and writing an
#    executing binary would fail with ETXTBSY. The restart below re-execs the
#    restored file. kubectl was never tampered but we restore it idempotently.
cp -f "$PRISTINE/kubelet" /usr/bin/kubelet.restore
mv -f /usr/bin/kubelet.restore /usr/bin/kubelet
cp -f "$PRISTINE/kubectl" /usr/bin/kubectl.restore
mv -f /usr/bin/kubectl.restore /usr/bin/kubectl

# 2. Restart the affected service so it runs the restored binary.
systemctl restart kubelet
for i in $(seq 1 30); do
  [ "$(systemctl is-active kubelet)" = active ] && break
  sleep 1
done

# 3. Write /root/binary-verification.txt in the required fixed format:
#    "<name>: expected=<sha512> computed=<sha512> status=OK|MISMATCH"
: > "$REPORT"
for b in kubelet kubectl; do
  path="/usr/bin/$b"
  exp=$(grep " $path\$" "$REF" | awk '{print $1}')
  got=$(sha512sum "$path" | awk '{print $1}')
  if [ "$exp" = "$got" ]; then st=OK; else st=MISMATCH; fi
  printf '%s: expected=%s computed=%s status=%s\n' "$b" "$exp" "$got" "$st" >> "$REPORT"
done

echo "kubelet is now: $(systemctl is-active kubelet)"
echo "--- /root/binary-verification.txt ---"
cat "$REPORT"
REMOTE

echo "Done. Restored the tampered binary, restarted kubelet, wrote the report."
