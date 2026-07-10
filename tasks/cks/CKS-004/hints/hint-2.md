# CKS-004 — Hint 2 (cost 25)

Do the four steps on `cp-1`, in order.

## 1. Compare each binary to the reference

```bash
sha512sum /usr/bin/kubelet /usr/bin/kubectl   # computed hashes
cat /root/checksums/SHA512SUMS                # expected hashes
```

The tampered binary is `kubelet` (bytes were appended — the ELF still runs, but
the hash no longer matches). `kubectl` matches.

## 2. Write `/root/binary-verification.txt` in the EXACT format

One line per binary — the checks grep for `expected=`, `computed=`, `status=`:

```
kubelet: expected=<sha512> computed=<sha512> status=MISMATCH
kubectl: expected=<sha512> computed=<sha512> status=OK
```

`status=OK` only when `expected == computed`. You will regenerate this after the
fix so **both** lines read `status=OK`.

## 3. Restore the tampered binary

```bash
cp -f /root/pristine/kubelet /usr/bin/kubelet.restore
mv -f /usr/bin/kubelet.restore /usr/bin/kubelet
```

Why `mv` (swap the file) instead of writing `/usr/bin/kubelet` directly? The
running kubelet holds that file open for execution, so writing it in place fails
with **`ETXTBSY` (text file busy)**. `mv` replaces the directory entry; the next
step re-execs the fresh file.

## 4. Restart kubelet and re-check

```bash
systemctl restart kubelet
sha512sum -c /root/checksums/SHA512SUMS     # both must say OK now
kubectl get nodes                           # node returns to Ready
```

Then rewrite the report so both lines show `status=OK` with matching hashes.

## Traps

- **The report alone is not the fix.** A separate check recomputes the live hash
  on the node and compares it to `SHA512SUMS` — you must actually restore the
  binary, not just edit the text file.
- **Restart the service.** Replacing the file does not change the already-running
  process; `systemctl restart kubelet` makes it exec the restored binary.
- On a real node you would re-download from `dl.k8s.io` and verify against the
  published `.sha512` — see `solution/README.md`.
