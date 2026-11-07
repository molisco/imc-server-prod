package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"imc-server/internal/models"
	"imc-server/internal/repository"

	"github.com/google/uuid"
)

var videoStyleRepo = repository.NewVideoStyleRepository()

type VideoStylePageData struct {
	Title          string
	LoggedUser     string
	ErrorMessage   string
	SuccessMessage string
	Form           map[string]string
	Styles         []models.VideoStyle
	User           map[string]string
	Theme          map[string]string
}

func HandleVideoStylesPage(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	stylesList, err := videoStyleRepo.GetAll(context.Background())
	if err != nil {
		stylesList = []models.VideoStyle{}
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "#161925",
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}

	data := VideoStylePageData{
		Title:      "Gerenciamento de Estilos de Vídeo",
		LoggedUser: session.UserName,
		Styles:     stylesList,
		Form:       make(map[string]string),
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme: theme,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/video_styles.html"))
	tmpl.Execute(w, data)
}

func HandleCreateVideoStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/dashboard/video-styles", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	name := r.FormValue("name")
	description := r.FormValue("description")
	baseDirectory := r.FormValue("base_directory")

	formData := map[string]string{
		"name":           name,
		"description":    description,
		"base_directory": baseDirectory,
	}

	nameClean := strings.TrimSpace(name)

	if nameClean == "" {
		renderVideoStylesError(w, session, "❌ ATENÇÃO: O campo Nome do Estilo é obrigatório!", formData)
		return
	}

	style := models.VideoStyle{
		ID:            uuid.New(),
		Name:          nameClean,
		Description:   strings.TrimSpace(description),
		BaseDirectory: strings.TrimSpace(baseDirectory),
	}

	err = videoStyleRepo.Create(context.Background(), style)
	if err != nil {
		renderVideoStylesError(w, session, "❌ Erro ao salvar o Estilo de Vídeo: "+err.Error(), formData)
		return
	}

	http.Redirect(w, r, "/video-styles", http.StatusSeeOther)
}

func HandleUpdateVideoStyle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/video-styles", http.StatusSeeOther)
		return
	}

	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.ParseForm()
	idStr := r.FormValue("id")
	styleID, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/video-styles", http.StatusSeeOther)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	baseDirectory := r.FormValue("base_directory")

	formData := map[string]string{
		"id":             idStr,
		"name":           name,
		"description":    description,
		"base_directory": baseDirectory,
	}

	nameClean := strings.TrimSpace(name)

	if nameClean == "" {
		renderVideoStylesError(w, session, "❌ ATENÇÃO: O campo Nome do Estilo é obrigatório!", formData)
		return
	}

	style := models.VideoStyle{
		ID:            styleID,
		Name:          nameClean,
		Description:   strings.TrimSpace(description),
		BaseDirectory: strings.TrimSpace(baseDirectory),
	}

	err = videoStyleRepo.Update(context.Background(), style)
	if err != nil {
		renderVideoStylesError(w, session, "❌ Erro ao atualizar o Estilo de Vídeo: "+err.Error(), formData)
		return
	}

	http.Redirect(w, r, "/video-styles", http.StatusSeeOther)
}

func HandleDeleteVideoStyle(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("imc_session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	decodedValue, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	var session models.SessionData
	if err := json.Unmarshal([]byte(decodedValue), &session); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := r.URL.Query().Get("id")
	styleID, err := uuid.Parse(idStr)
	if err == nil {
		_ = videoStyleRepo.Delete(context.Background(), styleID)
	}

	http.Redirect(w, r, "/video-styles", http.StatusSeeOther)
}

func renderVideoStylesError(w http.ResponseWriter, session models.SessionData, errMsg string, formData map[string]string) {
	stylesList, _ := videoStyleRepo.GetAll(context.Background())

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7",
		"SecondaryColor":       "#161925",
		"BackgroundColor":      "#0f111a",
		"FontColor":            "#ffffff",
		"FieldBackgroundColor": "#161925",
	}

	data := VideoStylePageData{
		Title:        "Gerenciamento de Estilos de Vídeo",
		LoggedUser:   session.UserName,
		ErrorMessage: errMsg,
		Form:         formData,
		Styles:       stylesList,
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme: theme,
	}

	w.WriteHeader(http.StatusBadRequest)
	tmpl := template.Must(template.ParseFiles("web/templates/video_styles.html"))
	tmpl.Execute(w, data)
}
