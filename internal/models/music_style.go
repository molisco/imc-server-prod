package models

import (
	"time"

	"github.com/google/uuid"
)

// MusicStyle represents a musical category in the system (entirely global)
type MusicStyle struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Description   string    `json:"description,omitempty" db:"description"`
	BaseDirectory string    `json:"base_directory,omitempty" db:"base_directory"`
	SourceType    string    `json:"source_type" db:"source_type"` // 'LOCAL' ou 'STREAM'
	StreamURL     string    `json:"stream_url,omitempty" db:"stream_url"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

