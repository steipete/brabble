# Brabble

🎙️ Brabble — Open hailing frequencies… and run the command.

Local, always-on voice daemon for macOS. Listens for a wake word (“clawd” by default), transcribes locally, and fires a configurable hook (`../warelay send "Voice brabble from ${hostname}: <text>"` by default). Go binary with daemon-friendly control.

## Install / Run
- Requirements: Go 1.25+.
- Stub (no audio deps): `go run ./cmd/brabble serve` then type lines containing “clawd …”.
- Full audio (macOS):
  - `brew install portaudio`
  - Download a whisper.cpp model, e.g.:
    ```sh
    mkdir -p "$HOME/Library/Application Support/brabble/models"
    curl -L https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium-q5_1.bin \
      -o "$HOME/Library/Application Support/brabble/models/ggml-medium-q5_1.bin"
    ```
  - Build with whisper + VAD: `go build -tags whisper ./cmd/brabble`
  - Run: `./brabble start` (daemon) or `./brabble serve` (foreground)

## Commands
- `start | stop | restart` — daemon lifecycle (PID + UNIX socket).
- `status` — running?, uptime, recent transcripts.
- `tail-log` — last 50 log lines.
- `list-mics` (whisper build) — enumerate inputs.
- `set-mic "<name>"` — persist preferred device.
- `test-hook "text"` — invoke hook manually.
- `doctor` — check config, model path, warelay, PortAudio (whisper build).
- `install-service` — write a user launchd plist (macOS) for autostart.
- `uninstall-service` — remove the launchd plist; use bootout to stop.
- `setup` — download default whisper model if missing.
- `health` — ping the daemon for liveness.
- `models list|download|set` — manage whisper.cpp models.
- pnpm shortcuts (Go under the hood):
  - `pnpm build` → `go build -o bin/brabble ./cmd/brabble`
  - `pnpm start|stop|restart` → manages the daemon via built binary
  - `pnpm brabble --help` → build then show full Go help (root command)
  - `pnpm brabble serve` → build then run foreground
  - `pnpm lint` → `golangci-lint run`
  - `pnpm format` → `gofmt -w .`
  - `pnpm test` → `go test ./...`

## Requirements to actually listen
- Go 1.25+ and pnpm installed.
- macOS: `brew install portaudio pkg-config`.
- Whisper model present, e.g.:
  ```sh
  mkdir -p "$HOME/Library/Application Support/brabble/models"
  curl -L https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium-q5_1.bin \
    -o "$HOME/Library/Application Support/brabble/models/ggml-medium-q5_1.bin"
  ```
  or run `pnpm setup` to download the default.
- Hook target exists (default `../warelay send …`); set `hook.command` to an absolute path if needed.
- Build with audio: `pnpm build` then `./bin/brabble start --metrics-addr 127.0.0.1:9317` (whisper build); or `go build -tags whisper ./cmd/brabble`.
- Pick mic: `./bin/brabble list-mics` (whisper build) then `./bin/brabble set-mic "<name>"`.
- Verify: `./bin/brabble doctor` (deps/model/hook) and `./bin/brabble health` after start.
- Optional autostart: `./bin/brabble install-service --env BRABBLE_METRICS_ADDR=127.0.0.1:9317`; load/kickstart/bootout commands are printed.

## CI
- GitHub Actions: lint (`golangci-lint`), `go test` (stub build), macOS whisper build (PortAudio).

## Hook
- Default: `../warelay send "<prefix><text>"`, prefix `Voice brabble from ${hostname}: `.
- Env vars added: `BRABBLE_TEXT`, `BRABBLE_PREFIX` + configured env.
- Cooldown + `min_chars`; wake word is stripped before dispatch.

## Wake, VAD, ASR
- Wake word: “clawd” (configurable, case-insensitive).
- VAD: WebRTC; `silence_ms` ends a segment, `max_segment_ms` caps length.
- ASR: whisper.cpp (quantized ggml). Default model path `~/Library/Application Support/brabble/models/ggml-medium-q5_1.bin`.

## Config (auto-created at `~/.config/brabble/config.toml`)
```toml
[audio]
device_name = ""
sample_rate = 16000
channels = 1
frame_ms = 20        # 10/20/30 only

[vad]
enabled = true
silence_ms = 1000
aggressiveness = 2
min_speech_ms = 300
max_segment_ms = 10000

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

[transcripts]
enabled = true
```
State/logs live under `~/Library/Application Support/brabble/` (socket, pid, logs, transcripts).

## Behavior
- Wake word must be present; removed before sending hook.
- Transcripts are tailed by `status`; logs rotate (20 MB, 3 backups, 30 days).
- Default build uses stdin as mic; whisper build uses PortAudio + VAD + whisper.cpp.
- Metrics: optional Prometheus-style endpoint at `/metrics` when enabled in config.
- Env overrides: `BRABBLE_WAKE_ENABLED=0` to disable wake; `BRABBLE_METRICS_ADDR=127.0.0.1:9317` to enable metrics at a custom address; `BRABBLE_LOG_LEVEL=debug`, `BRABBLE_LOG_FORMAT=json`; `BRABBLE_TRANSCRIPTS_ENABLED=0` to skip transcript writes; `BRABBLE_REDACT_PII=1` to mask emails/phones before hooks/logs.
- Launchd: `install-service --env KEY=VAL` adds EnvironmentVariables; load with `launchctl load -w ~/Library/LaunchAgents/com.brabble.agent.plist`, start `launchctl kickstart gui/$(id -u)/com.brabble.agent`, stop `launchctl bootout gui/$(id -u)/com.brabble.agent`.

## Roadmap
- Optional Silero VAD (onnxruntime) and Porcupine wake front-end.
- `reload` command for hot config.
- launchd plist for macOS autostart.
