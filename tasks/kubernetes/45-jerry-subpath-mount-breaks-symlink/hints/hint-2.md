`subPath` mounts a single file or directory from a volume. The tradeoff is that it
breaks the live-update mechanism that normal ConfigMap volume mounts rely on.

Normal ConfigMap mount: kubelet atomically replaces the directory via a symlink swap
→ container sees updated files automatically within ~60 seconds.

subPath mount: bind-mounts the original file descriptor directly → the symlink swap
doesn't propagate to it → container sees only the value at pod creation time.

Two fixes:

**Option A — Restart the pod** (quick fix, subPath still won't live-update in future):
```bash
kubectl delete pod jerry-config-app
# Then recreate it
```

**Option B — Remove subPath** (proper fix, enables live updates):
Change the volumeMount to mount the whole ConfigMap as a directory,
then read `db-host` as a file within that directory.
