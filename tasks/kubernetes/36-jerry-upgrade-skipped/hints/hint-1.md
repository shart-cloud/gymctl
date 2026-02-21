# Hint 1: Read the Script and Focus on the Version Numbers

Open the upgrade script and look at every version flag:

```bash
cat /tmp/jerry-upgrade.sh
```

What version does `kubeadm upgrade apply` target? What version is the cluster
currently running? kubeadm's version skew policy allows upgrading only
**one minor version at a time** — skipping a minor version will be rejected.
