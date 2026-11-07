package models

import (
	"time"

	"github.com/google/uuid"
)

// VideoStyle represents a video category in the system (entirely global)
type VideoStyle struct {
	ID            uuid.UUID `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Description   string    `json:"description,omitempty" db:"description"`
	BaseDirectory string    `json:"base_directory,omitempty" db:"base_directory"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}
