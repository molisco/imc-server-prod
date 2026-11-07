package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"
	"testing"

	"github.com/google/uuid"
)

func TestMusicTrackCRUD(t *testing.T) {
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		t.Skip("Database not available. Skipping integration test.")
		return
	}

	styleRepo := NewMusicStyleRepository()
	trackRepo := NewMusicTrackRepository()
	ctx := context.Background()

	// Clean up potential leftovers
	database.Pool.Exec(ctx, "DELETE FROM music_styles WHERE name = $1", "Music Style for Track Test")

	styleID := uuid.New()
	style := models.MusicStyle{
		ID:            styleID,
		Name:          "Music Style for Track Test",
		Description:   "Description for track testing",
		BaseDirectory: "/media/audio/test_track",
	}

	err := styleRepo.Create(ctx, style)
	if err != nil {
		t.Fatalf("Failed to create parent music style: %v", err)
	}
	defer styleRepo.Delete(ctx, styleID)

	trackID := uuid.New()
	track := models.MusicTrack{
		ID:              trackID,
		MusicStyleID:    styleID,
		FileName:        "test_track_file.mp3",
		DisplayName:     "Test Track File",
		DurationSeconds: 150,
		Album:           "Test Album",
		Title:           "Test Title",
		Artist:          "Test Artist",
		Year:            2024,
		LUFS:            -14.2,
		LRA:             4.5,
		LoudnessScore:   "A",
		CueIn:           2.5,
		CueOut:          145.0,
		FileSizeBytes:   5000000,
		BitRateKbps:     320,
		SampleRateHz:    48000,
	}

	// 1. Create Track
	err = trackRepo.Create(ctx, track)
	if err != nil {
		t.Fatalf("Failed to create music track: %v", err)
	}

	// 2. Get By Style ID
	tracks, err := trackRepo.GetByStyleID(ctx, styleID)
	if err != nil {
		t.Fatalf("Failed to retrieve tracks by style: %v", err)
	}
	if len(tracks) != 1 {
		t.Errorf("Expected 1 track, got %d", len(tracks))
	} else {
		if tracks[0].FileName != "test_track_file.mp3" {
			t.Errorf("Expected filename 'test_track_file.mp3', got '%s'", tracks[0].FileName)
		}
		if tracks[0].CueIn != 2.5 || tracks[0].CueOut != 145.0 {
			t.Errorf("Expected CueIn 2.5 and CueOut 145.0, got %f and %f", tracks[0].CueIn, tracks[0].CueOut)
		}
		if tracks[0].FileSizeBytes != 5000000 || tracks[0].BitRateKbps != 320 || tracks[0].SampleRateHz != 48000 {
			t.Errorf("Expected size 5000000, bitrate 320, rate 48000, got %d, %d, %d", tracks[0].FileSizeBytes, tracks[0].BitRateKbps, tracks[0].SampleRateHz)
		}
	}

	// 3. Get By Style and FileName
	fetched, err := trackRepo.GetByStyleAndFileName(ctx, styleID, "test_track_file.mp3")
	if err != nil {
		t.Fatalf("Failed to retrieve track by style and filename: %v", err)
	}
	if fetched.ID != trackID {
		t.Errorf("Expected track ID '%s', got '%s'", trackID, fetched.ID)
	}
	if fetched.CueIn != 2.5 || fetched.CueOut != 145.0 {
		t.Errorf("Expected CueIn 2.5 and CueOut 145.0, got %f and %f", fetched.CueIn, fetched.CueOut)
	}
	if fetched.FileSizeBytes != 5000000 || fetched.BitRateKbps != 320 || fetched.SampleRateHz != 48000 {
		t.Errorf("Expected size 5000000, bitrate 320, rate 48000, got %d, %d, %d", fetched.FileSizeBytes, fetched.BitRateKbps, fetched.SampleRateHz)
	}

	// 3.5 Get By ID
	fetchedByID, err := trackRepo.GetByID(ctx, trackID)
	if err != nil {
		t.Fatalf("Failed to retrieve track by ID: %v", err)
	}
	if fetchedByID.ID != trackID {
		t.Errorf("Expected track ID '%s', got '%s'", trackID, fetchedByID.ID)
	}
	if fetchedByID.CueIn != 2.5 || fetchedByID.CueOut != 145.0 {
		t.Errorf("Expected CueIn 2.5 and CueOut 145.0, got %f and %f", fetchedByID.CueIn, fetchedByID.CueOut)
	}

	// 3.75 Test Update
	fetched.DisplayName = "Updated Test Track File"
	fetched.FileSizeBytes = 6000000
	fetched.BitRateKbps = 192
	fetched.SampleRateHz = 44100
	err = trackRepo.Update(ctx, *fetched)
	if err != nil {
		t.Fatalf("Failed to update music track: %v", err)
	}
	updated, err := trackRepo.GetByID(ctx, trackID)
	if err != nil {
		t.Fatalf("Failed to retrieve track after update: %v", err)
	}
	if updated.DisplayName != "Updated Test Track File" || updated.FileSizeBytes != 6000000 || updated.BitRateKbps != 192 || updated.SampleRateHz != 44100 {
		t.Errorf("Expected updated values, got Name: %s, Size: %d, Bitrate: %d, SampleRate: %d", updated.DisplayName, updated.FileSizeBytes, updated.BitRateKbps, updated.SampleRateHz)
	}

	// 4. Get Counts Map
	counts, err := trackRepo.GetCountsMap(ctx)
	if err != nil {
		t.Fatalf("Failed to get track counts: %v", err)
	}
	if counts[styleID] != 1 {
		t.Errorf("Expected count for style '%s' to be 1, got %d", styleID, counts[styleID])
	}

	// 5. Delete Track
	err = trackRepo.DeleteByID(ctx, trackID)
	if err != nil {
		t.Fatalf("Failed to delete music track: %v", err)
	}

	// Verify deletion
	tracksAfter, err := trackRepo.GetByStyleID(ctx, styleID)
	if err != nil {
		t.Fatalf("Failed to query tracks after delete: %v", err)
	}
	if len(tracksAfter) != 0 {
		t.Errorf("Expected 0 tracks, got %d", len(tracksAfter))
	}
}
