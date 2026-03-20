# agent-plugins/

The `agent-plugins/` directory contains local plugin bundles used alongside the core `agent` repository for development, testing, and examples.

Important distinction:

- this directory is intentionally separate from the core `agent` repository
- but the bundles inside it are not core agent runtime code
- they are plugin manifests and assets that the agent can discover locally

So in practice:

- `../agent/cmd/agent` and `../agent/internal/` are the core framework and host
- `agent-plugins/` contains example or first-party plugin bundles
- plugin runtime executables still live separately under `../agent/_testing/runtimes/`, for example:
  - `_testing/runtimes/send-email-plugin`
  - `_testing/runtimes/web-research-plugin`
- plugin-owned example profiles now live with the matching plugin package, for example:
  - `send-email/examples/profiles/`
  - `web-research/examples/profiles/`
  - `spawn-sub-agent/examples/profiles/`

This keeps the core agent small while giving plugin bundles their own home.

If we build a real plugin registry later, this repository is the natural starting point for published plugin packages.

The core `agent` repository still keeps testing profiles under `../agent/_testing/profiles/`.

Registry server planning and build docs:

- `../agent-registry/README.md`
- `../agent-registry/docs/plugin-registry-server-plan.md`
- `../agent-registry/docs/plugin-registry-contract.md`
- `../agent-registry/docs/registry-server-build-guide.md`
