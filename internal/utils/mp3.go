package utils

import (
	"fmt"
	"io"
	"os"
)

// GetMP3Duration returns the duration of an MP3 file in seconds.
func GetMP3Duration(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Bitrates for Layer III (kbps)
	// index: [MPEG version (0: V2/V2.5, 1: V1)][bitrate index (0-15)]
	bitrates := [2][16]int{
		{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},     // MPEG 2 / 2.5
		{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}, // MPEG 1
	}

	// Sampling rates (Hz)
	// index: [MPEG version][sampling rate index]
	// MPEG version: 00: 2.5, 01: reserved, 10: V2, 11: V1
	samplerates := [4][4]int{
		{11025, 12000, 8000, 0},  // MPEG 2.5 (value 0)
		{0, 0, 0, 0},              // Reserved (value 1)
		{22050, 24000, 16000, 0}, // MPEG 2 (value 2)
		{44100, 48000, 32000, 0}, // MPEG 1 (value 3)
	}

	var duration float64
	var buf [4]byte

	// Skip ID3v2 tag if present
	_, _ = file.Seek(0, io.SeekStart)
	if _, err := file.Read(buf[:3]); err == nil && string(buf[:3]) == "ID3" {
		// Read ID3v2 header size (4 bytes starting at offset 6)
		var header [7]byte
		if _, err := file.Read(header[:]); err == nil {
			// Size is a synchsafe integer (4 bytes, 7 bits each)
			size := int(header[6]) & 0x7F |
				(int(header[5])&0x7F)<<7 |
				(int(header[4])&0x7F)<<14 |
				(int(header[3])&0x7F)<<21
			// Skip the ID3v2 tag (10 bytes header + size)
			_, _ = file.Seek(int64(10+size), io.SeekStart)
		}
	} else {
		_, _ = file.Seek(0, io.SeekStart)
	}

	tempBuf := make([]byte, 4)
	for {
		// Read 4 bytes for frame header
		_, err := io.ReadFull(file, tempBuf)
		if err != nil {
			break
		}

		// Look for syncword (11 bits set: 0xFF and top 3 bits of byte 1 set)
		if tempBuf[0] == 0xFF && (tempBuf[1]&0xE0) == 0xE0 {
			header := tempBuf

			// MPEG Audio Version
			versionIndex := (header[1] & 0x18) >> 3 // 00: 2.5, 02: V2, 03: V1
			version := 0                            // 0: MPEG 2/2.5, 1: MPEG 1
			if versionIndex == 3 {
				version = 1
			}

			// Layer
			layer := (header[1] & 0x06) >> 1 // 01: Layer III (MP3), 10: Layer II, 11: Layer I
			if layer != 1 {
				// Not Layer III, skip 1 byte and search again
				_, _ = file.Seek(-3, io.SeekCurrent)
				continue
			}

			// Bitrate
			bitrateIdx := (header[2] & 0xF0) >> 4
			bitrate := bitrates[version][bitrateIdx] * 1000 // bits per sec
			if bitrate == 0 {
				_, _ = file.Seek(-3, io.SeekCurrent)
				continue
			}

			// Sampling Rate
			srIdx := (header[2] & 0x0C) >> 2
			sampleRate := samplerates[versionIndex][srIdx]
			if sampleRate == 0 {
				_, _ = file.Seek(-3, io.SeekCurrent)
				continue
			}

			// Padding
			padding := (header[2] & 0x02) >> 1

			// Calculate Frame Size (Layer III formula)
			// Frame Size = (Samples Per Frame * Bitrate / (8 * Sample Rate)) + Padding
			samplesPerFrame := 1152
			if versionIndex != 3 { // MPEG 2 / 2.5
				samplesPerFrame = 576
			}

			frameSize := (samplesPerFrame * bitrate) / (8 * sampleRate) + int(padding)
			if frameSize < 4 {
				_, _ = file.Seek(-3, io.SeekCurrent)
				continue
			}

			// Add frame duration to total
			duration += float64(samplesPerFrame) / float64(sampleRate)

			// Skip the rest of the frame
			_, err = file.Seek(int64(frameSize-4), io.SeekCurrent)
			if err != nil {
				break
			}
		} else {
			// Skip 1 byte and scan again
			_, _ = file.Seek(-3, io.SeekCurrent)
		}
	}

	if duration == 0 {
		return 0, fmt.Errorf("no valid audio frames found")
	}

	return int(duration), nil
}
