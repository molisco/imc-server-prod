package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://neoroute:@@Mo1isc00@@@192.168.15.250:5432/imc_server"
	ctx := context.Background()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	// Get store audio_grid_id
	var gridID string
	err = conn.QueryRow(ctx, "SELECT audio_grid_id FROM stores WHERE token = 'token_1_1'").Scan(&gridID)
	if err != nil {
		log.Fatalf("Store query failed: %v\n", err)
	}
	fmt.Printf("Store audio_grid_id: %s\n", gridID)

	// Query grid schedule IDs
	var sun, mon, tue, wed, thu, fri, sat, hol *string
	err = conn.QueryRow(ctx, `
		SELECT sunday_schedule_id, monday_schedule_id, tuesday_schedule_id, wednesday_schedule_id,
		       thursday_schedule_id, friday_schedule_id, saturday_schedule_id, holiday_schedule_id
		FROM weekly_master_grids WHERE id = $1`, gridID).Scan(&sun, &mon, &tue, &wed, &thu, &fri, &sat, &hol)
	if err != nil {
		log.Fatalf("Grid query failed: %v\n", err)
	}

	schedIDs := []string{}
	add := func(id *string) {
		if id != nil {
			schedIDs = append(schedIDs, *id)
		}
	}
	add(sun); add(mon); add(tue); add(wed); add(thu); add(fri); add(sat); add(hol)

	fmt.Printf("Programmed schedule IDs count: %d\n", len(schedIDs))

	for _, sid := range schedIDs {
		var name string
		var customStyles *string
		err = conn.QueryRow(ctx, "SELECT name, custom_styles_json FROM basic_schedules WHERE id = $1", sid).Scan(&name, &customStyles)
		if err == nil {
			stylesStr := "NULL"
			if customStyles != nil {
				stylesStr = *customStyles
			}
			fmt.Printf("  Schedule Name: %s | Styles: %s\n", name, stylesStr[:100])
		}
	}
}
