# 🎙️ Brabble — Open hailing frequencies… and run the command.

Always-on, local-only voice daemon for macOS. Hears your wake word (“clawd” by default), transcribes with whisper.cpp, then fires a configurable hook (user-defined, e.g., warelay heartbeat). Written in Go; ships with a daemon lifecycle, status socket, and launchd helper.

## Quick start
- Requirements: Go 1.25+, `brew install cmake portaudio pkg-config`, whisper.cpp headers/libs, and a whisper.cpp model.
- One-liner: `pnpm brabble setup && pnpm start` (downloads medium Q5_1, writes config, starts daemon).
- Foreground run: `go run ./cmd/brabble serve` (mic + PortAudio required).

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
- `pnpm brabble` — build then start daemon (default); extra args pass through, e.g. `pnpm brabble --help`, `pnpm brabble status`.
- `pnpm start|stop|restart` — lifecycle wrappers.
- `pnpm build` — build to `bin/brabble`; `pnpm lint` — `golangci-lint run`; `pnpm format` — `gofmt -w .`; `pnpm test` — `go test ./...`.
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
energy_threshold = -35.0  # dBFS gate; raise (e.g., -30) to suppress low-noise hallucinations
min_speech_ms = 300
max_segment_ms = 10000
partial_flush_ms = 4000  # emit partial segments (not sent to hook)

[asr]
 model_path = "~/Library/Application Support/brabble/models/ggml-large-v3-turbo-q8_0.bin"
language = "auto"
compute_type = "q5_1"
device = "auto"       # auto/metal/cpu

[wake]
enabled = true
word = "clawd"
aliases = ["claude"]
sensitivity = 0.6

[hook]
command = ""                       # REQUIRED: set to your warelay binary path
args = []                          # e.g., ["heartbeat", "--message"]
prefix = "Voice brabble from ${hostname}: "
cooldown_sec = 1
min_chars = 24
max_latency_ms = 5000
queue_size = 16
timeout_sec = 30
redact_pii = false
env = {}

[logging]
level = "info"   # debug|info|warn|error
format = "text"  # text|json
stdout = false   # also log to stdout (defaults to file-only)

[daemon]
stop_timeout_sec = 5     # wait for PID to clear on restart

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
- Wake word (case-insensitive) is stripped before dispatch; disable with `--no-wake` or `BRABBLE_WAKE_ENABLED=0`. If wake word is “clawd”, “Claude” is also accepted.
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
- Tests: `go test ./...` plus config/env/hook coverage.
- Build: build whisper.cpp once. On macOS the Makefile auto-detects a user-local install at `~/.local/opt/whisper`; this avoids relying on Homebrew's `whisper-cpp` formula, which may not ship the `ggml.h` header required by the Go binding.
  ```sh
  git clone --depth 1 https://github.com/ggerganov/whisper.cpp.git /tmp/whisper.cpp-brabble
  cmake -S /tmp/whisper.cpp-brabble -B /tmp/whisper.cpp-brabble/build -DGGML_METAL=ON -DGGML_BLAS=ON -DBUILD_SHARED_LIBS=ON
  cmake --build /tmp/whisper.cpp-brabble/build --target whisper --parallel
  mkdir -p ~/.local/opt/whisper/{include,lib}
  cp /tmp/whisper.cpp-brabble/build/src/libwhisper*.dylib ~/.local/opt/whisper/lib/
  cp /tmp/whisper.cpp-brabble/build/ggml/src/libggml*.dylib ~/.local/opt/whisper/lib/
  cp /tmp/whisper.cpp-brabble/build/ggml/src/ggml-metal/libggml-metal*.dylib ~/.local/opt/whisper/lib/ 2>/dev/null || true
  cp /tmp/whisper.cpp-brabble/build/ggml/src/ggml-blas/libggml-blas*.dylib ~/.local/opt/whisper/lib/ 2>/dev/null || true
  cp -R /tmp/whisper.cpp-brabble/include/* /tmp/whisper.cpp-brabble/ggml/include/* ~/.local/opt/whisper/include/
  make test build
  ```
- Models: defaults to `ggml-large-v3-turbo-q8_0.bin`; best quality `ggml-large-v3-turbo.bin`; lighter option `ggml-medium-q5_1.bin`. Use `brabble models download <name>` then `brabble models set <name>`.
- CI: GitHub Actions (`.github/workflows/ci.yml`) runs gofmt check, golangci-lint, and go test.

🎙️ Brabble. Make it say.
[hooks]
# Optional per-wake hooks. First matching entry wins.
# [[hooks]]
# wake    = ["clawd", "claude"]
# aliases = ["clawd"]
# command = "/path/to/warelay"
# args    = ["heartbeat", "--message"]
# prefix  = "Voice brabble from ${hostname}: "
# min_chars = 16
# cooldown_sec = 1
# timeout_sec = 5
# queue_size = 16
# redact_pii = false
