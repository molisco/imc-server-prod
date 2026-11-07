package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB represents our database connection pool instance
var Pool *pgxpool.Pool

// ConnectDatabase initializes the connection pool with PostgreSQL
func ConnectDatabase() {
	// 1. Get the Database URL from Environment Variables
	// In production, never hardcode credentials. We use os.Getenv for security.
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		// Fallback for local/production server if the environment variable is not set
		databaseURL = "postgres://neoroute:@@Mo1isc00@@@127.0.0.1:5432/imc_server?sslmode=disable"
	}

	// 2. Parse configuration string into a pgx config struct
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("Unable to parse database URL configuration: %v\n", err)
	}

	// 3. Configure Pool Settings (Crucial for high performance)
	config.MaxConns = 20                      // Maximum number of open connections
	config.MinConns = 5                       // Minimum number of idle connections always open
	config.MaxConnIdleTime = 30 * time.Minute // How long an idle connection stays alive

	// 4. Create the context with a timeout for the initial connection attempt
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 5. Establish the connection pool
	Pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	// 6. Ping the database to verify the connection is live and working
	err = Pool.Ping(ctx)
	if err != nil {
		log.Fatalf("Database ping failed. Connection is dead: %v\n", err)
	}

	fmt.Println("🚀 Magic inside the Linux: Successfully connected to PostgreSQL Pool!")

	// 7. Automatic Schema Patching (White Label Colors & Missing Schema Columns)
	// We run these in separate statements to ensure that errors on non-exist/pre-exist columns do not abort the process.
	queries := []string{
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS primary_color VARCHAR(7) DEFAULT '#4364f7'",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS background_color VARCHAR(7) DEFAULT '#0f111a'",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS font_color VARCHAR(7) DEFAULT '#ffffff'",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS field_background_color VARCHAR(7) DEFAULT '#161925'",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS district VARCHAR(100)",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS country VARCHAR(100) DEFAULT 'Brasil'",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS logo_url TEXT",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS operational_contacts JSONB DEFAULT '[]'::jsonb",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS financial_contacts JSONB DEFAULT '[]'::jsonb",
		"ALTER TABLE tenants ALTER COLUMN number DROP NOT NULL",
		"ALTER TABLE stores ADD COLUMN IF NOT EXISTS district VARCHAR(100)",
		"ALTER TABLE stores ALTER COLUMN regional_id DROP NOT NULL",
		"ALTER TABLE stores ADD COLUMN IF NOT EXISTS timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo'",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS default_audio_grid_id UUID",
		"ALTER TABLE tenants ADD COLUMN IF NOT EXISTS default_video_grid_id UUID",
		"ALTER TABLE basic_schedules ADD COLUMN IF NOT EXISTS custom_durations_json JSONB",
		"ALTER TABLE basic_schedules ADD COLUMN IF NOT EXISTS custom_styles_json JSONB",
		"ALTER TABLE basic_schedules ADD COLUMN IF NOT EXISTS off_hours_behavior VARCHAR(50) DEFAULT 'SILENCE'",
		"CREATE TABLE IF NOT EXISTS music_styles (id UUID PRIMARY KEY, name VARCHAR(100) NOT NULL, description VARCHAR(200), base_directory VARCHAR(150), created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE IF NOT EXISTS video_styles (id UUID PRIMARY KEY, name VARCHAR(100) NOT NULL, description VARCHAR(200), base_directory VARCHAR(150), created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP)",
		"CREATE TABLE IF NOT EXISTS music_library (id UUID PRIMARY KEY, music_style_id UUID NOT NULL REFERENCES music_styles(id) ON DELETE CASCADE, file_name VARCHAR(255) NOT NULL, display_name VARCHAR(255) NOT NULL, duration_seconds INT NOT NULL, album VARCHAR(255), title VARCHAR(255), artist VARCHAR(255), year INT, lufs NUMERIC(5,2), lra NUMERIC(5,2), loudness_score VARCHAR(5), cue_in NUMERIC(6,2) DEFAULT 0.0, cue_out NUMERIC(6,2) DEFAULT 0.0, file_size_bytes BIGINT DEFAULT 0, bitrate_kbps INT DEFAULT 0, sample_rate_hz INT DEFAULT 0, created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP)",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS album VARCHAR(255)",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS title VARCHAR(255)",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS artist VARCHAR(255)",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS year INT",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS lufs NUMERIC(5,2)",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS lra NUMERIC(5,2)",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS loudness_score VARCHAR(5)",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS cue_in NUMERIC(6,2) DEFAULT 0.0",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS cue_out NUMERIC(6,2) DEFAULT 0.0",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS file_size_bytes BIGINT DEFAULT 0",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS bitrate_kbps INT DEFAULT 0",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS sample_rate_hz INT DEFAULT 0",
		"ALTER TABLE music_styles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE music_styles ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) DEFAULT 'LOCAL'",
		"ALTER TABLE music_styles ADD COLUMN IF NOT EXISTS stream_url TEXT",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS risco_ruido INT DEFAULT 0",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS manual_cue BOOLEAN DEFAULT FALSE",
		"ALTER TABLE music_library ADD COLUMN IF NOT EXISTS is_sonic BOOLEAN DEFAULT FALSE",
		"ALTER TYPE commercial_type_enum ADD VALUE IF NOT EXISTS 'GENERAL_INFORMATION'",
		"ALTER TYPE commercial_type_enum ADD VALUE IF NOT EXISTS 'RIGHT_TIME'",
		"ALTER TYPE commercial_type_enum ADD VALUE IF NOT EXISTS 'TEMPERATURE'",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS distribute_to_all BOOLEAN DEFAULT TRUE",
		"CREATE TABLE IF NOT EXISTS commercial_distributions (commercial_id UUID NOT NULL REFERENCES commercials_and_events(id) ON DELETE CASCADE, store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE, PRIMARY KEY (commercial_id, store_id))",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS lufs DOUBLE PRECISION DEFAULT -14.0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS lra DOUBLE PRECISION DEFAULT 5.0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS loudness_score VARCHAR(10) DEFAULT 'A'",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS cue_in DOUBLE PRECISION DEFAULT 0.0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS cue_out DOUBLE PRECISION DEFAULT 0.0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS file_size_bytes BIGINT DEFAULT 0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS bitrate_kbps INT DEFAULT 0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS sample_rate_hz INT DEFAULT 0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS risco_ruido INT DEFAULT 0",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS manual_cue BOOLEAN DEFAULT FALSE",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS is_sonic BOOLEAN DEFAULT FALSE",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS execution_time TIME DEFAULT NULL",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS event_interruption_mode VARCHAR(20) DEFAULT 'SOFT'",
		"ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE stores ADD COLUMN IF NOT EXISTS radio_type VARCHAR(20) NOT NULL DEFAULT 'server'",
		"ALTER TABLE stores ADD COLUMN IF NOT EXISTS harbor_port INT DEFAULT NULL",
		"ALTER TABLE weekly_master_grids ADD COLUMN IF NOT EXISTS allowed_manual_styles_json TEXT",
		"ALTER TABLE music_library ADD CONSTRAINT unique_music_style_file UNIQUE (music_style_id, file_name)",
		"CREATE TABLE IF NOT EXISTS prelaunch_leads (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(150) NOT NULL, company VARCHAR(150) NOT NULL, contact VARCHAR(180) NOT NULL, plan_interest VARCHAR(120) NOT NULL, source_host VARCHAR(120), ip_address VARCHAR(64), user_agent TEXT, created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP)",
	}

	hasErrors := false
	for _, q := range queries {
		if _, err := Pool.Exec(ctx, q); err != nil {
			// Some errors (like column 'number' not existing) are harmless if the user's active table has a different structure.
			// So we log them as warnings and proceed.
			fmt.Printf("ℹ️ Info: Schema patch query ignored/failed: %q -> %v\n", q, err)
			hasErrors = true
		}
	}

	if !hasErrors {
		fmt.Println("✅ Database schema patches applied successfully (district, country, colors, music_styles, video_styles).")
	} else {
		fmt.Println("✅ Database schema patching process finished.")
	}

	// Insert the Global Tenant
	insertGlobalTenantQuery := `
		INSERT INTO tenants (
			id, trade_name, legal_name, document_number, zip_code, street, city, state, is_active
		) VALUES (
			'00000000-0000-0000-0000-000000000000', 'Global / Geral', 'Sistema Global Default', '00.000.000/0000-00', '00000-000', 'Rua Principal', 'Brasília', 'DF', true
		) ON CONFLICT (id) DO NOTHING
	`
	if _, err := Pool.Exec(ctx, insertGlobalTenantQuery); err != nil {
		fmt.Printf("ℹ️ Info: Could not insert Global Tenant: %v\n", err)
	} else {
		fmt.Println("✅ Default Global Tenant verified/inserted.")
	}

	// Clean up Default Music Styles (Estilo 01-10) as requested by the user
	deleteStylesQuery := `
		DELETE FROM music_styles 
		WHERE name IN ('Estilo 01', 'Estilo 02', 'Estilo 03', 'Estilo 04', 'Estilo 05', 'Estilo 06', 'Estilo 07', 'Estilo 08', 'Estilo 09', 'Estilo 10')
	`
	if _, err := Pool.Exec(ctx, deleteStylesQuery); err != nil {
		fmt.Printf("ℹ️ Info: Could not delete old default styles: %v\n", err)
	} else {
		fmt.Println("✅ Default Music Styles (Estilo 01-10) cleaned up/removed.")
	}

	// Insert official Hotmix streaming music styles if not present
	hotmixStations := []struct {
		Name        string
		Description string
		StreamURL   string
	}{
		{"Streaming: Best-of", "Hotmix Best-of (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-best-of-pre"},
		{"Streaming: Just Lounge", "Hotmix Just Lounge (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-just-lounge-pre"},
		{"Streaming: Shopping Vibes", "Hotmix Shopping Vibes (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-shopping-vibes-pre"},
		{"Streaming: Cozy & Coffee", "Hotmix Cozy & Coffee Shop (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-cozy-coffee-shop-pre"},
		{"Streaming: Gentle Luxury", "Hotmix Gentle Luxury (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-gentle-luxury-pre"},
		{"Streaming: Cocktail & Dinner", "Hotmix Cocktail & Dinner Groove (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-cocktail-and-dinner-pre"},
		{"Streaming: Cool & Calm", "Hotmix Cool & Calm (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-cool-and-calm-pre"},
		{"Streaming: On the Rocks", "Hotmix On the Rocks (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-on-the-rocks-pre"},
		{"Streaming: Spa Garden", "Hotmix Spa Garden (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-spa-garden-pre"},
		{"Streaming: Workout", "Hotmix Workout (Streaming Ao Vivo)", "https://streaming.hotmixradio.com/hotmixpro-workout-pre"},
	}

	for _, hs := range hotmixStations {
		insertHotmixQuery := `
			INSERT INTO music_styles (id, name, description, base_directory, source_type, stream_url)
			SELECT gen_random_uuid(), $1::text, $2::text, '', 'STREAM', $3::text
			WHERE NOT EXISTS (SELECT 1 FROM music_styles WHERE name = $1::text)
		`
		if _, err := Pool.Exec(ctx, insertHotmixQuery, hs.Name, hs.Description, hs.StreamURL); err != nil {
			fmt.Printf("⚠️ Warning: Failed to insert Hotmix station %s: %v\n", hs.Name, err)
		}
	}

	// Padrão Ouro: Auto-heal basic schedules JSON to resolve old style names to persistent UUIDs
	AutoHealSchedulesCustomStylesJSON(ctx)
}

// AutoHealSchedulesCustomStylesJSON scans basic schedules and updates custom_styles_json to use style UUIDs instead of names
func AutoHealSchedulesCustomStylesJSON(ctx context.Context) {
	rows, err := Pool.Query(ctx, "SELECT id, name FROM music_styles")
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to query music styles for auto-healing: %v\n", err)
		return
	}
	defer rows.Close()

	styleMap := make(map[string]uuid.UUID)
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err == nil {
			styleMap[name] = id
			cleanName := cleanStyleName(name)
			styleMap[cleanName] = id
		}
	}
	rows.Close()

	rowsSched, err := Pool.Query(ctx, "SELECT id, custom_styles_json::text FROM basic_schedules WHERE custom_styles_json IS NOT NULL")
	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to query basic schedules for auto-healing: %v\n", err)
		return
	}
	defer rowsSched.Close()

	type SchedUpdate struct {
		ID   uuid.UUID
		JSON string
	}
	var updates []SchedUpdate

	for rowsSched.Next() {
		var id uuid.UUID
		var jsonStr string
		if err := rowsSched.Scan(&id, &jsonStr); err == nil {
			var stylesMap map[string][]string
			if err := json.Unmarshal([]byte(jsonStr), &stylesMap); err == nil {
				changed := false
				for blockKey, items := range stylesMap {
					for idx, item := range items {
						if _, errParse := uuid.Parse(item); errParse == nil {
							continue
						}
						if targetID, ok := styleMap[item]; ok {
							items[idx] = targetID.String()
							changed = true
						} else if targetID, ok := styleMap[cleanStyleName(item)]; ok {
							items[idx] = targetID.String()
							changed = true
						}
					}
					stylesMap[blockKey] = items
				}
				if changed {
					newJSON, _ := json.Marshal(stylesMap)
					updates = append(updates, SchedUpdate{ID: id, JSON: string(newJSON)})
				}
			}
		}
	}
	rowsSched.Close()

	for _, u := range updates {
		_, err := Pool.Exec(ctx, "UPDATE basic_schedules SET custom_styles_json = $1 WHERE id = $2", u.JSON, u.ID)
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to update basic schedule %s during auto-healing: %v\n", u.ID, err)
		} else {
			fmt.Printf("🔧 Auto-healed basic schedule %s custom_styles_json to use persistent UUIDs.\n", u.ID)
		}
	}
}

// cleanStyleName strips common prefix patterns (e.g. "0 - ", "Z - ", "(TESTE) ")
func cleanStyleName(name string) string {
	clean := name
	if len(clean) > 4 && (strings.HasPrefix(clean, "0 - ") || strings.HasPrefix(clean, "Z - ") || strings.HasPrefix(clean, "z - ")) {
		clean = clean[4:]
	}
	clean = strings.TrimPrefix(clean, "(TESTE) ")
	clean = strings.TrimPrefix(clean, "TESTE - ")
	return strings.TrimSpace(clean)
}

// CloseDatabase closes all active connections in the pool gracefully
func CloseDatabase() {
	if Pool != nil {
		Pool.Close()
		fmt.Println("🛑 Database connection pool closed safely.")
	}
}
