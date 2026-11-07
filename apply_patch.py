import re

with open('/home/neoroute/imc-server/internal/handlers/customer_player_handler.go', 'r', encoding='utf-8') as f:
    text = f.read()

# 1. Add resolveActiveStreamName if missing
resolve_func = """// resolveActiveStreamName descobre o nome do streaming ativo para a loja
func resolveActiveStreamName(ctx context.Context, store *models.Store) string {
	if store.ExternalStreamName != nil && strings.TrimSpace(*store.ExternalStreamName) != "" {
		return strings.TrimSpace(*store.ExternalStreamName)
	}

	var scheduledName string
	query := `
		SELECT COALESCE(st.name, ms.name, s.name, 'Streaming Ao Vivo')
		FROM schedules s
		LEFT JOIN streamings st ON s.streaming_id = st.id
		LEFT JOIN media_styles ms ON s.media_style_id = ms.id
		WHERE s.store_id = $1 
		  AND s.is_active = true
		  AND (s.start_date IS NULL OR s.start_date <= CURRENT_DATE)
		  AND (s.end_date IS NULL OR s.end_date >= CURRENT_DATE)
		  AND (s.start_time <= CURRENT_TIME AND s.end_time >= CURRENT_TIME)
		LIMIT 1
	`
	err := database.Pool.QueryRow(ctx, query, store.ID).Scan(&scheduledName)
	if err == nil && strings.TrimSpace(scheduledName) != "" {
		return strings.TrimSpace(scheduledName)
	}

	return "Streaming Ao Vivo"
}
"""

if "func resolveActiveStreamName" not in text:
    pos = text.find("// HandleCustomerPlayerPage")
    if pos != -1:
        text = text[:pos] + resolve_func + "\n" + text[pos:]

# 2. Update HandleCustomerPlayerPage
old_p = """	isStreaming := (store.RadioType == "external" || (store.ExternalStreamURL != nil && *store.ExternalStreamURL != ""))
	streamName := ""
	title := "Radio no Ar"
	artist := store.StoreName
	mediaType := "Musica"
	formattedRemaining := "--:--"
	formattedElapsed := "00:00"
	progressPercent := 0.0

	if isStreaming {
		streamName = "Streaming Externo"
		if store.ExternalStreamName != nil && strings.TrimSpace(*store.ExternalStreamName) != "" {
			streamName = strings.TrimSpace(*store.ExternalStreamName)
		}
		title = streamName
		artist = "Transmissão Streaming Ao Vivo"
		mediaType = "Streaming Ao Vivo"
		formattedRemaining = "--:--"
		formattedElapsed = "--:--"
	} else {
		// Tenta buscar metadados detalhados com tempo restante do Liquidsoap
		meta, errMeta := getLiquidsoapMeta(store)
		if errMeta == nil && meta != nil && meta.Title != "" {
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			if math.IsInf(meta.RemainingSeconds, 0) || meta.RemainingSeconds < 0 || meta.Filename == "" || strings.HasPrefix(meta.Filename, "http") || meta.Type == "stream" || meta.Type == "streaming" {
				isStreaming = true
				mediaType = "Streaming Ao Vivo"
				formattedRemaining = "--:--"
				formattedElapsed = "--:--"
				progressPercent = 0.0
			} else {
				if meta.Type == "commercial" {
					mediaType = "Comercial / Evento"
				} else if meta.Type == "warning" {
					mediaType = "Aviso / Comunicado"
				} else {
					mediaType = "Música"
				}

				if meta.RemainingSeconds > 0 {
					remM := int(meta.RemainingSeconds) / 60
					remS := int(meta.RemainingSeconds) % 60
					formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remM, remS)
				} else if meta.ElapsedSeconds > 0 {
					formattedRemaining = "Quase no fim"
				}
				if meta.ElapsedSeconds > 0 {
					elM := int(meta.ElapsedSeconds) / 60
					elS := int(meta.ElapsedSeconds) % 60
					formattedElapsed = fmt.Sprintf("%02d:%02d", elM, elS)
				}
				tot := meta.ElapsedSeconds + meta.RemainingSeconds
				if tot > 0 {
					progressPercent = (meta.ElapsedSeconds / tot) * 100.0
				}
			}
		} else if isOnline && source.Title != "" {
			title = source.Title
			if source.Artist != "" {
				artist = source.Artist
			}
		}
	}"""

new_p = """	isStreaming := (store.RadioType == "external" || (store.ExternalStreamURL != nil && *store.ExternalStreamURL != ""))
	streamName := resolveActiveStreamName(ctx, store)
	title := "Radio no Ar"
	artist := store.StoreName
	mediaType := "Musica"
	formattedRemaining := "--:--"
	formattedElapsed := "00:00"
	progressPercent := 0.0

	// Tenta buscar metadados detalhados com tempo restante do Liquidsoap
	meta, errMeta := getLiquidsoapMeta(store)
	if errMeta == nil && meta != nil {
		isInfRemaining := math.IsInf(meta.RemainingSeconds, 0) || meta.RemainingSeconds < 0
		isCommercialActive := (meta.Type == "commercial" || meta.Type == "institutional" || meta.Type == "warning") && meta.RemainingSeconds > 0 && !isInfRemaining

		if isCommercialActive {
			if meta.Type == "warning" {
				mediaType = "Aviso / Comunicado"
			} else {
				mediaType = "Comercial / Evento"
			}
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			if meta.RemainingSeconds > 0 {
				remM := int(meta.RemainingSeconds) / 60
				remS := int(meta.RemainingSeconds) % 60
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remM, remS)
			}
			if meta.ElapsedSeconds > 0 {
				elM := int(meta.ElapsedSeconds) / 60
				elS := int(meta.ElapsedSeconds) % 60
				formattedElapsed = fmt.Sprintf("%02d:%02d", elM, elS)
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}
		} else if isStreaming || isInfRemaining || meta.Filename == "" || strings.HasPrefix(meta.Filename, "http") || meta.Type == "stream" || meta.Type == "streaming" {
			isStreaming = true
			mediaType = "Streaming Ao Vivo"
			title = streamName
			artist = "Transmissão Streaming Ao Vivo"
			formattedRemaining = "--:--"
			formattedElapsed = "--:--"
			progressPercent = 0.0
		} else if meta.Title != "" {
			mediaType = "Música"
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			if meta.RemainingSeconds > 0 {
				remM := int(meta.RemainingSeconds) / 60
				remS := int(meta.RemainingSeconds) % 60
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remM, remS)
			} else if meta.ElapsedSeconds > 0 {
				formattedRemaining = "Quase no fim"
			}
			if meta.ElapsedSeconds > 0 {
				elM := int(meta.ElapsedSeconds) / 60
				elS := int(meta.ElapsedSeconds) % 60
				formattedElapsed = fmt.Sprintf("%02d:%02d", elM, elS)
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}
		}
	} else if isOnline && source.Title != "" {
		title = source.Title
		if source.Artist != "" {
			artist = source.Artist
		}
	}"""

text = text.replace(old_p, new_p)

# 3. Update HandlePlayerNowPlayingAPI
old_a = """	isStreaming := (store.RadioType == "external" || (store.ExternalStreamURL != nil && *store.ExternalStreamURL != ""))
	streamName := ""
	title := "Radio no Ar"
	artist := store.StoreName
	album := ""
	mediaType := "Musica"
	durationSec := 0
	elapsedSec := 0
	remainingSec := 0
	formattedDuration := "--:--"
	formattedElapsed := "00:00"
	formattedRemaining := "--:--"
	progressPercent := 0.0

	if isStreaming {
		streamName = "Streaming Externo"
		if store.ExternalStreamName != nil && strings.TrimSpace(*store.ExternalStreamName) != "" {
			streamName = strings.TrimSpace(*store.ExternalStreamName)
		}
		title = streamName
		artist = "Transmissão Streaming Ao Vivo"
		mediaType = "Streaming Ao Vivo"
		formattedRemaining = "--:--"
		formattedElapsed = "--:--"
		formattedDuration = "--:--"
	} else {
		// 1. Busca metadados de execução do Liquidsoap
		meta, errMeta := getLiquidsoapMeta(store)
		if errMeta == nil && meta != nil && meta.Title != "" {
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}

			if math.IsInf(meta.RemainingSeconds, 0) || meta.RemainingSeconds < 0 || meta.Filename == "" || strings.HasPrefix(meta.Filename, "http") || meta.Type == "stream" || meta.Type == "streaming" {
				isStreaming = true
				mediaType = "Streaming Ao Vivo"
				formattedRemaining = "--:--"
				formattedElapsed = "--:--"
				formattedDuration = "--:--"
				progressPercent = 0.0
			} else {
				durationSec = int(meta.DurationSeconds)
				elapsedSec = int(meta.ElapsedSeconds)
				remainingSec = int(meta.RemainingSeconds)

				if meta.Type == "commercial" {
					mediaType = "Comercial / Evento"
				} else if meta.Type == "warning" {
					mediaType = "Aviso / Comunicado"
				} else {
					mediaType = "Música"
				}

				if durationSec > 0 {
					m := durationSec / 60
					s := durationSec % 60
					formattedDuration = fmt.Sprintf("%02d:%02d", m, s)
				}
				if elapsedSec > 0 {
					m := elapsedSec / 60
					s := elapsedSec % 60
					formattedElapsed = fmt.Sprintf("%02d:%02d", m, s)
				}
				if remainingSec > 0 {
					m := remainingSec / 60
					s := remainingSec % 60
					formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", m, s)
				} else if elapsedSec > 0 {
					formattedRemaining = "Quase no fim"
				}
				tot := meta.ElapsedSeconds + meta.RemainingSeconds
				if tot > 0 {
					progressPercent = (meta.ElapsedSeconds / tot) * 100.0
				}
			}
		} else if isOnline && source.Title != "" {
			title = source.Title
			if source.Artist != "" {
				artist = source.Artist
			}
		}

		// 2. Busca informações do álbum no acervo
		var dbAlbum string
		errQuery := database.Pool.QueryRow(ctx, `
			SELECT COALESCE(album, '')
			FROM media_library 
			WHERE (display_name = $1 OR file_name = $1 OR title = $1)
			LIMIT 1
		`, title).Scan(&dbAlbum)
		if errQuery == nil {
			album = dbAlbum
		}
	}"""

new_a = """	isStreaming := (store.RadioType == "external" || (store.ExternalStreamURL != nil && *store.ExternalStreamURL != ""))
	streamName := resolveActiveStreamName(ctx, store)
	title := "Radio no Ar"
	artist := store.StoreName
	album := ""
	mediaType := "Musica"
	durationSec := 0
	elapsedSec := 0
	remainingSec := 0
	formattedDuration := "--:--"
	formattedElapsed := "00:00"
	formattedRemaining := "--:--"
	progressPercent := 0.0

	// 1. Busca metadados de execução do Liquidsoap
	meta, errMeta := getLiquidsoapMeta(store)
	if errMeta == nil && meta != nil {
		isInfRemaining := math.IsInf(meta.RemainingSeconds, 0) || meta.RemainingSeconds < 0
		isCommercialActive := (meta.Type == "commercial" || meta.Type == "institutional" || meta.Type == "warning") && meta.RemainingSeconds > 0 && !isInfRemaining

		if isCommercialActive {
			if meta.Type == "warning" {
				mediaType = "Aviso / Comunicado"
			} else {
				mediaType = "Comercial / Evento"
			}
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			durationSec = int(meta.DurationSeconds)
			elapsedSec = int(meta.ElapsedSeconds)
			remainingSec = int(meta.RemainingSeconds)

			if durationSec > 0 {
				formattedDuration = fmt.Sprintf("%02d:%02d", durationSec/60, durationSec%60)
			}
			if elapsedSec > 0 {
				formattedElapsed = fmt.Sprintf("%02d:%02d", elapsedSec/60, elapsedSec%60)
			}
			if remainingSec > 0 {
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remainingSec/60, remainingSec%60)
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}
		} else if isStreaming || isInfRemaining || meta.Filename == "" || strings.HasPrefix(meta.Filename, "http") || meta.Type == "stream" || meta.Type == "streaming" {
			isStreaming = true
			mediaType = "Streaming Ao Vivo"
			title = streamName
			artist = "Transmissão Streaming Ao Vivo"
			formattedRemaining = "--:--"
			formattedElapsed = "--:--"
			formattedDuration = "--:--"
			progressPercent = 0.0
		} else if meta.Title != "" {
			mediaType = "Música"
			title = meta.Title
			if meta.Artist != "" {
				artist = meta.Artist
			}
			durationSec = int(meta.DurationSeconds)
			elapsedSec = int(meta.ElapsedSeconds)
			remainingSec = int(meta.RemainingSeconds)

			if durationSec > 0 {
				formattedDuration = fmt.Sprintf("%02d:%02d", durationSec/60, durationSec%60)
			}
			if elapsedSec > 0 {
				formattedElapsed = fmt.Sprintf("%02d:%02d", elapsedSec/60, elapsedSec%60)
			}
			if remainingSec > 0 {
				formattedRemaining = fmt.Sprintf("Faltam %02d:%02d", remainingSec/60, remainingSec%60)
			} else if elapsedSec > 0 {
				formattedRemaining = "Quase no fim"
			}
			tot := meta.ElapsedSeconds + meta.RemainingSeconds
			if tot > 0 {
				progressPercent = (meta.ElapsedSeconds / tot) * 100.0
			}

			// 2. Busca informações do álbum no acervo
			var dbAlbum string
			errQuery := database.Pool.QueryRow(ctx, `
				SELECT COALESCE(album, '')
				FROM media_library 
				WHERE (display_name = $1 OR file_name = $1 OR title = $1)
				LIMIT 1
			`, title).Scan(&dbAlbum)
			if errQuery == nil {
				album = dbAlbum
			}
		}
	} else if isOnline && source.Title != "" {
		title = source.Title
		if source.Artist != "" {
			artist = source.Artist
		}
	}"""

text = text.replace(old_a, new_a)

with open('/home/neoroute/imc-server/internal/handlers/customer_player_handler.go', 'w', encoding='utf-8') as f:
    f.write(text)

print("Applied patch successfully.")
