package models

import (
	"time"

	"github.com/google/uuid"
)

// BasicSchedule represents a time window for playlists/musical blocks
type BasicSchedule struct {
	ID                         uuid.UUID `json:"id" db:"id"`
	TenantID                   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	MediaType                  MediaType `json:"media_type" db:"media_type"`
	Name                       string    `json:"name" db:"name"`
	Description                string    `json:"description,omitempty" db:"description"`
	StartTime                  string    `json:"start_time" db:"start_time"` // "HH:MM:SS"
	EndTime                    string    `json:"end_time" db:"end_time"`     // "HH:MM:SS"
	MusicalBlockDurationMinutes int       `json:"musical_block_duration_minutes" db:"musical_block_duration_minutes"`
	CustomDurationsJSON        *string   `json:"custom_durations_json,omitempty" db:"custom_durations_json"`
	CustomStylesJSON           *string   `json:"custom_styles_json,omitempty" db:"custom_styles_json"`
	OffHoursBehavior           string    `json:"off_hours_behavior" db:"off_hours_behavior"`
	CreatedAt                  time.Time `json:"created_at" db:"created_at"`
}

// CommercialBlockConfig represents interval rules and commercial limits
type CommercialBlockConfig struct {
	ID                         uuid.UUID `json:"id" db:"id"`
	TenantID                   uuid.UUID `json:"tenant_id" db:"tenant_id"`
	MediaType                  MediaType `json:"media_type" db:"media_type"`
	Name                       string    `json:"name" db:"name"`
	Description                string    `json:"description,omitempty" db:"description"`
	IntervalMinutes            int       `json:"interval_minutes" db:"interval_minutes"`
	BlockLimitSeconds          int       `json:"block_limit_seconds" db:"block_limit_seconds"`
	PercentageTimeAnnouncement int       `json:"percentage_time_announcement" db:"percentage_time_announcement"`
	PercentageTemperature      int       `json:"percentage_temperature" db:"percentage_temperature"`
	PercentageInstitutional    int       `json:"percentage_institutional" db:"percentage_institutional"`
	PercentageOffer            int       `json:"percentage_offer" db:"percentage_offer"`
	CreatedAt                  time.Time `json:"created_at" db:"created_at"`
}

// WeeklyMasterGrid binds days of the week to basic schedules and commercial block configs
type WeeklyMasterGrid struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	TenantTradeName string    `json:"tenant_trade_name" db:"tenant_trade_name"` // Auxiliary field for JOIN
	MediaType       MediaType `json:"media_type" db:"media_type"`
	Name            string    `json:"name" db:"name"`
	Description     string    `json:"description,omitempty" db:"description"`

	SundayScheduleID    *uuid.UUID `json:"sunday_schedule_id,omitempty" db:"sunday_schedule_id"`
	SundayBlockID       *uuid.UUID `json:"sunday_block_id,omitempty" db:"sunday_block_id"`
	MondayScheduleID    *uuid.UUID `json:"monday_schedule_id,omitempty" db:"monday_schedule_id"`
	MondayBlockID       *uuid.UUID `json:"monday_block_id,omitempty" db:"monday_block_id"`
	TuesdayScheduleID   *uuid.UUID `json:"tuesday_schedule_id,omitempty" db:"tuesday_schedule_id"`
	TuesdayBlockID      *uuid.UUID `json:"tuesday_block_id,omitempty" db:"tuesday_block_id"`
	WednesdayScheduleID *uuid.UUID `json:"wednesday_schedule_id,omitempty" db:"wednesday_schedule_id"`
	WednesdayBlockID    *uuid.UUID `json:"wednesday_block_id,omitempty" db:"wednesday_block_id"`
	ThursdayScheduleID  *uuid.UUID `json:"thursday_schedule_id,omitempty" db:"thursday_schedule_id"`
	ThursdayBlockID     *uuid.UUID `json:"thursday_block_id,omitempty" db:"thursday_block_id"`
	FridayScheduleID    *uuid.UUID `json:"friday_schedule_id,omitempty" db:"friday_schedule_id"`
	FridayBlockID       *uuid.UUID `json:"friday_block_id,omitempty" db:"friday_block_id"`
	SaturdayScheduleID  *uuid.UUID `json:"saturday_schedule_id,omitempty" db:"saturday_schedule_id"`
	SaturdayBlockID     *uuid.UUID `json:"saturday_block_id,omitempty" db:"saturday_block_id"`
	HolidayScheduleID   *uuid.UUID `json:"holiday_schedule_id,omitempty" db:"holiday_schedule_id"`
	HolidayBlockID      *uuid.UUID `json:"holiday_block_id,omitempty" db:"holiday_block_id"`
	AllowedManualStylesJSON *string   `json:"allowed_manual_styles_json,omitempty" db:"allowed_manual_styles_json"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
