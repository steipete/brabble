# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.3] - 2026-09-13

**Highlights:** Release binaries launch cleanly from a quarantined download, and source builds keep a valid signature layout.

### Fixed
- Release binaries: bundled whisper.cpp libraries are signed and notarized so quarantined launches no longer hang.
- Source builds with signed Go toolchains keep a valid code-signature layout by placing the runtime search path in the linker flags.

## [0.1.2] - 2026-09-13

**Highlights:** Protect private voice data, restore reproducible macOS builds, and manage Linux systemd user services.

### Added
- Add systemd user-service management and Linux whisper.cpp runtime linkage. (#11, #12)

### Fixed
- Sign and notarize macOS release executables so quarantined downloads can run; set the release minimum to macOS 14.
- Protect local voice data with private transcript, control-socket, and launchd-plist permissions; omit hook environment values from logs.
- Restore documented single-hook configs and `test-hook` routing through the per-wake dispatcher; validate every configured hook in `doctor`.
- Bound metrics request headers to prevent slow-client resource exhaustion.
- Restore reproducible macOS releases by building the binding-matched whisper.cpp revision and bundling its runtime libraries.
- Keep systemd unit tests inside temporary directories on macOS.
- Fix macOS tests failing to load whisper.cpp with signed Go toolchains by linking the native runtime search path into test binaries.

### Changed
- Build with Go 1.27.1 by default while retaining Go 1.27.0 source compatibility.
- Update to Go 1.27 and refresh whisper.cpp with upstream inference fixes; keep its Go binding and native runtime pinned together and update GoReleaser and CI tooling.

## [0.1.1] - 2026-06-11
### Fixed
- Clean up interrupted model downloads, honor configured model state paths, report control and transcript I/O errors, and wait for ASR and control workers during shutdown. (#1, thanks @Wang200935)

### Changed
- Update Go dependencies and make the macOS release build reproducible with pinned whisper.cpp and GoReleaser versions.

## [0.1.0] - 2025-12-03
### Added
- Daemon with PID + UNIX socket; lifecycle commands (`start|stop|restart|status|tail-log|test-hook`).
- Audio pipeline: PortAudio capture, WebRTC VAD, whisper.cpp transcription, wake word (“clawd”), partial flush segments (flagged, not hooked).
- Mic management via `mic list|set` (aliases `mics`/`microphone`, supports `--index`); model management (`models list|download|set`), setup downloads default model.
- Hook runner to `../warelay send` with prefix, envs, cooldown, timeout, queue, PII redaction toggle.
- Config defaults + logging level/format, metrics endpoint, transcript logging toggle.
- Doctor checks deps/model/portaudio; service command writes launchd plist with env; health check.
- Rich, colored Go help output with examples; pnpm scripts for build/run.

[0.1.3]: https://github.com/steipete/brabble/releases/tag/v0.1.3
[0.1.2]: https://github.com/steipete/brabble/releases/tag/v0.1.2
[0.1.1]: https://github.com/steipete/brabble/releases/tag/v0.1.1
[0.1.0]: https://github.com/steipete/brabble/releases/tag/v0.1.0
