package handlers

import (
	"testing"
)

func TestCNPJValidation(t *testing.T) {
	// Teste 1: CNPJ Inválido reportado pelo usuário
	invalidCNPJ := "19097485000102"
	if IsValidCNPJ(invalidCNPJ) {
		t.Errorf("Esperado que IsValidCNPJ(%s) fosse falso, mas retornou verdadeiro", invalidCNPJ)
	}

	// Teste 2: Sequencial Inválido
	seqCNPJ := "11111111111111"
	if IsValidCNPJ(seqCNPJ) {
		t.Errorf("Esperado que IsValidCNPJ(%s) fosse falso, mas retornou verdadeiro", seqCNPJ)
	}

	// Teste 3: CNPJ Válido gerado matematicamente
	validCNPJ := "12345678000195"
	if !IsValidCNPJ(validCNPJ) {
		t.Errorf("Esperado que IsValidCNPJ(%s) fosse verdadeiro, mas retornou falso", validCNPJ)
	}
}
