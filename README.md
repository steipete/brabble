# Brabble

🎙️ Brabble — Open hailing frequencies… and run the command.

Local, always-on voice daemon with a wake word and a configurable hook. Brabble listens on your Mac, waits for “Clawd” (configurable), transcribes what you say, then runs a command — by default `../warelay send "Voice brabble from ${hostname}: <text>"`. Built in Go for a single static binary and daemon-friendly control.

> Status: **0.2.0** — daemon/control/hook/wake/logging done; real audio pipeline available with `-tags whisper` (PortAudio + WebRTC VAD + whisper.cpp). Default build still works everywhere using stdin as the “mic”.

## Features
- Daemon lifecycle: `start | stop | restart | status | tail-log | list-mics | set-mic | test-hook`.
- Wake word gate (default “clawd”), stripped from payload before sending the hook.
- Hook runner with prefix substitution (`${hostname}`), cooldown, env vars, and argv payload.
- Rolling transcripts + status via UNIX socket; rotating logs under `~/Library/Application Support/brabble/`.
- Config auto-created at `~/.config/brabble/config.toml` with sane defaults for macOS + Metal.

## Quick start (stub build, no audio deps)
The stub uses stdin as the “microphone” so you can exercise the daemon and hook without audio libs.

```sh
go build ./cmd/brabble
./brabble serve   # foreground; type lines that include "clawd ..."
# in another terminal
./brabble status
./brabble test-hook "clawd ship it"
```

## Full audio build (macOS)
Dependencies:
- `brew install portaudio` (required for capture).
- A Whisper model file, e.g. `ggml-medium-q5_1.bin`:
  ```sh
  mkdir -p "$HOME/Library/Application Support/brabble/models"
  curl -L https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium-q5_1.bin \
    -o "$HOME/Library/Application Support/brabble/models/ggml-medium-q5_1.bin"
  ```

Build and run:
```sh
go build -tags whisper ./cmd/brabble
./brabble start        # daemonize with real mic + VAD + whisper
# or foreground for debugging:
./brabble serve
```

Useful commands:
- `./brabble list-mics` — enumerate inputs (uses PortAudio).
- `./brabble set-mic "MacBook Pro Microphone"` — persist preferred device.
- `./brabble status` — running?, uptime, recent transcripts.

## Configuration (auto-created)
`~/.config/brabble/config.toml`

```toml
[audio]
device_name = ""      # set via `brabble set-mic`
sample_rate = 16000
channels = 1
frame_ms = 20         # must be 10/20/30 for VAD

[vad]
enabled = true
silence_ms = 1000
aggressiveness = 2
min_speech_ms = 300
max_segment_ms = 10000

[asr]
model_path = "~/.local/state/brabble/models/ggml-medium-q5_1.bin"
language = "auto"
compute_type = "q5_1"   # q5_1, q8_0, float16
device = "auto"         # auto/metal/cpu

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
env = {}

[paths]
state_dir = "~/Library/Application Support/brabble"
log_path = ".../brabble.log"
transcript_path = ".../transcripts.log"
socket_path = ".../brabble.sock"
pid_path = ".../brabble.pid"

[ui]
status_tail = 10
```

Rules:
- Wake word must appear (case-insensitive); it is removed before sending the hook.
- `min_chars` prevents firing on tiny fragments; `silence_ms` ends a segment.
- `cooldown_sec` prevents rapid re-trigger; `prefix` supports `${hostname}` substitution.

## Process control
- `start` forks and runs `serve` in background; PID at `~/Library/Application Support/brabble/brabble.pid`.
- `stop` sends SIGTERM via PID.
- `status` queries UNIX socket at `.../brabble.sock` and prints uptime + last transcripts.
- `tail-log` shows last 50 log lines; logs rotate (20 MB, 3 backups, 30 days).

## Hook contract
- Called with argv: `hook.command` + `hook.args` + `<prefix><text>`.
- Env vars added: `BRABBLE_TEXT`, `BRABBLE_PREFIX` (plus config-provided env).
- Async execution; stdout/stderr captured to log; cooldown enforced.

## Developing
- Code layout in `internal/`:
  - `config`: defaults + TOML load/save
  - `logging`: logrus + rotation
  - `run`: daemon loop, control socket, transcript log
  - `daemon`: CLI lifecycle commands
  - `control`: CLI client commands
  - `hook`: hook runner
  - `asr`: recognizer abstraction; stdin stub + whisper build tag placeholder
- Spec/documentation: `docs/spec.md`.

## Roadmap to 0.2.0
1) Wire PortAudio/CoreAudio input and make `list-mics` real.
2) Add VAD (WebRTC default); expose `silence_ms`/`min_speech_ms` in config.
3) Implement whisper.cpp streaming recognizer and build with `-tags whisper`.
4) Optional Porcupine wake-word front-end for lower idle cost.
5) Ship launchd plist for autostart on macOS.

## Versioning
- 0.1.0 — initial skeleton with daemon/control/hook/wake/config/logging; ASR/audio stubbed.
