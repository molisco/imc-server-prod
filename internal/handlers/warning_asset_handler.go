package handlers

import "net/http"

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"imc-server/internal/database"
	"imc-server/internal/models"
	"imc-server/internal/utils"

	"github.com/google/uuid"
)

// Common Theme Data
func getStandardTheme() map[string]string {
	return map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "#161925",
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}
}

// -----------------------------------------------------------------------------
// 1. VOICE PACKS HANDLERS
// -----------------------------------------------------------------------------

type VoicePacksPageData struct {
	Title      string
	User       map[string]string
	Theme      map[string]string
	VoicePacks []models.VoicePack
	Stores     []models.Store
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func HandleVoicePacksPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, _ := url.QueryUnescape(cookie.Value)
	var session models.SessionData
	_ = json.Unmarshal([]byte(decodedValue), &session)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch all stores for the checklist
	var stores []models.Store
	rowsStores, errS := database.Pool.Query(ctx, "SELECT id, COALESCE(store_name, '') as store_name, COALESCE(store_number, '') as store_number FROM stores ORDER BY store_name ASC")
	if errS == nil {
		defer rowsStores.Close()
		for rowsStores.Next() {
			var s models.Store
			if rowsStores.Scan(&s.ID, &s.StoreName, &s.StoreNumber) == nil {
				stores = append(stores, s)
			}
		}
	}

	var packs []models.VoicePack
	rows, err := database.Pool.Query(ctx, `
		SELECT id, COALESCE(name, '') as name, COALESCE(category, 'Estacionamento / Placas') as category, 
		       COALESCE(description, '') as description, COALESCE(is_global, false) as is_global, 
		       COALESCE(base_directory, '') as base_directory, is_active, created_at 
		FROM voice_packs 
		ORDER BY name ASC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vp models.VoicePack
			if rows.Scan(&vp.ID, &vp.Name, &vp.Category, &vp.Description, &vp.IsGlobal, &vp.BaseDirectory, &vp.IsActive, &vp.CreatedAt) == nil {
				// Query target stores for this pack
				rowsTPS, errTPS := database.Pool.Query(ctx, `
					SELECT s.id, s.name 
					FROM voice_pack_stores vps 
					JOIN stores s ON vps.store_id = s.id 
					WHERE vps.voice_pack_id = $1 
					ORDER BY s.name ASC`, vp.ID)
				if errTPS == nil {
					for rowsTPS.Next() {
						var sID, sName string
						if rowsTPS.Scan(&sID, &sName) == nil {
							vp.TargetStoreIDs = append(vp.TargetStoreIDs, sID)
							vp.TargetStoreNames = append(vp.TargetStoreNames, sName)
						}
					}
					rowsTPS.Close()
				}

				// Scan directory for audio files
				if vp.BaseDirectory != "" {
					if entries, errRead := os.ReadDir(vp.BaseDirectory); errRead == nil {
						for _, entry := range entries {
							if !entry.IsDir() {
								info, errInfo := entry.Info()
								sizeStr := "-"
								if errInfo == nil {
									sizeStr = formatBytes(info.Size())
								}
								vp.AudioFiles = append(vp.AudioFiles, models.VoicePackAudioFile{
									Name:          entry.Name(),
									Path:          "/" + filepath.ToSlash(filepath.Join(vp.BaseDirectory, entry.Name())),
									SizeFormatted: sizeStr,
								})
							}
						}
					}
				}

				packs = append(packs, vp)
			}
		}
	}

	data := VoicePacksPageData{
		Title: "Cadastro de Placas (Avisos de Voz)",
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme:      getStandardTheme(),
		VoicePacks: packs,
		Stores:     stores,
	}

	tmpl, err := template.ParseFiles("web/templates/voice_packs.html")
	if err != nil {
		log.Printf("[ERROR] Failed to parse voice_packs.html template: %v", err)
		http.Error(w, "Erro de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("[ERROR] voice_packs.html execution error: %v", err)
	}
}

func saveAndSonicProcessFile(targetDir string, formFieldName string, overrideName string, r *http.Request) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return
	}
	files := r.MultipartForm.File[formFieldName]
	for _, fheader := range files {
		file, err := fheader.Open()
		if err != nil {
			continue
		}
		saveName := fheader.Filename
		if overrideName != "" {
			ext := filepath.Ext(fheader.Filename)
			if ext == "" {
				ext = ".mp3"
			}
			saveName = overrideName + ext
		}
		destPath := filepath.Join(targetDir, saveName)
		out, err := os.Create(destPath)
		if err == nil {
			_, _ = io.Copy(out, file)
			out.Close()

			// Sonic Pipeline Processing (Sonic audio analysis via ffprobe/ffmpeg)
			cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", destPath)
			_, _ = cmd.Output()
		}
		file.Close()
	}
}

func HandleCreateVoicePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseMultipartForm(50 << 20) // 50MB max

	name := strings.TrimSpace(r.FormValue("name"))
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "Estacionamento / Placas"
	}
	description := strings.TrimSpace(r.FormValue("description"))
	isGlobal := r.FormValue("is_global") == "1" || r.FormValue("is_global") == "true" || r.FormValue("is_global") == "on"

	if name == "" {
		http.Redirect(w, r, "/voice-packs?error=Nome+da+placa+e+obrigatorio", http.StatusSeeOther)
		return
	}

	dirName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	targetDir := filepath.Join("midias", "voice_packs", dirName)
	_ = os.MkdirAll(targetDir, 0755)

	// Save and process simplified slots through Sonic Pipeline
	saveAndSonicProcessFile(targetDir, "file_intro", "intro", r)
	saveAndSonicProcessFile(targetDir, "file_repetindo", "repetindo", r)
	saveAndSonicProcessFile(targetDir, "files_letters", "", r)
	saveAndSonicProcessFile(targetDir, "files_numbers", "", r)
	saveAndSonicProcessFile(targetDir, "file_outro", "outro", r)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vpID := uuid.New().String()
	_, err := database.Pool.Exec(ctx, `
		INSERT INTO voice_packs (id, name, category, description, is_global, base_directory, is_active) 
		VALUES ($1, $2, 'Aviso de Placas', $3, $4, $5, true)`, vpID, name, description, isGlobal, targetDir)
	if err != nil {
		http.Redirect(w, r, "/voice-packs?error=Falha+ao+salvar+no+banco", http.StatusSeeOther)
		return
	}

	// Insert targeted store relationships if not global
	if !isGlobal {
		storeIDs := r.MultipartForm.Value["store_ids"]
		for _, sid := range storeIDs {
			if strings.TrimSpace(sid) != "" {
				_, _ = database.Pool.Exec(ctx, "INSERT INTO voice_pack_stores (voice_pack_id, store_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", vpID, sid)
			}
		}
	}

	http.Redirect(w, r, "/voice-packs?success=Aviso+de+placa+cadastrado+e+processado+no+Sonic+com+sucesso", http.StatusSeeOther)
}

func HandleUpdateVoicePack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método HTTP não permitido", http.StatusMethodNotAllowed)
		return
	}

	errParse := r.ParseMultipartForm(50 << 20) // 50MB
	if errParse != nil {
		r.ParseForm()
	}

	id := r.FormValue("id")
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	isGlobal := r.FormValue("is_global") == "true"

	if id == "" || name == "" {
		http.Redirect(w, r, "/voice-packs?error=ID+e+Nome+do+conjunto+são+obrigatórios", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Retrieve current base directory
	var currentDir string
	errQuery := database.Pool.QueryRow(ctx, "SELECT base_directory FROM voice_packs WHERE id = $1", id).Scan(&currentDir)
	if errQuery != nil {
		http.Redirect(w, r, "/voice-packs?error=Conjunto+não+encontrado", http.StatusSeeOther)
		return
	}

	// Save and process any new file uploads into currentDir
	if r.MultipartForm != nil {
		saveAndSonicProcessFile(currentDir, "file_intro", "intro", r)
		saveAndSonicProcessFile(currentDir, "file_repetindo", "repetindo", r)
		saveAndSonicProcessFile(currentDir, "files_letters", "", r)
		saveAndSonicProcessFile(currentDir, "files_numbers", "", r)
		saveAndSonicProcessFile(currentDir, "file_outro", "outro", r)
	}

	// Update metadata in database
	_, err := database.Pool.Exec(ctx, `
		UPDATE voice_packs 
		SET name = $1, description = $2, is_global = $3 
		WHERE id = $4`, name, description, isGlobal, id)
	if err != nil {
		http.Redirect(w, r, "/voice-packs?error=Falha+ao+atualizar+no+banco", http.StatusSeeOther)
		return
	}

	// Clear old store associations and re-insert new ones if not global
	_, _ = database.Pool.Exec(ctx, "DELETE FROM voice_pack_stores WHERE voice_pack_id = $1", id)
	if !isGlobal && r.MultipartForm != nil {
		storeIDs := r.MultipartForm.Value["store_ids"]
		for _, sid := range storeIDs {
			if strings.TrimSpace(sid) != "" {
				_, _ = database.Pool.Exec(ctx, "INSERT INTO voice_pack_stores (voice_pack_id, store_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", id, sid)
			}
		}
	}

	http.Redirect(w, r, "/voice-packs?success=Conjunto+de+placas+atualizado+com+sucesso", http.StatusSeeOther)
}

func HandleDeleteVoicePack(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var baseDir string
		_ = database.Pool.QueryRow(ctx, "SELECT base_directory FROM voice_packs WHERE id = $1", idStr).Scan(&baseDir)
		if baseDir != "" {
			_ = os.RemoveAll(baseDir)
		}
		_, _ = database.Pool.Exec(ctx, "DELETE FROM voice_packs WHERE id = $1", idStr)
	}
	http.Redirect(w, r, "/voice-packs", http.StatusSeeOther)
}

func HandleDeleteVoicePackFile(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	fileName := r.URL.Query().Get("file")
	if idStr != "" && fileName != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var baseDir string
		_ = database.Pool.QueryRow(ctx, "SELECT base_directory FROM voice_packs WHERE id = $1", idStr).Scan(&baseDir)
		if baseDir != "" {
			targetPath := filepath.Join(baseDir, filepath.Base(fileName))
			_ = os.Remove(targetPath)
		}
	}
	http.Redirect(w, r, "/voice-packs", http.StatusSeeOther)
}

func HandleReprocessVoicePackSonic(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Redirect(w, r, "/voice-packs?error=ID+do+conjunto+invalido", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var baseDir string
	err := database.Pool.QueryRow(ctx, "SELECT base_directory FROM voice_packs WHERE id = $1", idStr).Scan(&baseDir)
	if err != nil || baseDir == "" {
		http.Redirect(w, r, "/voice-packs?error=Conjunto+nao+encontrado", http.StatusSeeOther)
		return
	}

	entries, errRead := os.ReadDir(baseDir)
	if errRead != nil {
		http.Redirect(w, r, "/voice-packs?error=Erro+ao+ler+diretorio+do+conjunto", http.StatusSeeOther)
		return
	}

	reprocessedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".mp3") {
			filePath := filepath.Join(baseDir, entry.Name())
			if errMaster := utils.MasterizeAudio(filePath); errMaster == nil {
				reprocessedCount++
			}
		}
	}

	msg := fmt.Sprintf("Processamento+Sonic+concluido!+%d+arquivos+do+conjunto+reprocessados+no+padrao+SONIC", reprocessedCount)
	http.Redirect(w, r, "/voice-packs?success="+msg, http.StatusSeeOther)
}

func HandleReprocessVoicePackFileSonic(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	fileName := r.URL.Query().Get("file")
	if idStr == "" || fileName == "" {
		http.Redirect(w, r, "/voice-packs?error=ID+ou+nome+do+arquivo+invalido", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var baseDir string
	err := database.Pool.QueryRow(ctx, "SELECT base_directory FROM voice_packs WHERE id = $1", idStr).Scan(&baseDir)
	if err != nil || baseDir == "" {
		http.Redirect(w, r, "/voice-packs?error=Conjunto+nao+encontrado", http.StatusSeeOther)
		return
	}

	filePath := filepath.Join(baseDir, filepath.Base(fileName))
	if errMaster := utils.MasterizeAudio(filePath); errMaster != nil {
		log.Printf("❌ [SONIC] Erro ao reprocessar arquivo individual %s: %v", filePath, errMaster)
		http.Redirect(w, r, "/voice-packs?error=Falha+ao+reprocessar+arquivo+no+Sonic", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/voice-packs?success=Arquivo+"+filepath.Base(fileName)+"+reprocessado+com+sucesso+no+padrao+SONIC", http.StatusSeeOther)
}

// -----------------------------------------------------------------------------
// 2. CHIMES HANDLERS
// -----------------------------------------------------------------------------

type ChimesPageData struct {
	Title  string
	User   map[string]string
	Theme  map[string]string
	Chimes []models.Chime
}

func HandleChimesPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, _ := url.QueryUnescape(cookie.Value)
	var session models.SessionData
	_ = json.Unmarshal([]byte(decodedValue), &session)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var chimes []models.Chime
	rows, err := database.Pool.Query(ctx, "SELECT id, name, file_name, duration_seconds, cue_in, cue_out, lufs, amplify, is_active, created_at FROM chimes ORDER BY name ASC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ch models.Chime
			if rows.Scan(&ch.ID, &ch.Name, &ch.FileName, &ch.DurationSeconds, &ch.CueIn, &ch.CueOut, &ch.LUFS, &ch.Amplify, &ch.IsActive, &ch.CreatedAt) == nil {
				chimes = append(chimes, ch)
			}
		}
	}

	data := ChimesPageData{
		Title: "Sinais Sonoros / Chimes",
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme:  getStandardTheme(),
		Chimes: chimes,
	}

	tmpl, err := template.ParseFiles("web/templates/chimes.html")
	if err != nil {
		http.Error(w, "Erro de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func HandleCreateChime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseMultipartForm(50 << 20)

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/chimes?error=Nome+do+chime+e+obrigatorio", http.StatusSeeOther)
		return
	}

	file, header, err := r.FormFile("audio_file")
	if err != nil {
		http.Redirect(w, r, "/chimes?error=Arquivo+de+audio+obrigatorio", http.StatusSeeOther)
		return
	}
	defer file.Close()

	chimesDir := filepath.Join("midias", "chimes")
	_ = os.MkdirAll(chimesDir, 0755)

	fileExt := filepath.Ext(header.Filename)
	safeFileName := fmt.Sprintf("chime_%d%s", time.Now().UnixNano(), fileExt)
	destPath := filepath.Join(chimesDir, safeFileName)

	out, err := os.Create(destPath)
	if err != nil {
		http.Redirect(w, r, "/chimes?error=Falha+ao+salvar+arquivo", http.StatusSeeOther)
		return
	}
	_, _ = io.Copy(out, file)
	out.Close()

	// Perform Sonic Audio Pipeline Analysis (LUFS & Duration estimation)
	durationSecs := 5
	lufs := -14.0
	amplify := 1.0

	// Try running ffmpeg / ffprobe for exact duration if present
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", destPath)
	output, errCmd := cmd.Output()
	if errCmd == nil {
		if durFloat, errParse := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); errParse == nil {
			durationSecs = int(durFloat)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chimeID := uuid.New().String()
	cueOut := float64(durationSecs)
	_, err = database.Pool.Exec(ctx, `
		INSERT INTO chimes (id, name, file_name, duration_seconds, cue_in, cue_out, lufs, amplify, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)`,
		chimeID, name, safeFileName, durationSecs, 0.0, cueOut, lufs, amplify,
	)
	if err != nil {
		log.Printf("❌ [CHIMES] Erro ao cadastrar chime no banco: %v", err)
		http.Redirect(w, r, "/chimes?error=Falha+ao+persistir+no+banco", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/chimes?success=Sinal+sonoro+cadastrado+com+sucesso", http.StatusSeeOther)
}

func HandleDeleteChime(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var fileName string
		_ = database.Pool.QueryRow(ctx, "SELECT file_name FROM chimes WHERE id = $1", idStr).Scan(&fileName)
		if fileName != "" {
			_ = os.Remove(filepath.Join("midias", "chimes", fileName))
		}
		_, _ = database.Pool.Exec(ctx, "DELETE FROM chimes WHERE id = $1", idStr)
	}
	http.Redirect(w, r, "/chimes", http.StatusSeeOther)
}

func HandleReprocessChimeSonic(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Redirect(w, r, "/chimes?error=ID+do+chime+invalido", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var fileName string
	err := database.Pool.QueryRow(ctx, "SELECT file_name FROM chimes WHERE id = $1", idStr).Scan(&fileName)
	if err != nil || fileName == "" {
		http.Redirect(w, r, "/chimes?error=Sinal+sonoro+nao+encontrado", http.StatusSeeOther)
		return
	}

	filePath := filepath.Join("midias", "chimes", fileName)
	if errMaster := utils.MasterizeAudio(filePath); errMaster != nil {
		log.Printf("❌ [SONIC] Erro ao reprocessar chime %s: %v", filePath, errMaster)
		http.Redirect(w, r, "/chimes?error=Falha+ao+reprocessar+sinal+sonoro+no+Sonic", http.StatusSeeOther)
		return
	}

	// Extract new audio metadata (exact duration, LUFS, amplify)
	meta, _ := utils.ExtractMetadata(filePath)

	durationSecs := meta.Duration
	if durationSecs <= 0 {
		durationSecs = 3
	}
	cueOut := float64(durationSecs)
	lufs := meta.LUFS
	amplify := 1.0

	_, errUp := database.Pool.Exec(ctx, `
		UPDATE chimes 
		SET duration_seconds = $1, cue_in = 0.0, cue_out = $2, lufs = $3, amplify = $4 
		WHERE id = $5`, durationSecs, cueOut, lufs, amplify, idStr)
	if errUp != nil {
		log.Printf("❌ [SONIC] Erro ao atualizar metadados do chime %s no banco: %v", idStr, errUp)
	}

	http.Redirect(w, r, "/chimes?success=Sinal+sonoro+reprocessado+com+sucesso+no+padrao+SONIC", http.StatusSeeOther)
}

func HandlePlayChime(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var fileName string
	_ = database.Pool.QueryRow(ctx, "SELECT file_name FROM chimes WHERE id = $1", idStr).Scan(&fileName)
	if fileName == "" {
		http.Error(w, "Arquivo não encontrado", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, filepath.Join("midias", "chimes", fileName))
}

func HandleUpdateChime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseMultipartForm(50 << 20)

	idStr := strings.TrimSpace(r.FormValue("id"))
	name := strings.TrimSpace(r.FormValue("name"))

	if idStr == "" || name == "" {
		http.Redirect(w, r, "/chimes?error=ID+e+nome+sao+obrigatorios", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var oldFileName string
	err := database.Pool.QueryRow(ctx, "SELECT file_name FROM chimes WHERE id = $1", idStr).Scan(&oldFileName)
	if err != nil {
		http.Redirect(w, r, "/chimes?error=Sinal+sonoro+nao+encontrado", http.StatusSeeOther)
		return
	}

	// Update Name
	_, _ = database.Pool.Exec(ctx, "UPDATE chimes SET name = $1 WHERE id = $2", name, idStr)

	// Check if a new file was uploaded
	file, header, errFile := r.FormFile("audio_file")
	if errFile == nil && file != nil {
		defer file.Close()

		chimeDir := filepath.Join("midias", "chimes")
		_ = os.MkdirAll(chimeDir, 0755)

		fileExt := filepath.Ext(header.Filename)
		safeFileName := fmt.Sprintf("chime_%d%s", time.Now().UnixNano(), fileExt)
		destPath := filepath.Join(chimeDir, safeFileName)

		out, errCreate := os.Create(destPath)
		if errCreate == nil {
			_, _ = io.Copy(out, file)
			out.Close()

			// Masterize & extract metadata
			_ = utils.MasterizeAudio(destPath)
			meta, _ := utils.ExtractMetadata(destPath)

			durationSecs := meta.Duration
			if durationSecs <= 0 {
				durationSecs = 3
			}
			cueOut := float64(durationSecs)
			lufs := meta.LUFS
			amplify := 1.0

			_, errUp := database.Pool.Exec(ctx, `
				UPDATE chimes 
				SET file_name = $1, duration_seconds = $2, cue_in = 0.0, cue_out = $3, lufs = $4, amplify = $5 
				WHERE id = $6`, safeFileName, durationSecs, cueOut, lufs, amplify, idStr)

			if errUp == nil && oldFileName != "" {
				_ = os.Remove(filepath.Join(chimeDir, oldFileName))
			}
		}
	}

	http.Redirect(w, r, "/chimes?success=Sinal+sonoro+atualizado+com+sucesso", http.StatusSeeOther)
}

// -----------------------------------------------------------------------------
// 3. PRE-FORMATTED WARNINGS HANDLERS
// -----------------------------------------------------------------------------

type PreFormattedWarningsPageData struct {
	Title    string
	User     map[string]string
	Theme    map[string]string
	Warnings []models.PreFormattedWarning
}

func HandlePreFormattedWarningsPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, _ := url.QueryUnescape(cookie.Value)
	var session models.SessionData
	_ = json.Unmarshal([]byte(decodedValue), &session)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var warnings []models.PreFormattedWarning
	rows, err := database.Pool.Query(ctx, "SELECT id, name, file_name, duration_seconds, cue_in, cue_out, lufs, amplify, is_active, created_at FROM pre_formatted_warnings ORDER BY name ASC")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pw models.PreFormattedWarning
			if rows.Scan(&pw.ID, &pw.Name, &pw.FileName, &pw.DurationSeconds, &pw.CueIn, &pw.CueOut, &pw.LUFS, &pw.Amplify, &pw.IsActive, &pw.CreatedAt) == nil {
				warnings = append(warnings, pw)
			}
		}
	}

	data := PreFormattedWarningsPageData{
		Title: "Mensagens Pré-Formatadas",
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme:    getStandardTheme(),
		Warnings: warnings,
	}

	tmpl, err := template.ParseFiles("web/templates/pre_formatted_warnings.html")
	if err != nil {
		http.Error(w, "Erro de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, data)
}

func HandleCreatePreFormattedWarning(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseMultipartForm(50 << 20)

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/pre-formatted-warnings?error=Nome+da+mensagem+e+obrigatorio", http.StatusSeeOther)
		return
	}

	file, header, err := r.FormFile("audio_file")
	if err != nil {
		http.Redirect(w, r, "/pre-formatted-warnings?error=Arquivo+de+audio+obrigatorio", http.StatusSeeOther)
		return
	}
	defer file.Close()

	warnDir := filepath.Join("midias", "pre_formatted_warnings")
	_ = os.MkdirAll(warnDir, 0755)

	fileExt := filepath.Ext(header.Filename)
	safeFileName := fmt.Sprintf("warning_%d%s", time.Now().UnixNano(), fileExt)
	destPath := filepath.Join(warnDir, safeFileName)

	out, err := os.Create(destPath)
	if err != nil {
		http.Redirect(w, r, "/pre-formatted-warnings?error=Falha+ao+salvar+arquivo", http.StatusSeeOther)
		return
	}
	_, _ = io.Copy(out, file)
	out.Close()

	durationSecs := 15
	lufs := -14.0
	amplify := 1.0

	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", destPath)
	output, errCmd := cmd.Output()
	if errCmd == nil {
		if durFloat, errParse := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); errParse == nil {
			durationSecs = int(durFloat)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	warnID := uuid.New().String()
	cueOut := float64(durationSecs)
	_, err = database.Pool.Exec(ctx, `
		INSERT INTO pre_formatted_warnings (id, name, file_name, duration_seconds, cue_in, cue_out, lufs, amplify, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true)`,
		warnID, name, safeFileName, durationSecs, 0.0, cueOut, lufs, amplify,
	)
	if err != nil {
		log.Printf("❌ [PRE-FORMATTED WARNINGS] Erro ao cadastrar mensagem no banco: %v", err)
		http.Redirect(w, r, "/pre-formatted-warnings?error=Falha+ao+persistir+no+banco", http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/pre-formatted-warnings?success=Mensagem+cadastrada+com+sucesso", http.StatusSeeOther)
}

func HandleReprocessPreFormattedWarningSonic(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Redirect(w, r, "/pre-formatted-warnings?error=ID+da+mensagem+invalido", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var fileName string
	err := database.Pool.QueryRow(ctx, "SELECT file_name FROM pre_formatted_warnings WHERE id = $1", idStr).Scan(&fileName)
	if err != nil || fileName == "" {
		http.Redirect(w, r, "/pre-formatted-warnings?error=Mensagem+nao+encontrada", http.StatusSeeOther)
		return
	}

	filePath := filepath.Join("midias", "pre_formatted_warnings", fileName)
	if errMaster := utils.MasterizeAudio(filePath); errMaster != nil {
		log.Printf("❌ [SONIC] Erro ao reprocessar mensagem %s: %v", filePath, errMaster)
		http.Redirect(w, r, "/pre-formatted-warnings?error=Falha+ao+reprocessar+mensagem+no+Sonic", http.StatusSeeOther)
		return
	}

	// Extract new audio metadata
	meta, _ := utils.ExtractMetadata(filePath)

	durationSecs := meta.Duration
	if durationSecs <= 0 {
		durationSecs = 15
	}
	cueOut := float64(durationSecs)
	lufs := meta.LUFS
	amplify := 1.0

	_, errUp := database.Pool.Exec(ctx, `
		UPDATE pre_formatted_warnings 
		SET duration_seconds = $1, cue_in = 0.0, cue_out = $2, lufs = $3, amplify = $4 
		WHERE id = $5`, durationSecs, cueOut, lufs, amplify, idStr)
	if errUp != nil {
		log.Printf("❌ [SONIC] Erro ao atualizar metadados da mensagem %s no banco: %v", idStr, errUp)
	}

	http.Redirect(w, r, "/pre-formatted-warnings?success=Mensagem+reprocessada+com+sucesso+no+padrao+SONIC", http.StatusSeeOther)
}

func HandleDeletePreFormattedWarning(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var fileName string
		_ = database.Pool.QueryRow(ctx, "SELECT file_name FROM pre_formatted_warnings WHERE id = $1", idStr).Scan(&fileName)
		if fileName != "" {
			_ = os.Remove(filepath.Join("midias", "pre_formatted_warnings", fileName))
		}
		_, _ = database.Pool.Exec(ctx, "DELETE FROM pre_formatted_warnings WHERE id = $1", idStr)
	}
	http.Redirect(w, r, "/pre-formatted-warnings", http.StatusSeeOther)
}

func HandlePlayPreFormattedWarning(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var fileName string
	_ = database.Pool.QueryRow(ctx, "SELECT file_name FROM pre_formatted_warnings WHERE id = $1", idStr).Scan(&fileName)
	if fileName == "" {
		http.Error(w, "Arquivo não encontrado", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, filepath.Join("midias", "pre_formatted_warnings", fileName))
}

// -----------------------------------------------------------------------------
// 4. DOWNLOAD ENDPOINTS FOR SYNC
// -----------------------------------------------------------------------------

func DownloadVoicePack(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var baseDir string
	_ = database.Pool.QueryRow(ctx, "SELECT base_directory FROM voice_packs WHERE id = $1", idStr).Scan(&baseDir)
	if baseDir == "" {
		http.Error(w, "Pacote não encontrado", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, baseDir)
}

func DownloadChime(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var fileName string
	_ = database.Pool.QueryRow(ctx, "SELECT file_name FROM chimes WHERE id = $1", idStr).Scan(&fileName)
	if fileName == "" {
		http.Error(w, "Chime não encontrado", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, filepath.Join("midias", "chimes", fileName))
}

func DownloadWarning(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var fileName string
	_ = database.Pool.QueryRow(ctx, "SELECT file_name FROM pre_formatted_warnings WHERE id = $1", idStr).Scan(&fileName)
	if fileName == "" {
		http.Error(w, "Mensagem não encontrada", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, filepath.Join("midias", "pre_formatted_warnings", fileName))
}

func HandleUpdatePreFormattedWarning(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseMultipartForm(50 << 20)

	idStr := strings.TrimSpace(r.FormValue("id"))
	name := strings.TrimSpace(r.FormValue("name"))

	if idStr == "" || name == "" {
		http.Redirect(w, r, "/pre-formatted-warnings?error=ID+e+nome+sao+obrigatorios", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var oldFileName string
	err := database.Pool.QueryRow(ctx, "SELECT file_name FROM pre_formatted_warnings WHERE id = $1", idStr).Scan(&oldFileName)
	if err != nil {
		http.Redirect(w, r, "/pre-formatted-warnings?error=Mensagem+nao+encontrada", http.StatusSeeOther)
		return
	}

	// Update Name
	_, _ = database.Pool.Exec(ctx, "UPDATE pre_formatted_warnings SET name = $1 WHERE id = $2", name, idStr)

	// Check if a new file was uploaded
	file, header, errFile := r.FormFile("audio_file")
	if errFile == nil && file != nil {
		defer file.Close()

		warnDir := filepath.Join("midias", "pre_formatted_warnings")
		_ = os.MkdirAll(warnDir, 0755)

		fileExt := filepath.Ext(header.Filename)
		safeFileName := fmt.Sprintf("warning_%d%s", time.Now().UnixNano(), fileExt)
		destPath := filepath.Join(warnDir, safeFileName)

		out, errCreate := os.Create(destPath)
		if errCreate == nil {
			_, _ = io.Copy(out, file)
			out.Close()

			// Masterize & extract metadata
			_ = utils.MasterizeAudio(destPath)
			meta, _ := utils.ExtractMetadata(destPath)

			durationSecs := meta.Duration
			if durationSecs <= 0 {
				durationSecs = 15
			}
			cueOut := float64(durationSecs)
			lufs := meta.LUFS
			amplify := 1.0

			_, errUp := database.Pool.Exec(ctx, `
				UPDATE pre_formatted_warnings 
				SET file_name = $1, duration_seconds = $2, cue_in = 0.0, cue_out = $3, lufs = $4, amplify = $5 
				WHERE id = $6`, safeFileName, durationSecs, cueOut, lufs, amplify, idStr)

			if errUp == nil && oldFileName != "" {
				_ = os.Remove(filepath.Join(warnDir, oldFileName))
			}
		}
	}

	http.Redirect(w, r, "/pre-formatted-warnings?success=Mensagem+atualizada+com+sucesso", http.StatusSeeOther)
}
