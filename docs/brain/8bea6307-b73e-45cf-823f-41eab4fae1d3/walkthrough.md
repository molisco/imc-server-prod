# Walkthrough: Separação de Módulos (Áudio/Vídeo), Suporte a Feriados, Configuração por Dia e Comparativo Gráfico Semanal

Implementamos com sucesso todas as melhorias solicitadas de separação de mundos de áudio e vídeo, suporte a feriados na tabela de grades de programação, configuração individual para cada um dos 8 dias e um gráfico visual comparativo empilhado no rodapé do cadastro.

---

## Modificações Realizadas

### 1. Separação de Áudio e Vídeo ("Mundos Separados")
* **Sidebar Navegação**:
  - Modificado o menu lateral nos templates principais ([regionals.html](file:///p:/imc-server/web/templates/regionals.html), [stores.html](file:///p:/imc-server/web/templates/stores.html), [tenants.html](file:///p:/imc-server/web/templates/tenants.html)) para conter links independentes: **Grades de Áudio** (`/dashboard/grids/audio`) e **Grades de Vídeo** (`/dashboard/grids/video`).
* **Rotas e Endpoints ([main.go](file:///p:/imc-server/cmd/server/main.go))**:
  - Removidas as antigas rotas genéricas `/dashboard/grids/*`.
  - Registradas rotas e endpoints específicos para `/dashboard/grids/audio` e `/dashboard/grids/video` cobrindo as ações de listagem, criação, atualização e exclusão independentes.
* **Templates Dedicados**:
  - Excluído o arquivo genérico `grids.html` e criados os templates **[grids_audio.html](file:///p:/imc-server/web/templates/grids_audio.html)** e **[grids_video.html](file:///p:/imc-server/web/templates/grids_video.html)** específicos para cada recurso, com estilos customizados, títulos e badges apropriados (ex: `ÁUDIO PORTAL` e `VÍDEO PORTAL`).

### 2. Suporte a Feriados (Holiday)
* **Banco de Dados (PostgreSQL)**:
  - Executada a migração alterando a tabela `weekly_master_grids` para incluir as colunas `holiday_schedule_id UUID` e `holiday_block_id UUID`.
  - Atualizada a representação SQL local no arquivo [schema.sql](file:///p:/imc-server/schema.sql).
* **Modelos e Repositório**:
  - Inseridos os campos `HolidayScheduleID` e `HolidayBlockID` na struct `WeeklyMasterGrid` em [grid.go](file:///p:/imc-server/internal/models/grid.go).
  - Atualizadas as queries de inserção, leitura, atualização e exclusão no repositório [grid_repository.go](file:///p:/imc-server/internal/repository/grid_repository.go) para contemplar as colunas de Feriado.

### 3. Configuração Individual por Dia da Semana
* **Visual com 8 Abas no Formulário**:
  - Criadas 8 abas independentes no formulário: **Segunda**, **Terça**, **Quarta**, **Quinta**, **Sexta**, **Sábado**, **Domingo** e **Feriado**.
  - O preenchimento da segunda-feira é mandatório. Os outros dias contam com um switch de ativação que habilita ou desabilita a programação específica desse dia (salvando como NULL no banco caso desativado).
* **Ferramentas de Replicação Rápida**:
  - Adicionado botão **"Replicar Seg para Ter-Sex"** no topo do formulário que duplica os dados configurados de segunda-feira para todos os dias de semana úteis com um único clique.
  - Adicionado botão **"Copiar de Segunda"** em cada aba específica de Terça-feira a Feriado para preenchimento ágil.
* **Processamento de Formulário**:
  - A função `parseGridForm` em [grid_handler.go](file:///p:/imc-server/internal/handlers/grid_handler.go) mapeia e valida cada dia da semana separadamente com chaves individuais (`mon`, `tue`, `wed`, `thu`, `fri`, `sat`, `sun`, `holiday`).

### 4. Visão Geral Semanal (Gráfico Comparativo)
* **Design Premium e Dinâmico**:
  - Inserida uma seção fixa no rodapé do formulário chamada **"Visão Geral Semanal"**.
  - O visual exibe 8 linhas horizontais empilhadas representando cada dia da semana.
  - Quando um dia está ativo, ele plota uma miniatura da distribuição entre Música (verde) e Comerciais (laranja) proporcional ao expediente.
  - Caso esteja desativado, exibe uma barra cinza escrito `Fechado / Sem Programação`.
  - As 8 linhas atualizam-se instantaneamente em tempo real à medida que o usuário ajusta os horários ou tempos de qualquer aba.

---

## Ajustes Visuais e Correções Recentes (03/06/2026)

### 1. Correção de Tela em Branco ao Acessar Grades de Áudio
* **Causa do Problema**: Identificamos que as páginas de Grades de Áudio e Vídeo utilizavam a função de template do Go `slice` de forma incorreta (ex: `{{ $days := slice "mon" "tue" ... }}`). No mecanismo do Go `html/template`, a função `slice` serve para recortar uma lista existente (equivalente a `arr[i:j]`) e não para construir uma lista. Isso gerava um erro de execução silencioso que abortava a renderização no início do bloco de conteúdo, deixando a tela do portal completamente preta/em branco.
* **Solução**:
  - Criamos a estrutura `DayInfo` no backend em [grid_handler.go](file:///p:/imc-server/internal/handlers/grid_handler.go) e passamos a lista dos 8 dias e seus respectivos nomes estruturados via payload do Go (`GridsPageData.Days`).
  - Atualizamos as páginas [grids_audio.html](file:///p:/imc-server/web/templates/grids_audio.html) e [grids_video.html](file:///p:/imc-server/web/templates/grids_video.html) para iterar diretamente sobre `.Days` (ex: `{{ range $index, $dayOpt := .Days }}`), eliminando a chamada problemática.

### 2. Exibição do Tipo de Usuário (Perfil/Role) no Topo
* **Causa do Problema**: Nos menus anteriores, a barra superior exibia o perfil do usuário (ex: `Administrador Mestre`). Porém, nos novos portais de áudio e vídeo, a barra superior exibia apenas o badge `ÁUDIO PORTAL` ou `VÍDEO PORTAL`, sem mostrar o nível de privilégio.
* **Solução**:
  - Atualizamos a estrutura de dados em [grid_handler.go](file:///p:/imc-server/internal/handlers/grid_handler.go) para passar a role `"Administrador Mestre"`, mantendo consistência com as demais páginas.
  - Modificamos os cabeçalhos de `grids_audio.html` e `grids_video.html` para renderizar os dois badges em paralelo (ex: `Administrador Mestre | ÁUDIO PORTAL`), deixando o canto superior direito alinhado e completo como nas outras telas.

---

## Verificação e Testes

### 1. Testes de Integração de Repositório (`TestGridCRUD`)
Atualizamos e executamos a suíte de testes de integração em **[grid_repository_test.go](file:///p:/imc-server/internal/repository/grid_repository_test.go)** para validar a criação dos 8 dias separados, a inserção e atualização de feriados, e a limpeza correta em transações.

#### Comando de Execução:
```bash
ssh dreams "cd imc-server && DATABASE_URL='...' go test -v ./..."
```

#### Resultado obtido:
```
=== RUN   TestHandleImportStores
--- PASS: TestHandleImportStores (0.17s)
=== RUN   TestCNPJValidation
--- PASS: TestCNPJValidation (0.00s)
PASS
ok  	imc-server/internal/handlers	0.191s

=== RUN   TestGridCRUD
🚀 Magic inside the Linux: Successfully connected to PostgreSQL Pool!
✅ Database schema patches applied successfully (district, country, colors).
🛑 Database connection pool closed safely.
--- PASS: TestGridCRUD (0.11s)
=== RUN   TestStoreCRUD
🚀 Magic inside the Linux: Successfully connected to PostgreSQL Pool!
✅ Database schema patches applied successfully (district, country, colors).
🛑 Database connection pool closed safely.
--- PASS: TestStoreCRUD (0.13s)
=== RUN   TestTenantCRUD
🚀 Magic inside the Linux: Successfully connected to PostgreSQL Pool!
✅ Database schema patches applied successfully (district, country, colors).
🛑 Database connection pool closed safely.
--- PASS: TestTenantCRUD (0.10s)
PASS
ok  	imc-server/internal/repository	(cached)
```
*(Todos os testes estão passando com 100% de sucesso e sem regressões, e os arquivos HTML de templates foram validados local e remotamente sem erros).*
