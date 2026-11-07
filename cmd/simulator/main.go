package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"imc-server/internal/database"
	"imc-server/internal/handlers"
	"imc-server/internal/models"
	"imc-server/internal/repository"
)

// Cores ANSI para deixar o console maravilhoso e fácil de ler
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

func logInfo(msg string, args ...interface{}) {
	fmt.Printf(Cyan+"[ INFO ] "+Reset+msg+"\n", args...)
}

func logSuccess(msg string, args ...interface{}) {
	fmt.Printf(Green+Bold+"[  OK  ] "+Reset+msg+"\n", args...)
}

func logWarning(msg string, args ...interface{}) {
	fmt.Printf(Yellow+"[ AVISO] "+Reset+msg+"\n", args...)
}

func logError(msg string, args ...interface{}) {
	fmt.Printf(Red+Bold+"[ ERRO ] "+Reset+msg+"\n", args...)
}

func main() {
	fmt.Println(Bold + Cyan + "=========================================================" + Reset)
	fmt.Println(Bold + Cyan + "       SIMULADOR DE INTEGRAÇÃO & TESTES: IMC-SERVER      " + Reset)
	fmt.Println(Bold + Cyan + "=========================================================" + Reset)

	// 1. Inicializa Conexão do Banco de Dados
	logInfo("Conectando ao banco de dados e aplicando patches...")
	database.ConnectDatabase()
	defer database.CloseDatabase()

	if database.Pool == nil {
		logError("Pool de conexões não inicializado. Abortando simulação.")
		os.Exit(1)
	}
	logSuccess("Conexão ao PostgreSQL estabelecida com sucesso!")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Instancia repositórios
	tenantRepo := repository.NewTenantRepository()
	regionalRepo := repository.NewRegionalRepository()

	fmt.Println("\n" + Bold + Yellow + "--- PASSO 1: VALIDAÇÃO MATEMÁTICA DE CNPJ ---" + Reset)
	
	// Teste 1: CNPJ Inválido (do usuário)
	cnpjInvalido := "19097485000102"
	if handlers.IsValidCNPJ(cnpjInvalido) {
		logError("Falha no validador: O CNPJ %s (inválido) foi aceito!", cnpjInvalido)
		os.Exit(1)
	} else {
		logSuccess("Sucesso: CNPJ matematicamente inválido %s foi rejeitado pelo algoritmo mod 11.", cnpjInvalido)
	}

	// Teste 2: Outro CNPJ repetido inválido
	cnpjInvalido2 := "11111111111111"
	if handlers.IsValidCNPJ(cnpjInvalido2) {
		logError("Falha no validador: O CNPJ %s (sequencial) foi aceito!", cnpjInvalido2)
		os.Exit(1)
	} else {
		logSuccess("Sucesso: CNPJ sequencial %s foi rejeitado.", cnpjInvalido2)
	}

	// Teste 3: CNPJ Válido
	cnpjValido := "12345678000195" // CNPJ Válido gerado matematicamente
	if !handlers.IsValidCNPJ(cnpjValido) {
		logError("Falha no validador: O CNPJ %s (válido) foi rejeitado!", cnpjValido)
		os.Exit(1)
	} else {
		logSuccess("Sucesso: CNPJ matematicamente válido %s foi aceito.", cnpjValido)
	}

	// Limpeza preventiva de execuções anteriores que possam ter sido interrompidas
	database.Pool.Exec(ctx, "DELETE FROM tenants WHERE document_number = $1", cnpjValido)

	fmt.Println("\n" + Bold + Yellow + "--- PASSO 2: CADASTRO DE MATRIZ (COM DADOS OPCIONAIS) ---" + Reset)

	tenantID := uuid.New()
	logoOriginal := "/static/uploads/logo-teste.png"
	
	// Criamos um tenant contendo contatos operacionais e financeiros parciais para comprovar
	// que o Nome não é obrigatório e que dados incompletos/opcionais gravam perfeitamente.
	testTenant := models.Tenant{
		ID:             tenantID,
		TradeName:      "Simulador Supermercados",
		LegalName:      "Simulador Comercio de Alimentos LTDA",
		DocumentNumber: cnpjValido,
		ZipCode:        "80010-010",
		Street:         "Rua XV de Novembro",
		Number:         "1230", // Campo número obrigatório agora
		District:       "Centro",
		City:           "Curitiba",
		State:          "PR",
		Country:        "Brasil",
		LogoURL:        &logoOriginal,
		PrimaryColor:   "#4364f7",
		BackgroundColor: "#0f111a",
		FontColor:      "#ffffff",
		FieldBackgroundColor: "#161925",
		IsActive:       true,
		OperationalContacts: []models.ContactDetails{
			{Name: "Op Completo", Email: "completo@teste.com", Phone: "11999998888"},
			{Name: "", Email: "apenas_email@teste.com", Phone: ""}, // Sem nome! Opcional
			{Name: "", Email: "", Phone: "11888887777"},          // Sem nome e sem email! Opcional
		},
		FinancialContacts: []models.ContactDetails{
			{Name: "Fin Apenas Nome", Email: "", Phone: ""}, // Sem email e sem telefone
		},
	}

	logInfo("Cadastrando matriz '%s' com contatos parciais e número do endereço...", testTenant.TradeName)
	err := tenantRepo.Create(ctx, testTenant)
	if err != nil {
		logError("Falha ao gravar matriz no banco: %v", err)
		os.Exit(1)
	}
	logSuccess("Matriz gravada no banco com sucesso!")

	fmt.Println("\n" + Bold + Yellow + "--- PASSO 3: RECUPERAÇÃO E VALIDAÇÃO DE CAMPOS ---" + Reset)
	
	list, err := tenantRepo.GetAll(ctx)
	if err != nil {
		logError("Falha ao listar matrizes: %v", err)
		os.Exit(1)
	}

	var foundTenant *models.Tenant
	for _, t := range list {
		if t.ID == tenantID {
			foundTenant = &t
			break
		}
	}

	if foundTenant == nil {
		logError("Erro: A matriz cadastrada não foi encontrada na lista de busca!")
		os.Exit(1)
	}

	// Verifica se os campos de endereço, número e contatos foram preservados
	if foundTenant.Number != "1230" {
		logError("Erro: Campo 'Número' corrompido! Esperado '1230', recebido '%s'", foundTenant.Number)
		os.Exit(1)
	}
	logSuccess("Sucesso: Campo 'Número' gravado e recuperado corretamente: %s", foundTenant.Number)

	if len(foundTenant.OperationalContacts) != 3 {
		logError("Erro: Quantidade de contatos operacionais incorreta! Esperado 3, recebido %d", len(foundTenant.OperationalContacts))
		os.Exit(1)
	}
	logSuccess("Sucesso: Gravou e recuperou contatos operacionais opcionais (total: %d)", len(foundTenant.OperationalContacts))

	// Confirma se o contato sem nome foi recuperado
	var contatoSemNomeEncontrado bool
	for _, c := range foundTenant.OperationalContacts {
		if c.Name == "" && c.Email == "apenas_email@teste.com" {
			contatoSemNomeEncontrado = true
			break
		}
	}
	if !contatoSemNomeEncontrado {
		logError("Erro: Contato opcional sem nome (apenas email) foi perdido no banco de dados!")
		os.Exit(1)
	}
	logSuccess("Sucesso: Contato opcional sem nome recuperado com integridade.")

	fmt.Println("\n" + Bold + Yellow + "--- PASSO 4: ATUALIZAÇÃO (EDIÇÃO/WHITE LABEL) ---" + Reset)
	
	// Alteramos as cores do white label e adicionamos um contato
	foundTenant.TradeName = "Simulador Supermercados Editado"
	foundTenant.PrimaryColor = "#00ff00" // Verde
	foundTenant.OperationalContacts = append(foundTenant.OperationalContacts, models.ContactDetails{
		Name: "Novo Contato Adicionado",
		Email: "novo@teste.com",
	})

	logInfo("Atualizando Matriz: novo nome '%s', cor destaque '%s'", foundTenant.TradeName, foundTenant.PrimaryColor)
	err = tenantRepo.Update(ctx, *foundTenant)
	if err != nil {
		logError("Falha ao atualizar matriz: %v", err)
		os.Exit(1)
	}
	logSuccess("Matriz editada com sucesso!")

	// Recarrega e valida
	list2, _ := tenantRepo.GetAll(ctx)
	for _, t := range list2 {
		if t.ID == tenantID {
			if t.TradeName != "Simulador Supermercados Editado" || t.PrimaryColor != "#00ff00" || len(t.OperationalContacts) != 4 {
				logError("Erro: Dados de atualização não foram salvos corretamente!")
				os.Exit(1)
			}
			break
		}
	}
	logSuccess("Matriz alterada validada com sucesso no banco!")

	fmt.Println("\n" + Bold + Yellow + "--- PASSO 5: CADASTRO E ASSOCIAÇÃO DE REGIONAL ---" + Reset)
	
	regionalID := uuid.New()
	coordName := "Carlos Coordenador Teste"
	coordPhone := "41999995555"
	testRegional := models.Regional{
		ID:              regionalID,
		TenantID:        tenantID,
		Name:            "Regional Simulada PR-Sul",
		CoordinatorName: &coordName,
		Phone:           &coordPhone,
		IsActive:        true,
	}

	logInfo("Cadastrando Regional '%s' vinculada à Matriz...", testRegional.Name)
	err = regionalRepo.Create(ctx, testRegional)
	if err != nil {
		logError("Erro ao cadastrar Regional: %v", err)
		os.Exit(1)
	}
	logSuccess("Regional cadastrada com sucesso!")

	// Listar e verificar regional
	regionals, err := regionalRepo.GetAll(ctx, nil)
	if err != nil {
		logError("Erro ao listar regionais: %v", err)
		os.Exit(1)
	}

	var foundRegional bool
	for _, r := range regionals {
		if r.ID == regionalID && r.TenantTradeName == "Simulador Supermercados Editado" {
			foundRegional = true
			break
		}
	}
	if !foundRegional {
		logError("Erro: Regional vinculada não foi encontrada ou não associou o nome fantasia da matriz!")
		os.Exit(1)
	}
	logSuccess("Regional validada com sucesso com vinculo correto!")

	fmt.Println("\n" + Bold + Yellow + "--- PASSO 6: EXCLUSÃO EM CASCATA ---" + Reset)
	
	logInfo("Removendo a Matriz de Testes (ID: %s)...", tenantID)
	err = tenantRepo.Delete(ctx, tenantID)
	if err != nil {
		logError("Erro ao excluir matriz: %v", err)
		os.Exit(1)
	}
	logSuccess("Matriz excluída do banco!")

	// Verifica se a regional vinculada sumiu do banco por cascata
	regionals2, _ := regionalRepo.GetAll(ctx, nil)
	regionalAindaExiste := false
	for _, r := range regionals2 {
		if r.ID == regionalID {
			regionalAindaExiste = true
			break
		}
	}

	if regionalAindaExiste {
		logError("Erro crítico: A regional de testes continuou no banco após a exclusão da Matriz! Cascata falhou.")
		os.Exit(1)
	}
	logSuccess("Sucesso: A Regional foi deletada automaticamente em cascata!")

	fmt.Println("\n" + Bold + Green + "=========================================================" + Reset)
	fmt.Println(Bold + Green + "      SIMULAÇÃO CONCLUÍDA: TODOS OS TESTES PASSARAM!     " + Reset)
	fmt.Println(Bold + Green + "=========================================================" + Reset)
}
