# CKS Track — Authoring Playbook

How to turn each `scaffold` CKS exercise into a `ready`, playtested lab.
CKS-001 is the worked reference — read it alongside this doc.

Status: `gymctl author status --track cks`

---

## 1. Anatomy of a "gold standard" exercise (what CKS-001 established)

```
CKS-NNN/
  task.yaml            # implementationStatus: ready; real environment + real checks
  setup/               # manifests/scripts that build the BROKEN/vulnerable state
    namespaces.yaml
    workloads.yaml
  solution/            # the intended fix + a README explaining WHY
    *.yaml
    README.md
  hints/               # 3 real tiered hints (cost 0 / 25 / 50)
    hint-1.md          # free: orientation + the one idea
    hint-2.md          # the required objects + the traps, no full YAML
    hint-3.md          # near-complete YAML; "full manifests in solution/"
```

Delete the scaffold's `check/` dir and `setup/README.md` placeholders — the
gold standard doesn't use them. Checks live in `task.yaml`.

### Rules that make it "gold", not just "passing"

1. **Broken state fails, solved state passes — and the delta is the learning.**
   Run `check` before solving: the security checks must FAIL. If a check passes
   on the untouched broken state, it isn't testing the control.
2. **Test real enforcement, not just object shape.** A NetworkPolicy that
   *exists* is not a NetworkPolicy that *blocks*. Include negative checks
   ("X CANNOT reach Y") that only pass when the control is truly enforced. This
   is why CKS-001 installs Calico — see §2.
3. **Both a positive and a negative per control.** Positive proves you didn't
   break the app; negative proves you actually restricted something.
4. **Deterministic checks.** Prefer fixed object names (instruct them in the
   task) + `kubectl`/`agnhost` over fuzzy matching. Avoid depending on `jq`
   (not guaranteed on the learner's host); use `kubectl -o jsonpath` + `grep`.
5. **Solution is idempotent and self-contained.** `kubectl apply -f solution/`
   must take the broken state to 100% green with no manual steps.

### Check types available (see `internal/checks/`)

| type | use |
| --- | --- |
| `resourceExists` | object is/ isn't present (`exists: false` to assert absence) |
| `jsonpath` | field value via `-o jsonpath`, with `operator` (equals/contains/regex/greaterThan/lessThan/exists) |
| `condition` | `.status.conditions[type==X].status` |
| `exec` | run a command *inside a pod* (`resource`, `command`, `expectExitCode`, `expectOutput`) |
| `podLogs` | grep a pod's logs (`selector`/`resource`, `operator`, `value`) |
| `nodeExec` | run a script *on a node* (`node`, `script`) — kind: `docker exec`; vagrant: ssh |
| `script` | arbitrary `bash -c` on the host with `$KUBECONFIG` set; exit 0 = pass |
| `docker-image`/`docker-container`/`docker-logs`/`dockerfile` | docker-env exercises |
| `http`/`file` | endpoint probe / file assertion |

The connectivity trick used throughout CKS-001: the `agnhost` image
(`registry.k8s.io/e2e-test-images/agnhost`) serves with `netexec` and acts as a
client with `connect host:port --timeout=Xs` — exit 0 on success, non-zero when
a packet is dropped. Ideal for allow/deny assertions via `exec`/`script`.

---

## 2. Environment decision: Docker (kind) vs VM (vagrant) — the research

This is the single most important authoring decision per exercise, because the
gym has two provisioning backends with very different fidelity, cost, and
reliability.

### How the two backends actually differ

| | **kind** (`provider: kind`) | **vagrant** (`provider: vagrant`) |
| --- | --- | --- |
| What a "node" is | a **container** on the host | a real **VM** (VirtualBox) |
| Kernel | **shared with the host** | **its own** |
| Node access | `docker exec` (root shell, files at `/etc/kubernetes`, `/var/lib/kubelet`) | ssh |
| Startup | ~30–90s | **5–15 min** |
| Multi-node / real kubelet-per-node | limited | yes |
| Kernel-level isolation (LSM, seccomp-at-kernel, runsc, eBPF drivers) | **unreliable — inherits host** | **full control** |
| Runs in this repo's dev/CI env | ✅ (docker up) | ⚠️ **needs VirtualBox — `VBoxManage` is MISSING on this WSL2 box** |

**The core rule:** *kind shares the host kernel.* Anything that is purely
Kubernetes API / manifest / control-plane-file is perfectly fine on kind (and
you can reach node files via `nodeExec` = `docker exec`). Anything that depends
on the **guest kernel** — Linux Security Modules (AppArmor), a runtime sandbox
(gVisor/runsc), a syscall-tap driver (Falco kernel module / eBPF), or a real
`kubeadm upgrade` — is where a container-as-node leaks host behaviour and gives
false pass/fail. Those want a VM.

**NetworkPolicy is the subtle one.** kind's default CNI (`kindnetd`) *silently
ignores NetworkPolicy*. A "cannot reach X" check would falsely pass. Two fixes:
- **kind + Calico** (what CKS-001 does): `disableDefaultCNI: true` in the kind
  config + install Calico in `customSetup`. Now denial is enforced and testable.
  Adds ~2 min. This is the recommended path for all NP exercises — no VM needed.
- Structural-only checks (object shape + positive connectivity). Cheaper but
  does **not** verify blocking — unacceptable for a security track's deny rules.

### Per-exercise classification (all 25)

Tiers: **K** = pure kind · **K+C** = kind + Calico · **K+N** = kind + nodeExec
(control-plane / node-file edits) · **D** = docker/local env · **VM** = vagrant
VM required or strongly preferred.

| ID | Title | Tier | Why |
| --- | --- | --- | --- |
| 001 | network-policy-default-deny | **K+C** ✅done | NP enforcement needs Calico |
| 002 | cis-benchmark-remediation-kube-bench | K+N | kube-bench reads node files; some host/kernel checks only real on **VM** |
| 003 | ingress-tls-dashboard-lockdown | K | ingress + TLS secret, API-level |
| 004 | binary-verification-cluster-integrity | K+N | checksum node binaries via `nodeExec` |
| 005 | rbac-least-privilege | K | RBAC is API-level |
| 006 | service-account-hardening | K | SA tokens/automount, API-level |
| 007 | api-server-access-restriction | K+N | edit kube-apiserver static pod on control-plane |
| 008 | kubernetes-version-upgrade-cve | **VM** | real `kubeadm upgrade` doesn't work on kind |
| 009 | apparmor-profile-enforcement | **VM** | AppArmor is a kernel LSM; load profile in guest kernel |
| 010 | seccomp-profiles | K+N | seccomp profiles broadly work on kind; place profile via `nodeExec` |
| 011 | node-kubelet-hardening | K+N (VM nicer) | edit kubelet config + restart on node |
| 012 | pod-security-context-lockdown | K | securityContext, manifest-level |
| 013 | opa-gatekeeper-policy-enforcement | K | Gatekeeper admission webhook |
| 014 | secrets-management-encryption-at-rest | K+N | EncryptionConfig on apiserver; verify via etcd in node |
| 015 | runtime-sandboxing-gvisor | **VM** | runsc needs guest kernel features; unsupported in kind |
| 016 | pod-security-admission | K | PSA namespace labels, built-in admission |
| 017 | base-image-minimization | **D** | Dockerfile/image exercise, `environment: docker` |
| 018 | container-image-vulnerability-scanning | **D** | trivy scan; docker/local |
| 019 | dockerfile-security-audit | **D** | static Dockerfile analysis; local |
| 020 | image-admission-validating-webhook | K | validating webhook, API-level |
| 021 | manifest-static-analysis-pipeline | **D** (local) | kubesec/kube-linter/conftest on files |
| 022 | falco-runtime-monitoring | **VM** | Falco driver taps syscalls; kernel-module won't build in kind, eBPF is host-dependent |
| 023 | api-server-audit-logging | K+N | audit policy + apiserver flags + read log on node |
| 024 | immutable-container-enforcement | K | readOnlyRootFilesystem / admission, API-level |
| 025 | incident-response-cluster-compromise | K (VM if Falco-based) | planted artifacts + log forensics |

Rough split: **~13 pure/near kind, ~3 docker/local, ~4 need a VM, rest kind+nodeExec.**
Build the kind ones first (they're testable in this env today); the 4 VM ones
(008, 009, 015, 022) are blocked here until VirtualBox (or a libvirt/docker
vagrant provider) is available — call that out when you get to them.

---

## 3. The build → test → playthrough loop

Do this for every exercise. `bin` = your freshly built `gymctl`.

```bash
go build -o /tmp/gymctl ./cmd/gymctl

# 0. sanity: schema + status
/tmp/gymctl validate tasks/cks/CKS-NNN/task.yaml
/tmp/gymctl author status --track cks | grep CKS-NNN     # expect: ready ok

# 1. provision the broken state
/tmp/gymctl start cks-NNN-<slug>

# 2. checks MUST fail on the broken state (esp. the negative/deny checks)
/tmp/gymctl check cks-NNN-<slug>        # expect: NOT complete; note which pass

# 3. apply the intended fix and confirm green
kubectl apply -f tasks/cks/CKS-NNN/solution/     # or the documented fix steps
/tmp/gymctl check cks-NNN-<slug>        # expect: 🎉 all checks passed

# 4. (optional) prove no false-greens: revert ONE policy/control, re-check,
#    confirm exactly the expected check(s) go red.

# 5. teardown
/tmp/gymctl stop cks-NNN-<slug>
```

Acceptance bar before flipping `implementationStatus: ready`:
- `validate` OK, `author status` shows `ready ok`.
- Broken `check` fails; **every negative check is red in the broken state**.
- `solution/` alone drives `check` to 100%.
- Start + solve + teardown leaves no stray kind cluster (`kind get clusters`).

For **VM** exercises, step 1 needs VirtualBox; if unavailable, mark the exercise
`draft` with a note rather than `ready`, and test on a box that has it.

---

## 4. Reusable agent prompt (one per exercise)

Copy-paste, fill the two placeholders, hand to an implementation agent:

```
Implement CKS-<NNN> (<title>) in tasks/cks/CKS-<NNN>/ to the "gold standard"
defined in tasks/cks/AUTHORING_PLAYBOOK.md, using tasks/cks/CKS-001 as the
worked reference.

1. Read tasks/cks/CKS-<NNN>/task.yaml (scaffold) for the intended scenario and
   the planned check criteria in its placeholder script + hint-2.
2. Pick the environment tier from the playbook's §2 table for this exercise and
   justify it in one line. If it needs a VM (vagrant) and VirtualBox is absent,
   stop and report that instead of forcing kind.
3. Write setup/ (the broken/vulnerable state), solution/ (the fix + README that
   explains WHY each piece is needed), and 3 real tiered hints (0/25/50).
4. Write real checks in task.yaml: for every control, one POSITIVE check (app
   still works) and one NEGATIVE check (the thing is actually restricted).
   Delete the check/ and setup/README.md placeholders. Set
   implementationStatus: ready.
5. Run the full build→test→playthrough loop from §3. Paste the broken-state
   check output (must fail) and the post-solution output (must be 100%). Fix
   until it genuinely passes; do NOT weaken checks to make them pass.
6. Report: env tier chosen, what the broken state looks like, the check matrix,
   and any host requirements (Calico, trivy, VirtualBox, etc.).

Constraints: deterministic checks, no jq dependency, idempotent solution,
teardown clean. Match the voice/structure of CKS-001's files.
```

For a batch, run one agent per exercise in the same tier (they share setup
patterns), review each playthrough transcript, then flip to `ready`.
```
