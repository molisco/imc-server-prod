# Adição da Aba "Eventos" no Painel de Diagnósticos

Este plano detalha o design e a implementação para incluir uma aba dedicada a "Eventos" (EVENT) no painel de controle web, mantendo o mesmo estilo visual premium de tabelas e cartões translúcidos (glassmorphism) do painel atual.

## User Review Required

> [!IMPORTANT]
> A nova aba exibirá a lista de todos os eventos agendados para a data selecionada (Hoje ou Amanhã) com base no seletor de data global da página.
> As informações mostradas incluirão:
> - **Horário agendado** (formato HH:MM)
> - **Modo de interrupção** (SOFT ou HARD)
> - **Status de sincronização/mídia** (PRONTO se baixado, PENDENTE caso contrário)
> - **Nome do Evento**

---

## Proposed Changes

### Component: Local Player (imc-audio-local)

#### [MODIFY] [web_dashboard.go](file:///c:/imc-audio-local/web_dashboard.go)

1. **Estrutura de Dados (Go):**
   - Criar a struct `DiagnosticEventSlot` para retornar os dados dos eventos via JSON.
   - Adicionar o campo `Events []DiagnosticEventSlot` à struct principal `DiagnosticResponse`.

2. **Endpoint `/api/diagnostics` (Go):**
   - Adicionar a consulta na tabela `commercials_and_events` para recuperar eventos ativos na data consultada (usando `targetDateStr`).
   - Mapear os resultados para preencher a lista de `Events` na resposta JSON.

3. **Layout e Tabs HTML (diagnosticsHTML):**
   - Inserir o botão da aba no menu de navegação: `<button class="tab-btn" id="btn-tab-events" onclick="switchTab('tab-events')">Eventos</button>` (posicionado logo após a aba de comerciais).
   - Inserir a div correspondente ao conteúdo da aba (`tab-events`), contendo a tabela com as colunas: **Horário**, **Modo (Interrupção)**, **Status**, e **Nome do Evento**.

4. **Lógica de Interface (JavaScript):**
   - Atualizar a função `switchTab` para gerenciar a classe `active` do botão da aba de eventos.
   - Atualizar a função `setCommercialsDate` para também atualizar o título da aba de eventos ("Eventos Agendados (Hoje)" ou "Eventos Agendados (Amanhã)").
   - Na função `updateDiagnostics`, adicionar o código para limpar e preencher dinamicamente a tabela de eventos (`tbl-events-body`) com estilos consistentes (badges para o Modo HARD/SOFT e Status PRONTO/PENDENTE).

---

## Verification Plan

### Automated Tests
- Compilar o player local para validar a sintaxe e a integridade de compilação:
  ```powershell
  go build -o imc-audio-local.exe
  ```

### Manual Verification
- Acessar o painel de diagnósticos no navegador (`http://localhost:5005/diagnostics`).
- Verificar se a aba "Eventos" aparece ao lado de "Distribuição de Comerciais".
- Alternar entre "Hoje" e "Amanhã" e constatar que os títulos e a lista de eventos agendados mudam de acordo.
- Validar se a estilização dos modos e status (badges coloridos) segue o padrão estético premium do dashboard.
