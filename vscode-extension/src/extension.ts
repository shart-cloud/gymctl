import * as os from "node:os";
import * as path from "node:path";
import * as fs from "node:fs/promises";
import { spawn } from "node:child_process";
import * as vscode from "vscode";

import {
  CheckResponse,
  DetectionResult,
  HintResponse,
  ListResponse,
  StatusResponse,
  WorkspaceMode,
} from "./types";

type SectionKey = "exercises" | "active" | "hints" | "progress";
type ItemKind = "section" | "exercise" | "info" | "check";

class LabItem extends vscode.TreeItem {
  constructor(
    label: string,
    collapsibleState: vscode.TreeItemCollapsibleState,
    public readonly kind: ItemKind,
    public readonly sectionKey?: SectionKey,
    public readonly exerciseName?: string,
  ) {
    super(label, collapsibleState);
  }
}

class LabTreeProvider implements vscode.TreeDataProvider<LabItem> {
  private readonly onDidChangeTreeDataEmitter = new vscode.EventEmitter<LabItem | void>();
  readonly onDidChangeTreeData = this.onDidChangeTreeDataEmitter.event;

  mode: WorkspaceMode = "none";
  exercises: ListResponse["exercises"] = [];
  listSummary: ListResponse["summary"] | null = null;
  statusSummary: StatusResponse["summary"] | null = null;
  currentExercise: string | null = null;
  currentTitle: string | null = null;
  lastCheck: CheckResponse | null = null;
  lastHint: HintResponse | null = null;

  refresh(): void {
    this.onDidChangeTreeDataEmitter.fire();
  }

  getTreeItem(element: LabItem): vscode.TreeItem {
    return element;
  }

  async getChildren(element?: LabItem): Promise<LabItem[]> {
    if (!element) {
      return this.getRootItems();
    }

    if (element.kind === "section") {
      switch (element.sectionKey) {
        case "exercises":
          return this.getExerciseItems();
        case "active":
          return this.getActiveItems();
        case "hints":
          return this.getHintItems();
        case "progress":
          return this.getProgressItems();
      }
    }

    return [];
  }

  private getRootItems(): LabItem[] {
    if (this.mode === "none") {
      return [
        new LabItem("No lab detected", vscode.TreeItemCollapsibleState.None, "info"),
        new LabItem("Set GYMCTL_TASKS_DIR, add ./tasks, or provide ~/.coder/lab-spec.yaml", vscode.TreeItemCollapsibleState.None, "info"),
      ];
    }

    const items: LabItem[] = [];
    if (this.mode === "gymctl") {
      items.push(new LabItem("EXERCISES", vscode.TreeItemCollapsibleState.Expanded, "section", "exercises"));
    }
    items.push(new LabItem("ACTIVE", vscode.TreeItemCollapsibleState.Expanded, "section", "active"));
    items.push(new LabItem("HINTS", vscode.TreeItemCollapsibleState.Expanded, "section", "hints"));
    items.push(new LabItem("PROGRESS", vscode.TreeItemCollapsibleState.Expanded, "section", "progress"));
    return items;
  }

  private getExerciseItems(): LabItem[] {
    if (this.mode !== "gymctl") {
      return [];
    }

    return this.exercises.map((ex) => {
      const statusIcon =
        ex.status === "completed"
          ? "$(check)"
          : ex.status === "in-progress"
            ? "$(circle-large-filled)"
            : "$(circle-outline)";
      const lock = ex.locked ? " $(lock-small)" : "";
      const item = new LabItem(`${statusIcon} ${ex.name}${lock}`, vscode.TreeItemCollapsibleState.None, "exercise", undefined, ex.name);
      item.description = ex.difficulty;
      item.tooltip = ex.title;
      item.contextValue = "exercise";
      return item;
    });
  }

  private getActiveItems(): LabItem[] {
    const activeName = this.lastCheck?.exercise ?? this.currentExercise;
    if (!activeName) {
      return [new LabItem("No active exercise", vscode.TreeItemCollapsibleState.None, "info")];
    }

    const title = this.resolveActiveTitle(activeName);
    const items: LabItem[] = [
      new LabItem(`$(target) ${activeName}`, vscode.TreeItemCollapsibleState.None, "info"),
    ];
    if (title && title !== activeName) {
      items.push(new LabItem(title, vscode.TreeItemCollapsibleState.None, "info"));
    }

    if (this.lastCheck && this.lastCheck.exercise === activeName) {
      for (const check of this.lastCheck.checks) {
        const icon = check.passed ? "$(pass-filled)" : "$(error)";
        const label = check.message ? `${icon} ${check.name}: ${check.message}` : `${icon} ${check.name}`;
        items.push(new LabItem(label, vscode.TreeItemCollapsibleState.None, "check"));
      }
      items.push(
        new LabItem(
          `${this.lastCheck.passedCount}/${this.lastCheck.totalCount} checks passing`,
          vscode.TreeItemCollapsibleState.None,
          "info",
        ),
      );
    } else {
      items.push(new LabItem("Run checks to see objectives", vscode.TreeItemCollapsibleState.None, "info"));
    }

    return items;
  }

  private getHintItems(): LabItem[] {
    if (!this.lastHint) {
      return [new LabItem("No hints shown yet", vscode.TreeItemCollapsibleState.None, "info")];
    }
    if (this.lastHint.content === null) {
      return [new LabItem("No hints left", vscode.TreeItemCollapsibleState.None, "info")];
    }

    const nextCost = this.lastHint.nextHintCost === 0 ? "free" : `${this.lastHint.nextHintCost} pts`;
    return [
      new LabItem(`Hint ${this.lastHint.hintIndex}: ${this.lastHint.content}`, vscode.TreeItemCollapsibleState.None, "info"),
      new LabItem(`Remaining: ${this.lastHint.hintsRemaining} (next ${nextCost})`, vscode.TreeItemCollapsibleState.None, "info"),
    ];
  }

  private getProgressItems(): LabItem[] {
    if (this.mode === "gymctl" && this.statusSummary) {
      return [
        new LabItem(
          `${this.statusSummary.completed}/${this.statusSummary.total} complete · ${this.statusSummary.earnedPoints}/${this.statusSummary.totalPoints} pts`,
          vscode.TreeItemCollapsibleState.None,
          "info",
        ),
        new LabItem(
          `${this.statusSummary.inProgress} in progress · ${this.statusSummary.notStarted} not started`,
          vscode.TreeItemCollapsibleState.None,
          "info",
        ),
      ];
    }

    if (this.mode === "coder-lab" && this.lastCheck) {
      return [
        new LabItem(
          `${this.lastCheck.passedCount}/${this.lastCheck.totalCount} objectives passing`,
          vscode.TreeItemCollapsibleState.None,
          "info",
        ),
      ];
    }

    return [new LabItem("Run checks to populate progress", vscode.TreeItemCollapsibleState.None, "info")];
  }

  private resolveActiveTitle(activeName: string): string | null {
    if (this.mode === "coder-lab") {
      return this.currentTitle ?? activeName;
    }
    return this.exercises.find((ex) => ex.name === activeName)?.title ?? activeName;
  }
}

interface ExecResult {
  code: number;
  stdout: string;
  stderr: string;
}

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const provider = new LabTreeProvider();
  vscode.window.createTreeView("gymctl-lab-companion.labView", { treeDataProvider: provider });

  let detection = await detectWorkspace();
  provider.mode = detection.mode;
  await refreshState(provider, detection);

  context.subscriptions.push(
    vscode.commands.registerCommand("gymctl.refresh", async () => {
      detection = await detectWorkspace();
      provider.mode = detection.mode;
      await refreshState(provider, detection);
    }),
    vscode.commands.registerCommand("gymctl.openTerminal", () => {
      const terminal = vscode.window.createTerminal({ name: "gymctl" });
      terminal.show();
    }),
    vscode.commands.registerCommand("gymctl.setActive", async (exerciseName?: string) => {
      if (detection.mode !== "gymctl") {
        return;
      }

      let target = exerciseName;
      if (!target) {
        const pick = await vscode.window.showQuickPick(
          provider.exercises.map((ex) => ({ label: ex.name, description: ex.title, picked: ex.name === provider.currentExercise })),
          { title: "Select active exercise" },
        );
        target = pick?.label;
      }

      if (!target) {
        return;
      }

      const result = await execGymctl(["start", target]);
      if (result.code !== 0) {
        vscode.window.showErrorMessage(result.stderr || `Failed to set active exercise: ${target}`);
        return;
      }

      await refreshState(provider, detection);
    }),
    vscode.commands.registerCommand("gymctl.runChecks", async (exerciseName?: string) => {
      if (detection.mode === "none") {
        vscode.window.showWarningMessage("No lab detected.");
        return;
      }

      const args = ["check"];
      if (detection.mode === "coder-lab" && detection.specPath) {
        args.push("--spec", detection.specPath);
      } else {
        const defaultExercise = provider.exercises.find((ex) => !ex.locked && ex.status !== "completed")?.name;
        const target = exerciseName ?? provider.currentExercise ?? defaultExercise;
        if (target) {
          args.push(target);
        }
      }
      args.push("--output", "json");

      await vscode.window.withProgress(
        { location: vscode.ProgressLocation.Notification, title: "Running gymctl checks..." },
        async () => {
          const result = await execGymctl(args);
          const parsed = parseJSON<CheckResponse>(result.stdout);
          if (!parsed) {
            vscode.window.showErrorMessage(result.stderr || "gymctl output was not valid JSON.");
            return;
          }

          provider.lastCheck = parsed;
          if (parsed.allPassed) {
            vscode.window.showInformationMessage(`Exercise complete: ${parsed.exercise}`);
          }
          await refreshState(provider, detection);
        },
      );
    }),
    vscode.commands.registerCommand("gymctl.nextHint", async () => {
      if (detection.mode === "none") {
        vscode.window.showWarningMessage("No lab detected.");
        return;
      }

      const args = ["hint"];
      if (detection.mode === "coder-lab" && detection.specPath) {
        args.push("--spec", detection.specPath);
      } else {
        const target = provider.currentExercise;
        if (target) {
          args.push(target);
        }
      }
      args.push("--output", "json");

      const result = await execGymctl(args);
      const parsed = parseJSON<HintResponse>(result.stdout);
      if (!parsed) {
        vscode.window.showErrorMessage(result.stderr || "gymctl hint output was not valid JSON.");
        return;
      }

      provider.lastHint = parsed;
      provider.refresh();
    }),
  );

  const progressWatcher = vscode.workspace.createFileSystemWatcher(
    new vscode.RelativePattern(os.homedir(), ".gym/progress.yaml"),
  );
  const refreshOnProgress = async (): Promise<void> => {
    if (detection.mode !== "none") {
      await refreshState(provider, detection);
    }
  };
  progressWatcher.onDidChange(refreshOnProgress);
  progressWatcher.onDidCreate(refreshOnProgress);
  context.subscriptions.push(progressWatcher);
}

async function refreshState(provider: LabTreeProvider, detection: DetectionResult): Promise<void> {
  provider.exercises = [];
  provider.listSummary = null;
  provider.statusSummary = null;
  provider.currentExercise = null;
  provider.currentTitle = null;

  if (detection.mode === "none") {
    provider.refresh();
    return;
  }

  if (detection.mode === "gymctl") {
    const listResult = await execGymctl(["list", "--output", "json"]);
    const list = parseJSON<ListResponse>(listResult.stdout);
    if (list) {
      provider.exercises = list.exercises;
      provider.listSummary = list.summary;
    }

    const statusResult = await execGymctl(["status", "--output", "json"]);
    const status = parseJSON<StatusResponse>(statusResult.stdout);
    if (status) {
      provider.currentExercise = status.current;
      provider.statusSummary = status.summary;
    }
  } else {
    provider.currentTitle = await readSpecTitle(detection.specPath) ?? "Coder Lab";
    provider.currentExercise = provider.lastCheck?.exercise ?? provider.currentTitle;
  }

  provider.refresh();
}

async function detectWorkspace(): Promise<DetectionResult> {
  const envTasksDir = process.env.GYMCTL_TASKS_DIR?.trim();
  if (envTasksDir) {
    return { mode: "gymctl", tasksDir: envTasksDir };
  }

  const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
  if (workspaceFolder) {
    const tasksPath = vscode.Uri.joinPath(workspaceFolder.uri, "tasks");
    if (await pathExists(tasksPath)) {
      return { mode: "gymctl", tasksDir: tasksPath.fsPath };
    }
  }

  const specPath = path.join(os.homedir(), ".coder", "lab-spec.yaml");
  for (let attempt = 0; attempt < 4; attempt++) {
    if (await pathExists(vscode.Uri.file(specPath))) {
      return { mode: "coder-lab", specPath };
    }
    if (attempt < 3) {
      await sleep(3000);
    }
  }

  return { mode: "none" };
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function pathExists(uri: vscode.Uri): Promise<boolean> {
  try {
    await vscode.workspace.fs.stat(uri);
    return true;
  } catch {
    return false;
  }
}

function parseJSON<T>(raw: string): T | null {
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

async function execGymctl(args: string[]): Promise<ExecResult> {
  return new Promise((resolve) => {
    const child = spawn("gymctl", args, { env: process.env });
    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk) => {
      stdout += String(chunk);
    });
    child.stderr.on("data", (chunk) => {
      stderr += String(chunk);
    });

    child.on("close", (code) => {
      resolve({ code: code ?? 1, stdout: stdout.trim(), stderr: stderr.trim() });
    });
    child.on("error", (err) => {
      resolve({ code: 1, stdout: "", stderr: err.message });
    });
  });
}

async function readSpecTitle(specPath?: string): Promise<string | null> {
  if (!specPath) {
    return null;
  }

  try {
    const raw = await fs.readFile(specPath, "utf8");
    const titleMatch = raw.match(/^\s*title:\s*"?(.+?)"?\s*$/m);
    if (titleMatch?.[1]) {
      return titleMatch[1].trim();
    }
    const nameMatch = raw.match(/^\s*name:\s*"?(.+?)"?\s*$/m);
    return nameMatch?.[1]?.trim() ?? null;
  } catch {
    return null;
  }
}

export function deactivate(): void {}
