# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2025-12-03
### Added
- Live audio pipeline (macOS): PortAudio capture, WebRTC VAD, segmentation by silence/max duration.
- Whisper.cpp integration behind `-tags whisper` with model loading, language selection, and transcription worker.
- Real microphone enumeration via `list-mics`; `set-mic` now meaningful.
- Updated README and spec to document build/deps (`brew install portaudio`, model download, whisper tag).
- `doctor` command to verify config/model/hook and PortAudio (when whisper build).
- `install-service` to write a user launchd plist; `setup` to fetch default model.
- Hook queue with drop-on-full, metrics endpoint (`/metrics`) optional via config, JSON output for `status` and `list-mics`.
- `reload` command to refresh hook/wake config without restart.

## [0.1.0] - 2025-12-03
### Added
- Daemon skeleton with PID file, UNIX control socket, and lifecycle CLI (`start|stop|restart|status|tail-log|list-mics|set-mic|test-hook`).
- Config defaults and auto-creation (`~/.config/brabble/config.toml`) with macOS state/log paths.
- Hook runner wired to `../warelay send` with hostname-prefixed payload, env vars, and cooldown.
- Wake-word gate (“clawd” default) with stripping before hook dispatch; `min_chars` guard.
- Rolling transcripts and rotating logs under `~/Library/Application Support/brabble/`.
- Stub ASR using stdin plus build-tag placeholder for whisper.cpp integration.
- Project spec at `docs/spec.md` and detailed README.

[0.2.0]: https://github.com/steipete/brabble/releases/tag/v0.2.0
[0.1.0]: https://github.com/steipete/brabble/releases/tag/v0.1.0
