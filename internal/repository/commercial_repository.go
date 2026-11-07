package repository

import (
	"context"
	"time"

	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

type CommercialRepository struct{}

func NewCommercialRepository() *CommercialRepository {
	return &CommercialRepository{}
}

func (r *CommercialRepository) Create(ctx context.Context, c models.CommercialsAndEvents) error {
	query := `
		INSERT INTO commercials_and_events (
			id, tenant_id, media_type, commercial_type, name, file_name, duration_seconds, insertions_per_day,
			start_date, end_date, start_time, end_time,
			run_on_sunday, run_on_monday, run_on_tuesday, run_on_wednesday, run_on_thursday, run_on_friday, run_on_saturday,
			distribute_to_all, is_active, created_at, updated_at,
			lufs, lra, loudness_score, cue_in, cue_out, file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, execution_time, event_interruption_mode
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36)
	`
	createdAt := c.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := c.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	_, err := database.Pool.Exec(ctx, query,
		c.ID, c.TenantID, c.MediaType, c.CommercialType, c.Name, c.FileName, c.DurationSeconds, c.InsertionsPerDay,
		c.StartDate, c.EndDate, c.StartTime, c.EndTime,
		c.RunOnSunday, c.RunOnMonday, c.RunOnTuesday, c.RunOnWednesday, c.RunOnThursday, c.RunOnFriday, c.RunOnSaturday,
		c.DistributeToAll, c.IsActive, createdAt, updatedAt,
		c.LUFS, c.LRA, c.LoudnessScore, c.CueIn, c.CueOut, c.FileSizeBytes, c.BitrateKbps, c.SampleRateHz, c.RiscoRuido, c.ManualCue, c.IsSonic, c.ExecutionTime, c.EventInterruptionMode,
	)
	return err
}

func (r *CommercialRepository) Update(ctx context.Context, c models.CommercialsAndEvents) error {
	query := `
		UPDATE commercials_and_events
		SET tenant_id = $1, media_type = $2, commercial_type = $3, name = $4, file_name = $5, duration_seconds = $6, insertions_per_day = $7,
		    start_date = $8, end_date = $9, start_time = $10, end_time = $11,
		    run_on_sunday = $12, run_on_monday = $13, run_on_tuesday = $14, run_on_wednesday = $15, run_on_thursday = $16,
		    run_on_friday = $17, run_on_saturday = $18, distribute_to_all = $19, is_active = $20,
		    lufs = $21, lra = $22, loudness_score = $23, cue_in = $24, cue_out = $25, file_size_bytes = $26,
		    bitrate_kbps = $27, sample_rate_hz = $28, risco_ruido = $29, manual_cue = $30, is_sonic = $31, execution_time = $32,
		    event_interruption_mode = $33,
		    updated_at = NOW()
		WHERE id = $34
	`
	_, err := database.Pool.Exec(ctx, query,
		c.TenantID, c.MediaType, c.CommercialType, c.Name, c.FileName, c.DurationSeconds, c.InsertionsPerDay,
		c.StartDate, c.EndDate, c.StartTime, c.EndTime,
		c.RunOnSunday, c.RunOnMonday, c.RunOnTuesday, c.RunOnWednesday, c.RunOnThursday, c.RunOnFriday, c.RunOnSaturday,
		c.DistributeToAll, c.IsActive,
		c.LUFS, c.LRA, c.LoudnessScore, c.CueIn, c.CueOut, c.FileSizeBytes,
		c.BitrateKbps, c.SampleRateHz, c.RiscoRuido, c.ManualCue, c.IsSonic, c.ExecutionTime,
		c.EventInterruptionMode,
		c.ID,
	)
	return err
}

func (r *CommercialRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM commercials_and_events WHERE id = $1", id)
	return err
}

func (r *CommercialRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.CommercialsAndEvents, error) {
	query := `
		SELECT id, tenant_id, media_type, commercial_type, name, file_name, duration_seconds, insertions_per_day,
		       start_date, end_date, start_time, end_time,
		       run_on_sunday, run_on_monday, run_on_tuesday, run_on_wednesday, run_on_thursday, run_on_friday, run_on_saturday,
		       distribute_to_all, is_active, created_at, updated_at,
		       lufs, lra, loudness_score, cue_in, cue_out, file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, execution_time, event_interruption_mode
		FROM commercials_and_events
		WHERE id = $1
	`
	var c models.CommercialsAndEvents
	err := database.Pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.TenantID, &c.MediaType, &c.CommercialType, &c.Name, &c.FileName, &c.DurationSeconds, &c.InsertionsPerDay,
		&c.StartDate, &c.EndDate, &c.StartTime, &c.EndTime,
		&c.RunOnSunday, &c.RunOnMonday, &c.RunOnTuesday, &c.RunOnWednesday, &c.RunOnThursday, &c.RunOnFriday, &c.RunOnSaturday,
		&c.DistributeToAll, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
		&c.LUFS, &c.LRA, &c.LoudnessScore, &c.CueIn, &c.CueOut, &c.FileSizeBytes, &c.BitrateKbps, &c.SampleRateHz, &c.RiscoRuido, &c.ManualCue, &c.IsSonic, &c.ExecutionTime, &c.EventInterruptionMode,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CommercialRepository) GetAll(ctx context.Context) ([]models.CommercialsAndEvents, error) {
	query := `
		SELECT id, tenant_id, media_type, commercial_type, name, file_name, duration_seconds, insertions_per_day,
		       start_date, end_date, start_time, end_time,
		       run_on_sunday, run_on_monday, run_on_tuesday, run_on_wednesday, run_on_thursday, run_on_friday, run_on_saturday,
		       distribute_to_all, is_active, created_at, updated_at,
		       lufs, lra, loudness_score, cue_in, cue_out, file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, execution_time, event_interruption_mode
		FROM commercials_and_events
		ORDER BY created_at DESC
	`
	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.CommercialsAndEvents
	for rows.Next() {
		var c models.CommercialsAndEvents
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.MediaType, &c.CommercialType, &c.Name, &c.FileName, &c.DurationSeconds, &c.InsertionsPerDay,
			&c.StartDate, &c.EndDate, &c.StartTime, &c.EndTime,
			&c.RunOnSunday, &c.RunOnMonday, &c.RunOnTuesday, &c.RunOnWednesday, &c.RunOnThursday, &c.RunOnFriday, &c.RunOnSaturday,
			&c.DistributeToAll, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
			&c.LUFS, &c.LRA, &c.LoudnessScore, &c.CueIn, &c.CueOut, &c.FileSizeBytes, &c.BitrateKbps, &c.SampleRateHz, &c.RiscoRuido, &c.ManualCue, &c.IsSonic, &c.ExecutionTime, &c.EventInterruptionMode,
		)
		if err == nil {
			list = append(list, c)
		}
	}
	return list, nil
}

func (r *CommercialRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) ([]models.CommercialsAndEvents, error) {
	query := `
		SELECT id, tenant_id, media_type, commercial_type, name, file_name, duration_seconds, insertions_per_day,
		       start_date, end_date, start_time, end_time,
		       run_on_sunday, run_on_monday, run_on_tuesday, run_on_wednesday, run_on_thursday, run_on_friday, run_on_saturday,
		       distribute_to_all, is_active, created_at, updated_at,
		       lufs, lra, loudness_score, cue_in, cue_out, file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, execution_time, event_interruption_mode
		FROM commercials_and_events
		WHERE tenant_id = $1 OR tenant_id = '00000000-0000-0000-0000-000000000000'
		ORDER BY created_at DESC
	`
	rows, err := database.Pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.CommercialsAndEvents
	for rows.Next() {
		var c models.CommercialsAndEvents
		err := rows.Scan(
			&c.ID, &c.TenantID, &c.MediaType, &c.CommercialType, &c.Name, &c.FileName, &c.DurationSeconds, &c.InsertionsPerDay,
			&c.StartDate, &c.EndDate, &c.StartTime, &c.EndTime,
			&c.RunOnSunday, &c.RunOnMonday, &c.RunOnTuesday, &c.RunOnWednesday, &c.RunOnThursday, &c.RunOnFriday, &c.RunOnSaturday,
			&c.DistributeToAll, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
			&c.LUFS, &c.LRA, &c.LoudnessScore, &c.CueIn, &c.CueOut, &c.FileSizeBytes, &c.BitrateKbps, &c.SampleRateHz, &c.RiscoRuido, &c.ManualCue, &c.IsSonic, &c.ExecutionTime, &c.EventInterruptionMode,
		)
		if err == nil {
			list = append(list, c)
		}
	}
	return list, nil
}

func (r *CommercialRepository) GetDistributedStores(ctx context.Context, commercialID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT store_id FROM commercial_distributions WHERE commercial_id = $1`
	rows, err := database.Pool.Query(ctx, query, commercialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []uuid.UUID
	for rows.Next() {
		var storeID uuid.UUID
		if err := rows.Scan(&storeID); err == nil {
			list = append(list, storeID)
		}
	}
	return list, nil
}

func (r *CommercialRepository) SaveDistribution(ctx context.Context, commercialID uuid.UUID, storeIDs []uuid.UUID) error {
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Delete current mappings
	_, err = tx.Exec(ctx, "DELETE FROM commercial_distributions WHERE commercial_id = $1", commercialID)
	if err != nil {
		return err
	}

	// 2. Insert new mappings
	for _, storeID := range storeIDs {
		_, err = tx.Exec(ctx, "INSERT INTO commercial_distributions (commercial_id, store_id) VALUES ($1, $2)", commercialID, storeID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *CommercialRepository) ExistsByName(ctx context.Context, tenantID uuid.UUID, name string) (bool, error) {
	query := `SELECT COUNT(*) FROM commercials_and_events WHERE tenant_id = $1 AND name = $2`
	var count int
	err := database.Pool.QueryRow(ctx, query, tenantID, name).Scan(&count)
	return count > 0, err
}
