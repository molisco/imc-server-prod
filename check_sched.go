package main

import (
	"context"
	"fmt"
	"imc-server/internal/database"
)

func main() {
	database.InitPool("postgres://neoroute:@@Mo1isc00@@@192.168.15.250:5432/imc_server?sslmode=disable")
	defer database.ClosePool()
	ctx := context.Background()
	rows, err := database.Pool.Query(ctx, "SELECT s.name, s.start_time, s.end_time, COALESCE(st.name, ''), COALESCE(ms.name, '') FROM schedules s LEFT JOIN streamings st ON s.streaming_id = st.id LEFT JOIN media_styles ms ON s.media_style_id = ms.id WHERE s.store_id = 'ce275c7f-9b5a-4892-a802-efe432016721'")
	if err != nil {
		fmt.Printf("Query err: %v\n", err)
		return
	}
	defer rows.Close()
	fmt.Println("--- SCHEDULES FOR SOROCABA ---")
	for rows.Next() {
		var sName, sStart, sEnd, streamName, styleName string
		_ = rows.Scan(&sName, &sStart, &sEnd, &streamName, &styleName)
		fmt.Printf("Schedule: %s [%s - %s] -> Stream: %s | Style: %s\n", sName, sStart, sEnd, streamName, styleName)
	}
}
