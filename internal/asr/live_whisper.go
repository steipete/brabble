package asr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
	"unsafe"

	"brabble/internal/config"
	"brabble/internal/logging"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/gordonklaus/portaudio"
	vad "github.com/maxhawkins/go-webrtcvad"
)

// whisperRecognizer captures audio, runs VAD, then transcribes with whisper.cpp.
type whisperRecognizer struct {
	cfg    *config.Config
	logger *logging.Logger
	model  whisper.Model
	vad    *vad.VAD
}

type segmentChunk struct {
	pcm     []int16
	partial bool
}

func newWhisperRecognizer(cfg *config.Config, logger *logging.Logger) (Recognizer, error) {
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
		_ = portaudio.Terminate()
		return nil, fmt.Errorf("load model: %w", err)
	}
	if err := warmup(model, cfg, logger); err != nil {
		logger.Warnf("warmup: %v", err)
	}
	v, err := vad.New()
	if err != nil {
		_ = model.Close()
		_ = portaudio.Terminate()
		return nil, fmt.Errorf("vad init: %w", err)
	}
	if err := v.SetMode(cfg.VAD.Aggressiveness); err != nil {
		_ = model.Close()
		_ = portaudio.Terminate()
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
	defer func() { _ = r.model.Close() }()
	defer func() { _ = portaudio.Terminate() }()

	frameSamples := r.cfg.Audio.SampleRate * r.cfg.Audio.FrameMS / 1000
	if ok := r.vad.ValidRateAndFrameLength(r.cfg.Audio.SampleRate, frameSamples); !ok {
		return fmt.Errorf("invalid frame_ms %d for sample_rate %d", r.cfg.Audio.FrameMS, r.cfg.Audio.SampleRate)
	}

	segments := make(chan segmentChunk, 8)
	go r.transcribeWorker(ctx, segments, out)

	// retry loop for device/stream failures
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		dev, err := selectDevice(r.cfg.Audio.DeviceName, r.cfg.Audio.DeviceIndex)
		if err != nil {
			r.logger.Warnf("select device: %v; retrying in 2s", err)
			time.Sleep(2 * time.Second)
			continue
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
			r.logger.Warnf("open stream: %v; retrying in 2s", err)
			time.Sleep(2 * time.Second)
			continue
		}
		r.logger.Infof("listening on mic: %s @ %d Hz", dev.Name, r.cfg.Audio.SampleRate)
		if err := r.captureLoop(ctx, stream, buf, segments); err != nil && !errors.Is(err, context.Canceled) {
			r.logger.Warnf("stream ended: %v; restarting in 2s", err)
			_ = stream.Close()
			time.Sleep(2 * time.Second)
			continue
		}
		_ = stream.Close()
		return nil
	}
}

func (r *whisperRecognizer) captureLoop(ctx context.Context, stream *portaudio.Stream, buf []int16, segments chan<- segmentChunk) error {
	if err := stream.Start(); err != nil {
		return fmt.Errorf("start stream: %w", err)
	}
	defer func() { _ = stream.Stop() }()

	var (
		chunk           []int16
		inSpeech        bool
		lastVoice       time.Time
		speechBegan     time.Time
		lastPartialSent time.Time
		silenceDur      = time.Duration(r.cfg.VAD.SilenceMS) * time.Millisecond
		maxSegDur       = time.Duration(r.cfg.VAD.MaxSegmentMS) * time.Millisecond
		partialFlush    = time.Duration(r.cfg.VAD.PartialFlushMS) * time.Millisecond
		sampleRate      = r.cfg.Audio.SampleRate
		minSpeech       = time.Duration(r.cfg.VAD.MinSpeechMS) * time.Millisecond
	)

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
		active, err := r.vad.Process(r.cfg.Audio.SampleRate, int16ToBytes(buf))
		if err != nil {
			r.logger.Warnf("vad process: %v", err)
			continue
		}

		if active {
			if !inSpeech {
				inSpeech = true
				speechBegan = time.Now()
				lastPartialSent = time.Now()
				chunk = chunk[:0]
			}
			chunk = append(chunk, buf...)
			lastVoice = time.Now()

			if partialFlush > 0 && time.Since(lastPartialSent) >= partialFlush && len(chunk) > 0 {
				chunkDur := time.Duration(len(chunk)) * time.Second / time.Duration(sampleRate)
				if chunkDur < minSpeech {
					continue
				}
				if skipForEnergy(chunk, r.cfg.VAD.EnergyThresh) {
					inSpeech = false
					chunk = chunk[:0]
					continue
				}
				cpy := make([]int16, len(chunk))
				copy(cpy, chunk)
				select {
				case segments <- segmentChunk{pcm: cpy, partial: true}:
					lastPartialSent = time.Now()
					chunk = chunk[:0]
					speechBegan = time.Now()
				default:
					r.logger.Warn("segment queue full, dropping partial")
				}
			}
		} else if inSpeech {
			now := time.Now()
			if (now.Sub(lastVoice) >= silenceDur && len(chunk) > 0) ||
				(maxSegDur > 0 && now.Sub(speechBegan) >= maxSegDur) {
				chunkDur := time.Duration(len(chunk)) * time.Second / time.Duration(sampleRate)
				if chunkDur >= minSpeech {
					if skipForEnergy(chunk, r.cfg.VAD.EnergyThresh) {
						inSpeech = false
						chunk = chunk[:0]
						continue
					}
					cpy := make([]int16, len(chunk))
					copy(cpy, chunk)
					select {
					case segments <- segmentChunk{pcm: cpy, partial: false}:
					default:
						r.logger.Warn("segment queue full, dropping segment")
					}
				}
				inSpeech = false
				chunk = chunk[:0]
			}
		}
	}
}

func (r *whisperRecognizer) transcribeWorker(ctx context.Context, segs <-chan segmentChunk, out chan<- Segment) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-segs:
			if len(data.pcm) == 0 {
				continue
			}
			text, err := r.transcribe(ctx, data.pcm)
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
				Partial:    data.partial,
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

	ctxWhisper, err := r.model.NewContext()
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

func warmup(model whisper.Model, cfg *config.Config, logger *logging.Logger) error {
	ctx, err := model.NewContext()
	if err != nil {
		return err
	}
	samples := make([]float32, cfg.Audio.SampleRate/2) // 0.5s silence
	if err := ctx.Process(samples, nil, nil, nil); err != nil {
		return err
	}
	logger.Debug("whisper warmup complete")
	return nil
}

func selectDevice(preferred string, index int) (*portaudio.DeviceInfo, error) {
	devs, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	if index >= 0 && index < len(devs) {
		d := devs[index]
		if d.MaxInputChannels > 0 {
			return d, nil
		}
	}
	if preferred != "" {
		for _, d := range devs {
			if d.MaxInputChannels > 0 && strings.Contains(strings.ToLower(d.Name), strings.ToLower(preferred)) {
				return d, nil
			}
		}
	}
	if def, err := portaudio.DefaultInputDevice(); err == nil && def != nil {
		return def, nil
	}
	for _, d := range devs {
		if d.MaxInputChannels > 0 {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no input devices found")
}

func int16ToBytes(samples []int16) []byte {
	if len(samples) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&samples[0])), len(samples)*2)
}

func skipForEnergy(pcm []int16, threshDb float64) bool {
	if threshDb == 0 {
		return false
	}
	db := rmsDbFS(pcm)
	return db < threshDb
}

func rmsDbFS(pcm []int16) float64 {
	if len(pcm) == 0 {
		return -120
	}
	var sum float64
	for _, s := range pcm {
		f := float64(s) / 32768.0
		sum += f * f
	}
	rms := math.Sqrt(sum / float64(len(pcm)))
	if rms == 0 {
		return -120
	}
	return 20 * math.Log10(rms)
}
