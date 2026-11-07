package models

import "time"

type VoicePackAudioFile struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SizeFormatted string `json:"size_formatted"`
}

type VoicePack struct {
	ID               string               `json:"id" db:"id"`
	TenantID         *string              `json:"tenant_id" db:"tenant_id"`
	Name             string               `json:"name" db:"name"`
	Category         string               `json:"category" db:"category"`
	Description      string               `json:"description" db:"description"`
	IsGlobal         bool                 `json:"is_global" db:"is_global"`
	BaseDirectory    string               `json:"base_directory" db:"base_directory"`
	IsActive         bool                 `json:"is_active" db:"is_active"`
	CreatedAt        time.Time            `json:"created_at" db:"created_at"`
	AudioFiles       []VoicePackAudioFile `json:"audio_files"`
	TargetStoreIDs   []string             `json:"target_store_ids"`
	TargetStoreNames []string             `json:"target_store_names"`
}

type Chime struct {
	ID              string    `json:"id" db:"id"`
	TenantID        *string   `json:"tenant_id" db:"tenant_id"`
	Name            string    `json:"name" db:"name"`
	FileName        string    `json:"file_name" db:"file_name"`
	DurationSeconds int       `json:"duration_seconds" db:"duration_seconds"`
	CueIn           float64   `json:"cue_in" db:"cue_in"`
	CueOut          float64   `json:"cue_out" db:"cue_out"`
	LUFS            float64   `json:"lufs" db:"lufs"`
	Amplify         float64   `json:"amplify" db:"amplify"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type PreFormattedWarning struct {
	ID              string    `json:"id" db:"id"`
	TenantID        *string   `json:"tenant_id" db:"tenant_id"`
	Name            string    `json:"name" db:"name"`
	FileName        string    `json:"file_name" db:"file_name"`
	DurationSeconds int       `json:"duration_seconds" db:"duration_seconds"`
	CueIn           float64   `json:"cue_in" db:"cue_in"`
	CueOut          float64   `json:"cue_out" db:"cue_out"`
	LUFS            float64   `json:"lufs" db:"lufs"`
	Amplify         float64   `json:"amplify" db:"amplify"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}
