package models

import (
	"time"

	"github.com/google/uuid"
)

// StreamingStation representa uma estação de rádio ou streaming cadastrado no sistema
type StreamingStation struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	StreamURL        string    `json:"stream_url"`
	Genre            string    `json:"genre"`
	Bitrate          string    `json:"bitrate"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	StoresCount      int       `json:"stores_count"`
	AssignedStores   []string  `json:"assigned_stores"`
	ConnectionStatus string    `json:"connection_status"`
	IsOnline         bool      `json:"is_online"`
	IsLiveInSystem   bool      `json:"is_live_in_system"`
}

type StreamingPageData struct {
	Title          string
	LoggedUser     string
	TotalStations  int
	InUseStations  int
	TotalStoresOn  int
	Stations       []StreamingStation
	ErrorMessage   string
	SuccessMessage string
}
