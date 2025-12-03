//go:build whisper

package asr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"

	"brabble/internal/config"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/gordonklaus/portaudio"
	vad "github.com/maxhawkins/go-webrtcvad"
	"github.com/sirupsen/logrus"
)

// whisperRecognizer captures audio, runs VAD, then transcribes with whisper.cpp.
type whisperRecognizer struct {
	cfg    *config.Config
	logger *logrus.Logger
	model  whisper.Model
	vad    *vad.VAD
}

func newWhisperRecognizer(cfg *config.Config, logger *logrus.Logger) (Recognizer, error) {
	if cfg.Audio.Channels != 1 {
		return nil, fmt.Errorf("only mono input supported; set audio.channels = 1")
	}
	if cfg.Audio.FrameMS != 10 && cfg.Audio.FrameMS != 20 && cfg.Audio.FrameMS != 30 {
		return nil, fmt.Errorf("audio.frame_ms must be 10, 20, or 30 (got %d)", cfg.Audio.FrameMS)
	}
	switch cfg.Audio.SampleRate {
	case 8000, 16000, 32000, 48000:
	default:
		return nil, fmt.Errorf("sample_rate must be 8k/16k/32k/48k for webrtc VAD (got %d)", cfg.Audio.SampleRate)
	}
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("portaudio init: %w", err)
	}
	model, err := whisper.New(cfg.ASR.ModelPath)
	if err != nil {
		portaudio.Terminate()
		return nil, fmt.Errorf("load model: %w", err)
	}
	v := vad.New()
	if err := v.SetMode(cfg.VAD.Aggressiveness); err != nil {
		model.Close()
		portaudio.Terminate()
		return nil, fmt.Errorf("vad mode: %w", err)
	}
	return &whisperRecognizer{
		cfg:    cfg,
		logger: logger,
		model:  model,
		vad:    v,
	}, nil
}

func (r *whisperRecognizer) Run(ctx context.Context, out chan<- Segment) error {
	defer r.model.Close()
	defer portaudio.Terminate()

	dev, err := selectDevice(r.cfg.Audio.DeviceName)
	if err != nil {
		return err
	}

	frameSamples := r.cfg.Audio.SampleRate * r.cfg.Audio.FrameMS / 1000
	if ok := vad.ValidRateAndFrameLength(r.cfg.Audio.SampleRate, frameSamples); !ok {
		return fmt.Errorf("invalid frame_ms %d for sample_rate %d", r.cfg.Audio.FrameMS, r.cfg.Audio.SampleRate)
	}

	buf := make([]int16, frameSamples)
	stream, err := portaudio.OpenStream(portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   dev,
			Channels: r.cfg.Audio.Channels,
			Latency:  dev.DefaultLowInputLatency,
		},
		SampleRate:      float64(r.cfg.Audio.SampleRate),
		FramesPerBuffer: frameSamples,
	}, &buf)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		return fmt.Errorf("start stream: %w", err)
	}
	defer stream.Stop()

	segments := make(chan []int16, 8)
	go r.transcribeWorker(ctx, segments, out)

	var (
		chunk      []int16
		inSpeech   bool
		lastVoice  time.Time
		speechBegan time.Time
		silenceDur = time.Duration(r.cfg.VAD.SilenceMS) * time.Millisecond
		maxSegDur  = time.Duration(r.cfg.VAD.MaxSegmentMS) * time.Millisecond
	)

	r.logger.Infof("listening on mic: %s @ %d Hz", dev.Name, r.cfg.Audio.SampleRate)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := stream.Read(); err != nil {
			if errors.Is(err, portaudio.InputOverflowed) {
				r.logger.Warn("input overflow")
				continue
			}
			return fmt.Errorf("stream read: %w", err)
		}
		voice := r.vad.Process(r.cfg.Audio.SampleRate, buf)

		if voice {
			if !inSpeech {
				inSpeech = true
				speechBegan = time.Now()
				chunk = chunk[:0]
			}
			chunk = append(chunk, buf...)
			lastVoice = time.Now()
		} else if inSpeech {
			// check if silence long enough or max segment exceeded
			now := time.Now()
			if (now.Sub(lastVoice) >= silenceDur && len(chunk) > 0) ||
				(maxSegDur > 0 && now.Sub(speechBegan) >= maxSegDur) {
				// finalize
				cpy := make([]int16, len(chunk))
				copy(cpy, chunk)
				select {
				case segments <- cpy:
				default:
					r.logger.Warn("segment queue full, dropping segment")
				}
				inSpeech = false
				chunk = chunk[:0]
			}
		}
	}
}

func (r *whisperRecognizer) transcribeWorker(ctx context.Context, segs <-chan []int16, out chan<- Segment) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-segs:
			if len(data) == 0 {
				continue
			}
			text, err := r.transcribe(ctx, data)
			if err != nil {
				r.logger.Errorf("transcribe: %v", err)
				continue
			}
			if strings.TrimSpace(text) == "" {
				continue
			}
			seg := Segment{
				Text:       strings.TrimSpace(text),
				Start:      time.Now(), // approximate; audio timestamps not tracked
				End:        time.Now(),
				Confidence: 0.0,
			}
			select {
			case out <- seg:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (r *whisperRecognizer) transcribe(ctx context.Context, pcm []int16) (string, error) {
	samples := make([]float32, len(pcm))
	for i, s := range pcm {
		samples[i] = float32(s) / 32768.0
	}

	params := whisper.NewParams(whisper.SAMPLING_GREEDY)
	params.SetNThreads(runtime.NumCPU())
	params.SetAudioCtx(0)

	ctxWhisper, err := r.model.NewContext(params)
	if err != nil {
		return "", err
	}

	if lang := strings.TrimSpace(r.cfg.ASR.Language); lang != "" {
		if err := ctxWhisper.SetLanguage(lang); err != nil {
			r.logger.Warnf("set language: %v", err)
		}
	}

	if err := ctxWhisper.Process(samples, nil, nil, nil); err != nil {
		return "", err
	}
	var b strings.Builder
	for {
		seg, err := ctxWhisper.NextSegment()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		b.WriteString(seg.Text)
		if !strings.HasSuffix(seg.Text, " ") {
			b.WriteRune(' ')
		}
	}
	return b.String(), nil
}

func selectDevice(preferred string) (*portaudio.DeviceInfo, error) {
	devs, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	if preferred != "" {
		for _, d := range devs {
			if d.MaxInputChannels > 0 && strings.Contains(strings.ToLower(d.Name), strings.ToLower(preferred)) {
				return d, nil
			}
		}
	}
	if def := portaudio.DefaultInputDevice(); def != nil {
		return def, nil
	}
	for _, d := range devs {
		if d.MaxInputChannels > 0 {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no input devices found")
}
