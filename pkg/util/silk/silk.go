package silk

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/sjzar/go-lame"
	"github.com/sjzar/go-silk"
)

func Silk2MP3(data []byte) ([]byte, error) {
	log.Info().Int("input_data_size", len(data)).Msg("开始SILK到MP3转换")

	if len(data) == 0 {
		log.Error().Msg("输入的SILK数据为空")
		return nil, fmt.Errorf("输入数据为空")
	}

	log.Debug().Msg("初始化SILK解码器")
	sd := silk.SilkInit()
	defer sd.Close()

	log.Debug().Int("silk_data_size", len(data)).Msg("开始解码SILK数据")
	pcmdata := sd.Decode(data)
	if len(pcmdata) == 0 {
		log.Error().Int("silk_data_size", len(data)).Msg("SILK解码失败，PCM数据为空")
		return nil, fmt.Errorf("silk decode failed")
	}

	log.Info().Int("silk_data_size", len(data)).Int("pcm_data_size", len(pcmdata)).Msg("SILK解码成功")

	log.Debug().Msg("初始化LAME编码器")
	le := lame.Init()
	defer le.Close()

	log.Debug().Msg("设置LAME编码参数")
	le.SetInSamplerate(24000)
	le.SetOutSamplerate(24000)
	le.SetNumChannels(1)
	le.SetBitrate(16)
	// IMPORTANT!
	le.InitParams()

	log.Debug().Int("pcm_data_size", len(pcmdata)).Msg("开始编码PCM数据为MP3")
	mp3data := le.Encode(pcmdata)
	if len(mp3data) == 0 {
		log.Error().Int("pcm_data_size", len(pcmdata)).Msg("MP3编码失败，MP3数据为空")
		return nil, fmt.Errorf("mp3 encode failed")
	}

	log.Info().Int("pcm_data_size", len(pcmdata)).Int("mp3_data_size", len(mp3data)).Msg("MP3编码成功")
	return mp3data, nil
}
