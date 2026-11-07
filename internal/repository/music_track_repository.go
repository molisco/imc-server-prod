package repository

import (
	"context"
	"imc-server/internal/database"
	"imc-server/internal/models"

	"github.com/google/uuid"
)

type MusicTrackRepository struct{}

func NewMusicTrackRepository() *MusicTrackRepository {
	return &MusicTrackRepository{}
}

func (r *MusicTrackRepository) Create(ctx context.Context, track models.MusicTrack) error {
	query := `
		INSERT INTO music_library (
			id, music_style_id, file_name, display_name, duration_seconds, 
			album, title, artist, year, lufs, lra, loudness_score, cue_in, cue_out, 
			file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), NOW()
		)
	`
	_, err := database.Pool.Exec(ctx, query,
		track.ID, track.MusicStyleID, track.FileName, track.DisplayName, track.DurationSeconds,
		track.Album, track.Title, track.Artist, track.Year, track.LUFS, track.LRA, track.LoudnessScore,
		track.CueIn, track.CueOut, track.FileSizeBytes, track.BitRateKbps, track.SampleRateHz, track.RiscoRuido,
		track.ManualCue, track.IsSonic,
	)
	return err
}

func (r *MusicTrackRepository) Update(ctx context.Context, track models.MusicTrack) error {
	query := `
		UPDATE music_library
		SET display_name = $1, duration_seconds = $2, album = $3, title = $4, artist = $5,
		    year = $6, lufs = $7, lra = $8, loudness_score = $9, cue_in = $10, cue_out = $11,
		    file_size_bytes = $12, bitrate_kbps = $13, sample_rate_hz = $14, risco_ruido = $15,
		    manual_cue = $16, is_sonic = $17, updated_at = NOW()
		WHERE id = $18
	`
	_, err := database.Pool.Exec(ctx, query,
		track.DisplayName, track.DurationSeconds, track.Album, track.Title, track.Artist,
		track.Year, track.LUFS, track.LRA, track.LoudnessScore, track.CueIn, track.CueOut,
		track.FileSizeBytes, track.BitRateKbps, track.SampleRateHz, track.RiscoRuido, track.ManualCue,
		track.IsSonic, track.ID,
	)
	return err
}

func (r *MusicTrackRepository) GetByStyleID(ctx context.Context, styleID uuid.UUID) ([]models.MusicTrack, error) {
	query := `
		SELECT id, music_style_id, file_name, display_name, duration_seconds, 
		       album, title, artist, year, lufs, lra, loudness_score, cue_in, cue_out,
		       file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, created_at, updated_at
		FROM music_library
		WHERE music_style_id = $1
		ORDER BY display_name ASC
	`
	rows, err := database.Pool.Query(ctx, query, styleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []models.MusicTrack
	for rows.Next() {
		var t models.MusicTrack
		err := rows.Scan(
			&t.ID, &t.MusicStyleID, &t.FileName, &t.DisplayName, &t.DurationSeconds,
			&t.Album, &t.Title, &t.Artist, &t.Year, &t.LUFS, &t.LRA, &t.LoudnessScore,
			&t.CueIn, &t.CueOut, &t.FileSizeBytes, &t.BitRateKbps, &t.SampleRateHz, &t.RiscoRuido, &t.ManualCue, &t.IsSonic, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, nil
}

func (r *MusicTrackRepository) GetByStyleAndFileName(ctx context.Context, styleID uuid.UUID, fileName string) (*models.MusicTrack, error) {
	query := `
		SELECT id, music_style_id, file_name, display_name, duration_seconds, 
		       album, title, artist, year, lufs, lra, loudness_score, cue_in, cue_out,
		       file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, created_at, updated_at
		FROM music_library
		WHERE music_style_id = $1 AND file_name = $2
	`
	var t models.MusicTrack
	err := database.Pool.QueryRow(ctx, query, styleID, fileName).Scan(
		&t.ID, &t.MusicStyleID, &t.FileName, &t.DisplayName, &t.DurationSeconds,
		&t.Album, &t.Title, &t.Artist, &t.Year, &t.LUFS, &t.LRA, &t.LoudnessScore,
		&t.CueIn, &t.CueOut, &t.FileSizeBytes, &t.BitRateKbps, &t.SampleRateHz, &t.RiscoRuido, &t.ManualCue, &t.IsSonic, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *MusicTrackRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MusicTrack, error) {
	query := `
		SELECT id, music_style_id, file_name, display_name, duration_seconds, 
		       album, title, artist, year, lufs, lra, loudness_score, cue_in, cue_out,
		       file_size_bytes, bitrate_kbps, sample_rate_hz, risco_ruido, manual_cue, is_sonic, created_at, updated_at
		FROM music_library
		WHERE id = $1
	`
	var t models.MusicTrack
	err := database.Pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.MusicStyleID, &t.FileName, &t.DisplayName, &t.DurationSeconds,
		&t.Album, &t.Title, &t.Artist, &t.Year, &t.LUFS, &t.LRA, &t.LoudnessScore,
		&t.CueIn, &t.CueOut, &t.FileSizeBytes, &t.BitRateKbps, &t.SampleRateHz, &t.RiscoRuido, &t.ManualCue, &t.IsSonic, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *MusicTrackRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM music_library WHERE id = $1", id)
	return err
}

func (r *MusicTrackRepository) DeleteByStyleID(ctx context.Context, styleID uuid.UUID) error {
	_, err := database.Pool.Exec(ctx, "DELETE FROM music_library WHERE music_style_id = $1", styleID)
	return err
}

func (r *MusicTrackRepository) GetCountsMap(ctx context.Context) (map[uuid.UUID]int, error) {
	query := `
		SELECT music_style_id, COUNT(*)
		FROM music_library
		GROUP BY music_style_id
	`
	rows, err := database.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int)
	for rows.Next() {
		var styleID uuid.UUID
		var count int
		if err := rows.Scan(&styleID, &count); err == nil {
			counts[styleID] = count
		}
	}
	return counts, nil
}
