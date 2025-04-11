package silk

import (
	"fmt"

	"github.com/sjzar/go-lame"
	"github.com/sjzar/go-silk"
)

func Silk2MP3(data []byte) ([]byte, error) {

	sd := silk.SilkInit()
	defer sd.Close()

	pcmdata := sd.Decode(data)
	if len(pcmdata) == 0 {
		return nil, fmt.Errorf("silk decode failed")
	}

	le := lame.Init()
	defer le.Close()

	le.SetInSamplerate(24000)
	le.SetOutSamplerate(24000)
	le.SetNumChannels(1)
	le.SetBitrate(16)
	// IMPORTANT!
	le.InitParams()

	mp3data := le.Encode(pcmdata)
	if len(mp3data) == 0 {
		return nil, fmt.Errorf("mp3 encode failed")
	}

	return mp3data, nil
}
