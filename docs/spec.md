# Brabble Specification (macOS, Go)

## Purpose
Always-on local voice daemon that listens via microphone, detects a wake word, transcribes speech, and triggers a configurable shell hook (default: `../warelay send "<prefix><text>"`). Optimized for offline use with a strong machine; runs as a controllable daemon with CLI surface.

## Scope
- **Targets**: macOS (Apple Silicon/Intel). Linux possible later.
- **ASR**: whisper.cpp via Go bindings (enabled with `-tags whisper`) using quantized medium/large models; stub build uses stdin.
- **VAD**: WebRTC VAD (current default); Silero VAD via onnxruntime remains optional future work.
- **Wake word**: Configurable, default “clawd”. Optional disable.
- **Hook**: Local shell command with prefix, env vars, cooldown, and payload on argv.
- **Control**: Start/stop/restart/status/tail-log/mic list|set/test-hook via CLI; status over UNIX socket.

## Architecture
1) **Daemon process** (`brabble serve` launched by `start`):
   - Writes PID file and owns a UNIX domain socket for control.
   - Captures audio from selected mic via PortAudio → WebRTC VAD segments speech (partial flush every `partial_flush_ms` for live feedback; partial segments are not sent to the hook).
   - Wake-word gate (string match) before dispatch.
   - ASR (whisper.cpp) transcribes segments; finished segments sent to hook runner and transcript log.
2) **CLI client**:
   - Subcommands send JSON requests over the UNIX socket or manage lifecycle (start/stop).
3) **State & logs** (macOS defaults):
   - State dir: `~/Library/Application Support/brabble/`
   - PID: `.../brabble.pid`
   - Socket: `.../brabble.sock`
   - Main log (rotating): `.../brabble.log`
   - Transcript log: `.../transcripts.log`

## CLI Commands
- `brabble start [-c path] [--foreground]` (foreground only via `serve`; start forks by default).
- `brabble stop [-c path]` sends SIGTERM using PID file.
- `brabble restart [-c path]` stop then start (best effort).
- `brabble status [-c path]` shows running?, uptime, last N transcripts.
- `brabble tail-log [-c path]` prints last 50 log lines.
- `brabble mic list` enumerates mics (whisper build).
- `brabble mic set [--index N] "<name>" [-c path]` writes preferred mic/index to config.
- `brabble models list|download|set` manage whisper models.
- `brabble setup` download default model and update config.
- `brabble doctor` run dependency checks (hook, model, portaudio).
- `brabble transcribe <wav>` transcribe a WAV file; `--hook` sends through configured hook; `--no-wake` skips wake gating.
- `brabble health` ping the control socket.
- `brabble service install|uninstall|status` manage launchd plist and print kickstart/bootout commands.
- `brabble test-hook "text" [-c path]` invokes hook once with sample text.
- Internal: `brabble serve [-c path]` runs daemon in foreground (used by start/launchd).

## Configuration (TOML)
Default path: `~/.config/brabble/config.toml` (auto-created). Key sections:
```toml
[audio]
device_name = ""      # set via mic set
device_index = -1     # optional numeric selection
sample_rate = 16000
channels = 1
frame_ms = 20

[vad]
enabled = true
silence_ms = 1000
aggressiveness = 2
energy_threshold = 0.0
min_speech_ms = 300
max_segment_ms = 10000
partial_flush_ms = 4000

[asr]
model_path = "~/.local/state/brabble/models/ggml-medium-q5_1.bin"
language = "auto"
compute_type = "q5_1"   # q5_1, q8_0, float16
device = "auto"         # auto/metal/cpu

[wake]
enabled = true
word = "clawd"
aliases = ["claude"]
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

[paths]
state_dir = "~/Library/Application Support/brabble"
log_path = ".../brabble.log"
transcript_path = ".../transcripts.log"
socket_path = ".../brabble.sock"
pid_path = ".../brabble.pid"

[ui]
status_tail = 10

[logging]
level = "info"   # debug|info|warn|error
format = "text"  # text|json
stdout = false   # also log to stdout when true

[metrics]
enabled = false
addr = "127.0.0.1:9317"

[transcripts]
enabled = true
```
Rules:
- Wake word must be present (case-insensitive); it is stripped before hook text.
- `min_chars` gate prevents firing on very short utterances.
- `silence_ms` ends a segment when no speech is detected for that long.
- `cooldown_sec` prevents rapid successive hook invocations.
- `partial_flush_ms` emits interim transcripts; marked `Partial=true` and skipped by the hook.
- `prefix` supports `${hostname}` substitution.

## Hook Execution
- Command: `hook.command` with `hook.args` plus final payload argument = `prefix + text`.
- Env vars: inherited plus `BRABBLE_TEXT`, `BRABBLE_PREFIX`.
- Runs asynchronously; stdout/stderr are logged.
- Cooldown enforced globally.

## Status & Logging
- Status reply: running flag, uptime seconds, last `status_tail` transcripts (text + timestamp).
- Logging: logrus + rotating file (20 MB, 3 backups, 30 days); also to stdout when foreground.
- Transcript log: tab-separated RFC3339 timestamp and text for history.

## Daemon Lifecycle
- PID file guards double start; removed on clean exit.
- SIGTERM/SIGINT trigger graceful shutdown: stop audio, flush pending, close socket.
- Control socket is removed on start and shutdown to avoid stale sockets.
- Doctor command checks config/model/hook binary presence and PortAudio availability (with whisper build).
- launchd helper writes a user plist for autostart on macOS.
- launchd supports custom env via `brabble service install --env KEY=VAL`; helper prints kickstart/bootout commands.
- CI: GitHub Actions runs lint/test on Linux and whisper-tag build on macOS with PortAudio installed.
- Setup command fetches default whisper model if missing.
- Models command supports listing known models, downloading into state dir, and setting `asr.model_path`.
- Optional `/metrics` endpoint (Prometheus text) gated by config.
- Health op exposed on the control socket; env overrides `BRABBLE_WAKE_ENABLED`, `BRABBLE_METRICS_ADDR`.
- Logging config (level/format) with env overrides `BRABBLE_LOG_LEVEL`, `BRABBLE_LOG_FORMAT`.
- Hook PII redaction toggle; transcript logging toggle.

## Audio & ASR Implementation Notes (to be filled)
- Replace stdin stub by implementing `internal/asr/whisper_whisper.go` using whisper.cpp Go bindings; build with `-tags whisper`.
- Audio capture: PortAudio/CoreAudio, expose device enumeration and selection for `mic list`.
- VAD: default WebRTC VAD with `silence_ms`; optional Silero VAD via onnxruntime for robustness.
- Wake word: initial pass can be string match on transcribed text; optional Porcupine/keyword spotter before ASR for lower cost.

## Build Flavors
- **Stub** (default): `go build ./cmd/brabble` — uses stdin recognizer; useful for wiring tests/control without audio deps.
- **Whisper**:
  1. Build whisper.cpp once (Metal+BLAS):
     ```sh
     git clone https://github.com/ggerganov/whisper.cpp.git /tmp/whisper.cpp-build
     cmake -S /tmp/whisper.cpp-build -B /tmp/whisper.cpp-build/build -DGGML_METAL=ON -DGGML_BLAS=ON
     cmake --build /tmp/whisper.cpp-build/build --target whisper
     sudo mkdir -p /usr/local/lib/whisper /usr/local/include/whisper
     sudo cp /tmp/whisper.cpp-build/build/src/libwhisper.dylib /tmp/whisper.cpp-build/build/ggml/src/libggml*.dylib /tmp/whisper.cpp-build/build/ggml/src/ggml-metal/libggml-metal.dylib /tmp/whisper.cpp-build/build/ggml/src/ggml-blas/libggml-blas.dylib /usr/local/lib/whisper/
     sudo cp -R /tmp/whisper.cpp-build/include/* /tmp/whisper.cpp-build/ggml/include/* /usr/local/include/whisper/
     ```
  2. Build Go binary:
     ```sh
     export CGO_CFLAGS='-I/usr/local/include/whisper'
     export CGO_LDFLAGS='-L/usr/local/lib/whisper -Wl,-rpath,/usr/local/lib/whisper -lwhisper -lggml -lggml-base -lggml-cpu -lggml-metal -lggml-blas -framework Accelerate -framework Metal -framework Foundation -framework CoreGraphics'
     go build -tags whisper -o bin/brabble-whisper ./cmd/brabble
     ```

## Dependencies
- Go 1.25+
- Runtime libs (planned): PortAudio (macOS: `brew install portaudio`), whisper.cpp built as dylib or static via cgo, optional onnxruntime (Silero VAD), optional Porcupine wake word SDK.
- Current tree vendors only Go libs: cobra, logrus, lumberjack, go-toml, shlex.

## Operational Defaults
- Wake word “clawd”, medium Q5 Whisper model, Metal device auto-detected.
- Hook target `../warelay send` with hostname-prefixed text.
- Silence timeout 1.0s; `min_chars` 24; cooldown 1s.

## Open Items / TODO
- Optional: Silero VAD via onnxruntime for better noise robustness.
- Optional: Porcupine/keyword front-end before whisper to save compute.
- Extra: smarter device hot-swap notifications.
