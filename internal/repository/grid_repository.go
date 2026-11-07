package repository

import (
	"context"
	"errors"
	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

type GridRepository struct{}

func NewGridRepository() *GridRepository {
	return &GridRepository{}
}

// Create insere uma nova grade e suas dependências em uma transação
func (r *GridRepository) Create(ctx context.Context, grid *models.WeeklyMasterGrid, schedules map[string]*models.BasicSchedule, configs map[string]*models.CommercialBlockConfig) error {
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	insertSchedule := func(s *models.BasicSchedule) (uuid.UUID, error) {
		if s.ID == uuid.Nil {
			s.ID = uuid.New()
		}
		if s.OffHoursBehavior == "" {
			s.OffHoursBehavior = "SILENCE"
		}
		query := `
			INSERT INTO basic_schedules (
				id, tenant_id, media_type, name, description, start_time, end_time, musical_block_duration_minutes,
				custom_durations_json, custom_styles_json, off_hours_behavior, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
			)
		`
		_, err := tx.Exec(ctx, query,
			s.ID, s.TenantID, s.MediaType, s.Name, s.Description, s.StartTime, s.EndTime, s.MusicalBlockDurationMinutes,
			s.CustomDurationsJSON, s.CustomStylesJSON, s.OffHoursBehavior,
		)
		return s.ID, err
	}

	insertConfig := func(c *models.CommercialBlockConfig) (uuid.UUID, error) {
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
		query := `
			INSERT INTO commercial_block_configs (
				id, tenant_id, media_type, name, description, interval_minutes, block_limit_seconds,
				percentage_time_announcement, percentage_temperature, percentage_institutional, percentage_offer, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
			)
		`
		_, err := tx.Exec(ctx, query,
			c.ID, c.TenantID, c.MediaType, c.Name, c.Description, c.IntervalMinutes, c.BlockLimitSeconds,
			c.PercentageTimeAnnouncement, c.PercentageTemperature, c.PercentageInstitutional, c.PercentageOffer,
		)
		return c.ID, err
	}

	// Auxiliar para processar o vinculo do dia
	processDay := func(dayKey string, schedPtr **uuid.UUID, blockPtr **uuid.UUID) error {
		if s, ok := schedules[dayKey]; ok && s != nil {
			id, err := insertSchedule(s)
			if err != nil {
				return err
			}
			*schedPtr = &id
		}
		if c, ok := configs[dayKey]; ok && c != nil {
			id, err := insertConfig(c)
			if err != nil {
				return err
			}
			*blockPtr = &id
		}
		return nil
	}

	if err := processDay("mon", &grid.MondayScheduleID, &grid.MondayBlockID); err != nil { return err }
	if err := processDay("tue", &grid.TuesdayScheduleID, &grid.TuesdayBlockID); err != nil { return err }
	if err := processDay("wed", &grid.WednesdayScheduleID, &grid.WednesdayBlockID); err != nil { return err }
	if err := processDay("thu", &grid.ThursdayScheduleID, &grid.ThursdayBlockID); err != nil { return err }
	if err := processDay("fri", &grid.FridayScheduleID, &grid.FridayBlockID); err != nil { return err }
	if err := processDay("sat", &grid.SaturdayScheduleID, &grid.SaturdayBlockID); err != nil { return err }
	if err := processDay("sun", &grid.SundayScheduleID, &grid.SundayBlockID); err != nil { return err }
	if err := processDay("holiday", &grid.HolidayScheduleID, &grid.HolidayBlockID); err != nil { return err }

	if grid.ID == uuid.Nil {
		grid.ID = uuid.New()
	}

	queryGrid := `
		INSERT INTO weekly_master_grids (
			id, tenant_id, media_type, name, description,
			sunday_schedule_id, sunday_block_id,
			monday_schedule_id, monday_block_id,
			tuesday_schedule_id, tuesday_block_id,
			wednesday_schedule_id, wednesday_block_id,
			thursday_schedule_id, thursday_block_id,
			friday_schedule_id, friday_block_id,
			saturday_schedule_id, saturday_block_id,
			holiday_schedule_id, holiday_block_id,
			allowed_manual_styles_json,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9,
			$10, $11,
			$12, $13,
			$14, $15,
			$16, $17,
			$18, $19,
			$20, $21,
			$22,
			NOW()
		)
	`
	_, err = tx.Exec(ctx, queryGrid,
		grid.ID, grid.TenantID, grid.MediaType, grid.Name, grid.Description,
		grid.SundayScheduleID, grid.SundayBlockID,
		grid.MondayScheduleID, grid.MondayBlockID,
		grid.TuesdayScheduleID, grid.TuesdayBlockID,
		grid.WednesdayScheduleID, grid.WednesdayBlockID,
		grid.ThursdayScheduleID, grid.ThursdayBlockID,
		grid.FridayScheduleID, grid.FridayBlockID,
		grid.SaturdayScheduleID, grid.SaturdayBlockID,
		grid.HolidayScheduleID, grid.HolidayBlockID, grid.AllowedManualStylesJSON,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetAll retorna todas as grades mestre cadastradas com JOIN do tenant
func (r *GridRepository) GetAll(ctx context.Context, filterTenantID *uuid.UUID) ([]models.WeeklyMasterGrid, error) {
	query := `
		SELECT g.id, g.tenant_id, t.trade_name AS tenant_trade_name, g.media_type, g.name, COALESCE(g.description, '') AS description,
		       g.sunday_schedule_id, g.sunday_block_id,
		       g.monday_schedule_id, g.monday_block_id,
		       g.tuesday_schedule_id, g.tuesday_block_id,
		       g.wednesday_schedule_id, g.wednesday_block_id,
		       g.thursday_schedule_id, g.thursday_block_id,
		       g.friday_schedule_id, g.friday_block_id,
		       g.saturday_schedule_id, g.saturday_block_id,
		       g.holiday_schedule_id, g.holiday_block_id,
		       g.allowed_manual_styles_json,
		       g.created_at
		FROM weekly_master_grids g
		INNER JOIN tenants t ON g.tenant_id = t.id
	`
	var args []interface{}
	if filterTenantID != nil {
		query += " WHERE g.tenant_id = $1"
		args = append(args, *filterTenantID)
	}
	query += " ORDER BY t.trade_name ASC, g.name ASC"

	rows, err := database.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grids []models.WeeklyMasterGrid
	for rows.Next() {
		var g models.WeeklyMasterGrid
		err := rows.Scan(
			&g.ID, &g.TenantID, &g.TenantTradeName, &g.MediaType, &g.Name, &g.Description,
			&g.SundayScheduleID, &g.SundayBlockID,
			&g.MondayScheduleID, &g.MondayBlockID,
			&g.TuesdayScheduleID, &g.TuesdayBlockID,
			&g.WednesdayScheduleID, &g.WednesdayBlockID,
			&g.ThursdayScheduleID, &g.ThursdayBlockID,
			&g.FridayScheduleID, &g.FridayBlockID,
			&g.SaturdayScheduleID, &g.SaturdayBlockID,
			&g.HolidayScheduleID, &g.HolidayBlockID, &g.AllowedManualStylesJSON,
			&g.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		grids = append(grids, g)
	}

	return grids, nil
}

// GetByID recupera uma grade semanal pelo seu ID, juntamente com seus schedules e block configs dos 8 dias
func (r *GridRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.WeeklyMasterGrid, map[string]*models.BasicSchedule, map[string]*models.CommercialBlockConfig, error) {
	queryGrid := `
		SELECT id, tenant_id, media_type, name, COALESCE(description, '') AS description,
		       sunday_schedule_id, sunday_block_id,
		       monday_schedule_id, monday_block_id,
		       tuesday_schedule_id, tuesday_block_id,
		       wednesday_schedule_id, wednesday_block_id,
		       thursday_schedule_id, thursday_block_id,
		       friday_schedule_id, friday_block_id,
		       saturday_schedule_id, saturday_block_id,
		       holiday_schedule_id, holiday_block_id,
		       allowed_manual_styles_json,
		       created_at
		FROM weekly_master_grids
		WHERE id = $1
	`
	var g models.WeeklyMasterGrid
	err := database.Pool.QueryRow(ctx, queryGrid, id).Scan(
		&g.ID, &g.TenantID, &g.MediaType, &g.Name, &g.Description,
		&g.SundayScheduleID, &g.SundayBlockID,
		&g.MondayScheduleID, &g.MondayBlockID,
		&g.TuesdayScheduleID, &g.TuesdayBlockID,
		&g.WednesdayScheduleID, &g.WednesdayBlockID,
		&g.ThursdayScheduleID, &g.ThursdayBlockID,
		&g.FridayScheduleID, &g.FridayBlockID,
		&g.SaturdayScheduleID, &g.SaturdayBlockID,
		&g.HolidayScheduleID, &g.HolidayBlockID, &g.AllowedManualStylesJSON,
		&g.CreatedAt,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	schedules := make(map[string]*models.BasicSchedule)
	configs := make(map[string]*models.CommercialBlockConfig)

	loadSchedule := func(sID *uuid.UUID, key string) error {
		if sID == nil || *sID == uuid.Nil {
			return nil
		}
		var s models.BasicSchedule
		query := `SELECT id, tenant_id, media_type, name, COALESCE(description, '') AS description, start_time, end_time,
		                 musical_block_duration_minutes, custom_durations_json, custom_styles_json, COALESCE(off_hours_behavior, 'SILENCE'), created_at
		          FROM basic_schedules WHERE id = $1`
		err := database.Pool.QueryRow(ctx, query, *sID).Scan(
			&s.ID, &s.TenantID, &s.MediaType, &s.Name, &s.Description, &s.StartTime, &s.EndTime,
			&s.MusicalBlockDurationMinutes, &s.CustomDurationsJSON, &s.CustomStylesJSON, &s.OffHoursBehavior, &s.CreatedAt,
		)
		if err != nil {
			return err
		}
		schedules[key] = &s
		return nil
	}

	loadConfig := func(cID *uuid.UUID, key string) error {
		if cID == nil || *cID == uuid.Nil {
			return nil
		}
		var c models.CommercialBlockConfig
		query := `SELECT id, tenant_id, media_type, name, COALESCE(description, '') AS description, interval_minutes, block_limit_seconds,
		                 percentage_time_announcement, percentage_temperature, percentage_institutional, percentage_offer, created_at
		          FROM commercial_block_configs WHERE id = $1`
		err := database.Pool.QueryRow(ctx, query, *cID).Scan(
			&c.ID, &c.TenantID, &c.MediaType, &c.Name, &c.Description, &c.IntervalMinutes, &c.BlockLimitSeconds,
			&c.PercentageTimeAnnouncement, &c.PercentageTemperature, &c.PercentageInstitutional, &c.PercentageOffer, &c.CreatedAt,
		)
		if err != nil {
			return err
		}
		configs[key] = &c
		return nil
	}

	// Carrega individualmente todos os 8 dias
	days := map[string]*uuid.UUID{
		"mon": g.MondayScheduleID,
		"tue": g.TuesdayScheduleID,
		"wed": g.WednesdayScheduleID,
		"thu": g.ThursdayScheduleID,
		"fri": g.FridayScheduleID,
		"sat": g.SaturdayScheduleID,
		"sun": g.SundayScheduleID,
		"holiday": g.HolidayScheduleID,
	}
	for k, sID := range days {
		if err := loadSchedule(sID, k); err != nil {
			return nil, nil, nil, err
		}
	}

	dayBlocks := map[string]*uuid.UUID{
		"mon": g.MondayBlockID,
		"tue": g.TuesdayBlockID,
		"wed": g.WednesdayBlockID,
		"thu": g.ThursdayBlockID,
		"fri": g.FridayBlockID,
		"sat": g.SaturdayBlockID,
		"sun": g.SundayBlockID,
		"holiday": g.HolidayBlockID,
	}
	for k, bID := range dayBlocks {
		if err := loadConfig(bID, k); err != nil {
			return nil, nil, nil, err
		}
	}

	return &g, schedules, configs, nil
}

// Update atualiza os dados da grade e de suas dependências dentro de uma transação
func (r *GridRepository) Update(ctx context.Context, grid *models.WeeklyMasterGrid, schedules map[string]*models.BasicSchedule, configs map[string]*models.CommercialBlockConfig) error {
	oldGrid, _, _, err := r.GetByID(ctx, grid.ID)
	if err != nil {
		return err
	}

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	processSchedule := func(s *models.BasicSchedule, oldID *uuid.UUID) (*uuid.UUID, error) {
		if s == nil {
			if oldID != nil && *oldID != uuid.Nil {
				_, err := tx.Exec(ctx, "DELETE FROM basic_schedules WHERE id = $1", *oldID)
				return nil, err
			}
			return nil, nil
		}

		if s.OffHoursBehavior == "" {
			s.OffHoursBehavior = "SILENCE"
		}

		if oldID != nil && *oldID != uuid.Nil {
			s.ID = *oldID
			query := `
				UPDATE basic_schedules SET
					tenant_id = $2, media_type = $3, name = $4, description = $5,
					start_time = $6, end_time = $7, musical_block_duration_minutes = $8,
					custom_durations_json = $9, custom_styles_json = $10, off_hours_behavior = $11
				WHERE id = $1
			`
			_, err := tx.Exec(ctx, query,
				s.ID, s.TenantID, s.MediaType, s.Name, s.Description, s.StartTime, s.EndTime, s.MusicalBlockDurationMinutes,
				s.CustomDurationsJSON, s.CustomStylesJSON, s.OffHoursBehavior,
			)
			return oldID, err
		} else {
			s.ID = uuid.New()
			query := `
				INSERT INTO basic_schedules (
					id, tenant_id, media_type, name, description, start_time, end_time, musical_block_duration_minutes,
					custom_durations_json, custom_styles_json, off_hours_behavior, created_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
				)
			`
			_, err := tx.Exec(ctx, query,
				s.ID, s.TenantID, s.MediaType, s.Name, s.Description, s.StartTime, s.EndTime, s.MusicalBlockDurationMinutes,
				s.CustomDurationsJSON, s.CustomStylesJSON, s.OffHoursBehavior,
			)
			return &s.ID, err
		}
	}

	processConfig := func(c *models.CommercialBlockConfig, oldID *uuid.UUID) (*uuid.UUID, error) {
		if c == nil {
			if oldID != nil && *oldID != uuid.Nil {
				_, err := tx.Exec(ctx, "DELETE FROM commercial_block_configs WHERE id = $1", *oldID)
				return nil, err
			}
			return nil, nil
		}

		if oldID != nil && *oldID != uuid.Nil {
			c.ID = *oldID
			query := `
				UPDATE commercial_block_configs SET
					tenant_id = $2, media_type = $3, name = $4, description = $5,
					interval_minutes = $6, block_limit_seconds = $7,
					percentage_time_announcement = $8, percentage_temperature = $9,
					percentage_institutional = $10, percentage_offer = $11
				WHERE id = $1
			`
			_, err := tx.Exec(ctx, query,
				c.ID, c.TenantID, c.MediaType, c.Name, c.Description, c.IntervalMinutes, c.BlockLimitSeconds,
				c.PercentageTimeAnnouncement, c.PercentageTemperature, c.PercentageInstitutional, c.PercentageOffer,
			)
			return oldID, err
		} else {
			c.ID = uuid.New()
			query := `
				INSERT INTO commercial_block_configs (
					id, tenant_id, media_type, name, description, interval_minutes, block_limit_seconds,
					percentage_time_announcement, percentage_temperature, percentage_institutional, percentage_offer, created_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW()
				)
			`
			_, err := tx.Exec(ctx, query,
				c.ID, c.TenantID, c.MediaType, c.Name, c.Description, c.IntervalMinutes, c.BlockLimitSeconds,
				c.PercentageTimeAnnouncement, c.PercentageTemperature, c.PercentageInstitutional, c.PercentageOffer,
			)
			return &c.ID, err
		}
	}

	// Processa os 8 dias um por um
	var errProc error
	grid.MondayScheduleID, errProc = processSchedule(schedules["mon"], oldGrid.MondayScheduleID); if errProc != nil { return errProc }
	grid.MondayBlockID, errProc = processConfig(configs["mon"], oldGrid.MondayBlockID); if errProc != nil { return errProc }

	grid.TuesdayScheduleID, errProc = processSchedule(schedules["tue"], oldGrid.TuesdayScheduleID); if errProc != nil { return errProc }
	grid.TuesdayBlockID, errProc = processConfig(configs["tue"], oldGrid.TuesdayBlockID); if errProc != nil { return errProc }

	grid.WednesdayScheduleID, errProc = processSchedule(schedules["wed"], oldGrid.WednesdayScheduleID); if errProc != nil { return errProc }
	grid.WednesdayBlockID, errProc = processConfig(configs["wed"], oldGrid.WednesdayBlockID); if errProc != nil { return errProc }

	grid.ThursdayScheduleID, errProc = processSchedule(schedules["thu"], oldGrid.ThursdayScheduleID); if errProc != nil { return errProc }
	grid.ThursdayBlockID, errProc = processConfig(configs["thu"], oldGrid.ThursdayBlockID); if errProc != nil { return errProc }

	grid.FridayScheduleID, errProc = processSchedule(schedules["fri"], oldGrid.FridayScheduleID); if errProc != nil { return errProc }
	grid.FridayBlockID, errProc = processConfig(configs["fri"], oldGrid.FridayBlockID); if errProc != nil { return errProc }

	grid.SaturdayScheduleID, errProc = processSchedule(schedules["sat"], oldGrid.SaturdayScheduleID); if errProc != nil { return errProc }
	grid.SaturdayBlockID, errProc = processConfig(configs["sat"], oldGrid.SaturdayBlockID); if errProc != nil { return errProc }

	grid.SundayScheduleID, errProc = processSchedule(schedules["sun"], oldGrid.SundayScheduleID); if errProc != nil { return errProc }
	grid.SundayBlockID, errProc = processConfig(configs["sun"], oldGrid.SundayBlockID); if errProc != nil { return errProc }

	grid.HolidayScheduleID, errProc = processSchedule(schedules["holiday"], oldGrid.HolidayScheduleID); if errProc != nil { return errProc }
	grid.HolidayBlockID, errProc = processConfig(configs["holiday"], oldGrid.HolidayBlockID); if errProc != nil { return errProc }

	// Atualizar a Weekly Master Grid
	queryGrid := `
		UPDATE weekly_master_grids SET
			tenant_id = $2, media_type = $3, name = $4, description = $5,
			sunday_schedule_id = $6, sunday_block_id = $7,
			monday_schedule_id = $8, monday_block_id = $9,
			tuesday_schedule_id = $10, tuesday_block_id = $11,
			wednesday_schedule_id = $12, wednesday_block_id = $13,
			thursday_schedule_id = $14, thursday_block_id = $15,
			friday_schedule_id = $16, friday_block_id = $17,
			saturday_schedule_id = $18, saturday_block_id = $19,
			holiday_schedule_id = $20, holiday_block_id = $21,
			allowed_manual_styles_json = $22
		WHERE id = $1
	`
	_, err = tx.Exec(ctx, queryGrid,
		grid.ID, grid.TenantID, grid.MediaType, grid.Name, grid.Description,
		grid.SundayScheduleID, grid.SundayBlockID,
		grid.MondayScheduleID, grid.MondayBlockID,
		grid.TuesdayScheduleID, grid.TuesdayBlockID,
		grid.WednesdayScheduleID, grid.WednesdayBlockID,
		grid.ThursdayScheduleID, grid.ThursdayBlockID,
		grid.FridayScheduleID, grid.FridayBlockID,
		grid.SaturdayScheduleID, grid.SaturdayBlockID,
		grid.HolidayScheduleID, grid.HolidayBlockID, grid.AllowedManualStylesJSON,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Delete remove uma grade semanal mestre e limpa suas dependências em transação
func (r *GridRepository) Delete(ctx context.Context, id uuid.UUID) error {
	g, _, _, err := r.GetByID(ctx, id)
	if err != nil {
		return nil
	}

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM weekly_master_grids WHERE id = $1", id)
	if err != nil {
		return err
	}

	// Deletar schedules únicos
	scheduleIDs := make(map[uuid.UUID]bool)
	days := []*uuid.UUID{
		g.MondayScheduleID, g.TuesdayScheduleID, g.WednesdayScheduleID, g.ThursdayScheduleID, g.FridayScheduleID,
		g.SaturdayScheduleID, g.SundayScheduleID, g.HolidayScheduleID,
	}
	for _, sID := range days {
		if sID != nil {
			scheduleIDs[*sID] = true
		}
	}
	for sID := range scheduleIDs {
		if sID != uuid.Nil {
			_, err = tx.Exec(ctx, "DELETE FROM basic_schedules WHERE id = $1", sID)
			if err != nil {
				return err
			}
		}
	}

	// Deletar blocks únicos
	blockIDs := make(map[uuid.UUID]bool)
	blocks := []*uuid.UUID{
		g.MondayBlockID, g.TuesdayBlockID, g.WednesdayBlockID, g.ThursdayBlockID, g.FridayBlockID,
		g.SaturdayBlockID, g.SundayBlockID, g.HolidayBlockID,
	}
	for _, bID := range blocks {
		if bID != nil {
			blockIDs[*bID] = true
		}
	}
	for bID := range blockIDs {
		if bID != uuid.Nil {
			_, err = tx.Exec(ctx, "DELETE FROM commercial_block_configs WHERE id = $1", bID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

// DeleteByInterface é um helper para apagar usando um tipo interface genérico
func (r *GridRepository) DeleteByInterface(ctx context.Context, id interface{}) error {
	idStr, ok := id.(string)
	if ok {
		parsed, err := uuid.Parse(idStr)
		if err != nil {
			return err
		}
		return r.Delete(ctx, parsed)
	}

	idUUID, ok := id.(uuid.UUID)
	if ok {
		return r.Delete(ctx, idUUID)
	}

	return errors.New("formato de ID inválido")
}
