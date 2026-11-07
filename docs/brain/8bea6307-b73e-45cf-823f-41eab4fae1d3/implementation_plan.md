# Plano de Implementação: Divisão de Módulos (Áudio/Vídeo), Cadastro por Dia e Comparativo Gráfico Semanal

Este plano detalha a reestruturação do módulo de Grades de Programação para separar as grades de Áudio e Vídeo em páginas dedicadas, incluir suporte a Feriados, e permitir a customização completa por cada dia da semana individual com uma visão comparativa gráfica de toda a semana.

---

## User Review Required

> [!IMPORTANT]
> 1. **Separação de Mundos (Áudio e Vídeo)**:
>    - Dividiremos a rota `/dashboard/grids` em duas rotas específicas: `/dashboard/grids/audio` e `/dashboard/grids/video`.
>    - Cada rota renderizará o formulário e a listagem apenas das grades correspondentes a esse tipo.
>    - Adicionaremos links correspondentes na barra lateral (Sidebar) para "Grades de Áudio" e "Grades de Vídeo".
> 2. **Suporte a Feriados (Holiday)**:
>    - Mapearemos `holiday_schedule_id` e `holiday_block_id` na tabela `weekly_master_grids`. (A migração do banco já foi executada com sucesso via SSH).
> 3. **Configuração Individual por Dia da Semana**:
>    - O formulário terá 8 abas de configuração independentes: **Segunda**, **Terça**, **Quarta**, **Quinta**, **Sexta**, **Sábado**, **Domingo** e **Feriado**.
>    - Para cada dia, haverá um seletor de ativação (exceto Segunda-feira, que é obrigatória).
>    - Para mitigar a fadiga de preenchimento, incluiremos botões rápidos de replicação no topo das abas: **"Copiar de Segunda-feira"** e **"Replicar Segunda-feira para todos os dias de semana (Ter-Sex)"**.
> 4. **Painel de Visão Geral Semanal (Gráfico Comparativo)**:
>    - Adicionaremos uma seção fixa de visualização chamada **"🔍 Visão Geral Semanal"**.
>    - Ela empilhará 8 barras de linha do tempo horizontal comparativas (Segunda a Feriado), atualizadas em tempo real a cada alteração nos formulários, facilitando enxergar e comparar o expediente e a distribuição sonora (Música vs. Comerciais) de toda a semana.

---

## Proposed Changes

### Component 1: Sidebar Layout

#### [MODIFY] [regionals.html](file:///p:/imc-server/web/templates/regionals.html), [stores.html](file:///p:/imc-server/web/templates/stores.html), [tenants.html](file:///p:/imc-server/web/templates/tenants.html)
- Atualizar a navegação na Sidebar para trocar a rota genérica por:
  ```html
  <li><a href="/dashboard/grids/audio" class="nav-link-custom">📅 Grades de Áudio</a></li>
  <li><a href="/dashboard/grids/video" class="nav-link-custom">📺 Grades de Vídeo</a></li>
  ```

---

### Component 2: Backend Models and Repository

#### [MODIFY] [grid.go](file:///p:/imc-server/internal/models/grid.go)
- Adicionar os campos de feriado à struct `WeeklyMasterGrid`:
  - `HolidayScheduleID *uuid.UUID`
  - `HolidayBlockID *uuid.UUID`
- Adicionar os campos específicos de cada dia da semana para mapear individualmente (Monday a Friday já existem como colunas no banco de dados).

#### [MODIFY] [grid_repository.go](file:///p:/imc-server/internal/repository/grid_repository.go)
- Atualizar as operações `Create`, `Update`, `GetByID` e `Delete` para processar de forma individual cada um dos 8 dias:
  - `mon`, `tue`, `wed`, `thu`, `fri`, `sat`, `sun`, `holiday`.
  - Mapear cada dia com seu respectivo ID de schedule e block config de forma independente no banco para evitar conflitos de compartilhamento de ponteiros.

---

### Component 3: HTTP Routing and Handlers

#### [MODIFY] [main.go](file:///p:/imc-server/cmd/server/main.go)
- Substituir as antigas rotas de grids pelas rotas separadas por tipo de mídia:
  - **Áudio**:
    - `r.Get("/dashboard/grids/audio", handlers.HandleAudioGridsPage)`
    - `r.Post("/dashboard/grids/audio/create", handlers.HandleCreateAudioGrid)`
    - `r.Post("/dashboard/grids/audio/update", handlers.HandleUpdateAudioGrid)`
    - `r.Get("/dashboard/grids/audio/delete", handlers.HandleDeleteAudioGrid)`
  - **Vídeo**:
    - `r.Get("/dashboard/grids/video", handlers.HandleVideoGridsPage)`
    - `r.Post("/dashboard/grids/video/create", handlers.HandleCreateVideoGrid)`
    - `r.Post("/dashboard/grids/video/update", handlers.HandleUpdateVideoGrid)`
    - `r.Get("/dashboard/grids/video/delete", handlers.HandleDeleteVideoGrid)`

#### [MODIFY] [grid_handler.go](file:///p:/imc-server/internal/handlers/grid_handler.go)
- Dividir o handler antigo em handlers focados por tipo de mídia (`AUDIO` / `VIDEO`).
- Atualizar a função `parseGridForm` para ler os parâmetros individuais de cada um dos 8 dias (ex: `mon_start_time`, `tue_start_time`, etc.).
- Tratar o redirecionamento correto para `/dashboard/grids/audio` ou `/dashboard/grids/video` com base no contexto.

---

### Component 4: Dynamic Templates (Grids)

#### [NEW] [grids_audio.html](file:///p:/imc-server/web/templates/grids_audio.html) & [grids_video.html](file:///p:/imc-server/web/templates/grids_video.html)
- Substituir o arquivo genérico `grids.html` por dois templates específicos contendo:
  - Formulário com 8 abas de dias (Segunda a Feriado).
  - Botão de replicação rápida ("Replicar Segunda-feira").
  - Painel de visualização comparativo fixo "Visão Geral Semanal", contendo 8 barras de timeline empilhadas verticalmente.
  - Ajustes de links e cabeçalhos para refletir "Grades de Áudio" ou "Grades de Vídeo".

---

## Verification Plan

### Automated Tests
- Atualizar `grid_repository_test.go` para testar os 8 dias (incluindo Holiday) de forma isolada e executar no servidor `dreams`:
  ```bash
  ssh dreams "cd imc-server && DATABASE_URL='...' go test -v ./..."
  ```

### Manual Verification
1. Acessar a barra lateral e navegar entre **Grades de Áudio** e **Grades de Vídeo**. Validar que as listagens trazem apenas os registros corretos.
2. Criar uma grade nova, habilitar Terça e Sábado, clicar em "Copiar de Segunda" na Terça-feira para verificar se os dados do formulário são duplicados instantaneamente.
3. Observar se o gráfico de "Visão Geral Semanal" no rodapé do formulário atualiza as 8 linhas simultaneamente de forma proporcional e harmônica.
