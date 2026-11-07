package models

import (
	"time"
	"github.com/google/uuid"
)

// Store represents a physical retail shop executing the edge C# Player.
type Store struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	TenantTradeName string    `json:"tenant_trade_name" db:"tenant_trade_name"` // Retornado via JOIN
	RegionalID      *uuid.UUID `json:"regional_id" db:"regional_id"`
	RegionalName    *string    `json:"regional_name" db:"regional_name"`       // Retornado via JOIN
	StoreName       string    `json:"store_name" db:"store_name"`
	StoreNumber     string    `json:"store_number,omitempty" db:"store_number"` // Customer internal tag (e.g. 'Loja 45')

	// Address Fields
	ZipCode  string `json:"zip_code,omitempty" db:"zip_code"`
	Street   string `json:"street,omitempty" db:"street"`
	Number   string `json:"number,omitempty" db:"number"`
	District string `json:"district,omitempty" db:"district"`
	City     string `json:"city" db:"city"`
	State    string `json:"state" db:"state"`
	Timezone string `json:"timezone" db:"timezone"`

	// Local Contact
	Phone       string `json:"phone,omitempty" db:"phone"`
	ManagerName string `json:"manager_name,omitempty" db:"manager_name"`

	// Media Flags (Audio/Video hardware setup)
	HasAudio bool `json:"has_audio" db:"has_audio"`
	HasVideo bool `json:"has_video" db:"has_video"`

	// Sincronismo (Last time the player checked in)
	LastAudioSync *time.Time `json:"last_audio_sync,omitempty" db:"last_audio_sync"`
	LastVideoSync *time.Time `json:"last_video_sync,omitempty" db:"last_video_sync"`

	// Authentication Token (Security key for config.ini)
	Token string `json:"token" db:"token"`

	// Master Grids Assigned to this specific store
	AudioGridID *uuid.UUID `json:"audio_grid_id,omitempty" db:"audio_grid_id"`
	VideoGridID *uuid.UUID `json:"video_grid_id,omitempty" db:"video_grid_id"`

	AudioGridName *string `json:"audio_grid_name,omitempty" db:"audio_grid_name"` // Retornado via JOIN
	VideoGridName *string `json:"video_grid_name,omitempty" db:"video_grid_name"` // Retornado via JOIN
	RadioType          string  `json:"radio_type" db:"radio_type"`
	HarborPort         *int    `json:"harbor_port,omitempty" db:"harbor_port"`
	ExternalStreamURL  *string `json:"external_stream_url,omitempty" db:"external_stream_url"`
	ExternalStreamName *string `json:"external_stream_name,omitempty" db:"external_stream_name"`

	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}