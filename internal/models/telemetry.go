package models

import (
	"time"

	"github.com/google/uuid"
)

// PlayHistoryBatchItem representa uma linha de execução de áudio enviada pelo player local
type PlayHistoryBatchItem struct {
	TrackID          string  `json:"track_id"`
	TrackType        string  `json:"track_type"`
	DisplayName      string  `json:"display_name"`
	Artist           string  `json:"artist"`
	PlayedAt         string  `json:"played_at"`
	DurationSeconds  int     `json:"duration_seconds"`
	SkippedByCatchup int     `json:"skipped_by_catchup"`
	StyleName        string  `json:"style_name"`
	PlayStatus       string  `json:"play_status"`
	PlannedBlockTime string  `json:"planned_block_time"`
	CueIn            float64 `json:"cue_in"`
	CueOut           float64 `json:"cue_out"`
	LoudnessScore    string  `json:"loudness_score"`
	RiscoRuido       int     `json:"risco_ruido"`
	SkipReason       string  `json:"skip_reason"`
}

// PlayHistoryBatchRequest payload enviado pela rádio da loja
type PlayHistoryBatchRequest struct {
	Records []PlayHistoryBatchItem `json:"records"`
}

// PlayHistoryBatchResponse resposta da API do servidor
type PlayHistoryBatchResponse struct {
	Status        string `json:"status"`
	IngestedCount int    `json:"ingested_count"`
	Message       string `json:"message,omitempty"`
}

// StorePlayHistoryRecord registro gravado no PostgreSQL central para relatórios
type StorePlayHistoryRecord struct {
	ID               uuid.UUID `json:"id"`
	StoreID          uuid.UUID `json:"store_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	TrackID          string    `json:"track_id"`
	TrackType        string    `json:"track_type"`
	DisplayName      string    `json:"display_name"`
	Artist           string    `json:"artist"`
	PlayedAt         time.Time `json:"played_at"`
	DurationSeconds  int       `json:"duration_seconds"`
	StyleName        string    `json:"style_name"`
	PlayStatus       string    `json:"play_status"`
	PlannedBlockTime string    `json:"planned_block_time"`
	CreatedAt        time.Time `json:"created_at"`
}
