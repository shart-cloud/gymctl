# gymctl VS Code Extension — Design Spec

**Status:** Pre-implementation  
**Scope:** gymctl JSON output, lab-spec.yaml convention, VS Code extension architecture  
**Repo:** `containers/gymctl` (CLI changes), new repo TBD for extension

---

## Overview

Students work in Coder workspaces provisioned from one of several templates (container-gym, kubeadm-lab, vcluster-lab, etc.). They solve exercises either by typing `gymctl check $EXERCISE` in the terminal or by clicking a button in a VS Code sidebar panel. Both paths use the same grading engine and produce the same result — the extension is a UI layer, not a replacement for the CLI.

The system has three moving parts:

1. **gymctl JSON output** — structured output flag added to existing commands so the extension can parse results without screen-scraping
2. **`lab-spec.yaml` convention** — a single-exercise spec file injected by Coder templates that don't use the gymctl task catalog (kubeadm, vcluster)
3. **VS Code extension** — sidebar panel that detects workspace type, runs checks, and surfaces results

---

## Part 1: gymctl JSON Output

### Goal

The extension needs machine-readable output from four commands: `list`, `status`, `check`, `hint`. Today these produce colored terminal output with Jerry dialogue, progress bars, and spinners — none of which is parseable.

Adding `--output json` (shorthand `--json`) suppresses all decorative output and writes clean JSON to stdout. The existing terminal experience is completely unchanged when the flag is absent. Errors go to stderr with a non-zero exit code regardless of output format.

### Command contracts

#### `gymctl list --output json`

```json
{
  "exercises": [
    {
      "name": "k8s-deploy-broken-pods",
      "title": "Fix Jerry's Broken Pods",
      "track": "kubernetes",
      "week": 4,
      "order": 1,
      "difficulty": "beginner",
      "estimatedTime": "20m",
      "points": 100,
      "status": "completed",
      "score": 90,
      "tags": ["pods", "debugging"],
      "locked": false,
      "missingPrereqs": []
    },
    {
      "name": "k8s-rbac-break-glass",
      "title": "RBAC Break Glass",
      "track": "kubernetes",
      "week": 4,
      "order": 2,
      "difficulty": "intermediate",
      "estimatedTime": "30m",
      "points": 150,
      "status": "not-started",
      "score": 0,
      "tags": ["rbac", "security"],
      "locked": true,
      "missingPrereqs": ["k8s-deploy-broken-pods"]
    }
  ],
  "summary": {
    "total": 56,
    "completed": 12,
    "inProgress": 1,
    "notStarted": 43,
    "totalPoints": 5000,
    "earnedPoints": 1150
  }
}
```

#### `gymctl status --output json`

```json
{
  "current": "k8s-rbac-break-glass",
  "summary": {
    "total": 56,
    "completed": 12,
    "inProgress": 1,
    "notStarted": 43,
    "totalPoints": 5000,
    "earnedPoints": 1150,
    "completionPercent": 21
  }
}
```

`current` is the active exercise name (the one set by `gymctl start`), or `null` if none is active.

#### `gymctl check [exercise] --output json`

```json
{
  "exercise": "k8s-rbac-break-glass",
  "allPassed": false,
  "passedCount": 2,
  "totalCount": 4,
  "score": 0,
  "pointsAvailable": 150,
  "checks": [
    {
      "name": "serviceaccount-exists",
      "passed": true,
      "message": ""
    },
    {
      "name": "role-bound-correctly",
      "passed": true,
      "message": ""
    },
    {
      "name": "pod-can-list-secrets",
      "passed": false,
      "message": "expected exit code 0, got 1"
    },
    {
      "name": "audit-log-shows-deny",
      "passed": false,
      "message": "output does not contain: \"forbid\""
    }
  ]
}
```

`score` is only populated (non-zero) when `allPassed` is true. `message` is empty string when passed; populated with the engine's diagnostic when failed. This is what the extension surfaces inline per check.

#### `gymctl hint [exercise] --output json`

```json
{
  "exercise": "k8s-rbac-break-glass",
  "hintIndex": 2,
  "content": "Check the RoleBinding subject — it needs to reference the ServiceAccount name exactly as defined, namespace included.",
  "hintsUsed": 2,
  "hintsTotal": 3,
  "hintsRemaining": 1,
  "nextHintCost": 25
}
```

When no hints remain:
```json
{
  "exercise": "k8s-rbac-break-glass",
  "hintIndex": null,
  "content": null,
  "hintsUsed": 3,
  "hintsTotal": 3,
  "hintsRemaining": 0,
  "nextHintCost": 0
}
```

### Implementation notes

- Add `--output` flag to `rootCmd.PersistentFlags()` so it's available on all subcommands, or add it individually to `list`, `status`, `check`, `hint`
- In each command's `RunE`, check `outputFormat == "json"` before rendering any UI (no spinners, no Jerry, no color)
- `--output json` and `--output table` (default) are the only valid values; error on anything else
- The flag should be `--output` / `-o` to match `kubectl` convention — familiar to students, consistent with the tooling they're already using

### New flag: `gymctl check --spec <path>`

The other change needed in `check.go`: a `--spec` flag that accepts a path to an exercise file directly, bypassing catalog lookup. When `--spec` is set, `check` does:

```go
exercise, err := scenario.LoadExerciseFile(opts.spec)
```

...instead of `scenario.LoadCatalog` + `scenario.FindByName`. Everything downstream (check engine, progress tracking, JSON output) is unchanged. This is what enables the extension to grade Coder lab objectives using the same engine as gymctl exercises.

The `--spec` flag can be combined with `--output json`:
```
gymctl check --spec ~/.coder/lab-spec.yaml --output json
```

---

## Part 2: `lab-spec.yaml` Convention

### Goal

Coder templates that don't use the gymctl task catalog (kubeadm-lab, vcluster-lab) need a way to declare objectives and checks that the extension can discover and grade. Rather than inventing a new format, these templates inject a single exercise file that the gymctl check engine already understands.

### Location

`~/.coder/lab-spec.yaml` — written by the Coder template's startup script or Terraform `local_file` resource during workspace provisioning. The extension looks for this file when no `tasks/` directory is found.

### Schema

The `lab-spec.yaml` is a standard gymctl exercise file. No new schema needed. The only additions are two optional fields on `metadata` and one on `spec` that are specific to the Coder lab context:

```yaml
apiVersion: gymctl.shart.cloud/v1
kind: Exercise
metadata:
  name: kubeadm-bootstrap          # used as exercise ID in progress tracking
  title: "Bootstrap a kubeadm Cluster"
  labType: kubeadm                 # NEW: kubeadm | vcluster | generic
  coderTemplate: kubeadm-lab       # NEW: which Coder template injected this

spec:
  difficulty: intermediate
  estimatedTime: 45m
  points: 200
  description: |
    Jerry tried to bootstrap a cluster and gave up. The VMs are provisioned and
    the packages are installed. Your job is to initialize the control plane,
    join the workers, and install a CNI so the nodes reach Ready state.

  environment:
    type: kubernetes
    kubernetes:
      createCluster: false         # cluster is pre-provisioned via KubeVirt
      provider: kind               # doesn't matter — checks use ambient kubeconfig

  checks:
    - name: control-plane-ready
      type: condition
      resource: "node/kubeadm-kubeadm-cp1"
      condition: Ready

    - name: workers-joined
      type: script
      script: |
        kubectl get nodes --no-headers | grep -v "control-plane" | grep -c "Ready"
      expectOutput:
        regex: "^2$"

    - name: cni-pods-running
      type: jsonpath
      resource: "pods -n kube-system -l k8s-app=cilium"
      jsonpath: "{.items[*].status.phase}"
      operator: contains
      value: "Running"

    - name: coredns-healthy
      type: condition
      resource: "deployment/coredns -n kube-system"
      condition: Available

  hints:
    - cost: 0
      content: "Start with the control plane. SSH in with: ssh kubeadm-cp1"
    - cost: 0
      content: "Run kubeadm init on the control plane first. Check: sudo kubeadm init --pod-network-cidr=10.244.0.0/16"
    - cost: 25
      content: "After init, copy the admin kubeconfig: mkdir -p $HOME/.kube && sudo cp /etc/kubernetes/admin.conf $HOME/.kube/config"
    - cost: 25
      content: "Install Cilium for CNI: cilium install. Then get the join command from: kubeadm token create --print-join-command"
```

### How the Coder template injects it

In the kubeadm-lab `main.tf`, add to the `coder_agent` startup script:

```bash
mkdir -p "$HOME/.coder"
cat > "$HOME/.coder/lab-spec.yaml" <<'SPEC'
# ... spec contents ...
SPEC
```

Or using Terraform heredoc with variable interpolation for dynamic values (e.g., node names derived from workspace name).

Each Coder template is responsible for writing a spec appropriate to that lab type. The spec content lives in the template repo alongside `main.tf`.

### Why reuse the gymctl exercise format

- Zero new parsing logic in the check engine
- The extension calls `gymctl check --spec ~/.coder/lab-spec.yaml --output json` — same command, same output shape
- Instructors who write gymctl exercises already know the format; writing a `lab-spec.yaml` is the same skill
- Progress tracking for Coder labs reuses `~/.gym/progress.yaml` — same file, same schema

---

## Part 3: VS Code Extension

### Overview

The extension is a private VS Code extension distributed through the Coder workspace — not published to the marketplace. It gets installed via `devcontainer.json` or the Coder template's startup script:

```bash
code-server --install-extension gymctl-lab-companion-*.vsix
```

Or via `devcontainer.json`:
```json
{
  "customizations": {
    "vscode": {
      "extensions": ["shart-cloud.gymctl-lab-companion"]
    }
  }
}
```

Extension ID: `shart-cloud.gymctl-lab-companion`  
Display name: **Lab Companion**

### Workspace type detection

On activation, the extension runs detection in order:

1. Check `GYMCTL_TASKS_DIRS` environment variable — if set, use the first path as the tasks directory
2. Check `GYMCTL_TASKS_DIR` as a compatibility fallback
3. Check for `./tasks/` directory in the workspace root — gymctl mode
4. Check for `~/.coder/lab-spec.yaml` — coder-lab mode
5. Neither found → extension stays silent; shows "No lab detected" in the panel with a setup link

Detection runs once at activation. If the workspace is freshly provisioned and the startup script hasn't finished yet, the extension should retry detection up to 3 times with a 3-second delay before giving up. In practice this matters most for coder-lab mode where `lab-spec.yaml` is written by the startup script.

Detection result is stored in extension state and determines which commands and UI are active for the session.

### Sidebar panel structure

The extension contributes a single sidebar view container with the icon `$(beaker)` (VS Code codicon).

**View: Lab Companion** (`gymctl-lab-companion.labView`)

Layout (top to bottom):

```
┌─────────────────────────────────────┐
│  LAB COMPANION          [↻] [⚙]    │
├─────────────────────────────────────┤
│  EXERCISES                          │  ← gymctl mode only; hidden in coder-lab mode
│  ▸ kubernetes (12/20)               │
│    ✓ broken-pods          beginner  │
│    ◉ rbac-break-glass  intermediate │  ← active exercise
│    ○ network-policy       beginner  │
│    ◌ ha-control-plane  advanced     │  ← locked
│  ▸ docker (6/6) ✓                   │
├─────────────────────────────────────┤
│  ACTIVE: rbac-break-glass           │
│  Fix Jerry's RBAC Disaster          │
│  ──────────────────────────────     │
│  OBJECTIVES                         │
│  ✓ serviceaccount-exists            │
│  ✓ role-bound-correctly             │
│  ✗ pod-can-list-secrets             │
│    expected exit code 0, got 1      │  ← failure message inline
│  ✗ audit-log-shows-deny             │
│    output does not contain: "forbid"│
│  ──────────────────────────────     │
│  2 / 4 checks passing               │
│  ──────────────────────────────     │
│  [  Run Checks  ]  [  Hint (2)  ]  │
├─────────────────────────────────────┤
│  PROGRESS  12/56 · 1150 pts        │  ← always visible footer
└─────────────────────────────────────┘
```

**In coder-lab mode**, the EXERCISES section is hidden and the ACTIVE section expands to fill the space, showing the `lab-spec.yaml` title, description, and objectives. The progress footer shows objectives count instead of exercise count.

### Commands

| Command | ID | Description |
|---|---|---|
| Run Checks | `gymctl.runChecks` | Runs checks for active exercise, updates panel |
| Next Hint | `gymctl.nextHint` | Fetches and shows next hint in panel |
| Set Active Exercise | `gymctl.setActive` | Sets exercise as active (gymctl mode; calls `gymctl start`) |
| Refresh | `gymctl.refresh` | Re-reads progress file and reloads panel state |
| Open Terminal | `gymctl.openTerminal` | Opens integrated terminal with `gymctl` in PATH |

`Run Checks` and `Next Hint` are also available via the panel buttons shown above.

### Check execution

When `Run Checks` is invoked:

1. Show a spinner/loading state in the panel
2. Shell out to the appropriate command:
   - gymctl mode: `gymctl check ${exerciseName} --output json`
   - coder-lab mode: `gymctl check --spec ~/.coder/lab-spec.yaml --output json`
3. Parse stdout as JSON (`CheckResult` shape from Part 1)
4. Update panel with per-check pass/fail and inline failure messages
5. If `allPassed: true`, show a completion celebration in the panel and refresh the progress footer

The extension does **not** intercept terminal runs of `gymctl check`. Instead it watches `~/.gym/progress.yaml` for file changes using VS Code's `FileSystemWatcher`. When the progress file changes (because the student ran `gymctl check` in the terminal), the panel automatically refreshes. This means terminal-first students still see their progress reflected without clicking anything.

### Hint display

Hints appear in a dedicated section below the objectives, revealed one at a time. Each hint shows its index and content. The `Next Hint` button is labeled with the remaining count and next cost (e.g., `Hint (2 left · free)` or `Hint (1 left · 25pts)`). When no hints remain the button is disabled and labeled `No hints left`.

Hints are not stored in extension state — they're fetched fresh from `gymctl hint --output json` each time so the hint count stays in sync with `~/.gym/progress.yaml`.

### Progress file watching

```typescript
const watcher = vscode.workspace.createFileSystemWatcher(
  new vscode.RelativePattern(os.homedir(), '.gym/progress.yaml')
);
watcher.onDidChange(() => refreshPanel());
watcher.onDidCreate(() => refreshPanel());
```

This is the passive sync mechanism. The extension never writes to the progress file directly — only gymctl does.

### Error handling

- If `gymctl` is not found in PATH: show a warning in the panel with a link to installation instructions
- If `gymctl check` exits non-zero (checks failed): parse stdout anyway — the JSON is still valid, `allPassed` will be `false`
- If stdout is not valid JSON: show a raw error message in the panel, offer to open the terminal
- If `~/.coder/lab-spec.yaml` is malformed: show parse error with the path, offer to open the file

### Distribution and installation

The extension is built as a `.vsix` and committed to the `containers/gymctl` repo (or a separate `gymctl-vscode` repo). Each Coder template that uses it adds it to the startup script:

```bash
# Install Lab Companion extension
if command -v code-server &>/dev/null; then
  code-server --install-extension /path/to/gymctl-lab-companion.vsix 2>/dev/null || true
fi
```

Alternatively, store the `.vsix` in the container image if the template uses a custom image.

The extension does not require an internet connection at runtime — all grading runs locally via gymctl.

---

## Implementation Order

### Step 1: gymctl `--output json` flag and `--spec` flag

**Files to change:** `internal/cli/check.go`, `internal/cli/list.go`, `internal/cli/status.go`, `internal/cli/hint.go`, `internal/cli/root.go`

This is the smallest change and unblocks everything else. Start here.

Acceptance criteria:
- `gymctl list --output json` returns valid JSON matching the schema above
- `gymctl check <exercise> --output json` returns valid JSON with per-check results
- `gymctl check --spec <path> --output json` loads the spec file directly and returns the same JSON shape
- `gymctl hint <exercise> --output json` returns hint content as JSON
- `gymctl status --output json` returns summary JSON
- All existing terminal output (without `--output json`) is unchanged
- `--output` / `-o` is the flag name; invalid values produce an error

### Step 2: `lab-spec.yaml` schema and Coder template injection

**Files to change:** `containers/coder-templates/kubeadm-lab/main.tf`, `containers/coder-templates/vcluster-lab/main.tf`

Write a `lab-spec.yaml` for each existing Coder template type. Verify that `gymctl check --spec ~/.coder/lab-spec.yaml --output json` runs correctly against each.

Acceptance criteria:
- kubeadm-lab template injects a `lab-spec.yaml` that checks control-plane Ready, workers joined, CNI running, CoreDNS healthy
- vcluster-lab template injects a `lab-spec.yaml` appropriate to its lab objectives
- `gymctl check --spec ~/.coder/lab-spec.yaml --output json` returns valid JSON in a fresh workspace

### Step 3: VS Code extension

**New repo or subdirectory:** `containers/gymctl-vscode` or `containers/gymctl/vscode-extension/`

Build with the VS Code extension API (TypeScript). The webview for the sidebar can use a simple TreeView for the exercise list; the active exercise detail can be a WebviewPanel or a custom TreeView with inline items.

Start with the detection logic and a minimal panel that shows exercise list and check results. Add the progress file watcher and hint display after the core flow is working.

Acceptance criteria:
- Extension activates correctly in gymctl-mode workspaces (shows exercise list)
- Extension activates correctly in coder-lab-mode workspaces (shows objectives from `lab-spec.yaml`)
- `Run Checks` button produces the same result as `gymctl check $EXERCISE` in the terminal
- Failed checks show their failure message inline
- Progress file changes from terminal runs are reflected in the panel within 2 seconds
- Extension is installable as a `.vsix` and works in both code-server and native VS Code via Coder's SSH helper

---

## Open Questions

1. **Exercise catalog for coder-lab mode**: Should `gymctl list` show Coder lab objectives alongside gymctl exercises, or are they separate catalogs? Current spec treats them as separate — the extension handles the split. Revisit if students find the context switch confusing.

2. **Multiple specs per template**: Some templates might have multiple lab stages (e.g., a kubeadm lab where stage 1 is bootstrap, stage 2 is upgrade). The current spec assumes one `lab-spec.yaml` per workspace. Could extend to a `~/.coder/lab-specs/` directory with multiple files if needed.

3. ~~**Progress persistence for coder-lab mode**~~ — **Resolved.** Coder lab workspaces are ephemeral by design; they are torn down after 2–4 hours regardless. When a student re-provisions, they start from scratch. Progress persistence across workspace sessions is not a concern — `~/.gym/progress.yaml` lives on the ephemeral workspace disk and disappears with it.

4. **Extension repo location**: Keeping it in `containers/gymctl/vscode-extension/` co-locates it with the CLI it depends on and simplifies versioning. A separate repo adds flexibility but splits concerns. Leaning toward co-location unless the extension grows large.
