# Commit Message Schema

## Format

```
<type>: <subject>
```

## Types

| Type | Purpose |
|------|---------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation changes |
| `refactor` | Code restructuring without behavior change |
| `test` | Adding or updating tests |
| `chore` | Build, CI, tooling, dependencies |
| `style` | Code formatting (no logic change) |

## Rules

- Subject: imperative mood, lowercase start, no trailing period
- Keep subject line under 72 characters
- Body (optional): separated by blank line, explains why not what
- Reference issues when applicable (e.g., `Closes #123`)

## Examples

```
feat: Add variable expansion support for base_dir
fix: resolve worktree path resolution on Windows
docs: update README with shell integration guide
refactor: extract hook execution into separate package
test: add E2E tests for prune command
chore: bump golangci-lint to v1.62
```
