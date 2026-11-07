package models

import (
	"time"
	"github.com/google/uuid"
)

// MediaType represents whether the file is AUDIO or VIDEO
type MediaType string

const (
	MediaAudio MediaType = "AUDIO"
	MediaVideo MediaType = "VIDEO"
)

// CommercialType defines the purpose of an advertisement track
type CommercialType string

const (
	CommercialInstitutional CommercialType = "INSTITUTIONAL"
	CommercialBlockOpening  CommercialType = "BLOCK_OPENING"
	CommercialBlockClosing  CommercialType = "BLOCK_CLOSING"
	CommercialStandard      CommercialType = "COMMERCIAL"
	CommercialOffer         CommercialType = "OFFER"
	CommercialEvent         CommercialType = "EVENT"
	CommercialGeneralInfo   CommercialType = "GENERAL_INFORMATION"
	CommercialRightTime     CommercialType = "RIGHT_TIME"
	CommercialTemperature   CommercialType = "TEMPERATURE"
)


// MediaLibrary represents a specific song or basic video file track
type MediaLibrary struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	MediaStyleID    uuid.UUID `json:"media_style_id" db:"media_style_id"`
	MediaType       MediaType `json:"media_type" db:"media_type"`
	DisplayName     string    `json:"display_name" db:"display_name"`
	FileName        string    `json:"file_name" db:"file_name"`
	DurationSeconds int       `json:"duration_seconds" db:"duration_seconds"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// CommercialsAndEvents represents targeted ads with specific weekday and date rules
type CommercialsAndEvents struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	TenantID        uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	MediaType       MediaType      `json:"media_type" db:"media_type"`
	CommercialType  CommercialType `json:"commercial_type" db:"commercial_type"`
	Name            string         `json:"name" db:"name"`
	FileName        string         `json:"file_name" db:"file_name"`
	DurationSeconds int            `json:"duration_seconds" db:"duration_seconds"`
	InsertionsPerDay int           `json:"insertions_per_day" db:"insertions_per_day"`

	// Date and Time Validity (Pointers handle nullable fields gracefully)
	StartDate *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty" db:"end_date"`
	StartTime string     `json:"start_time" db:"start_time"` // Kept as string for simple "HH:MM:SS" database matching
	EndTime   string     `json:"end_time" db:"end_time"`

	// Weekday execution flags
	RunOnSunday    bool `json:"run_on_sunday" db:"run_on_sunday"`
	RunOnMonday    bool `json:"run_on_monday" db:"run_on_monday"`
	RunOnTuesday   bool `json:"run_on_tuesday" db:"run_on_tuesday"`
	RunOnWednesday bool `json:"run_on_wednesday" db:"run_on_wednesday"`
	RunOnThursday  bool `json:"run_on_thursday" db:"run_on_thursday"`
	RunOnFriday    bool `json:"run_on_friday" db:"run_on_friday"`
	RunOnSaturday  bool `json:"run_on_saturday" db:"run_on_saturday"`

	DistributeToAll bool      `json:"distribute_to_all" db:"distribute_to_all"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`

	// Quality Metrics and Event specific fields
	LUFS            float64        `json:"lufs" db:"lufs"`
	LRA             float64        `json:"lra" db:"lra"`
	LoudnessScore   string         `json:"loudness_score" db:"loudness_score"`
	CueIn           float64        `json:"cue_in" db:"cue_in"`
	CueOut          float64        `json:"cue_out" db:"cue_out"`
	FileSizeBytes   int64          `json:"file_size_bytes" db:"file_size_bytes"`
	BitrateKbps     int            `json:"bitrate_kbps" db:"bitrate_kbps"`
	SampleRateHz    int            `json:"sample_rate_hz" db:"sample_rate_hz"`
	RiscoRuido      int            `json:"risco_ruido" db:"risco_ruido"`
	ManualCue       bool           `json:"manual_cue" db:"manual_cue"`
	IsSonic         bool           `json:"is_sonic" db:"is_sonic"`
	ExecutionTime   *string        `json:"execution_time,omitempty" db:"execution_time"`
	EventInterruptionMode string    `json:"event_interruption_mode" db:"event_interruption_mode"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
}