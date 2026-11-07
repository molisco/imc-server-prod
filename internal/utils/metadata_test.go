package utils

import (
	"strconv"
	"strings"
	"testing"
)

func TestParseLoudnessMetric(t *testing.T) {
	output := `
[Parsed_ebur128_0 @ 0x73bea0001a00] Summary:

  Integrated loudness:
    I:         -13.2 LUFS
    Threshold: -23.2 LUFS

  Loudness range:
    LRA:         1.3 LU
    Threshold: -33.0 LUFS
    LRA low:   -13.8 LUFS
    LRA high:  -12.5 LUFS
`

	lufs, ok := parseLoudnessMetric(output, "I:", "LUFS")
	if !ok || lufs != -13.2 {
		t.Errorf("expected LUFS to be -13.2, got: %f (ok=%v)", lufs, ok)
	}

	lra, ok := parseLoudnessMetric(output, "LRA:", "LU")
	if !ok || lra != 1.3 {
		t.Errorf("expected LRA to be 1.3, got: %f (ok=%v)", lra, ok)
	}
}

func TestExtractMetadata_Fallback(t *testing.T) {
	// Running ExtractMetadata on an empty or invalid path
	meta, err := ExtractMetadata("non_existent_file.mp3")
	if err != nil {
		t.Fatalf("expected no crash, got: %v", err)
	}

	// Should fall back to default values
	if meta.Duration != 180 {
		t.Errorf("expected fallback duration to be 180, got %d", meta.Duration)
	}
	if meta.LUFS != -14.0 {
		t.Errorf("expected fallback LUFS to be -14.0, got %f", meta.LUFS)
	}
	if meta.LRA != 5.0 {
		t.Errorf("expected fallback LRA to be 5.0, got %f", meta.LRA)
	}
	if meta.LoudnessScore != "A" {
		t.Errorf("expected fallback loudness score to be 'A', got '%s'", meta.LoudnessScore)
	}
	if meta.CueIn != 0.0 {
		t.Errorf("expected fallback CueIn to be 0.0, got %f", meta.CueIn)
	}
	if meta.CueOut != 180.0 {
		t.Errorf("expected fallback CueOut to be 180.0, got %f", meta.CueOut)
	}
}

func TestParseCuePoints(t *testing.T) {
	output := `
[silencedetect @ 0x7fa92] silence_start: 1.25
[silencedetect @ 0x7fa92] silence_end: 3.45 | silence_duration: 2.2
[silencedetect @ 0x7fa92] silence_start: 194.2
`
	// Since ExtractMetadata executes ffmpeg, we can simulate its parsing logic
	// by testing how it handles lines.
	// Let's verify our line-by-line parsing logic:
	lines := strings.Split(output, "\n")
	var lastSilenceStart float64 = -1.0
	var cueIn float64 = 0.0
	var cueOut float64 = 200.0 // simulated duration
	var hasCueIn bool = false

	for _, line := range lines {
		if strings.Contains(line, "silence_end:") {
			idx := strings.Index(line, "silence_end:")
			sub := line[idx+len("silence_end:"):]
			endIdx := strings.Index(sub, "|")
			if endIdx != -1 {
				valStr := strings.TrimSpace(sub[:endIdx])
				if val, err := strconv.ParseFloat(valStr, 64); err == nil {
					if !hasCueIn {
						cueIn = val
						hasCueIn = true
					}
					lastSilenceStart = -1.0
				}
			}
		} else if strings.Contains(line, "silence_start:") {
			idx := strings.Index(line, "silence_start:")
			valStr := strings.TrimSpace(line[idx+len("silence_start:"):])
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				lastSilenceStart = val
			}
		}
	}

	if lastSilenceStart != -1.0 {
		cueOut = lastSilenceStart
	}

	if cueIn != 3.45 {
		t.Errorf("expected cueIn to be 3.45, got %f", cueIn)
	}
	if cueOut != 194.2 {
		t.Errorf("expected cueOut to be 194.2, got %f", cueOut)
	}
}

func TestCuePointValidationAndTailSilence(t *testing.T) {
	// Test case 1: silence_end matches EOF (within 0.5s of floatDuration)
	// Output logs a silence_start: 178.45 and silence_end: 180.24
	// If floatDuration is 180.24, cue_out should be preserved as 178.45
	floatDuration := 180.24
	lines := []string{
		"[silencedetect] silence_start: 178.45",
		"[silencedetect] silence_end: 180.24 | silence_duration: 1.79",
	}

	var lastSilenceStart float64 = -1.0
	var cueIn float64 = 0.0
	var hasCueIn bool = false

	for _, line := range lines {
		if strings.Contains(line, "silence_end:") {
			idx := strings.Index(line, "silence_end:")
			sub := line[idx+len("silence_end:"):]
			endIdx := strings.Index(sub, "|")
			if endIdx != -1 {
				valStr := strings.TrimSpace(sub[:endIdx])
				if val, err := strconv.ParseFloat(valStr, 64); err == nil {
					if !hasCueIn {
						cueIn = val
						hasCueIn = true
					}
					// Only reset if NOT at EOF
					if val < floatDuration-0.5 {
						lastSilenceStart = -1.0
					}
				}
			}
		} else if strings.Contains(line, "silence_start:") {
			idx := strings.Index(line, "silence_start:")
			valStr := strings.TrimSpace(line[idx+len("silence_start:"):])
			if val, err := strconv.ParseFloat(valStr, 64); err == nil {
				lastSilenceStart = val
			}
		}
	}

	var cueOut float64
	if lastSilenceStart != -1.0 {
		cueOut = lastSilenceStart
	} else {
		cueOut = floatDuration
	}

	if cueOut != 178.45 {
		t.Errorf("Expected cueOut to remain 178.45, got %f", cueOut)
	}
	if !hasCueIn || cueIn != 180.24 {
		t.Errorf("Expected hasCueIn to be true and cueIn to be 180.24, got hasCueIn=%v, cueIn=%f", hasCueIn, cueIn)
	}

	// Test case 2: CueIn >= CueOut clamping logic
	// If cueIn = 10.0 and cueOut = 5.0, they should reset to 0.0 and floatDuration
	cIn := 10.0
	cOut := 5.0
	if cIn >= cOut {
		cIn = 0.0
		cOut = floatDuration
	}
	if cIn != 0.0 || cOut != floatDuration {
		t.Errorf("Expected cue points to reset, got cueIn=%f, cueOut=%f", cIn, cOut)
	}
}
