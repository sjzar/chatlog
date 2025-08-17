package dat2img

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/bits"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
)

const (
	ENV_FFMPEG_PATH = "FFMPEG_PATH"
	MinRatio        = 0.6
)

var (
	FFmpegMode = false
	FFMpegPath = "ffmpeg"
)

func init() {
	ffmpegPath := os.Getenv(ENV_FFMPEG_PATH)
	if len(ffmpegPath) > 0 {
		FFmpegMode = true
		FFMpegPath = ffmpegPath
	}
	if isFFmpegAvailable() {
		FFmpegMode = true
	}
}

func Wxam2pic(data []byte) ([]byte, string, error) {

	if len(data) < 15 || !bytes.Equal(data[0:4], WXGF.Header) {
		return nil, "", fmt.Errorf("invalid wxgf")
	}

	offset, size, err := findDataPartition(data)
	if err != nil {
		return nil, "", err
	}

	if FFmpegMode {
		jpgData, err := Convert2JPG(data[offset : offset+size])
		if err != nil {
			return nil, "", err
		}
		return jpgData, JPG.Ext, nil
	}

	mp4Data, err := Convert2MP4(data[offset : offset+size])
	if err != nil {
		return nil, "", err
	}
	return mp4Data, "mp4", nil
}

func findDataPartition(data []byte) (offset int, size int, err error) {

	headerLen := int(data[4])
	if headerLen >= len(data) {
		return 0, 0, fmt.Errorf("invalid wxgf")
	}

	patterns := [][]byte{
		{0x00, 0x00, 0x00, 0x01},
		{0x00, 0x00, 0x01},
	}

	for _, pattern := range patterns {
		offset := 0
		for {
			index := bytes.Index(data[headerLen+offset:], pattern)
			if index == -1 {
				break
			}

			absIndex := headerLen + offset + index
			offset += index + 1

			if absIndex < 4 {
				continue
			}

			length := int(data[absIndex-4])<<24 | int(data[absIndex-3])<<16 |
				int(data[absIndex-2])<<8 | int(data[absIndex-1])

			if length <= 0 || absIndex+length > len(data) {
				continue
			}

			ratio := float64(length) / float64(len(data))
			if ratio < MinRatio {
				continue
			}

			return absIndex, length, nil
		}

	}

	return 0, 0, fmt.Errorf("no partition found")
}

func Convert2JPG(data []byte) ([]byte, error) {
	cmd := exec.Command(FFMpegPath,
		"-i", "-",
		"-vframes", "1",
		"-c:v", "mjpeg",
		"-q:v", "4",
		"-f", "image2",
		"-")

	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	jpegData := stdout.Bytes()
	if len(jpegData) == 0 {
		return nil, fmt.Errorf("ffmpeg output is empty")
	}

	return jpegData, nil
}

func Convert2MP4(data []byte) ([]byte, error) {

	vpsNALUs, spsNALUs, ppsNALUs := hevc.GetParameterSetsFromByteStream(data)

	videoTimescale := uint32(1000)
	init := mp4.CreateEmptyInit()
	init.AddEmptyTrack(videoTimescale, "video", "und")

	trak := init.Moov.Trak
	err := trak.SetHEVCDescriptor("hvc1", vpsNALUs, spsNALUs, ppsNALUs, nil, true)
	if err != nil {
		return nil, err
	}

	seg := mp4.NewMediaSegment()
	seg.EncOptimize = mp4.OptimizeTrun
	frag, err := mp4.CreateFragment(1, mp4.DefaultTrakID)
	if err != nil {
		return nil, err
	}
	seg.AddFragment(frag)

	sampleData := avc.ConvertByteStreamToNaluSample(data)
	sample := mp4.FullSample{
		Sample: mp4.Sample{
			Flags:                 0x02000000,
			Dur:                   1000,
			Size:                  uint32(len(sampleData)),
			CompositionTimeOffset: 0,
		},
		DecodeTime: 0,
		Data:       sampleData,
	}

	frag.AddFullSample(sample)

	totalSize := init.Size() + seg.Size()
	sw := bits.NewFixedSliceWriter(int(totalSize))

	init.EncodeSW(sw)
	seg.EncodeSW(sw)

	return sw.Bytes(), nil
}

func isFFmpegAvailable() bool {
	cmd := exec.Command(FFMpegPath, "-version")
	return cmd.Run() == nil
}
