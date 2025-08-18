package dat2img

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Eyevinn/mp4ff/avc"
	"github.com/Eyevinn/mp4ff/bits"
	"github.com/Eyevinn/mp4ff/hevc"
	"github.com/Eyevinn/mp4ff/mp4"
	"github.com/google/uuid"
)

const (
	ENV_FFMPEG_PATH = "FFMPEG_PATH"
	MinRatio        = 0.6
	FixSliceHeaders = true
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

	partitions, err := findDataPartition(data)
	if err != nil {
		return nil, "", err
	}

	if partitions.LikeAnime() {
		// FIXME mask frame not work
		animeFrames := make([][]byte, 0)
		maskFrames := make([][]byte, 0)
		for i, partition := range partitions.Partitions {
			if i%2 == 0 {
				maskFrames = append(maskFrames, data[partition.Offset:partition.Offset+partition.Size])
			} else {
				animeFrames = append(animeFrames, data[partition.Offset:partition.Offset+partition.Size])
			}
		}
		if FFmpegMode {
			mp4Data, err := ConvertAnime2GIF(animeFrames, maskFrames)
			if err != nil {
				return nil, "", err
			}
			return mp4Data, "gif", nil
		}
		mp4Data, err := TransmuxAnime2MP4(animeFrames, maskFrames)
		if err != nil {
			return nil, "", err
		}
		return mp4Data, "mp4", nil
	}

	offset := partitions.Partitions[partitions.MaxIndex].Offset
	size := partitions.Partitions[partitions.MaxIndex].Size

	if FFmpegMode {
		jpgData, err := Convert2JPG(data[offset : offset+size])
		if err != nil {
			return nil, "", err
		}
		return jpgData, JPG.Ext, nil
	}

	mp4Data, err := Transmux2MP4(data[offset : offset+size])
	if err != nil {
		return nil, "", err
	}
	return mp4Data, "mp4", nil
}

type Partitions struct {
	Partitions []Partition
	MaxRatio   float64
	MaxIndex   int
}

func (p *Partitions) LikeAnime() bool {
	return len(p.Partitions) > 1 && p.MaxRatio < MinRatio
}

type Partition struct {
	Offset int
	Size   int
	Ratio  float64
}

func findDataPartition(data []byte) (*Partitions, error) {

	headerLen := int(data[4])
	if headerLen >= len(data) {
		return nil, fmt.Errorf("invalid wxgf")
	}

	patterns := [][]byte{
		{0x00, 0x00, 0x00, 0x01},
		{0x00, 0x00, 0x01},
	}

	for _, pattern := range patterns {
		ret := &Partitions{
			Partitions: make([]Partition, 0),
		}
		offset := 0
		for {
			if headerLen+offset > len(data) {
				break
			}

			index := bytes.Index(data[headerLen+offset:], pattern)
			if index == -1 {
				break
			}

			absIndex := headerLen + offset + index

			if absIndex < 4 {
				offset += index + 1
				continue
			}

			length := int(data[absIndex-4])<<24 | int(data[absIndex-3])<<16 |
				int(data[absIndex-2])<<8 | int(data[absIndex-1])

			if length <= 0 || absIndex+length > len(data) {
				offset += index + 1
				continue
			}

			partition := Partition{
				Offset: absIndex,
				Size:   length,
				Ratio:  float64(length) / float64(len(data)),
			}
			ret.Partitions = append(ret.Partitions, partition)
			if partition.Ratio > ret.MaxRatio {
				ret.MaxRatio = partition.Ratio
				ret.MaxIndex = len(ret.Partitions) - 1
			}
			offset += index + length
		}

		if len(ret.Partitions) > 0 {
			return ret, nil
		}
	}

	return nil, fmt.Errorf("no partition found")
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

func writeTempFile(data [][]byte) (string, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("anime-%s", uuid.New().String()))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open anime temp file: %w", err)
	}
	defer file.Close()
	for _, frame := range data {
		_, err := file.Write(frame)
		if err != nil {
			return "", fmt.Errorf("failed to write anime frame to temp file: %w", err)
		}
	}
	return path, nil
}

// ConvertAnime2GIF convert anime frames and mask frames to mp4
// FIXME No longer need to write to temporary files
func ConvertAnime2GIF(animeFrames [][]byte, maskFrames [][]byte) ([]byte, error) {
	animeFilePath, err := writeTempFile(animeFrames)
	if err != nil {
		return nil, fmt.Errorf("failed to write anime temp file: %w", err)
	}
	defer os.Remove(animeFilePath)

	maskFilePath, err := writeTempFile(maskFrames)
	if err != nil {
		return nil, fmt.Errorf("failed to write mask temp file: %w", err)
	}
	defer os.Remove(maskFilePath)

	cmd := exec.Command(FFMpegPath,
		"-i", animeFilePath,
		"-i", maskFilePath,
		"-filter_complex", "[0:v][1:v]alphamerge,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse",
		"-f", "gif",
		"-")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w", err)
	}

	gifData := stdout.Bytes()
	if len(gifData) == 0 {
		return nil, fmt.Errorf("ffmpeg output is empty")
	}

	return gifData, nil
}

func isFFmpegAvailable() bool {
	cmd := exec.Command(FFMpegPath, "-version")
	return cmd.Run() == nil
}

func Transmux2MP4(data []byte) ([]byte, error) {

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

func TransmuxAnime2MP4(animeFrames [][]byte, maskFrames [][]byte) ([]byte, error) {

	if len(maskFrames) != len(animeFrames) {
		return nil, fmt.Errorf("mask frame num (%d) not equal to anime frame num (%d)", len(maskFrames), len(animeFrames))
	}

	init := mp4.CreateEmptyInit()
	seg := mp4.NewMediaSegment()
	trackIDs := []uint32{1, 2}
	frag, err := mp4.CreateMultiTrackFragment(1, trackIDs)
	if err != nil {
		return nil, err
	}
	seg.AddFragment(frag)

	err = Add2Trak(init, frag, 0, animeFrames)
	if err != nil {
		return nil, fmt.Errorf("add full sample to track failed: %w", err)
	}

	err = Add2Trak(init, frag, 1, maskFrames)
	if err != nil {
		return nil, fmt.Errorf("add full sample to track failed: %w", err)
	}

	totalSize := init.Size() + seg.Size()
	sw := bits.NewFixedSliceWriter(int(totalSize))

	init.EncodeSW(sw)
	seg.EncodeSW(sw)

	return sw.Bytes(), nil
}

func Add2Trak(init *mp4.InitSegment, frag *mp4.Fragment, index int, data [][]byte) error {
	videoTimescale := uint32(90000)
	init.AddEmptyTrack(videoTimescale, "video", "und")
	trak := init.Moov.Traks[index]

	vps, sps, pps := hevc.GetParameterSetsFromByteStream(data[0])

	// FIXME  Two slices reporting being the first in the same frame.
	if FixSliceHeaders {
		spsMap, ppsMap, err := createSPSPPSMaps(vps, sps, pps)
		if err != nil {
			return fmt.Errorf("create sps pps map failed: %w", err)
		}
		for i := range data {
			fixedFrame, err := fixSliceHeadersInFrame(data[i], spsMap, ppsMap)
			if err != nil {
				return fmt.Errorf("fix slice header failed: %w", err)
			}
			data[i] = fixedFrame
		}
	}

	err := trak.SetHEVCDescriptor("hev1", vps, sps, pps, nil, true)
	if err != nil {
		return fmt.Errorf("set trak failed: %w", err)
	}

	var decodeTime uint64 = 0
	frameDuration := uint32(3000)
	for i := 0; i < len(data); i++ {
		sampleData := avc.ConvertByteStreamToNaluSample(removeParameterSets(data[i]))
		simple := mp4.FullSample{
			Sample: mp4.Sample{
				Flags: getSampleFlags(sampleData, i == 0),
				Dur:   frameDuration,
				Size:  uint32(len(sampleData)),
			},
			DecodeTime: decodeTime,
			Data:       sampleData,
		}

		err = frag.AddFullSampleToTrack(simple, uint32(index+1))
		if err != nil {
			return err
		}
		decodeTime += uint64(frameDuration)
	}

	return nil
}

func getSampleFlags(frameData []byte, isFirstFrame bool) uint32 {
	if isFirstFrame || hevc.IsRAPSample(frameData) {
		return 0x02000000
	}
	return 0x01010000
}

func createSPSPPSMaps(vpsNalus, spsNalus, ppsNalus [][]byte) (map[uint32]*hevc.SPS, map[uint32]*hevc.PPS, error) {
	spsMap := make(map[uint32]*hevc.SPS)
	for _, spsNalu := range spsNalus {
		sps, err := hevc.ParseSPSNALUnit(spsNalu)
		if err != nil {
			return nil, nil, fmt.Errorf("parse sps failed: %w", err)
		}
		spsMap[uint32(sps.SpsID)] = sps
	}

	ppsMap := make(map[uint32]*hevc.PPS)
	for _, ppsNalu := range ppsNalus {
		pps, err := hevc.ParsePPSNALUnit(ppsNalu, spsMap)
		if err != nil {
			return nil, nil, fmt.Errorf("parse pps failed: %w", err)
		}
		ppsMap[pps.PicParameterSetID] = pps
	}

	return spsMap, ppsMap, nil
}

func fixSliceHeadersInFrame(frameData []byte, spsMap map[uint32]*hevc.SPS, ppsMap map[uint32]*hevc.PPS) ([]byte, error) {
	nalus := avc.ExtractNalusFromByteStream(frameData)
	var fixedNalus [][]byte
	var firstSliceFound bool

	for _, nalu := range nalus {
		naluType := hevc.GetNaluType(nalu[0])

		if naluType < hevc.NALU_TRAIL_N || naluType > hevc.NALU_CRA {
			fixedNalus = append(fixedNalus, nalu)
			continue
		}

		if !firstSliceFound {
			fixedNalus = append(fixedNalus, nalu)
			firstSliceFound = true
		} else {
			sliceHeader, err := hevc.ParseSliceHeader(nalu, spsMap, ppsMap)
			if err != nil {
				return nil, fmt.Errorf("parse slice header failed: %w", err)
			}
			if !sliceHeader.FirstSliceSegmentInPicFlag {
				fixedNalus = append(fixedNalus, nalu)
			}
		}
	}

	return reconstructAnnexBStream(fixedNalus), nil
}

func removeParameterSets(annexBData []byte) []byte {
	nalus := avc.ExtractNalusFromByteStream(annexBData)
	var videoNalus [][]byte

	for _, nalu := range nalus {
		naluType := hevc.GetNaluType(nalu[0])
		if naluType <= 31 {
			videoNalus = append(videoNalus, nalu)
		}
	}

	return reconstructAnnexBStream(videoNalus)
}

func reconstructAnnexBStream(nalus [][]byte) []byte {
	var result []byte
	startCode := []byte{0x00, 0x00, 0x01}

	for _, nalu := range nalus {
		result = append(result, startCode...)
		result = append(result, nalu...)
	}
	return result
}
