package handlers

import (
	"regexp"
	"strconv"
)

// IsValidCNPJ aplica o algoritmo oficial matemático da Receita Federal para validar o CNPJ.
// Padrão Ouro: impede que o operador cadastre números falsos ou inválidos.
func IsValidCNPJ(cnpj string) bool {
	// Remove caracteres não numéricos (. , / -)
	re := regexp.MustCompile(`\D`)
	cnpj = re.ReplaceAllString(cnpj, "")

	// CNPJ precisa ter exatamente 14 dígitos numéricos
	if len(cnpj) != 14 {
		return false
	}

	// Rejeita sequências conhecidas de números repetidos
	invalidSequences := []string{
		"00000000000000", "11111111111111", "22222222222222",
		"33333333333333", "44444444444444", "55555555555555",
		"66666666666666", "77777777777777", "88888888888888", "99999999999999",
	}
	for _, seq := range invalidSequences {
		if cnpj == seq {
			return false
		}
	}

	// Validação do Primeiro Dígito Verificador
	weightsFirst := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	sum := 0
	for i := 0; i < 12; i++ {
		num, _ := strconv.Atoi(string(cnpj[i]))
		sum += num * weightsFirst[i]
	}
	remainder := sum % 11
	digit1 := 0
	if remainder >= 2 {
		digit1 = 11 - remainder
	}
	if strconv.Itoa(digit1) != string(cnpj[12]) {
		return false
	}

	// Validação do Segundo Dígito Verificador
	weightsSecond := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	sum = 0
	for i := 0; i < 13; i++ {
		num, _ := strconv.Atoi(string(cnpj[i]))
		sum += num * weightsSecond[i]
	}
	remainder = sum % 11
	digit2 := 0
	if remainder >= 2 {
		digit2 = 11 - remainder
	}
	return strconv.Itoa(digit2) == string(cnpj[13])
}
