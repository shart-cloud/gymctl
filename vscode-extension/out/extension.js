"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const os = __importStar(require("node:os"));
const path = __importStar(require("node:path"));
const fs = __importStar(require("node:fs/promises"));
const node_child_process_1 = require("node:child_process");
const vscode = __importStar(require("vscode"));
class LabItem extends vscode.TreeItem {
    constructor(label, collapsibleState, kind, sectionKey, exerciseName) {
        super(label, collapsibleState);
        this.kind = kind;
        this.sectionKey = sectionKey;
        this.exerciseName = exerciseName;
    }
}
class LabTreeProvider {
    constructor() {
        this.onDidChangeTreeDataEmitter = new vscode.EventEmitter();
        this.onDidChangeTreeData = this.onDidChangeTreeDataEmitter.event;
        this.mode = "none";
        this.exercises = [];
        this.listSummary = null;
        this.statusSummary = null;
        this.currentExercise = null;
        this.currentTitle = null;
        this.lastCheck = null;
        this.lastHint = null;
    }
    refresh() {
        this.onDidChangeTreeDataEmitter.fire();
    }
    getTreeItem(element) {
        return element;
    }
    async getChildren(element) {
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
    getRootItems() {
        if (this.mode === "none") {
            return [
                new LabItem("No lab detected", vscode.TreeItemCollapsibleState.None, "info"),
                new LabItem("Set GYMCTL_TASKS_DIR, add ./tasks, or provide ~/.coder/lab-spec.yaml", vscode.TreeItemCollapsibleState.None, "info"),
            ];
        }
        const items = [];
        if (this.mode === "gymctl") {
            items.push(new LabItem("EXERCISES", vscode.TreeItemCollapsibleState.Expanded, "section", "exercises"));
        }
        items.push(new LabItem("ACTIVE", vscode.TreeItemCollapsibleState.Expanded, "section", "active"));
        items.push(new LabItem("HINTS", vscode.TreeItemCollapsibleState.Expanded, "section", "hints"));
        items.push(new LabItem("PROGRESS", vscode.TreeItemCollapsibleState.Expanded, "section", "progress"));
        return items;
    }
    getExerciseItems() {
        if (this.mode !== "gymctl") {
            return [];
        }
        return this.exercises.map((ex) => {
            const statusIcon = ex.status === "completed"
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
    getActiveItems() {
        const activeName = this.lastCheck?.exercise ?? this.currentExercise;
        if (!activeName) {
            return [new LabItem("No active exercise", vscode.TreeItemCollapsibleState.None, "info")];
        }
        const title = this.resolveActiveTitle(activeName);
        const items = [
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
            items.push(new LabItem(`${this.lastCheck.passedCount}/${this.lastCheck.totalCount} checks passing`, vscode.TreeItemCollapsibleState.None, "info"));
        }
        else {
            items.push(new LabItem("Run checks to see objectives", vscode.TreeItemCollapsibleState.None, "info"));
        }
        return items;
    }
    getHintItems() {
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
    getProgressItems() {
        if (this.mode === "gymctl" && this.statusSummary) {
            return [
                new LabItem(`${this.statusSummary.completed}/${this.statusSummary.total} complete · ${this.statusSummary.earnedPoints}/${this.statusSummary.totalPoints} pts`, vscode.TreeItemCollapsibleState.None, "info"),
                new LabItem(`${this.statusSummary.inProgress} in progress · ${this.statusSummary.notStarted} not started`, vscode.TreeItemCollapsibleState.None, "info"),
            ];
        }
        if (this.mode === "coder-lab" && this.lastCheck) {
            return [
                new LabItem(`${this.lastCheck.passedCount}/${this.lastCheck.totalCount} objectives passing`, vscode.TreeItemCollapsibleState.None, "info"),
            ];
        }
        return [new LabItem("Run checks to populate progress", vscode.TreeItemCollapsibleState.None, "info")];
    }
    resolveActiveTitle(activeName) {
        if (this.mode === "coder-lab") {
            return this.currentTitle ?? activeName;
        }
        return this.exercises.find((ex) => ex.name === activeName)?.title ?? activeName;
    }
}
async function activate(context) {
    const provider = new LabTreeProvider();
    vscode.window.createTreeView("gymctl-lab-companion.labView", { treeDataProvider: provider });
    let detection = await detectWorkspace();
    provider.mode = detection.mode;
    await refreshState(provider, detection);
    context.subscriptions.push(vscode.commands.registerCommand("gymctl.refresh", async () => {
        detection = await detectWorkspace();
        provider.mode = detection.mode;
        await refreshState(provider, detection);
    }), vscode.commands.registerCommand("gymctl.openTerminal", () => {
        const terminal = vscode.window.createTerminal({ name: "gymctl" });
        terminal.show();
    }), vscode.commands.registerCommand("gymctl.setActive", async (exerciseName) => {
        if (detection.mode !== "gymctl") {
            return;
        }
        let target = exerciseName;
        if (!target) {
            const pick = await vscode.window.showQuickPick(provider.exercises.map((ex) => ({ label: ex.name, description: ex.title, picked: ex.name === provider.currentExercise })), { title: "Select active exercise" });
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
    }), vscode.commands.registerCommand("gymctl.runChecks", async (exerciseName) => {
        if (detection.mode === "none") {
            vscode.window.showWarningMessage("No lab detected.");
            return;
        }
        const args = ["check"];
        if (detection.mode === "coder-lab" && detection.specPath) {
            args.push("--spec", detection.specPath);
        }
        else {
            const defaultExercise = provider.exercises.find((ex) => !ex.locked && ex.status !== "completed")?.name;
            const target = exerciseName ?? provider.currentExercise ?? defaultExercise;
            if (target) {
                args.push(target);
            }
        }
        args.push("--output", "json");
        await vscode.window.withProgress({ location: vscode.ProgressLocation.Notification, title: "Running gymctl checks..." }, async () => {
            const result = await execGymctl(args);
            const parsed = parseJSON(result.stdout);
            if (!parsed) {
                vscode.window.showErrorMessage(result.stderr || "gymctl output was not valid JSON.");
                return;
            }
            provider.lastCheck = parsed;
            if (parsed.allPassed) {
                vscode.window.showInformationMessage(`Exercise complete: ${parsed.exercise}`);
            }
            await refreshState(provider, detection);
        });
    }), vscode.commands.registerCommand("gymctl.nextHint", async () => {
        if (detection.mode === "none") {
            vscode.window.showWarningMessage("No lab detected.");
            return;
        }
        const args = ["hint"];
        if (detection.mode === "coder-lab" && detection.specPath) {
            args.push("--spec", detection.specPath);
        }
        else {
            const target = provider.currentExercise;
            if (target) {
                args.push(target);
            }
        }
        args.push("--output", "json");
        const result = await execGymctl(args);
        const parsed = parseJSON(result.stdout);
        if (!parsed) {
            vscode.window.showErrorMessage(result.stderr || "gymctl hint output was not valid JSON.");
            return;
        }
        provider.lastHint = parsed;
        provider.refresh();
    }));
    const progressWatcher = vscode.workspace.createFileSystemWatcher(new vscode.RelativePattern(os.homedir(), ".gym/progress.yaml"));
    const refreshOnProgress = async () => {
        if (detection.mode !== "none") {
            await refreshState(provider, detection);
        }
    };
    progressWatcher.onDidChange(refreshOnProgress);
    progressWatcher.onDidCreate(refreshOnProgress);
    context.subscriptions.push(progressWatcher);
}
async function refreshState(provider, detection) {
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
        const list = parseJSON(listResult.stdout);
        if (list) {
            provider.exercises = list.exercises;
            provider.listSummary = list.summary;
        }
        const statusResult = await execGymctl(["status", "--output", "json"]);
        const status = parseJSON(statusResult.stdout);
        if (status) {
            provider.currentExercise = status.current;
            provider.statusSummary = status.summary;
        }
    }
    else {
        provider.currentTitle = await readSpecTitle(detection.specPath) ?? "Coder Lab";
        provider.currentExercise = provider.lastCheck?.exercise ?? provider.currentTitle;
    }
    provider.refresh();
}
async function detectWorkspace() {
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
function sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
}
async function pathExists(uri) {
    try {
        await vscode.workspace.fs.stat(uri);
        return true;
    }
    catch {
        return false;
    }
}
function parseJSON(raw) {
    try {
        return JSON.parse(raw);
    }
    catch {
        return null;
    }
}
async function execGymctl(args) {
    return new Promise((resolve) => {
        const child = (0, node_child_process_1.spawn)("gymctl", args, { env: process.env });
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
async function readSpecTitle(specPath) {
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
    }
    catch {
        return null;
    }
}
function deactivate() { }
//# sourceMappingURL=extension.js.map