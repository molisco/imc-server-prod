package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"imc-server/internal/database"
)

func main() {
	os.Setenv("DATABASE_URL", "postgres://neoroute:@@Mo1isc00@@@192.168.15.250:5432/imc_server?sslmode=disable")
	database.ConnectDatabase()
	defer database.CloseDatabase()

	ctx := context.Background()
	rows, err := database.Pool.Query(ctx, "SELECT id, name, start_time, end_time, custom_styles_json FROM basic_schedules LIMIT 10")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("=== BASIC SCHEDULES ===")
	for rows.Next() {
		var id, name, startTime, endTime string
		var customStyles *string
		err := rows.Scan(&id, &name, &startTime, &endTime, &customStyles)
		if err != nil {
			log.Fatalf("Scan failed: %v", err)
		}
		stylesStr := "NULL"
		if customStyles != nil {
			stylesStr = *customStyles
		}
		fmt.Printf("ID: %s | Name: %s | Start: %s | End: %s | Styles: %s\n", id, name, startTime, endTime, stylesStr)
	}
}
