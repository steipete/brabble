# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-12-03
### Added
- Daemon with PID + UNIX socket; lifecycle commands (`start|stop|restart|status|tail-log|test-hook`).
- Audio pipeline: PortAudio capture, WebRTC VAD, whisper.cpp transcription, wake word (“clawd”), partial flush segments (flagged, not hooked).
- Mic management via `mic list|set` (aliases `mics`/`microphone`, supports `--index`); model management (`models list|download|set`), setup downloads default model.
- Hook runner to `../warelay send` with prefix, envs, cooldown, timeout, queue, PII redaction toggle.
- Config defaults + logging level/format, metrics endpoint, transcript logging toggle.
- Doctor checks deps/model/portaudio; service command writes launchd plist with env; health check.
- Rich, colored Go help output with examples; pnpm scripts for build/run.

[0.1.0]: https://github.com/steipete/brabble/releases/tag/v0.1.0
