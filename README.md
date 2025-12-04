# 🎙️ Brabble — Open hailing frequencies… and run the command.

Always-on, local-only voice daemon for macOS. Hears your wake word (“clawd” by default), transcribes with whisper.cpp, then fires a configurable hook (default: `../warelay send "Voice brabble from ${hostname}: <text>"`). Written in Go; ships with a daemon lifecycle, status socket, and launchd helper.

## Quick start
- Requirements (full audio build): Go 1.25+, `brew install portaudio pkg-config`, a whisper.cpp model.
- One-liner: `pnpm brabble setup && pnpm start` (downloads medium Q5_1, writes config, starts daemon).
- Foreground test without audio deps: `go run ./cmd/brabble serve` then type lines containing “clawd”.

## CLI surface
- `start | stop | restart` — daemon lifecycle (PID + UNIX socket).
- `status [--json]` — uptime + last transcripts; `tail-log` shows recent logs.
- `mic list|set [--index N]` — enumerate or select microphone (aliases: `mics`, `microphone`).
- `models list|download|set` — manage whisper.cpp models under `~/Library/Application Support/brabble/models`.
- `setup` — download default model and update config; `doctor` — check deps/model/hook/portaudio.
- `test-hook "text"` — invoke hook manually; `health` — ping daemon; `service install|uninstall|status` — launchd helper (prints kickstart/bootout commands).
- `transcribe <wav>` — run whisper on a WAV file; add `--hook` to send it through your configured hook (respects wake/min_chars unless `--no-wake`).
- Hidden internal: `serve` runs the foreground daemon (used by `start`/launchd).
- `--metrics-addr` enables Prometheus text endpoint; `--no-wake` bypasses wake word.

## PNPM helpers (all build Go, no JS runtime)
- `pnpm brabble` — build (whisper) then start daemon (default); extra args pass through, e.g. `pnpm brabble --help`, `pnpm brabble status`.
- `pnpm start|stop|restart` — lifecycle wrappers.
- `pnpm build` — whisper build to `bin/brabble`; `pnpm build-stub` — stub build without audio deps; `pnpm lint` — `golangci-lint run`; `pnpm format` — `gofmt -w .`; `pnpm test` — `go test ./...`.
- Lint deps: `brew install golangci-lint`; CI runs gofmt+golangci-lint+tests (see `.github/workflows/ci.yml`).

## File-based testing
- Transcribe without the daemon: `pnpm brabble transcribe samples/clip.wav`
- Send through your hook (wake+min_chars enforced): `pnpm brabble transcribe samples/clip.wav --hook`
- Ignore wake gating for a file: `pnpm brabble transcribe samples/clip.wav --hook --no-wake`
- Input: any WAV; we downmix to mono and resample to 16 kHz internally.

## Config (auto-created at `~/.config/brabble/config.toml`)
```toml
[audio]
device_name = ""
device_index = -1
sample_rate = 16000
channels = 1
frame_ms = 20          # 10/20/30 only

[vad]
enabled = true
silence_ms = 1000      # end-of-speech detector
aggressiveness = 2
min_speech_ms = 300
max_segment_ms = 10000
partial_flush_ms = 4000  # emit partial segments (not sent to hook)

[asr]
model_path = "~/Library/Application Support/brabble/models/ggml-medium-q5_1.bin"
language = "auto"
compute_type = "q5_1"
device = "auto"       # auto/metal/cpu

[wake]
enabled = true
word = "clawd"
sensitivity = 0.6

[hook]
command = "../warelay"
args = ["send"]
prefix = "Voice brabble from ${hostname}: "
cooldown_sec = 1
min_chars = 24
max_latency_ms = 5000
queue_size = 16
timeout_sec = 5
redact_pii = false
env = {}

[logging]
level = "info"   # debug|info|warn|error
format = "text"  # text|json

[metrics]
enabled = false
addr = "127.0.0.1:9317"

[transcripts]
enabled = true
```
State & logs: `~/Library/Application Support/brabble/` (pid, socket, logs, transcripts, models).

## Models
- Registry: `ggml-small-q5_1.bin`, `ggml-medium-q5_1.bin` (default), `ggml-large-v3-q5_0.bin`.
- `brabble models download <name>` fetches to the models dir; `brabble models set <name|path>` updates config.
- `brabble setup` fetches the default model and writes `asr.model_path`; reruns `doctor` afterward.

## Audio & wake
- PortAudio capture → WebRTC VAD → partial segments every `partial_flush_ms` (suppressed from hook) → final segment; retries device open on failure.
- Wake word (case-insensitive) is stripped before dispatch; disable with `--no-wake` or `BRABBLE_WAKE_ENABLED=0`.
- Partial transcripts are logged with `Partial=true` and skipped by the hook; full segments respect `hook.min_chars` and cooldown.

## Hook
- Default hook: `../warelay send "<prefix><text>"`, prefix includes hostname.
- Extra env: `BRABBLE_TEXT`, `BRABBLE_PREFIX` plus any `hook.env`; redaction toggle masks obvious emails/phones.
- Queue + timeout + cooldown prevent flooding; `test-hook` is the dry-run.

## Service (launchd)
- `brabble service install --env KEY=VAL` writes `~/Library/LaunchAgents/com.brabble.agent.plist` and prints:
  - `launchctl load -w <plist>`
  - `launchctl kickstart gui/$(id -u)/com.brabble.agent`
  - `launchctl bootout gui/$(id -u)/com.brabble.agent`
- `service status` reports whether the plist exists; `service uninstall` removes the plist file.

## Env overrides
`BRABBLE_WAKE_ENABLED`, `BRABBLE_METRICS_ADDR`, `BRABBLE_LOG_LEVEL`, `BRABBLE_LOG_FORMAT`, `BRABBLE_TRANSCRIPTS_ENABLED`, `BRABBLE_REDACT_PII` (1/0).

## Notes on VAD options
- WebRTC VAD ships by default. Silero VAD (onnxruntime) remains an optional future path; onnxruntime is the runtime library for ONNX models and would be pulled in only if we add Silero.

## Development / testing
- Go style: gofmt tabs (default). `golangci-lint` config lives at `.golangci.yml`.
- Tests: `go test ./...` (stub ASR) plus config/env/hook coverage.
- Whisper build: build whisper.cpp once, then:
  ```sh
  # headers + libs placed in /usr/local/{include,lib}/whisper (see docs/spec.md)
  export CGO_CFLAGS='-I/usr/local/include/whisper'
  export CGO_LDFLAGS='-L/usr/local/lib/whisper -Wl,-rpath,/usr/local/lib/whisper -lwhisper -lggml -lggml-base -lggml-cpu -lggml-metal -lggml-blas -framework Accelerate -framework Metal -framework Foundation -framework CoreGraphics'
  go build -tags whisper -o bin/brabble-whisper ./cmd/brabble
  ```
- CI: GitHub Actions (`.github/workflows/ci.yml`) runs gofmt check, golangci-lint, and go test.

🎙️ Brabble. Make it say.
