package doctor

import (
	"fmt"

	"github.com/gordonklaus/portaudio"
)

func checkPortAudio(_ bool) Result {
	if err := portaudio.Initialize(); err != nil {
		return Result{Name: "portaudio", Pass: false, Detail: fmt.Sprintf("init failed: %v (install with: brew install portaudio)", err)}
	}
	defer func() {
		_ = portaudio.Terminate()
	}()
	return Result{Name: "portaudio", Pass: true, Detail: "ok"}
}
