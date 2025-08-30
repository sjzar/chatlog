package silk

import (
	"fmt"

	"github.com/sjzar/go-lame"
	"github.com/sjzar/go-silk"
)

// Silk2MP3 将SILK格式的音频数据转换为MP3格式
// 使用上游版本的实现，支持微信语音消息的SILK到MP3转换
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
