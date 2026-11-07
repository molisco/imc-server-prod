package models

import (
	"time"

	"github.com/google/uuid"
)

// ContactDetails representa a estrutura padrão ouro para qualquer contato no sistema.
// Agora, tanto o operacional quanto o financeiro usam este mesmo modelo modular.
type ContactDetails struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Tenant espelha a tabela de Matrizes/Clientes no banco de dados.
type Tenant struct {
	ID             uuid.UUID `db:"id" json:"id"`
	TradeName      string    `db:"trade_name" json:"trade_name"`
	LegalName      string    `db:"legal_name" json:"legal_name"`
	DocumentNumber string    `db:"document_number" json:"document_number"`

	// Usamos fatias (slices) da nossa struct de detalhes para permitir múltiplos contatos
	OperationalContacts []ContactDetails `db:"operational_contacts" json:"operational_contacts"`
	FinancialContacts   []ContactDetails `db:"financial_contacts" json:"financial_contacts"`

	// Endereço e Localização
	ZipCode    string  `db:"zip_code" json:"zip_code"`
	Street     string  `db:"street" json:"street"`
	Number     string  `db:"number" json:"number"`
	District   string  `db:"district" json:"district"`
	City       string  `db:"city" json:"city"`
	State      string  `db:"state" json:"state"`
	Country    string  `db:"country" json:"country"`
	Complement *string `db:"complement" json:"complement,omitempty"` // Ponteiro permite valores nulos (NULL) de forma elegante

	// Identidade Visual (White Label)
	LogoURL              *string `db:"logo_url" json:"logo_url,omitempty"`
	PrimaryColor         string  `db:"primary_color" json:"primary_color"`
	BackgroundColor      string  `db:"background_color" json:"background_color"`
	FontColor            string  `db:"font_color" json:"font_color"`
	FieldBackgroundColor string  `db:"field_background_color" json:"field_background_color"`

	IsActive           bool       `db:"is_active" json:"is_active"`
	DefaultAudioGridID *uuid.UUID `db:"default_audio_grid_id" json:"default_audio_grid_id,omitempty"`
	DefaultVideoGridID *uuid.UUID `db:"default_video_grid_id" json:"default_video_grid_id,omitempty"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
}
