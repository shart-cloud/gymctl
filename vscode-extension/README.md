# Lab Companion VS Code Extension

Private VS Code extension for gymctl-driven labs in Coder workspaces.

## Features

- Detects workspace mode:
  - `GYMCTL_TASKS_DIRS` (preferred)
  - `GYMCTL_TASKS_DIR` (compatibility fallback)
  - `./tasks`
  - `~/.coder/lab-spec.yaml` (with startup retry)
- Sidebar panel with sections for Exercises, Active checks, Hints, and Progress
- Runs checks through gymctl JSON contracts:
  - `gymctl check <exercise> --output json`
  - `gymctl check --spec ~/.coder/lab-spec.yaml --output json`
- Fetches hints with:
  - `gymctl hint <exercise> --output json`
  - `gymctl hint --spec ~/.coder/lab-spec.yaml --output json`
- Watches `~/.gym/progress.yaml` and refreshes automatically

## Local Development

From `vscode-extension/`:

```bash
npm install
npm run compile
```

In VS Code, press `F5` to launch an Extension Development Host.

## Packaging

```bash
npm run package
```

This produces a `.vsix` file in `vscode-extension/`.

## Install in code-server

```bash
code-server --install-extension /path/to/gymctl-lab-companion-<version>.vsix
```

## Notes

- The extension is a UI layer over `gymctl` and does not grade directly.
- It never writes `~/.gym/progress.yaml`; only `gymctl` mutates progress.
