//go:build !whisper

package doctor

func checkPortAudio(_ bool) Result {
	return Result{Name: "portaudio", Pass: true, Detail: "skipped (build without whisper tag)"}
}
