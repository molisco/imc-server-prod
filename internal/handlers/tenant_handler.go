package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"imc-server/internal/database"
	"imc-server/internal/models"
	"imc-server/internal/repository"
	"imc-server/internal/utils"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Instancia o repositório de dados de forma desacoplada (Design Pattern Ouro)
var tenantRepo = repository.NewTenantRepository()

// PageData estrutura a mala de dados unificada que o Go envia para os templates HTML
type TenantPageData struct {
	Title           string
	LoggedUser      string
	ErrorMessage    string
	SuccessMessage  string
	Form            map[string]string // Guarda os dados digitados temporariamente para não perder em caso de erro
	FormOpContacts  []models.ContactDetails
	FormFinContacts []models.ContactDetails
	Tenants         []models.Tenant
	Grids           []models.WeeklyMasterGrid
	User            map[string]string
	Theme           map[string]string
}

// HandleTenantsPage renderiza a tela principal carregando a lista de empresas do banco
func HandleTenantsPage(w http.ResponseWriter, r *http.Request) {
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

	// Busca a lista atualizada de matrizes no Postgres
	list, err := tenantRepo.GetAll(context.Background())
	if err != nil {
		list = []models.Tenant{}
	}

	gridsList, err := gridRepo.GetAll(context.Background(), nil)
	if err != nil {
		gridsList = []models.WeeklyMasterGrid{}
	}

	theme := map[string]string{
		"LogoURL":              "/static/img/logo-imc.png",
		"PrimaryColor":         "#4364f7", // Azul IMC
		"SecondaryColor":       "#161925", // Cinza Tech
		"BackgroundColor":      "#0f111a", // Preto profundo
		"FontColor":            "#ffffff", // Branco
		"FieldBackgroundColor": "#161925", // Cinza Tech
	}

	data := TenantPageData{
		Title:      "Gerenciamento de Matrizes",
		LoggedUser: session.UserName,
		Tenants:    list,
		Grids:      gridsList,
		Form:       make(map[string]string), // Inicia vazio
		User: map[string]string{
			"Name": session.UserName,
			"Role": "Administrador Mestre",
		},
		Theme: theme,
	}

	tmpl := template.Must(template.ParseFiles("web/templates/tenants.html"))
	tmpl.Execute(w, data)
}

// HandleCreateTenant processa o envio do formulário, valida dados e trata o upload opcional
func HandleCreateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tenants", http.StatusSeeOther)
		return
	}

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

	// Limita o tamanho máximo de upload (Segurança Ouro contra arquivos gigantes)
	r.ParseMultipartForm(5 << 20) // Máximo 5MB

	// 2. CAPTURA DOS DADOS DO FORMULÁRIO
	tradeName := r.FormValue("trade_name")
	legalName := r.FormValue("legal_name")
	cnpj := r.FormValue("document_number")
	zipCode := r.FormValue("zip_code")
	street := r.FormValue("street")
	number := r.FormValue("number")
	district := r.FormValue("district")
	city := r.FormValue("city")
	state := r.FormValue("state")
	country := r.FormValue("country")
	complement := r.FormValue("complement")
	primaryColor := r.FormValue("primary_color")
	backgroundColor := r.FormValue("background_color")
	fontColor := r.FormValue("font_color")
	fieldBackgroundColor := r.FormValue("field_background_color")

	if primaryColor == "" {
		primaryColor = "#4364f7" // Fallback Azul IMC
	}
	if backgroundColor == "" {
		backgroundColor = "#0f111a" // Fallback Background
	}
	if fontColor == "" {
		fontColor = "#ffffff" // Fallback Font
	}
	if fieldBackgroundColor == "" {
		fieldBackgroundColor = "#161925" // Fallback Field background
	}

	tempLogoURL := r.FormValue("temp_logo_url")

	// Guarda o estado atual do formulário para o caso de precisarmos devolver por erro (Item 1)
	formData := map[string]string{
		"trade_name":             tradeName,
		"legal_name":             legalName,
		"document_number":        cnpj,
		"zip_code":               zipCode,
		"street":                 street,
		"number":                 number,
		"district":               district,
		"city":                   city,
		"state":                  state,
		"country":                country,
		"complement":             complement,
		"primary_color":          primaryColor,
		"background_color":       backgroundColor,
		"font_color":             fontColor,
		"field_background_color": fieldBackgroundColor,
		"temp_logo_url":          tempLogoURL,
		"default_audio_grid_id":  r.FormValue("default_audio_grid_id"),
		"default_video_grid_id":  r.FormValue("default_video_grid_id"),
	}

	// 3. CAPTURA ARRAYS DINÂMICOS DE CONTATOS (Item 2)
	var opNames, opEmails, opPhones []string
	if r.MultipartForm != nil {
		opNames = r.MultipartForm.Value["op_name[]"]
		opEmails = r.MultipartForm.Value["op_email[]"]
		opPhones = r.MultipartForm.Value["op_phone[]"]
	} else {
		opNames = r.Form["op_name[]"]
		opEmails = r.Form["op_email[]"]
		opPhones = r.Form["op_phone[]"]
	}

	var opContacts []models.ContactDetails
	for i := 0; i < len(opNames); i++ {
		name := ""
		if i < len(opNames) { name = strings.TrimSpace(opNames[i]) }
		email := ""
		if i < len(opEmails) { email = strings.TrimSpace(opEmails[i]) }
		phone := ""
		if i < len(opPhones) { phone = strings.TrimSpace(opPhones[i]) }

		if name != "" || email != "" || phone != "" {
			opContacts = append(opContacts, models.ContactDetails{
				Name:  name,
				Email: email,
				Phone: phone,
			})
		}
	}

	var finNames, finEmails, finPhones []string
	if r.MultipartForm != nil {
		finNames = r.MultipartForm.Value["fin_name[]"]
		finEmails = r.MultipartForm.Value["fin_email[]"]
		finPhones = r.MultipartForm.Value["fin_phone[]"]
	} else {
		finNames = r.Form["fin_name[]"]
		finEmails = r.Form["fin_email[]"]
		finPhones = r.Form["fin_phone[]"]
	}

	var finContacts []models.ContactDetails
	for i := 0; i < len(finNames); i++ {
		name := ""
		if i < len(finNames) { name = strings.TrimSpace(finNames[i]) }
		email := ""
		if i < len(finEmails) { email = strings.TrimSpace(finEmails[i]) }
		phone := ""
		if i < len(finPhones) { phone = strings.TrimSpace(finPhones[i]) }

		if name != "" || email != "" || phone != "" {
			finContacts = append(finContacts, models.ContactDetails{
				Name:  name,
				Email: email,
				Phone: phone,
			})
		}
	}

	// 4. PROCESSAMENTO DO LOGOTIPO OPCIONAL (Item 4 & 7)
	var finalLogoURL *string = nil
	file, header, err := r.FormFile("logo_file")

	if err == nil {
		defer file.Close()

		// Cria um nome único usando UUID para evitar que clientes sobreponham arquivos com mesmo nome
		ext := filepath.Ext(header.Filename)
		newFileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
		uploadPath := filepath.Join("web", "static", "uploads", newFileName)

		// Cria fisicamente o arquivo no disco do Linux
		out, err := os.Create(uploadPath)
		if err == nil {
			defer out.Close()
			io.Copy(out, file)
			urlPath := "/static/uploads/" + newFileName
			finalLogoURL = &urlPath
		}
	}

	// Se nenhum novo arquivo de logotipo foi carregado nesta requisição,
	// mas existe um logotipo temporário anteriormente carregado, preserva-o!
	if finalLogoURL == nil && tempLogoURL != "" {
		finalLogoURL = &tempLogoURL
	}

	// Atualiza o logotipo temporário na mala de formulário caso tenhamos processado um novo upload
	if finalLogoURL != nil {
		formData["temp_logo_url"] = *finalLogoURL
	}

	// 3. VALIDAÇÃO MATEMÁTICA DO CNPJ (Item 1)
	// Chamamos o validador do arquivo validators.go
	if !IsValidCNPJ(cnpj) {
		// Busca a lista para não quebrar a tabela da tela ao recarregar
		list, _ := tenantRepo.GetAll(context.Background())
		gridsList, _ := gridRepo.GetAll(context.Background(), nil)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7", // Azul IMC
			"SecondaryColor":       "#161925", // Cinza Tech
			"BackgroundColor":      "#0f111a", // Preto profundo
			"FontColor":            "#ffffff", // Branco
			"FieldBackgroundColor": "#161925", // Cinza Tech
		}

		data := TenantPageData{
			Title:           "Gerenciamento de Matrizes",
			LoggedUser:      session.UserName, // Pega o usuário da sessão real
			ErrorMessage:    "❌ ATENÇÃO: O CNPJ digitado é matematicamente INVÁLIDO! Verifique os dígitos.",
			Form:            formData, // Enviamos os dados de volta para o HTML preencher os inputs!
			FormOpContacts:  opContacts,
			FormFinContacts: finContacts,
			Tenants:         list,
			Grids:           gridsList,
			User: map[string]string{
				"Name": session.UserName,
				"Role": "Administrador Mestre",
			},
			Theme:           theme,
		}

		w.WriteHeader(http.StatusBadRequest)
		tmpl := template.Must(template.ParseFiles("web/templates/tenants.html"))
		tmpl.Execute(w, data)
		return
	}

	// 5. MONTAGEM DA STRUCT FINAL E GRAVAÇÃO
	newID := uuid.New()
	compParam := &complement
	if complement == "" {
		compParam = nil
	}

	defaultAudioGridIDStr := r.FormValue("default_audio_grid_id")
	defaultVideoGridIDStr := r.FormValue("default_video_grid_id")
	var defaultAudioGridID *uuid.UUID = nil
	if defaultAudioGridIDStr != "" {
		if parsed, err := uuid.Parse(defaultAudioGridIDStr); err == nil {
			defaultAudioGridID = &parsed
		}
	}
	var defaultVideoGridID *uuid.UUID = nil
	if defaultVideoGridIDStr != "" {
		if parsed, err := uuid.Parse(defaultVideoGridIDStr); err == nil {
			defaultVideoGridID = &parsed
		}
	}

	tenant := models.Tenant{
		ID:                  newID,
		TradeName:           tradeName,
		LegalName:           legalName,
		DocumentNumber:      cnpj,
		OperationalContacts: opContacts,
		FinancialContacts:   finContacts,
		ZipCode:             zipCode,
		Street:              street,
		Number:              number,
		District:            district,
		City:                city,
		State:               state,
		Country:             country,
		Complement:          compParam,
		LogoURL:             finalLogoURL,
		PrimaryColor:        primaryColor,
		BackgroundColor:      backgroundColor,
		FontColor:            fontColor,
		FieldBackgroundColor: fieldBackgroundColor,
		DefaultAudioGridID:  defaultAudioGridID,
		DefaultVideoGridID:  defaultVideoGridID,
		IsActive:            true,
	}

	// Salva no banco de dados através do repositório
	err = tenantRepo.Create(context.Background(), tenant)
	if err != nil {
		fmt.Println("Erro crítico ao salvar no banco:", err)

		// Busca a lista para não quebrar a tabela da tela ao recarregar
		list, _ := tenantRepo.GetAll(context.Background())
		gridsList, _ := gridRepo.GetAll(context.Background(), nil)

		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7", // Azul IMC
			"SecondaryColor":       "#161925", // Cinza Tech
			"BackgroundColor":      "#0f111a", // Preto profundo
			"FontColor":            "#ffffff", // Branco
			"FieldBackgroundColor": "#161925", // Cinza Tech
		}

		data := TenantPageData{
			Title:           "Gerenciamento de Matrizes",
			LoggedUser:      session.UserName,
			ErrorMessage:    "❌ Erro ao salvar no banco de dados: " + err.Error(),
			Form:            formData, // Enviamos todos os dados digitados de volta para evitar perda!
			FormOpContacts:  opContacts,
			FormFinContacts: finContacts,
			Tenants:         list,
			Grids:           gridsList,
			User: map[string]string{
				"Name": session.UserName,
				"Role": "Administrador Mestre",
			},
			Theme:           theme,
		}

		w.WriteHeader(http.StatusInternalServerError)
		tmpl := template.Must(template.ParseFiles("web/templates/tenants.html"))
		tmpl.Execute(w, data)
		return
	}

	// Redireciona de volta com a lista limpa e atualizada se der tudo certo
	http.Redirect(w, r, "/tenants", http.StatusSeeOther)

}


// HandleUpdateTenant atualiza os dados de uma matriz existente
func HandleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/tenants", http.StatusSeeOther)
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

	r.ParseMultipartForm(5 << 20)

	idStr := r.FormValue("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		http.Redirect(w, r, "/tenants", http.StatusSeeOther)
		return
	}

	tradeName := r.FormValue("trade_name")
	legalName := r.FormValue("legal_name")
	cnpj := r.FormValue("document_number")
	zipCode := r.FormValue("zip_code")
	street := r.FormValue("street")
	number := r.FormValue("number")
	district := r.FormValue("district")
	city := r.FormValue("city")
	state := r.FormValue("state")
	country := r.FormValue("country")
	complement := r.FormValue("complement")
	primaryColor := r.FormValue("primary_color")
	backgroundColor := r.FormValue("background_color")
	fontColor := r.FormValue("font_color")
	fieldBackgroundColor := r.FormValue("field_background_color")

	if primaryColor == "" { primaryColor = "#4364f7" }
	if backgroundColor == "" { backgroundColor = "#0f111a" }
	if fontColor == "" { fontColor = "#ffffff" }
	if fieldBackgroundColor == "" { fieldBackgroundColor = "#161925" }

	tempLogoURL := r.FormValue("temp_logo_url")

	formData := map[string]string{
		"id":                     idStr,
		"trade_name":             tradeName,
		"legal_name":             legalName,
		"document_number":        cnpj,
		"zip_code":               zipCode,
		"street":                 street,
		"number":                 number,
		"district":               district,
		"city":                   city,
		"state":                  state,
		"country":                country,
		"complement":             complement,
		"primary_color":          primaryColor,
		"background_color":       backgroundColor,
		"font_color":             fontColor,
		"field_background_color": fieldBackgroundColor,
		"temp_logo_url":          tempLogoURL,
		"default_audio_grid_id":  r.FormValue("default_audio_grid_id"),
		"default_video_grid_id":  r.FormValue("default_video_grid_id"),
	}

	// Parse contatos
	var opNames, opEmails, opPhones []string
	if r.MultipartForm != nil {
		opNames = r.MultipartForm.Value["op_name[]"]
		opEmails = r.MultipartForm.Value["op_email[]"]
		opPhones = r.MultipartForm.Value["op_phone[]"]
	} else {
		opNames = r.Form["op_name[]"]
		opEmails = r.Form["op_email[]"]
		opPhones = r.Form["op_phone[]"]
	}

	var opContacts []models.ContactDetails
	for i := 0; i < len(opNames); i++ {
		name := ""
		if i < len(opNames) { name = strings.TrimSpace(opNames[i]) }
		email := ""
		if i < len(opEmails) { email = strings.TrimSpace(opEmails[i]) }
		phone := ""
		if i < len(opPhones) { phone = strings.TrimSpace(opPhones[i]) }

		if name != "" || email != "" || phone != "" {
			opContacts = append(opContacts, models.ContactDetails{
				Name:  name,
				Email: email,
				Phone: phone,
			})
		}
	}

	var finNames, finEmails, finPhones []string
	if r.MultipartForm != nil {
		finNames = r.MultipartForm.Value["fin_name[]"]
		finEmails = r.MultipartForm.Value["fin_email[]"]
		finPhones = r.MultipartForm.Value["fin_phone[]"]
	} else {
		finNames = r.Form["fin_name[]"]
		finEmails = r.Form["fin_email[]"]
		finPhones = r.Form["fin_phone[]"]
	}

	var finContacts []models.ContactDetails
	for i := 0; i < len(finNames); i++ {
		name := ""
		if i < len(finNames) { name = strings.TrimSpace(finNames[i]) }
		email := ""
		if i < len(finEmails) { email = strings.TrimSpace(finEmails[i]) }
		phone := ""
		if i < len(finPhones) { phone = strings.TrimSpace(finPhones[i]) }

		if name != "" || email != "" || phone != "" {
			finContacts = append(finContacts, models.ContactDetails{
				Name:  name,
				Email: email,
				Phone: phone,
			})
		}
	}

	var finalLogoURL *string = nil
	file, header, err := r.FormFile("logo_file")
	if err == nil {
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		newFileName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
		uploadPath := filepath.Join("web", "static", "uploads", newFileName)
		out, err := os.Create(uploadPath)
		if err == nil {
			defer out.Close()
			io.Copy(out, file)
			urlPath := "/static/uploads/" + newFileName
			finalLogoURL = &urlPath
		}
	}

	// Se nenhum novo arquivo de logotipo foi carregado nesta requisição,
	// mas existe um logotipo temporário anteriormente carregado, preserva-o!
	if finalLogoURL == nil && tempLogoURL != "" {
		finalLogoURL = &tempLogoURL
	}

	// Atualiza o logotipo temporário na mala de formulário caso tenhamos processado um novo upload
	if finalLogoURL != nil {
		formData["temp_logo_url"] = *finalLogoURL
	}

	if !IsValidCNPJ(cnpj) {
		list, _ := tenantRepo.GetAll(context.Background())
		gridsList, _ := gridRepo.GetAll(context.Background(), nil)
		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "#161925",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}
		data := TenantPageData{
			Title:           "Gerenciamento de Matrizes",
			LoggedUser:      session.UserName,
			ErrorMessage:    "❌ ATENÇÃO: O CNPJ digitado é inválido!",
			Form:            formData,
			FormOpContacts:  opContacts,
			FormFinContacts: finContacts,
			Tenants:         list,
			Grids:           gridsList,
			User:            map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:           theme,
		}
		w.WriteHeader(http.StatusBadRequest)
		tmpl := template.Must(template.ParseFiles("web/templates/tenants.html"))
		tmpl.Execute(w, data)
		return
	}

	compParam := &complement
	if complement == "" { compParam = nil }

	defaultAudioGridIDStr := r.FormValue("default_audio_grid_id")
	defaultVideoGridIDStr := r.FormValue("default_video_grid_id")
	var defaultAudioGridID *uuid.UUID = nil
	if defaultAudioGridIDStr != "" {
		if parsed, err := uuid.Parse(defaultAudioGridIDStr); err == nil {
			defaultAudioGridID = &parsed
		}
	}
	var defaultVideoGridID *uuid.UUID = nil
	if defaultVideoGridIDStr != "" {
		if parsed, err := uuid.Parse(defaultVideoGridIDStr); err == nil {
			defaultVideoGridID = &parsed
		}
	}

	tenant := models.Tenant{
		ID:                  tenantID,
		TradeName:           tradeName,
		LegalName:           legalName,
		DocumentNumber:      cnpj,
		OperationalContacts: opContacts,
		FinancialContacts:   finContacts,
		ZipCode:             zipCode,
		Street:              street,
		Number:              number,
		District:            district,
		City:                city,
		State:               state,
		Country:             country,
		Complement:          compParam,
		LogoURL:             finalLogoURL,
		PrimaryColor:        primaryColor,
		BackgroundColor:      backgroundColor,
		FontColor:            fontColor,
		FieldBackgroundColor: fieldBackgroundColor,
		DefaultAudioGridID:  defaultAudioGridID,
		DefaultVideoGridID:  defaultVideoGridID,
		IsActive:            true,
	}

	err = tenantRepo.Update(context.Background(), tenant)
	if err != nil {
		list, _ := tenantRepo.GetAll(context.Background())
		gridsList, _ := gridRepo.GetAll(context.Background(), nil)
		theme := map[string]string{
			"LogoURL":              "/static/img/logo-imc.png",
			"PrimaryColor":         "#4364f7",
			"SecondaryColor":       "#161925",
			"BackgroundColor":      "#0f111a",
			"FontColor":            "#ffffff",
			"FieldBackgroundColor": "#161925",
		}
		data := TenantPageData{
			Title:           "Gerenciamento de Matrizes",
			LoggedUser:      session.UserName,
			ErrorMessage:    "❌ Erro ao atualizar no banco: " + err.Error(),
			Form:            formData,
			FormOpContacts:  opContacts,
			FormFinContacts: finContacts,
			Tenants:         list,
			Grids:           gridsList,
			User:            map[string]string{"Name": session.UserName, "Role": "Administrador Mestre"},
			Theme:           theme,
		}
		w.WriteHeader(http.StatusInternalServerError)
		tmpl := template.Must(template.ParseFiles("web/templates/tenants.html"))
		tmpl.Execute(w, data)
		return
	}

	http.Redirect(w, r, "/tenants", http.StatusSeeOther)
}

// HandleDeleteTenant exclui permanentemente uma matriz
func HandleDeleteTenant(w http.ResponseWriter, r *http.Request) {
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
	tenantID, err := uuid.Parse(idStr)
	if err == nil {
		_ = tenantRepo.Delete(context.Background(), tenantID)
	}

	http.Redirect(w, r, "/tenants", http.StatusSeeOther)
}

// HandleToggleTenantStatus ativa ou suspende uma matriz/cliente e todas as suas lojas
func HandleToggleTenantStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		_ = r.ParseForm()
		idStr = r.FormValue("id")
	}

	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid tenant id"}`, http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tenant, err := tenantRepo.GetByID(ctx, tenantID)
	if err != nil || tenant == nil {
		http.Error(w, `{"error":"tenant not found"}`, http.StatusNotFound)
		return
	}

	newStatus := !tenant.IsActive
	_, err = database.Pool.Exec(ctx, "UPDATE tenants SET is_active = $1, updated_at = NOW() WHERE id = $2", newStatus, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to update tenant status: %v"}`, err), http.StatusInternalServerError)
		return
	}
	tenant.IsActive = newStatus

	// Atualiza e para/inicia containers de todas as lojas deste tenant
	stores, _ := storeRepo.GetAll(ctx, &tenantID)
	for _, s := range stores {
		if s.RadioType == "server" || s.RadioType == "" {
			if !newStatus {
				_ = utils.StopStoreDockerContainer(s.ID)
			} else if s.IsActive {
				_ = utils.StartStoreDockerContainer(s)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"id":          tenant.ID.String(),
		"is_active":   newStatus,
		"tenant_name": tenant.TradeName,
		"message":     fmt.Sprintf("Empresa %s %s com sucesso!", tenant.TradeName, func() string {
			if newStatus { return "ativada" }
			return "suspensa"
		}()),
	})
}
