import * as os from "node:os";
import * as path from "node:path";
import * as fs from "node:fs/promises";
import { spawn } from "node:child_process";
import * as vscode from "vscode";

import {
  CheckResponse,
  DetectionResult,
  DescribeResponse,
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
      if (this.lastCheck.checks.length > 0) {
        items.push(new LabItem("Practical Checks", vscode.TreeItemCollapsibleState.None, "info"));
        for (const check of this.lastCheck.checks) {
          const icon = check.passed ? "$(pass-filled)" : "$(error)";
          const label = check.message ? `${icon} ${check.name}: ${check.message}` : `${icon} ${check.name}`;
          items.push(new LabItem(label, vscode.TreeItemCollapsibleState.None, "check"));
        }
      }

      if (this.lastCheck.mcqs && this.lastCheck.mcqs.length > 0) {
        items.push(new LabItem("Multiple Choice", vscode.TreeItemCollapsibleState.None, "info"));
        for (const mcq of this.lastCheck.mcqs) {
          const icon = mcq.passed ? "$(pass-filled)" : "$(error)";
          const label = mcq.message ? `${icon} ${mcq.id}: ${mcq.message}` : `${icon} ${mcq.id}`;
          items.push(new LabItem(label, vscode.TreeItemCollapsibleState.None, "check"));
        }
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

interface LabPanelState {
  exerciseName?: string;
  describe: DescribeResponse;
}

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const provider = new LabTreeProvider();
  vscode.window.createTreeView("gymctl-lab-companion.labView", { treeDataProvider: provider });
  let labPanel: vscode.WebviewPanel | undefined;
  let labPanelState: LabPanelState | null = null;

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
    vscode.commands.registerCommand("gymctl.openLab", async (exerciseName?: string) => {
      if (detection.mode === "none") {
        vscode.window.showWarningMessage("No lab detected.");
        return;
      }

      const describe = await loadDescribeResponse(provider, detection, exerciseName);
      if (!describe) {
        return;
      }
      const resolvedExerciseName = detection.mode === "coder-lab"
        ? provider.currentExercise ?? describe.name
        : exerciseName ?? provider.currentExercise ?? describe.name;

      if (!labPanel) {
        labPanel = vscode.window.createWebviewPanel(
          "gymctlLabReader",
          describe.title || describe.name,
          vscode.ViewColumn.Beside,
          { enableScripts: true, retainContextWhenHidden: true },
        );
        labPanel.webview.onDidReceiveMessage(async (message) => {
          if (message?.type !== "select-answer" || !labPanelState?.describe.labPath) {
            return;
          }

          try {
            await updateLabAnswer(labPanelState.describe.labPath, String(message.questionId ?? ""), String(message.letter ?? ""));

            const exerciseForChecks = detection.mode === "coder-lab"
              ? labPanelState.exerciseName
              : labPanelState.exerciseName ?? provider.currentExercise ?? labPanelState.describe.name;

            const refreshedDescribe = await loadDescribeResponse(provider, detection, labPanelState.exerciseName);
            if (!refreshedDescribe) {
              return;
            }

            if (exerciseForChecks) {
              const latestCheck = await runChecksForExercise(provider, detection, exerciseForChecks, false);
              if (latestCheck) {
                provider.lastCheck = latestCheck;
              }
            }

            labPanelState = {
              exerciseName: labPanelState.exerciseName,
              describe: refreshedDescribe,
            };
            const currentPanel = labPanel;
            if (!currentPanel) {
              return;
            }
            currentPanel.title = refreshedDescribe.title || refreshedDescribe.name;
            currentPanel.webview.html = renderLabWebview(currentPanel.webview, refreshedDescribe, provider.lastCheck);
            provider.refresh();
          } catch (err) {
            const messageText = err instanceof Error ? err.message : String(err);
            vscode.window.showErrorMessage(`Failed to update lab answer: ${messageText}`);
          }
        });
        labPanel.onDidDispose(() => {
          labPanel = undefined;
          labPanelState = null;
        });
      } else {
        labPanel.reveal(vscode.ViewColumn.Beside, true);
        labPanel.title = describe.title || describe.name;
      }

      labPanelState = { exerciseName: resolvedExerciseName, describe };
      labPanel.webview.html = renderLabWebview(labPanel.webview, describe, provider.lastCheck);
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

async function loadDescribeResponse(
  provider: LabTreeProvider,
  detection: DetectionResult,
  exerciseName?: string,
): Promise<DescribeResponse | null> {
  const args = ["describe", "--output", "json"];
  if (detection.mode === "coder-lab" && detection.specPath) {
    args.push("--spec", detection.specPath);
  } else {
    const defaultExercise = provider.exercises.find((ex) => !ex.locked && ex.status !== "completed")?.name;
    const target = exerciseName ?? provider.currentExercise ?? defaultExercise;
    if (target) {
      args.push(target);
    }
  }

  const result = await execGymctl(args);
  const parsed = parseJSON<DescribeResponse>(result.stdout);
  if (!parsed) {
    vscode.window.showErrorMessage(result.stderr || "gymctl describe output was not valid JSON.");
    return null;
  }
  return parsed;
}

async function runChecksForExercise(
  provider: LabTreeProvider,
  detection: DetectionResult,
  exerciseName?: string,
  notifyOnPass = true,
): Promise<CheckResponse | null> {
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

  const result = await execGymctl(args);
  const parsed = parseJSON<CheckResponse>(result.stdout);
  if (!parsed) {
    if (result.stderr) {
      vscode.window.showErrorMessage(result.stderr || "gymctl output was not valid JSON.");
    }
    return null;
  }

  if (notifyOnPass && parsed.allPassed) {
    vscode.window.showInformationMessage(`Exercise complete: ${parsed.exercise}`);
  }
  return parsed;
}

async function updateLabAnswer(labPath: string, questionID: string, letter: string): Promise<void> {
  const raw = await fs.readFile(labPath, "utf8");
  const updated = setMCQSelection(raw, questionID, letter.toUpperCase());
  if (updated === raw) {
    throw new Error(`question ${questionID} not found in ${labPath}`);
  }
  await fs.writeFile(labPath, updated, "utf8");
}

function setMCQSelection(content: string, questionID: string, letter: string): string {
  const lines = content.replace(/\r\n/g, "\n").split("\n");
  const openFencePattern = /^ {0,3}([`~]{3,})(.*)$/;
  const optionLinePattern = /^(\s*)- \[([ xX])\](.*)$/;
  const idPattern = /(^|\s)id=([^\s]+)/;

  let updated = false;
  for (let i = 0; i < lines.length; i++) {
    const open = lines[i].match(openFencePattern);
    if (!open) {
      continue;
    }

    const info = open[2].trim();
    const fields = info.split(/\s+/).filter(Boolean);
    if (fields[0] !== "mcq") {
      continue;
    }

    const idMatch = info.match(idPattern);
    if (!idMatch || idMatch[2] !== questionID) {
      continue;
    }

    const fenceChar = open[1][0];
    const fenceLen = open[1].length;
    let optionIndex = 0;

    for (let j = i + 1; j < lines.length; j++) {
      const trimmed = lines[j].trim();
      if (isClosingFenceLine(trimmed, fenceChar, fenceLen)) {
        updated = true;
        i = j;
        break;
      }

      const optionMatch = lines[j].match(optionLinePattern);
      if (!optionMatch) {
        continue;
      }

      const targetLetter = String.fromCharCode(65 + optionIndex);
      lines[j] = `${optionMatch[1]}- [${targetLetter === letter ? "x" : " "}]${optionMatch[3]}`;
      optionIndex++;
    }

    break;
  }

  return updated ? lines.join("\n") : content;
}

function isClosingFenceLine(line: string, fenceChar: string, fenceLen: number): boolean {
  if (!line.startsWith(fenceChar)) {
    return false;
  }

  let count = 0;
  while (count < line.length && line[count] === fenceChar) {
    count++;
  }
  if (count < fenceLen) {
    return false;
  }
  return line.slice(count).trim() === "";
}

function renderLabWebview(
  webview: vscode.Webview,
  describe: DescribeResponse,
  lastCheck: CheckResponse | null,
): string {
  const nonce = String(Date.now());
  const body = (describe.labSections ?? [])
    .map((section) => {
      if (section.type === "mcq" && section.mcq) {
        return renderMCQSection(section.mcq, lastCheck);
      }
      return renderMarkdownSection(section.markdown ?? "");
    })
    .join("\n");

  const fallback = !body
    ? `<section class="card"><h2>Lab content unavailable</h2><p>This exercise does not have a sibling <code>lab.md</code> file to render.</p></section>`
    : body;

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src ${webview.cspSource} https:; style-src 'unsafe-inline' ${webview.cspSource}; script-src 'nonce-${nonce}';" />
  <title>${escapeHtml(describe.title || describe.name)}</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #0b0e14;
      --bg-alt: #141922;
      --panel: rgba(12, 17, 24, 0.82);
      --panel-strong: rgba(18, 24, 34, 0.95);
      --fg: #eef2f7;
      --muted: #a7b0bf;
      --border: rgba(169, 182, 205, 0.18);
      --accent: #8bd3ff;
      --accent-strong: #c3f36b;
      --accent-soft: rgba(139, 211, 255, 0.12);
      --card: rgba(14, 19, 28, 0.88);
      --card-edge: rgba(255, 255, 255, 0.05);
      --success: var(--vscode-testing-iconPassed, #2ea043);
      --danger: var(--vscode-testing-iconFailed, #d73a49);
      --shadow: 0 24px 80px rgba(0, 0, 0, 0.34);
    }
    body {
      margin: 0;
      padding: 24px 18px 40px;
      background:
        radial-gradient(circle at top left, rgba(139, 211, 255, 0.1), transparent 30%),
        radial-gradient(circle at top right, rgba(195, 243, 107, 0.08), transparent 24%),
        linear-gradient(180deg, #0a0d12 0%, #0d1118 45%, #0b0e14 100%);
      color: var(--fg);
      font: 15px/1.7 Georgia, "Iowan Old Style", "Palatino Linotype", "Book Antiqua", serif;
    }
    main {
      max-width: 980px;
      margin: 0 auto;
    }
    .hero, .card, .mcq-card {
      border: 1px solid var(--border);
      background:
        linear-gradient(180deg, var(--panel-strong), var(--panel)),
        var(--card);
      border-radius: 20px;
      padding: 24px;
      margin-bottom: 18px;
      box-shadow: var(--shadow);
      position: relative;
      overflow: hidden;
    }
    .hero::before, .card::before, .mcq-card::before {
      content: "";
      position: absolute;
      inset: 0;
      border-radius: inherit;
      padding: 1px;
      background: linear-gradient(135deg, rgba(139, 211, 255, 0.28), transparent 35%, rgba(195, 243, 107, 0.18));
      -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
      -webkit-mask-composite: xor;
      mask-composite: exclude;
      pointer-events: none;
    }
    .eyebrow {
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.18em;
      font-size: 11px;
      margin-bottom: 10px;
      font-family: "Trebuchet MS", "Segoe UI", sans-serif;
    }
    h1, h2, h3, h4, h5, h6 { line-height: 1.14; }
    h1 {
      margin: 0 0 14px;
      font-size: clamp(2.4rem, 6vw, 4.4rem);
      letter-spacing: -0.04em;
      max-width: 12ch;
    }
    h2 {
      margin: 0 0 10px;
      font-size: 1.6rem;
      letter-spacing: -0.03em;
    }
    h3, h4, h5, h6 {
      margin-top: 1.2em;
      margin-bottom: 0.5em;
    }
    .meta {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 18px;
    }
    .meta span, .status, .reader-chip {
      border: 1px solid var(--border);
      border-radius: 999px;
      padding: 6px 12px;
      color: var(--muted);
      font-size: 11px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      font-family: "Trebuchet MS", "Segoe UI", sans-serif;
      background: rgba(255, 255, 255, 0.02);
    }
    .status.pass { color: var(--success); }
    .status.fail { color: var(--danger); }
    .hero-grid {
      display: grid;
      grid-template-columns: minmax(0, 1.6fr) minmax(220px, 0.9fr);
      gap: 20px;
      align-items: end;
    }
    .hero-side {
      display: grid;
      gap: 14px;
      align-self: stretch;
    }
    .metric {
      padding: 14px 16px;
      border: 1px solid var(--border);
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.025);
    }
    .metric-label {
      display: block;
      font-size: 11px;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: var(--muted);
      font-family: "Trebuchet MS", "Segoe UI", sans-serif;
      margin-bottom: 6px;
    }
    .metric-value {
      font-size: 1.3rem;
      letter-spacing: -0.03em;
    }
    pre {
      padding: 16px;
      border-radius: 14px;
      overflow-x: auto;
      background: rgba(0, 0, 0, 0.26);
      border: 1px solid rgba(255, 255, 255, 0.05);
    }
    code {
      font-family: var(--vscode-editor-font-family, "Cascadia Code", monospace);
    }
    p, ul, ol { margin: 0 0 14px; }
    li { margin: 6px 0; }
    .reader-copy {
      color: #d8deea;
      font-size: 1.02rem;
    }
    .reader-copy p:first-of-type {
      font-size: 1.12rem;
      color: #f3f6fb;
    }
    .mcq-card {
      padding-top: 22px;
    }
    .mcq-head {
      display: flex;
      justify-content: space-between;
      gap: 14px;
      align-items: center;
      margin-bottom: 12px;
    }
    .mcq-question {
      font-size: 1.2rem;
      margin: 0 0 10px;
      max-width: 48rem;
    }
    .mcq-actions {
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
      margin-top: 16px;
    }
    .option {
      display: flex;
      gap: 14px;
      align-items: flex-start;
      padding: 14px 16px;
      border: 1px solid var(--border);
      border-radius: 16px;
      margin-top: 10px;
      background: rgba(255, 255, 255, 0.025);
      transition: transform 120ms ease, border-color 120ms ease, background 120ms ease, box-shadow 120ms ease;
      cursor: pointer;
    }
    .option:hover {
      transform: translateY(-1px);
      border-color: rgba(139, 211, 255, 0.42);
      box-shadow: 0 10px 24px rgba(0, 0, 0, 0.16);
    }
    .option.selected {
      border-color: rgba(139, 211, 255, 0.72);
      background:
        linear-gradient(135deg, rgba(139, 211, 255, 0.13), rgba(195, 243, 107, 0.08)),
        rgba(255, 255, 255, 0.03);
    }
    .letter {
      width: 34px;
      height: 34px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      border-radius: 999px;
      border: 1px solid var(--border);
      font-weight: 700;
      flex: 0 0 auto;
      font-family: "Trebuchet MS", "Segoe UI", sans-serif;
    }
    .selected .letter {
      border-color: var(--accent);
      color: #05111a;
      background: linear-gradient(135deg, var(--accent), #d6fbff);
    }
    .option-copy { flex: 1 1 auto; }
    .option-label {
      font-family: "Trebuchet MS", "Segoe UI", sans-serif;
      font-size: 11px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.12em;
      margin-bottom: 4px;
    }
    .option-text {
      color: #eef2f8;
      font-size: 1rem;
    }
    button.option {
      width: 100%;
      text-align: left;
      color: inherit;
      font: inherit;
    }
    button.option:focus-visible {
      outline: 2px solid var(--accent-strong);
      outline-offset: 2px;
    }
    .panel-note {
      margin-top: 16px;
      color: var(--muted);
      font-size: 0.92rem;
    }
    a { color: var(--accent); }
    @media (max-width: 820px) {
      body { padding: 16px 12px 28px; }
      .hero-grid { grid-template-columns: 1fr; }
      .hero, .card, .mcq-card { padding: 18px; border-radius: 18px; }
      h1 { max-width: none; }
      .mcq-head { flex-direction: column; align-items: flex-start; }
    }
  </style>
</head>
<body>
  <main>
    <section class="hero">
      <div class="hero-grid">
        <div>
          <div class="eyebrow">Field Manual</div>
          <h1>${escapeHtml(describe.title || describe.name)}</h1>
          ${describe.description ? `<p class="reader-copy">${escapeHtml(describe.description)}</p>` : ""}
          <div class="meta">
            <span>${escapeHtml(describe.name)}</span>
            <span>${escapeHtml(describe.difficulty)}</span>
            ${describe.estimatedTime ? `<span>${escapeHtml(describe.estimatedTime)}</span>` : ""}
            <span>${escapeHtml(String(describe.points))} pts</span>
          </div>
        </div>
        <div class="hero-side">
          <div class="metric">
            <span class="metric-label">Reader Mode</span>
            <div class="metric-value">Interactive MCQ</div>
          </div>
          <div class="metric">
            <span class="metric-label">Source File</span>
            <div class="metric-value">${escapeHtml(describe.labPath ? path.basename(describe.labPath) : "lab.md unavailable")}</div>
          </div>
        </div>
      </div>
    </section>
    ${fallback}
  </main>
  <script nonce="${nonce}">
    const vscode = acquireVsCodeApi();
    for (const option of document.querySelectorAll('[data-question-id][data-letter]')) {
      option.addEventListener('click', () => {
        const button = option;
        vscode.postMessage({
          type: 'select-answer',
          questionId: button.getAttribute('data-question-id'),
          letter: button.getAttribute('data-letter'),
        });
      });
    }
  </script>
</body>
</html>`;
}

function renderMCQSection(
  mcq: {
    id: string;
    question?: string;
    options: Array<{ letter: string; text: string; selected: boolean }>;
    selected?: string[];
  },
  lastCheck: CheckResponse | null,
): string {
  const result = lastCheck?.mcqs?.find((item) => item.id === mcq.id);
  const statusClass = result ? (result.passed ? "pass" : "fail") : "";
  const statusLabel = result
    ? result.passed
      ? "Passed"
      : result.message
        ? `Not passed: ${result.message}`
        : "Not passed"
    : mcq.selected && mcq.selected.length > 0
      ? `Selected: ${mcq.selected.join(", ")}`
      : "No answer selected";

  return `<section class="mcq-card">
    <div class="mcq-head">
      <div>
        <div class="eyebrow">Multiple Choice</div>
        <h2>${escapeHtml(mcq.id)}</h2>
      </div>
      <div class="status ${statusClass}">${escapeHtml(statusLabel)}</div>
    </div>
    ${mcq.question ? `<p class="mcq-question">${escapeHtml(mcq.question)}</p>` : ""}
    ${mcq.options
      .map(
        (option) => `<button class="option ${option.selected ? "selected" : ""}" type="button" data-question-id="${escapeHtml(mcq.id)}" data-letter="${escapeHtml(option.letter)}">
          <span class="letter">${escapeHtml(option.letter)}</span>
          <span class="option-copy">
            <span class="option-label">Option ${escapeHtml(option.letter)}</span>
            <span class="option-text">${escapeHtml(option.text)}</span>
          </span>
        </button>`,
      )
      .join("")}
    <div class="panel-note">Select an option to write your answer back to <code>lab.md</code> and refresh the grading state.</div>
  </section>`;
}

function renderMarkdownSection(markdown: string): string {
  if (!markdown.trim()) {
    return "";
  }
  return `<section class="card"><div class="reader-copy">${renderSimpleMarkdown(markdown)}</div></section>`;
}

function renderSimpleMarkdown(markdown: string): string {
  const lines = markdown.replace(/\r\n/g, "\n").split("\n");
  const html: string[] = [];
  let paragraph: string[] = [];
  let unordered: string[] = [];
  let ordered: string[] = [];
  let inCode = false;
  let codeLines: string[] = [];

  const flushParagraph = (): void => {
    if (paragraph.length === 0) {
      return;
    }
    html.push(`<p>${paragraph.map(escapeHtml).join(" ")}</p>`);
    paragraph = [];
  };
  const flushUnordered = (): void => {
    if (unordered.length === 0) {
      return;
    }
    html.push(`<ul>${unordered.map((item) => `<li>${escapeHtml(item)}</li>`).join("")}</ul>`);
    unordered = [];
  };
  const flushOrdered = (): void => {
    if (ordered.length === 0) {
      return;
    }
    html.push(`<ol>${ordered.map((item) => `<li>${escapeHtml(item)}</li>`).join("")}</ol>`);
    ordered = [];
  };
  const flushAll = (): void => {
    flushParagraph();
    flushUnordered();
    flushOrdered();
  };

  for (const rawLine of lines) {
    const line = rawLine.trimEnd();

    if (line.startsWith("```")) {
      flushAll();
      if (inCode) {
        html.push(`<pre><code>${escapeHtml(codeLines.join("\n"))}</code></pre>`);
        codeLines = [];
        inCode = false;
      } else {
        inCode = true;
      }
      continue;
    }

    if (inCode) {
      codeLines.push(rawLine);
      continue;
    }

    if (!line.trim()) {
      flushAll();
      continue;
    }

    const heading = line.match(/^(#{1,6})\s+(.+)$/);
    if (heading) {
      flushAll();
      const level = heading[1].length;
      html.push(`<h${level}>${escapeHtml(heading[2])}</h${level}>`);
      continue;
    }

    const unorderedItem = line.match(/^[-*]\s+(.+)$/);
    if (unorderedItem) {
      flushParagraph();
      flushOrdered();
      unordered.push(unorderedItem[1]);
      continue;
    }

    const orderedItem = line.match(/^\d+\.\s+(.+)$/);
    if (orderedItem) {
      flushParagraph();
      flushUnordered();
      ordered.push(orderedItem[1]);
      continue;
    }

    paragraph.push(line.trim());
  }

  flushAll();
  if (inCode) {
    html.push(`<pre><code>${escapeHtml(codeLines.join("\n"))}</code></pre>`);
  }
  return html.join("\n");
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
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
