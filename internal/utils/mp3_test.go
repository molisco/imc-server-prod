package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetMP3Duration_Valid(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_valid.mp3")

	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create temp mp3 file: %v", err)
	}

	// Write 40 valid MP3 frames (MPEG 1 Layer III, 32 kbps, 44100 Hz)
	// Frame Header: 0xFF, 0xFA, 0x10, 0x00
	// frameSize = (1152 * 32000) / (8 * 44100) = 104 bytes
	// We need 104 bytes per frame, so 4 bytes header + 100 bytes of zero payload.
	frameHeader := []byte{0xFF, 0xFA, 0x10, 0x00}
	framePayload := make([]byte, 100)

	for i := 0; i < 40; i++ {
		_, _ = file.Write(frameHeader)
		_, _ = file.Write(framePayload)
	}
	file.Close()

	duration, err := GetMP3Duration(filePath)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// 40 * (1152 / 44100) = 1.044 seconds, which should cast to 1
	if duration != 1 {
		t.Errorf("expected duration to be 1 second, got: %d", duration)
	}
}

func TestGetMP3Duration_Invalid(t *testing.T) {
	// Create a temporary file with random bytes
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_invalid.mp3")

	err := os.WriteFile(filePath, []byte("random non-mp3 content here"), 0644)
	if err != nil {
		t.Fatalf("failed to write temp invalid file: %v", err)
	}

	_, err = GetMP3Duration(filePath)
	if err == nil {
		t.Error("expected error for invalid mp3, but got nil")
	}
}
