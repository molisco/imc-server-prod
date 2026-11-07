package models

import (
	"time"

	"github.com/google/uuid"
)

// MusicTrack represents an MP3 file registered in a specific global music style.
type MusicTrack struct {
	ID              uuid.UUID `json:"id" db:"id"`
	MusicStyleID    uuid.UUID `json:"music_style_id" db:"music_style_id"`
	FileName        string    `json:"file_name" db:"file_name"`
	DisplayName     string    `json:"display_name" db:"display_name"`
	DurationSeconds int       `json:"duration_seconds" db:"duration_seconds"`
	Album           string    `json:"album" db:"album"`
	Title           string    `json:"title" db:"title"`
	Artist          string    `json:"artist" db:"artist"`
	Year            int       `json:"year" db:"year"`
	LUFS            float64   `json:"lufs" db:"lufs"`
	LRA             float64   `json:"lra" db:"lra"`
	LoudnessScore   string    `json:"loudness_score" db:"loudness_score"`
	CueIn           float64   `json:"cue_in" db:"cue_in"`
	CueOut          float64   `json:"cue_out" db:"cue_out"`
	FileSizeBytes   int64     `json:"file_size_bytes" db:"file_size_bytes"`
	BitRateKbps     int       `json:"bitrate_kbps" db:"bitrate_kbps"`
	SampleRateHz    int       `json:"sample_rate_hz" db:"sample_rate_hz"`
	RiscoRuido      int       `json:"risco_ruido" db:"risco_ruido"`
	ManualCue       bool      `json:"manual_cue" db:"manual_cue"`
	IsSonic         bool      `json:"is_sonic" db:"is_sonic"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
