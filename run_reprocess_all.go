package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"imc-server/internal/utils"

	"github.com/jackc/pgx/v5/pgxpool"
)

// folder name mapping helper
func GetFolderName(cType string) string {
	switch cType {
	case "INSTITUTIONAL":
		return "institutional"
	case "BLOCK_OPENING":
		return "open_blocks"
	case "BLOCK_CLOSING":
		return "close_blocks"
	case "COMMERCIAL":
		return "advertisements"
	case "OFFER":
		return "offers"
	case "EVENT":
		return "events"
	case "GENERAL_INFORMATION":
		return "general_information"
	case "RIGHT_TIME":
		return "right_time"
	case "TEMPERATURE":
		return "temperature"
	default:
		return "advertisements"
	}
}

func main() {
	dbURL := "postgres://neoroute:@@Mo1isc00@@@localhost:5432/imc_server?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	rows, err := pool.Query(context.Background(), `
		SELECT id, tenant_id, commercial_type, name, file_name, manual_cue
		FROM commercials_and_events
	`)
	if err != nil {
		log.Fatalf("query failed: %v", err)
	}
	defer rows.Close()

	type Comm struct {
		ID        string
		TenantID  string
		Type      string
		Name      string
		FileName  string
		ManualCue bool
	}

	var list []Comm
	for rows.Next() {
		var c Comm
		err = rows.Scan(&c.ID, &c.TenantID, &c.Type, &c.Name, &c.FileName, &c.ManualCue)
		if err == nil {
			list = append(list, c)
		}
	}

	fmt.Printf("Total commercials to process: %d\n", len(list))

	for i, c := range list {
		folderName := GetFolderName(c.Type)
		filePath := filepath.Join("midias", "comerciais", c.TenantID, folderName, c.FileName)

		// check if file exists
		if _, errStat := os.Stat(filePath); os.IsNotExist(errStat) {
			fmt.Printf("[%d/%d] Skipping %s: file not found at %s\n", i+1, len(list), c.Name, filePath)
			continue
		}

		fmt.Printf("[%d/%d] Processing %s (%s)... ", i+1, len(list), c.Name, c.ID)
		meta, errMeta := utils.ExtractMetadata(filePath)
		if errMeta != nil {
			fmt.Printf("FAIL: %v\n", errMeta)
			continue
		}

		// update DB
		_, errUpdate := pool.Exec(context.Background(), `
			UPDATE commercials_and_events
			SET lufs = $1, lra = $2, loudness_score = $3, file_size_bytes = $4,
			    bitrate_kbps = $5, sample_rate_hz = $6, risco_ruido = $7, is_sonic = $8
			WHERE id = $9
		`, meta.LUFS, meta.LRA, meta.LoudnessScore, meta.FileSizeBytes,
			meta.BitRateKbps, meta.SampleRateHz, meta.NoiseRisk, meta.IsSonic, c.ID)

		if errUpdate != nil {
			fmt.Printf("DB UPDATE FAIL: %v\n", errUpdate)
		} else {
			fmt.Printf("OK (LUFS: %0.1f, Sonic: %v)\n", meta.LUFS, meta.IsSonic)
		}
	}
}
