# CKS-019 — Hint 1 (free)

Let the linter do the first pass, then read the `DL####` codes it prints:

```bash
cd dockerfiles
for f in df-a df-b df-c; do
  echo "== $f =="
  docker run --rm -i hadolint/hadolint:v2.12.0 hadolint - < "$f"
done
```

Each code is a rule you can look up (e.g. DL3007 = don't use `:latest`). Fix the
finding in a new file named `<name>.fixed` (so `df-a.fixed`, `df-b.fixed`,
`df-c.fixed`) sitting next to the original, then build it:

```bash
docker build -f df-a.fixed -t cks019-df-a:fixed .
```

hadolint is mechanical — it will not catch *every* problem. Read each line and
ask: does this leak a secret, fetch something unverified, or run as root?

Docs: <https://github.com/hadolint/hadolint#rules>
