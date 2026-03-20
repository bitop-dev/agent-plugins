# Roadmap — agent-plugins

## Current state

13 official plugins across 4 runtimes (host, command, mcp, http). All plugins are tested and serve as both real tools and reference implementations for plugin authors.

---

## Near term

### Per-plugin README files
Each plugin should have its own `README.md` covering:
- What it does and what runtime it uses
- Prerequisites (e.g. `gh` installed, Slack webhook configured)
- Config schema reference
- Example profile snippets
- Known limitations

### Version bumps
All plugins are at `0.1.0`. As plugins mature and config schemas stabilize, bump versions and add release notes to this changelog.

### Plugin tests / validation
Add a lightweight validation step to CI: run `agent plugins validate` against each plugin directory to catch broken manifests before they reach the registry.

---

## Medium term

### `slack` v2 — thread and reaction support
Add tools for replying in threads and adding reactions, not just posting top-level messages.

### `grafana-alerts` v2 — alert deduplication
Current output includes repeated alert instances. Group by alert name + host, collapse to "fired N times, last at X" for cleaner agent summaries.

### `github-cli` expansion
Add more tool coverage beyond the basic argv-template wrapper:
- `gh pr list`, `gh pr review`
- `gh issue create`, `gh issue comment`
- Consider moving to a Go binary for richer output shaping

### New plugin: `postgres`
Query a PostgreSQL database. Command runtime with a Go binary. Tools: `db/query`, `db/schema`.

### New plugin: `linear`
Linear issue management. HTTP runtime. Tools: `linear/issues`, `linear/create`, `linear/update`.

### New plugin: `browser`
Headless browser for page fetch and interaction. Command runtime wrapping a Go binary with `chromedp`.

---

## Long term

### Profile packages
Bundle example profiles as installable packages. `agent profiles install research-starter --source official`.

### Plugin dependency resolution
If a plugin declares `requires.plugins`, the registry and CLI should auto-install dependencies when that plugin is installed.

### Plugin versioning and immutable releases
Pin artifact versions to git tags. The registry serves specific versions, not just "whatever's in the directory".
