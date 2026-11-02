# Using Octo as a Zero-Configuration CI Smoke Gate

Octo-CLI enables open-source maintainers and engineering teams to turn any repository into a self-verifying, automated smoke gate in GitHub Actions—without maintaining fragile custom bash scripts or hardcoding environment dependencies.

```yaml
- name: Octo CI Smoke Gate
  uses: octo-sh/octo-cli@v1
```

---

## Why Use Octo in CI?

1. **Zero Configuration**: Octo automatically analyzes repository technologies (Node.js, Go, Python, Rust, Java, Docker Compose, etc.) and generates deterministic build, run, and verification steps.
2. **Deterministic Verification**: `octo verify` checks live listening TCP ports, HTTP endpoints, container health, and lockfile integrity.
3. **Actionable Failure Diagnostics**: Runtime failures output structured root-cause explanations and suggested fixes in machine-readable JSON (`--json`) or readable terminal output.
4. **Fast & Lightweight**: Standalone binary installation takes under 2 seconds; static analysis and planning run in milliseconds.
5. **Verified Strategy Caching**: Maintainers can commit `.octo.lock` to ensure CI executes only evidence-backed, verified strategies (`octo verify --strict`).

---

## Quickstart Workflow

Add the official Octo composite GitHub Action to `.github/workflows/smoke-gate.yml` in your repository:

```yaml
name: CI Smoke Gate

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  smoke-gate:
    name: Octo Smoke Gate
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Octo Smoke Gate
        uses: octo-sh/octo-cli@v1
        with:
          command: verify
          args: --json
```

### Parity Between Local AI Agents and CI Gate

Developers and AI coding agents (Cursor, Claude Code, Windsurf) can configure Octo in 1 second locally:

```bash
octo mcp init
```

This connects your repository to agent tools (`octo_verify`, `octo_run_and_verify`, `octo_diagnose`) so agents catch runtime issues before opening a pull request. Adding `uses: octo-sh/octo-cli@v1` to your repository ensures CI enforces the exact same verification gate on every commit.

### Action Configuration Reference

The `octo-sh/octo-cli` action works across Linux (`ubuntu-latest`), macOS (`macos-latest`), and Windows (`windows-latest`) runners:

| Input | Description | Default | Example |
| :--- | :--- | :--- | :--- |
| `version` | Octo release tag to install (`latest`, or specific tag like `v1.2.2`, or `local`) | `latest` | `version: v1.2.2` |
| `working-directory` | Directory to execute Octo commands within | `.` | `working-directory: services/api` |
| `command` | Octo command to run (`verify`, `run`, `inspect`, `plan`) | `verify` | `command: verify` |
| `args` | Additional arguments or flags passed directly to `octo` | `--json` | `args: --strict --json` |

---

## Manual Script Setup (Alternative CI Systems)

For non-GitHub CI environments (such as GitLab CI, Bitbucket Pipelines, or CircleCI) or manual shell steps, install via the standalone install script:

```yaml
      - name: Install Octo CLI
        run: curl -fsSL https://raw.githubusercontent.com/Octo-sh/Octo-CLI/main/install.sh | sh

      - name: Smoke Run & Verify Stack
        run: |
          octo run --detach
          octo verify --json
```

---

## Advanced Configurations

### 1. Enforcing Strict Lockfile Integrity (`--strict`)

In production release pipelines or protected branch gates, enforce that all execution strategies match `.octo.lock` fingerprints without any stale or invalidated candidates:

```yaml
      - name: Enforce Verified Strategy Lockfile
        uses: octo-sh/octo-cli@v1
        with:
          command: verify
          args: --strict
```

If `.octo.lock` is missing or code modifications invalidated existing candidates without re-verification, the gate fails immediately with a descriptive error.

### 2. Multi-Target / Monorepo Smoke Checks

For monorepos with multiple runnable services, select specific components or run all targets:

```yaml
      # Run only specific critical components:
      - name: Smoke Run Frontend & API
        run: octo run --only component:web --only component:api --detach

      # Or execute all detected targets:
      - name: Smoke Run All Components
        run: octo run --all --detach

      - name: Verify All Services
        run: octo verify --json
```

### 3. Ephemeral Container Sandboxing (`--sandbox` / `-s`)

Run the smoke execution inside an ephemeral Docker container for clean environment isolation:

```yaml
      - name: Smoke Run in Container Sandbox
        run: octo run --sandbox --detach
      - name: Verify
        run: octo verify
```

---

## Output & Exit Codes

- **Exit Code 0**: All verification checks passed and lockfile integrity is confirmed.
- **Exit Code 1**: One or more verification checks failed, or `--strict` requirement was violated.
- **JSON Mode (`--json` / `-j`)**: Emits structured JSON to `stdout` containing `project_name`, `success`, `checks`, `error_code`, `root_cause`, and `suggested_fix`.

Example failure output:

```json
{
  "project_name": "web-app",
  "success": false,
  "checks": [
    {
      "check_id": "port.web-app",
      "passed": false,
      "reason": "TCP port 3000 is not reachable: connection refused",
      "error_code": "PORT_CONFLICT",
      "root_cause": "TCP port 3000 is already in use by another process or strictly declared.",
      "suggested_fix": "Free port 3000 (e.g., 'lsof -i :3000' or 'kill <pid>'), or run with port override: octo run --port 3001"
    }
  ],
  "error": "verification \"port.web-app\" failed: TCP port 3000 is not reachable: connection refused",
  "error_code": "PORT_CONFLICT",
  "root_cause": "TCP port 3000 is already in use by another process or strictly declared.",
  "suggested_fix": "Free port 3000 (e.g., 'lsof -i :3000' or 'kill <pid>'), or run with port override: octo run --port 3001"
}
```

