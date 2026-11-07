# Plano de Implementação: Esteira Universal de Masterização de Áudio, Selo SONIC e Ajustes no Cadastro

Este plano descreve as alterações técnicas para implementar a esteira universal de processamento de áudio (FFmpeg), o selo de certificação **SONIC** e melhorias no cadastro e exibição de comerciais e músicas.

## User Review Required

> [!IMPORTANT]
> **Selo SONIC:**
> Será exibida uma coluna visual com gradiente verde com o texto **SONIC** na listagem de Músicas e de Comerciais para todas as mídias em que `IsSonic` é verdadeiro.
>
> **Sincronização com o Player:**
> Conforme solicitado, a sincronização de mídias enviada para os players locais via `/api/sync` será restrita estritamente a arquivos com `is_sonic = true`.
>
> **Cadastro de Comerciais & Seleção em Massa:**
> Adição da opção "Selecionar Todas" no dropdown de empresas pertencentes no formulário de cadastro de comerciais para facilitar o envio em lote. A validação de nomes duplicados verificará todas as empresas selecionadas em vez de apenas a primeira.

---

## Proposed Changes

### Database & Models

#### [MODIFY] [postgres.go](file:///p:/imc-server/internal/database/postgres.go)
*   Adicionar as seguintes migrações para atualização automática do banco:
    *   `ALTER TABLE music_library ADD COLUMN IF NOT EXISTS is_sonic BOOLEAN DEFAULT FALSE`
    *   `ALTER TABLE commercials_and_events ADD COLUMN IF NOT EXISTS is_sonic BOOLEAN DEFAULT FALSE`

#### [MODIFY] [music_track.go](file:///p:/imc-server/internal/models/music_track.go)
*   Adicionar o campo `IsSonic bool` na struct `MusicTrack` com tags `json:"is_sonic"` e `db:"is_sonic"`.

#### [MODIFY] [media..go](file:///p:/imc-server/internal/models/media..go)
*   Adicionar o campo `IsSonic bool` na struct `CommercialsAndEvents` com tags `json:"is_sonic"` e `db:"is_sonic"`.

---

### Repository Layer

#### [MODIFY] [music_track_repository.go](file:///p:/imc-server/internal/repository/music_track_repository.go)
*   Atualizar as consultas SQL de `SELECT`, `INSERT` e `UPDATE` para incluir e mapear a nova coluna `is_sonic`.

#### [MODIFY] [commercial_repository.go](file:///p:/imc-server/internal/repository/commercial_repository.go)
*   Atualizar as consultas SQL de `SELECT`, `INSERT` e `UPDATE` para incluir e mapear a nova coluna `is_sonic`.

---

### Handlers & API Synchronization

#### [MODIFY] [commercial_handler.go](file:///p:/imc-server/internal/handlers/commercial_handler.go)
*   Atualizar `HandleValidateCommercialNames` para suportar `tenant_ids` (array de IDs de empresas) e verificar duplicidade de nomes em todas as empresas selecionadas.

#### [MODIFY] [store_handler.go](file:///p:/imc-server/internal/handlers/store_handler.go)
*   Garantir que a sincronização de músicas e comerciais via `/api/sync` envia apenas arquivos com `is_sonic = true`.

---

### Web Views & Templates

#### [MODIFY] [tracks_table_partial.html](file:///p:/imc-server/web/templates/tracks_table_partial.html)
*   Adicionar coluna `Selo` (SONIC badge) após a coluna `Nota`.
*   Ajustar os índices das colunas seguintes para ordenação correta (`sortTracks`).

#### [MODIFY] [commercials.html](file:///p:/imc-server/web/templates/commercials.html)
*   Adicionar coluna `Selo` (SONIC badge) após a coluna `Qualidade`.
*   Ajustar os índices das colunas seguintes para ordenação correta (`sortCommercials`).
*   Adicionar opção "Selecionar Todas" no dropdown de empresas pertencentes no formulário de cadastro.
*   Atualizar a função `handleTenantCheckboxChange` e o envio de formulário para interagir com o novo checkbox e realizar a validação com múltiplos `tenant_ids`.
*   Ajustar o `colspan` da linha sem itens na tabela de comerciais de 17 para 18.

---

## Verification Plan

### Automated Tests
*   Executar testes manuais de upload múltiplo com seleção de todas as empresas.
*   Executar e verificar o processamento bem-sucedido e inserção no banco de dados.

### Manual Verification
1.  **Exibição das Colunas SONIC:** Abrir a tela de Músicas e Comerciais. Confirmar que a coluna "Selo" exibe o badge estilizado "SONIC" se o arquivo passou pelas regras.
2.  **Cadastro Múltiplo:** Acessar a tela de comerciais, abrir o formulário, clicar em "Selecionar Todas" no dropdown de empresas, carregar múltiplos arquivos e salvar. Verificar que todos os arquivos são criados e processados para todas as empresas.
3.  **Sincronização do Player:** Chamar `/api/sync` e verificar que apenas arquivos com `is_sonic = true` são inclusos nos arrays de mídias.

