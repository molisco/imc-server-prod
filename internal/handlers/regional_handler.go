package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"imc-server/internal/models"
	"imc-server/internal/repository"

	"github.com/google/uuid"
)

var regionalRepo = repository.NewRegionalRepository()

// RegionalPageData representa o payload de dados enviado ao template HTML de regionais
type RegionalPageData struct {
	Title          string
	LoggedUser     string
	ErrorMessage   string
	SuccessMessage string
	Form           map[string]string // Preserva dados digitados em caso de erro
	Tenants        []models.Tenant
	Regionals      []models.Regional
	User           map[string]string
	Theme          map[string]string
}

// HandleRegionalsPage renderiza a tela de gerenciamento de regionais
func HandleRegionalsPage(w http.ResponseWriter, r *http.Request) {
	// 1. Validação de Sessão (Padrão Ouro)
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

	// 2. Busca lista de Clientes (Tenants) ativos para o dropdown do formulário
	tenantsList, err := tenantRepo.GetAll(context.Background())
	if err != nil {
		tenantsList = []models.Tenant{}
	}
	if session.TenantID != nil {
		var filtered []models.Tenant
		for _, t := range tenantsList {
			if t.ID == *session.TenantID {
				filtered = append(filtered, t)
				break
			}
		}
		tenantsList = filtered
	}

	// 3. Busca lista de regionais cadastradas no PostgreSQL
	regionalsList, err := regionalRepo.GetAll(context.Background(), session.TenantID)
	if err != nil {
		regionalsList = []models.Regional{}
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7", // Azul IMC
		"SecondaryColor":       "#161925", // Cinza Tech
		"BackgroundColor":      "#0f111a", // Preto profundo
		"FontColor":            "#ffffff", // Branco
		"FieldBackgroundColor": "#161925", // Cinza Tech
	}

	data := RegionalPageData{
		Title:      "Gerenciamento de Regionais",
		LoggedUser: session.UserName,
		Tenants:    tenantsList,
		Regionals:  regionalsList,
		Form:       make(map[string]string),
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme: theme,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/regionals.html"))
	tmpl.Execute(w, data)
}

// HandleCreateRegional valida os campos e cria uma nova regional no banco de dados
func HandleCreateRegional(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/regionals", http.StatusSeeOther)
		return
	}

	// 1. Validação de Sessão
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

	// 2. Coleta dos dados do formulário
	r.ParseForm()
	tenantIDStr := r.FormValue("tenant_id")
	name := r.FormValue("name")
	coordinatorName := r.FormValue("coordinator_name")
	phone := r.FormValue("phone")

	formData := map[string]string{
		"tenant_id":        tenantIDStr,
		"name":             name,
		"coordinator_name": coordinatorName,
		"phone":            phone,
	}

	// 3. Validação dos campos obrigatórios
	tenantID, parseErr := uuid.Parse(tenantIDStr)
	nameClean := strings.TrimSpace(name)

	if parseErr != nil || nameClean == "" {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7", // Azul IMC
			"SecondaryColor":       "#161925", // Cinza Tech
			"BackgroundColor":      "#0f111a", // Preto profundo
			"FontColor":            "#ffffff", // Branco
			"FieldBackgroundColor": "#161925", // Cinza Tech
		}

		data := RegionalPageData{
			Title:        "Gerenciamento de Regionais",
			LoggedUser:   session.UserName,
			ErrorMessage: "❌ ATENÇÃO: Os campos Matriz e Nome da Regional são obrigatórios!",
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			User: map[string]string{
				"Name": session.UserName,
				"Role": "Administrador Mestre",
			},
			Theme:        theme,
		}

		w.WriteHeader(http.StatusBadRequest)
		tmpl := template.Must(template.ParseFiles("web/templates/regionals.html"))
		tmpl.Execute(w, data)
		return
	}

	// 4. Monta a struct de persistência
	var coord *string = nil
	if strings.TrimSpace(coordinatorName) != "" {
		coordStr := strings.TrimSpace(coordinatorName)
		coord = &coordStr
	}
	var ph *string = nil
	if strings.TrimSpace(phone) != "" {
		phStr := strings.TrimSpace(phone)
		ph = &phStr
	}

	reg := models.Regional{
		ID:              uuid.New(),
		TenantID:        tenantID,
		Name:            nameClean,
		CoordinatorName: coord,
		Phone:           ph,
		IsActive:        true,
	}

	// 5. Salva no banco
	err = regionalRepo.Create(context.Background(), reg)
	if err != nil {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7", // Azul IMC
			"SecondaryColor":       "#161925", // Cinza Tech
			"BackgroundColor":      "#0f111a", // Preto profundo
			"FontColor":            "#ffffff", // Branco
			"FieldBackgroundColor": "#161925", // Cinza Tech
		}

		data := RegionalPageData{
			Title:        "Gerenciamento de Regionais",
			LoggedUser:   session.UserName,
			ErrorMessage: fmt.Sprintf("❌ Erro ao salvar a Regional no PostgreSQL: %v", err),
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			User: map[string]string{
				"Name": session.UserName,
				"Role": "Administrador Mestre",
			},
			Theme:        theme,
		}

		w.WriteHeader(http.StatusInternalServerError)
		tmpl := template.Must(template.ParseFiles("web/templates/regionals.html"))
		tmpl.Execute(w, data)
		return
	}

	// Redireciona de volta para listar a nova regional
	http.Redirect(w, r, "/regionals", http.StatusSeeOther)
}

// HandleUpdateRegional atualiza os dados de uma regional existente
func HandleUpdateRegional(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/regionals", http.StatusSeeOther)
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
	regionalID, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/regionals", http.StatusSeeOther)
		return
	}

	tenantIDStr := r.FormValue("tenant_id")
	name := r.FormValue("name")
	coordinatorName := r.FormValue("coordinator_name")
	phone := r.FormValue("phone")

	formData := map[string]string{
		"id":               idStr,
		"tenant_id":        tenantIDStr,
		"name":             name,
		"coordinator_name": coordinatorName,
		"phone":            phone,
	}

	tenantID, parseErr := uuid.Parse(tenantIDStr)
	nameClean := strings.TrimSpace(name)

	if parseErr != nil || nameClean == "" {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)
		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "#161925",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}
		data := RegionalPageData{
			Title:        "Gerenciamento de Regionais",
			LoggedUser:   session.UserName,
			ErrorMessage: "❌ ATENÇÃO: Os campos Matriz e Nome da Regional são obrigatórios!",
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			User:         map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:        theme,
		}
		w.WriteHeader(http.StatusBadRequest)
		tmpl := template.Must(template.ParseFiles("web/templates/regionals.html"))
		tmpl.Execute(w, data)
		return
	}

	var coord *string = nil
	if strings.TrimSpace(coordinatorName) != "" {
		cStr := strings.TrimSpace(coordinatorName)
		coord = &cStr
	}
	var ph *string = nil
	if strings.TrimSpace(phone) != "" {
		phStr := strings.TrimSpace(phone)
		ph = &phStr
	}

	reg := models.Regional{
		ID:              regionalID,
		TenantID:        tenantID,
		Name:            nameClean,
		CoordinatorName: coord,
		Phone:           ph,
		IsActive:        true,
	}

	err = regionalRepo.Update(context.Background(), reg)
	if err != nil {
		tenantsList, _ := tenantRepo.GetAll(context.Background())
		if session.TenantID != nil {
			var filtered []models.Tenant
			for _, t := range tenantsList {
				if t.ID == *session.TenantID {
					filtered = append(filtered, t)
					break
				}
			}
			tenantsList = filtered
		}
		regionalsList, _ := regionalRepo.GetAll(context.Background(), session.TenantID)
		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "#161925",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}
		data := RegionalPageData{
			Title:        "Gerenciamento de Regionais",
			LoggedUser:   session.UserName,
			ErrorMessage: "❌ Erro ao atualizar no banco: " + err.Error(),
			Form:         formData,
			Tenants:      tenantsList,
			Regionals:    regionalsList,
			User:         map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:        theme,
		}
		w.WriteHeader(http.StatusInternalServerError)
		tmpl := template.Must(template.ParseFiles("web/templates/regionals.html"))
		tmpl.Execute(w, data)
		return
	}

	http.Redirect(w, r, "/regionals", http.StatusSeeOther)
}

// HandleDeleteRegional exclui permanentemente uma regional
func HandleDeleteRegional(w http.ResponseWriter, r *http.Request) {
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
	regionalID, err := uuid.Parse(idStr)
	if err == nil {
		_ = regionalRepo.Delete(context.Background(), regionalID)
	}

	http.Redirect(w, r, "/regionals", http.StatusSeeOther)
}
