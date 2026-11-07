package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"imc-server/internal/database"
	"imc-server/internal/models"
)

// checkStreamHTTP testa rapidamente se a URL remota responde e está online
func checkStreamHTTP(streamURL string) (bool, string) {
	if streamURL == "" {
		return false, "Sem URL"
	}
	client := &http.Client{
		Timeout: 2500 * time.Millisecond,
	}
	req, err := http.NewRequest(http.MethodGet, streamURL, nil)
	if err != nil {
		return false, "URL Inválida"
	}
	req.Header.Set("User-Agent", "IMC-Stream-HealthCheck/1.0")
	req.Header.Set("Range", "bytes=0-1024")

	resp, err := client.Do(req)
	if err != nil {
		return false, "Inacessível"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent {
		return true, "OK (200)"
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

// EnsureStreamingTableExists garante que a tabela streaming_stations exista no banco
func EnsureStreamingTableExists() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
	CREATE TABLE IF NOT EXISTS streaming_stations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		stream_url TEXT NOT NULL,
		genre VARCHAR(100) DEFAULT '',
		bitrate VARCHAR(50) DEFAULT '128 kbps MP3',
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := database.Pool.Exec(ctx, query)
	if err != nil {
		fmt.Println("[DB Error] Falha ao criar tabela streaming_stations:", err)
		return
	}

	// Insere as estações Hotmix padrão se a tabela estiver vazia
	var count int
	err = database.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM streaming_stations").Scan(&count)
	if err == nil && count == 0 {
		defaultStations := []struct {
			Name    string
			URL     string
			Genre   string
			Bitrate string
		}{
			{"Best-of", "https://streaming.hotmixradio.com/hotmixpro-best-of-pre", "Pop / Hits", "128 kbps MP3"},
			{"Cocktail & Dinner Groove", "https://streaming.hotmixradio.com/hotmixpro-cocktail-and-dinner-pre", "Lounge / Jazz", "128 kbps MP3"},
			{"Cool & Calm", "https://streaming.hotmixradio.com/hotmixpro-cool-and-calm-pre", "Chillout / Calm", "128 kbps MP3"},
			{"Cozy & Coffee", "https://streaming.hotmixradio.com/hotmixpro-cozy-coffee-shop-pre", "Acoustic / Indie", "128 kbps MP3"},
			{"Gentle Luxury", "https://streaming.hotmixradio.com/hotmixpro-gentle-luxury-pre", "Sofisticado / Luxury", "128 kbps MP3"},
			{"Just Lounge", "https://streaming.hotmixradio.com/hotmixpro-just-lounge-pre", "Lounge / Ambient", "128 kbps MP3"},
			{"On the Rocks", "https://streaming.hotmixradio.com/hotmixpro-on-the-rocks-pre", "Rock / Classic", "128 kbps MP3"},
			{"Shopping Vibes", "https://streaming.hotmixradio.com/hotmixpro-shopping-vibes-pre", "Varejo / Shopping", "128 kbps MP3"},
			{"Spa Garden", "https://streaming.hotmixradio.com/hotmixpro-spa-garden-pre", "Relax / Instrumental", "128 kbps MP3"},
			{"Workout", "https://streaming.hotmixradio.com/hotmixpro-workout-pre", "Dance / Fitness", "128 kbps MP3"},
		}

		for _, s := range defaultStations {
			_, _ = database.Pool.Exec(ctx, `
				INSERT INTO streaming_stations (name, stream_url, genre, bitrate, is_active)
				VALUES ($1, $2, $3, $4, true)
			`, s.Name, s.URL, s.Genre, s.Bitrate)
		}
	}
}

// HandleStreamingsPage renderiza a tela de listagem e monitoramento de streamings
func HandleStreamingsPage(w http.ResponseWriter, r *http.Request) {
	EnsureStreamingTableExists()

	session, _ := GetSessionFromRequest(r)
	loggedUser := "Administrador"
	if session != nil && session.UserName != "" {
		loggedUser = session.UserName
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Busca todas as estações de streaming
	rows, err := database.Pool.Query(ctx, `
		SELECT id, name, stream_url, COALESCE(genre, ''), COALESCE(bitrate, '128 kbps MP3'), is_active, created_at, updated_at
		FROM streaming_stations
		ORDER BY name ASC
	`)
	if err != nil {
		http.Error(w, "Erro ao buscar estações de streaming: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stations []models.StreamingStation
	for rows.Next() {
		var s models.StreamingStation
		if err := rows.Scan(&s.ID, &s.Name, &s.StreamURL, &s.Genre, &s.Bitrate, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err == nil {
			stations = append(stations, s)
		}
	}

	now := time.Now()
	weekdayCol := strings.ToLower(now.Weekday().String()) + "_schedule_id" // e.g. sunday_schedule_id

	// 1. Busca todos os estilos cadastrados do tipo STREAM
	type streamStyleInfo struct {
		ID        string
		Name      string
		StreamURL string
	}
	var streamStyles []streamStyleInfo
	sRows, errS := database.Pool.Query(ctx, `SELECT id::text, name, COALESCE(stream_url, '') FROM music_styles WHERE source_type = 'STREAM'`)
	if errS == nil {
		defer sRows.Close()
		for sRows.Next() {
			var ss streamStyleInfo
			if errScan := sRows.Scan(&ss.ID, &ss.Name, &ss.StreamURL); errScan == nil {
				streamStyles = append(streamStyles, ss)
			}
		}
	}

	// 2. Busca todas as lojas ativas com sua respectiva grade de áudio
	type storeData struct {
		ID              uuid.UUID
		StoreName       string
		StoreNumber     string
		TenantTradeName string
		ExternalURL     string
		ExternalName    string
		GridID          *uuid.UUID
	}
	var allActiveStores []storeData
	storeRows, err := database.Pool.Query(ctx, `
		SELECT s.id, s.store_name, s.store_number, COALESCE(t.trade_name, ''),
		       COALESCE(s.external_stream_url, ''), COALESCE(s.external_stream_name, ''),
		       s.audio_grid_id
		FROM stores s
		LEFT JOIN tenants t ON s.tenant_id = t.id
		WHERE s.is_active = true
	`)
	if err == nil {
		defer storeRows.Close()
		for storeRows.Next() {
			var sd storeData
			_ = storeRows.Scan(&sd.ID, &sd.StoreName, &sd.StoreNumber, &sd.TenantTradeName, &sd.ExternalURL, &sd.ExternalName, &sd.GridID)
			allActiveStores = append(allActiveStores, sd)
		}
	}

	// 3. Mapa de Loja -> StreamURL / StreamName ativo agora no ar
	type activeStreamEntry struct {
		StreamURL  string
		StreamName string
	}
	storeActiveStreamMap := make(map[uuid.UUID]activeStreamEntry)

	for _, sd := range allActiveStores {
		// A. Verifica a telemetria ao vivo da loja (o que está tocando agora no play_history)
		var dispName, styleName string
		var playedAt time.Time
		errLatest := database.Pool.QueryRow(ctx, `
			SELECT COALESCE(display_name, ''), COALESCE(style_name, ''), played_at 
			FROM play_history 
			WHERE store_id = $1 
			ORDER BY played_at DESC, id DESC 
			LIMIT 1
		`, sd.ID).Scan(&dispName, &styleName, &playedAt)

		if errLatest == nil && (time.Since(playedAt) <= 45*time.Minute || dispName != "") {
			cleanDisp := strings.TrimSpace(dispName)
			if cleanDisp != "" {
				storeActiveStreamMap[sd.ID] = activeStreamEntry{
					StreamURL:  "",
					StreamName: cleanDisp,
				}
				continue
			}
		}

		// B. Se não tiver histórico recente, verifica stream estático configurado na loja:
		if sd.ExternalURL != "" || sd.ExternalName != "" {
			storeActiveStreamMap[sd.ID] = activeStreamEntry{
				StreamURL:  strings.TrimSpace(sd.ExternalURL),
				StreamName: strings.TrimSpace(sd.ExternalName),
			}
			continue
		}

		// C. Caso contrário, resolve pela grade de áudio agendada para o bloco atual:
		if sd.GridID != nil {
			var schedID *uuid.UUID
			queryGrid := fmt.Sprintf(`SELECT %s FROM weekly_master_grids WHERE id = $1`, weekdayCol)
			_ = database.Pool.QueryRow(ctx, queryGrid, *sd.GridID).Scan(&schedID)

			if schedID != nil {
				var startTime, endTime string
				var customStylesJSON *string
				var musicalBlockDurationMinutes int
				errSched := database.Pool.QueryRow(ctx, `
					SELECT start_time, end_time, musical_block_duration_minutes, custom_styles_json
					FROM basic_schedules WHERE id = $1`, *schedID).Scan(&startTime, &endTime, &musicalBlockDurationMinutes, &customStylesJSON)

				if errSched == nil && customStylesJSON != nil && *customStylesJSON != "" {
					var stylesMap map[string][]string
					if errJSON := json.Unmarshal([]byte(*customStylesJSON), &stylesMap); errJSON == nil {
						var sh, sm, ss int
						_, _ = fmt.Sscanf(startTime, "%d:%d:%d", &sh, &sm, &ss)
						startSecs := sh*3600 + sm*60 + ss

						currTimeSecs := now.Hour()*3600 + now.Minute()*60 + now.Second()
						relMins := (currTimeSecs - startSecs) / 60
						if relMins < 0 {
							relMins = 0
						}

						musicalBlockMins := musicalBlockDurationMinutes
						if musicalBlockMins <= 0 {
							musicalBlockMins = 60
						}
						intervalMins := 15

						blockIdx := 0
						currPos := 0
						for currPos <= relMins {
							currPos += musicalBlockMins
							blockIdx++
							if currPos <= relMins {
								currPos += intervalMins
								blockIdx++
							}
						}

						targetBlockIdx := blockIdx - 1
						if targetBlockIdx < 0 {
							targetBlockIdx = 0
						}
						if targetBlockIdx%2 != 0 {
							targetBlockIdx--
						}
						if targetBlockIdx < 0 {
							targetBlockIdx = 0
						}

						activeBlockStyles := stylesMap[fmt.Sprintf("%d", targetBlockIdx)]
						for _, styleKey := range activeBlockStyles {
							for _, ss := range streamStyles {
								if strings.EqualFold(ss.ID, styleKey) || strings.EqualFold(ss.Name, styleKey) ||
								   (ss.Name != "" && strings.Contains(strings.ToLower(styleKey), strings.ToLower(ss.Name))) ||
								   (ss.Name != "" && strings.Contains(strings.ToLower(ss.Name), strings.ToLower(styleKey))) {
									storeActiveStreamMap[sd.ID] = activeStreamEntry{
										StreamURL:  strings.TrimSpace(ss.StreamURL),
										StreamName: strings.TrimSpace(ss.Name),
									}
									break
								}
							}
							if _, found := storeActiveStreamMap[sd.ID]; found {
								break
							}
						}
					}
				}
			}
		}
	}

	// 4. Associa as lojas ativas a cada estação de streaming
	totalStoresOnSet := make(map[uuid.UUID]bool)
	inUseCount := 0

	for i := range stations {
		st := &stations[i]
		stURL := strings.TrimSpace(st.StreamURL)
		stName := strings.TrimSpace(st.Name)

		for _, sd := range allActiveStores {
			activeStream, isStreaming := storeActiveStreamMap[sd.ID]
			if !isStreaming {
				continue
			}

			isMatch := false
			if stURL != "" && activeStream.StreamURL != "" && strings.EqualFold(activeStream.StreamURL, stURL) {
				isMatch = true
			} else if stName != "" && activeStream.StreamName != "" {
				s1 := strings.ToLower(stName)
				s2 := strings.ToLower(activeStream.StreamName)
				if s1 == s2 || strings.Contains(s2, s1) || strings.Contains(s1, s2) {
					isMatch = true
				}
			}

			if isMatch {
				displayName := fmt.Sprintf("%s (%s - %s)", sd.StoreName, sd.StoreNumber, sd.TenantTradeName)
				st.AssignedStores = append(st.AssignedStores, displayName)
				totalStoresOnSet[sd.ID] = true
			}
		}

		st.StoresCount = len(st.AssignedStores)
		if st.StoresCount > 0 {
			inUseCount++
			st.IsLiveInSystem = true
		}
	}

	totalStoresOn := len(totalStoresOnSet)

	// Executa teste de conexão concorrente nas URLs das estações
	var wg sync.WaitGroup
	for i := range stations {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			st := &stations[idx]
			isOk, statusStr := checkStreamHTTP(st.StreamURL)
			st.IsOnline = isOk
			st.ConnectionStatus = statusStr
		}(i)
	}
	wg.Wait()

	data := models.StreamingPageData{
		Title:          "Gerenciador de Streamings & Rádios",
		LoggedUser:     loggedUser,
		TotalStations:  len(stations),
		InUseStations:  inUseCount,
		TotalStoresOn:  totalStoresOn,
		Stations:       stations,
		ErrorMessage:   r.URL.Query().Get("error"),
		SuccessMessage: r.URL.Query().Get("success"),
	}

	tmpl, err := template.ParseFiles("web/templates/streamings.html")
	if err != nil {
		http.Error(w, "Erro ao carregar template de streamings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

// HandleCreateStreaming cadastra uma nova estação de streaming
func HandleCreateStreaming(w http.ResponseWriter, r *http.Request) {
	EnsureStreamingTableExists()

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/streamings?error=Dados+do+formulário+inválidos", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	streamURL := strings.TrimSpace(r.FormValue("stream_url"))
	genre := strings.TrimSpace(r.FormValue("genre"))
	bitrate := strings.TrimSpace(r.FormValue("bitrate"))
	isActive := r.FormValue("is_active") == "true" || r.FormValue("is_active") == "on" || r.FormValue("is_active") == "1"

	if name == "" || streamURL == "" {
		http.Redirect(w, r, "/streamings?error=Nome+e+URL+da+estação+são+obrigatórios", http.StatusSeeOther)
		return
	}

	if bitrate == "" {
		bitrate = "128 kbps MP3"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := database.Pool.Exec(ctx, `
		INSERT INTO streaming_stations (name, stream_url, genre, bitrate, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`, name, streamURL, genre, bitrate, isActive)

	if err != nil {
		http.Redirect(w, r, "/streamings?error="+url.QueryEscape("Erro ao cadastrar streaming: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/streamings?success="+url.QueryEscape("Estação de streaming cadastrada com sucesso!"), http.StatusSeeOther)
}

// HandleUpdateStreaming atualiza os dados de uma estação de streaming existente
func HandleUpdateStreaming(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/streamings?error=Dados+do+formulário+inválidos", http.StatusSeeOther)
		return
	}

	idStr := strings.TrimSpace(r.FormValue("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/streamings?error=ID+da+estação+inválido", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	streamURL := strings.TrimSpace(r.FormValue("stream_url"))
	genre := strings.TrimSpace(r.FormValue("genre"))
	bitrate := strings.TrimSpace(r.FormValue("bitrate"))
	isActive := r.FormValue("is_active") == "true" || r.FormValue("is_active") == "on" || r.FormValue("is_active") == "1"

	if name == "" || streamURL == "" {
		http.Redirect(w, r, "/streamings?error=Nome+e+URL+da+estação+são+obrigatórios", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = database.Pool.Exec(ctx, `
		UPDATE streaming_stations
		SET name = $1, stream_url = $2, genre = $3, bitrate = $4, is_active = $5, updated_at = CURRENT_TIMESTAMP
		WHERE id = $6
	`, name, streamURL, genre, bitrate, isActive, id)

	if err != nil {
		http.Redirect(w, r, "/streamings?error="+url.QueryEscape("Erro ao atualizar streaming: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/streamings?success="+url.QueryEscape("Estação de streaming atualizada com sucesso!"), http.StatusSeeOther)
}

// HandleDeleteStreaming exclui uma estação de streaming
func HandleDeleteStreaming(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	if idStr == "" {
		idStr = strings.TrimSpace(r.FormValue("id"))
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/streamings?error=ID+inválido", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Verifica se há lojas usando esta estação
	var stationName, stationURL string
	err = database.Pool.QueryRow(ctx, "SELECT name, stream_url FROM streaming_stations WHERE id = $1", id).Scan(&stationName, &stationURL)
	if err == nil {
		var storeCount int
		_ = database.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM stores 
			WHERE is_active = true AND radio_type = 'ONLINE_STREAM' 
			AND (external_stream_url = $1 OR external_stream_name = $2)
		`, stationURL, stationName).Scan(&storeCount)

		if storeCount > 0 {
			http.Redirect(w, r, "/streamings?error="+url.QueryEscape(fmt.Sprintf("Não é possível excluir: existem %d lojas utilizando esta estação no momento.", storeCount)), http.StatusSeeOther)
			return
		}
	}

	_, err = database.Pool.Exec(ctx, "DELETE FROM streaming_stations WHERE id = $1", id)
	if err != nil {
		http.Redirect(w, r, "/streamings?error="+url.QueryEscape("Erro ao excluir: "+err.Error()), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/streamings?success="+url.QueryEscape("Estação de streaming excluída com sucesso!"), http.StatusSeeOther)
}

// HandleGetStreamingsAPI retorna lista JSON das estações ativas (para dropdown de lojas)
func HandleGetStreamingsAPI(w http.ResponseWriter, r *http.Request) {
	EnsureStreamingTableExists()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := database.Pool.Query(ctx, `
		SELECT id, name, stream_url, genre, bitrate
		FROM streaming_stations
		WHERE is_active = true
		ORDER BY name ASC
	`)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type apiStation struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		StreamURL string    `json:"stream_url"`
		Genre     string    `json:"genre"`
		Bitrate   string    `json:"bitrate"`
	}

	var list []apiStation
	for rows.Next() {
		var s apiStation
		if err := rows.Scan(&s.ID, &s.Name, &s.StreamURL, &s.Genre, &s.Bitrate); err == nil {
			list = append(list, s)
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(list)
}
