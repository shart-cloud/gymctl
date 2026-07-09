# CKS-021 — Hint 1 (free)

Don't apply the manifests yet — scan them first. The whole idea is *shift-left*:
catch the problems in the YAML before it reaches the cluster.

Your manifests are in `manifests/` in your work directory. Run both scanners:

```bash
# kubesec scores one manifest at a time on stdin; look at the `critical` list
docker run --rm -i kubesec/kubesec:v2 scan /dev/stdin < manifests/web.yaml
docker run --rm -i kubesec/kubesec:v2 scan /dev/stdin < manifests/api.yaml

# kube-linter lints the whole directory
docker run --rm -v "$PWD/manifests:/m" stackrox/kube-linter:v0.7.4 lint /m
```

Save the outputs — the checks look for them:

```bash
mkdir -p reports
docker run --rm -i kubesec/kubesec:v2 scan /dev/stdin < manifests/web.yaml > reports/kubesec.json
docker run --rm -v "$PWD/manifests:/m" stackrox/kube-linter:v0.7.4 lint /m > reports/kube-linter.txt 2>&1
```

Read the CRITICAL / HIGH findings. They point at exactly the fields you need to
change: `privileged`, `runAsNonRoot`, `readOnlyRootFilesystem`, dropped
capabilities, resource limits, and the host-namespace / hostPath access.

Docs: <https://kubesec.io/> · <https://docs.kubelinter.io/#/generated/checks>
