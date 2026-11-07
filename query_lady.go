package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	databaseURL := "postgres://neoroute:@@Mo1isc00@@@192.168.15.250:5432/imc_server?sslmode=disable"
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, "SELECT id, file_name, display_name, cue_in, cue_out, is_sonic, manual_cue, updated_at::text FROM music_library WHERE file_name LIKE '%Lady%' OR display_name LIKE '%Lady%'")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		var id, fileName, displayName, updatedAt string
		var cueIn, cueOut float64
		var isSonic, manualCue bool
		err = rows.Scan(&id, &fileName, &displayName, &cueIn, &cueOut, &isSonic, &manualCue, &updatedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Scan failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("ID: %s, File: %s, Display: %s, CueIn: %f, CueOut: %f, IsSonic: %t, ManualCue: %t, UpdatedAt: %s\n", id, fileName, displayName, cueIn, cueOut, isSonic, manualCue, updatedAt)
	}
}
