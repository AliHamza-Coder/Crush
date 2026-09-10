# Release Checklist & Changelog

Read and update this file before every push to `main` that should ship a release.
The pipeline reads the **topmost** entry under `## History` as the source of truth
for the version and changelog of the next release — not this repo's README, and
not a commit message. If you don't add a new entry here, nothing new ships (the
pipeline just skips quietly, per `version-gate.yml`).

Do not delete or rewrite old entries. They stay as a running changelog.

## Before you push

- [ ] The version in `crush-core/src/lib.rs` (`VERSION` constant) is bumped from the last released version
- [ ] A new entry has been added below under `## History`, dated today, with the same version as the manifest
- [ ] The bullets under the new entry describe real user-facing changes — not "misc fixes" or "various improvements"
- [ ] Any new secrets this release needs are already set in the repo's Settings → Secrets
- [ ] The target platforms below still match what you want built for this release

## Target platforms

- [x] Windows (`crush.exe` — x64)
- [x] Linux (`crush` binary — x64)

## History

<!--
Newest entry goes on top. The pipeline parses ONLY the first "### v..." block
below — everything after the second "### v..." heading is ignored by CI and
exists purely as human-readable history.
-->

### v3.0.0 — 2026-09-09

- Rewritten from Go to Rust with professional TUI (Ratatui + Crossterm)
- Three processing engines: FFmpeg, Native Image, ONNX AI Upscale
- Interactive terminal UI with mouse support, analysis dashboard, live queue
- CLI commands: analyse, install, uninstall, setup, update, ai-check, version
- Direct mode: compress/convert from command line with quality/format options
- Smart backup system with configurable directories
- Cross-platform: Windows and Linux support

