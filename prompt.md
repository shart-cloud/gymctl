# gymctl Phase 2 — Generalize for GX Exam Tracks

## Context

gymctl is a Go CLI (cobra-based) that orchestrates hands-on security/infrastructure exercises. It currently has ~55 tasks across Docker and Kubernetes tracks, a CKA-specific exam mode, and a progress tracker. The codebase lives at `containers/gymctl/` within the shart-cloud monorepo.

We are extending gymctl to support GIAC GX-series exams (GX-CS, GX-IA, GX-IH, GX-FA, GX-FE, GX-PT). These are CyberLive hands-on exams covering network analysis, Linux/Windows forensics, malware analysis, password cracking, etc. Tasks for these exams live in a separate repo (`gx-prep`) and use a new `local` environment type — they don't spin up clusters or containers, they just need tools and artifact files already present in the workspace.

The existing CKA exam engine is hardcoded in several places. This rework generalizes the exam system so any exam (CKA, GX-CS, GX-IA, etc.) can define its own domains, weights, and filters — and adds new commands for task authoring and objective tracking.

## What exists today (read these files first)

| File | What it does | Why it matters |
|------|-------------|----------------|
| `internal/cli/exam.go` | CKA exam mode — hardcoded domains, weights, track allowlist, playlist filter | **Primary refactor target** |
| `internal/cli/root.go` | Cobra command registration, global flags (`--tasks-dir`, `--progress-file`) | Registration point for new commands |
| `internal/scenario/types.go` | `Exercise`, `ExerciseMeta`, `ExerciseSpec`, `Check`, `Hint` structs | Needs new fields: `gxObjective`, `local` env type |
| `internal/scenario/catalog.go` | `LoadCatalog()` walks `tasks/` for `task.yaml` files | Already track-agnostic — no changes needed |
| `internal/scenario/schemas/exercise.schema.json` | JSON Schema for task.yaml validation | Needs `"local"` added to environment type enum |
| `internal/progress/progress.go` | `File` and `ExerciseStatus` types, Load/Save | Already track-agnostic — may need objective-level tracking |
| `internal/cli/list.go` | `gymctl list` with track/difficulty/week filters | Already works for arbitrary tracks |
| `internal/cli/next.go` | `gymctl next` picks next exercise by (track, week, order) | Works but has no track filter — should accept `--track` |
| `internal/cli/status.go` | `gymctl status` shows progress by track | Works for arbitrary tracks already |
| `internal/cli/start.go` | `gymctl start <name>` — switches on env type (docker/kubernetes/hybrid) | Needs `"local"` case that's a no-op (just marks started) |
| `internal/cli/check.go` | Runs validation checks | Already supports `script` and `file` check types needed for GX |
| `internal/cli/tasks_resolver.go` | Resolves tasks directory from flag/env/well-known paths | May want to support multiple task dirs |

## Changes required

### 1. Add `local` environment type

GX exercises don't need cluster or container setup. They assume the workspace already has the right tools and artifacts mounted. `gymctl start` for a `local` exercise should just record "started" in progress and print the description.

**Files to change:**

- `internal/scenario/schemas/exercise.schema.json` (line 54): Add `"local"` to the environment type enum:
  ```json
  "enum": ["docker", "kubernetes", "hybrid", "local"]
  ```

- `internal/cli/start.go` (~line 54): Add a `case "local":` to the environment type switch. It should:
  - Print the exercise description and any Jerry dialog
  - Write "started" to progress
  - Print `gymctl check <name>` as the next step
  - Do NO environment setup

- `internal/cli/stop.go` (~line 46): Add a `case "local":` that's a no-op (nothing to tear down).

- `internal/cli/reset.go` (~line 142): Add a `case "local":` that just resets progress state.

### 2. Add `gxObjective` field to exercise metadata

GX tasks map to specific exam objectives (e.g., "3.2" maps to GX-CS domain 3, objective 2). This is needed for the `objectives` command and for domain classification in exam mode.

**Files to change:**

- `internal/scenario/types.go` — add to `ExerciseMeta`:
  ```go
  type ExerciseMeta struct {
      Name        string `yaml:"name"`
      Title       string `yaml:"title"`
      Track       string `yaml:"track"`
      Week        int    `yaml:"week,omitempty"`
      Order       int    `yaml:"order,omitempty"`
      GXObjective string `yaml:"gxObjective,omitempty"`
  }
  ```

- `internal/scenario/schemas/exercise.schema.json` — add `"gxObjective": {"type": "string"}` to metadata properties.

### 3. Generalize the exam engine

The current `exam.go` has CKA concepts baked in everywhere: domain names, weights, track allowlists, playlist matching, domain classification. Generalize this into an exam definition system.

**Design: exam definition files**

Create `internal/exam/definition.go` with a data-driven exam definition:

```go
package exam

type Definition struct {
    ID            string            `yaml:"id"`
    Name          string            `yaml:"name"`
    Domains       []Domain          `yaml:"domains"`
    DomainWeights map[string]int    `yaml:"domainWeights"`
    DomainOrder   []string          `yaml:"domainOrder"`
    TrackPrefix   string            `yaml:"trackPrefix"`   // e.g. "gx-cs" or "k8s"
    AllowedTracks []string          `yaml:"allowedTracks"`  // explicit track list
    DefaultDuration int             `yaml:"defaultDuration"` // minutes
    DefaultBackend  string          `yaml:"defaultBackend"`
}

type Domain struct {
    ID          string `yaml:"id"`
    Name        string `yaml:"name"`
    Weight      int    `yaml:"weight"`
}
```

**Built-in exam definitions:**

Register CKA and GX-CS as built-in definitions. CKA keeps its current behavior. GX-CS gets:

```yaml
id: gx-cs
name: "GIAC Experienced Cyber Security"
trackPrefix: "gx-cs"
defaultDuration: 240  # 4 hours
defaultBackend: "local"
domains:
  - id: "1"
    name: "Advanced Network Analysis"
    weight: 15
  - id: "2"
    name: "Evaluating Linux Systems"
    weight: 15
  - id: "3"
    name: "Evaluating Windows Systems"
    weight: 15
  - id: "4"
    name: "File Analysis"
    weight: 15
  - id: "5"
    name: "Malicious Program Execution & Exploitation"
    weight: 12
  - id: "6"
    name: "Network Security"
    weight: 16
  - id: "7"
    name: "Password Cracking"
    weight: 12
```

**Refactor `exam.go`:**

- Change `gymctl exam cka` to `gymctl exam <exam-id>` where exam-id is `cka`, `gx-cs`, `gx-ia`, etc.
- Keep `gymctl exam cka` working (it's just the CKA definition).
- `matchesPlaylist()` → use `Definition.TrackPrefix` or `Definition.AllowedTracks`.
- `isCKAAlignedTrack()` → use `Definition.AllowedTracks`.
- `classifyCKADomain()` → generalized `classifyDomain()` that uses `gxObjective` field for GX exams, and falls back to tag-based classification for CKA.
- `ckaDomainWeights` / `ckaDomainOrder` → come from the definition.
- `isBackendCompatible()` → for `local` backend, all `local`-environment exercises are compatible; skip the vagrant/kind logic.
- `printExamPlan()` → use definition name instead of hardcoded "CKA Exam Mode".
- `examSession.Mode` → set to the exam ID, not hardcoded `"cka"`.

**Domain classification for GX exams:**

GX exercises have a `gxObjective` field like `"3.2"`. The domain is the integer prefix (e.g., `"3"` → domain 3 → "Evaluating Windows Systems"). This is much simpler than CKA's tag-based heuristic:

```go
func classifyGXDomain(ex *Exercise, def *Definition) string {
    if ex.Metadata.GXObjective == "" {
        return def.Domains[0].Name // fallback
    }
    domainID := strings.Split(ex.Metadata.GXObjective, ".")[0]
    for _, d := range def.Domains {
        if d.ID == domainID {
            return d.Name
        }
    }
    return def.Domains[0].Name
}
```

### 4. New command: `gymctl task new`

Scaffolds a new task directory with a `task.yaml` skeleton.

```
gymctl task new --track gx-cs --objective 3.2 --title "Assess Windows firewall rules"
```

Creates:
```
tasks/gx-cs/gx-cs-windows-firewall-001/
├── task.yaml    # pre-filled skeleton
└── hints/
    └── hint-1.md
```

The skeleton `task.yaml` should have:
- `apiVersion`, `kind` pre-filled
- `metadata.name` derived from track + slugified title + auto-incrementing suffix
- `metadata.track` from `--track`
- `metadata.gxObjective` from `--objective`
- `spec.environment.type: local` (default for gx-* tracks)
- Placeholder `checks:` and `hints:` sections with comments
- `spec.description` as a TODO placeholder

**Implementation:** New file `internal/cli/task.go` with `newTaskCmd()` registered as a subcommand group:
```go
rootCmd.AddCommand(newTaskCmd())
// gymctl task new ...
```

### 5. New command: `gymctl objectives <track>`

Displays objective progress for a GX exam track, driven by the exercises and progress data.

```
$ gymctl objectives gx-cs

GX-CS — GIAC Experienced Cyber Security
4 mastered / 8 practiced / 13 not started / 25 total

1. Advanced Network Analysis
   ✓ 1.1 Extract data from pcaps using tcpdump filters
         Tasks: gx-cs-tcpdump-filter-001 (completed, 8:42, no hints)
   ◌ 1.2 Analyze packet captures with Wireshark display filters
         Tasks: none yet
   ...
```

**Logic:**
1. Load the objectives definition from `gx-prep/exams/<track>/objectives.yaml`.
   - Or: embed objective lists in the exam definitions from change #3.
   - Or: derive objectives from discovered tasks' `gxObjective` fields.
   - **Recommendation:** Load from an objectives file path. Add a `--objectives-file` flag, defaulting to `exams/<track>/objectives.yaml` relative to tasks-dir parent.
2. For each objective, find all exercises with matching `gxObjective`.
3. For each exercise, look up progress.
4. Roll up to objective-level status:
   - **mastered** = at least one exercise completed with score >= 90 and time under estimated
   - **practiced** = at least one exercise attempted (started or completed with lower score)
   - **not started** = no exercises attempted for this objective

**Output options:**
- `--output table` (default): the formatted view above
- `--output json`: machine-readable for dashboards
- `--output markdown`: generates the `exams/<track>/README.md` content (this replaces hand-maintaining those READMEs)

**Implementation:** New file `internal/cli/objectives.go`.

### 6. Add `--track` filter to `gymctl next`

Currently `next` picks from all tracks. Add `--track gx-cs` to limit scope:

```
gymctl next --track gx-cs
```

**File:** `internal/cli/next.go` — add a `--track` flag, filter `entries` before partitioning. Small change.

### 7. Support multiple tasks directories

The workspace will have gymctl's built-in `tasks/` (Docker/K8s) AND `gx-prep/tasks/` (GX exams). Support a comma-separated or repeated `--tasks-dir` flag, or an env var `GYMCTL_TASKS_DIRS`.

**File:** `internal/cli/tasks_resolver.go` — update `setupTasksDirectory()` to handle multiple dirs. `LoadCatalog` already works on a single dir, so just call it for each dir and merge results.

This is lower priority — you can also just symlink `gx-prep/tasks/*` into gymctl's tasks dir, or set `--tasks-dir` to point at gx-prep. But native multi-dir support is cleaner.

## Implementation order

Do these in order — each builds on the previous:

1. **`local` environment type** (#1) — unblocks writing and testing GX tasks at all
2. **`gxObjective` field** (#2) — small, needed by everything else
3. **`--track` on `next`** (#6) — tiny change, immediately useful
4. **Generalize exam engine** (#3) — biggest change, but well-contained in exam.go + new exam package
5. **`gymctl task new`** (#4) — authoring workflow
6. **`gymctl objectives`** (#5) — progress dashboard
7. **Multiple tasks dirs** (#7) — nice-to-have, do last

## Testing strategy

- Existing tests in `internal/checks/engine_test.go` should still pass unchanged.
- Add test exercises in `tasks/gx-cs/` with `environment.type: local` and `script`/`file` checks to validate the local flow.
- Exam engine tests: create a small set of GX-CS test exercises, verify domain classification from `gxObjective`, verify weighted selection produces balanced output.
- `task new` tests: verify directory creation, yaml validity of generated skeleton.

## Non-goals for this phase

- No changes to the Docker or Kubernetes environment engines.
- No changes to the check engine (script/file checks already cover GX needs).
- No `gymctl publish` command (that's a later phase, and may live in a separate wrapper).
- No UI/TUI changes to the shell command.
- No changes to the GRIM-9/Jerry narrative system (GX tasks can optionally use it, but it's not required).
