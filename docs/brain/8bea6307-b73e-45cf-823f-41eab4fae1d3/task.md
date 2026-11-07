# Checklist de Execução: Divisão de Grades de Programação (Áudio/Vídeo) e Ajuste por Dia

- [x] Modelos e Repositório do Banco de Dados
  - [x] Atualizar struct `WeeklyMasterGrid` em `internal/models/grid.go` com campos de feriado e todos os dias da semana
  - [x] Atualizar `Create`, `Update`, `GetByID` e `Delete` em `internal/repository/grid_repository.go` para processar individualmente os 8 dias (Mon, Tue, Wed, Thu, Fri, Sat, Sun, Holiday)
- [x] Handlers do Servidor e Rotas
  - [x] Registrar novas rotas específicas de áudio e vídeo em `cmd/server/main.go`
  - [x] Atualizar `internal/handlers/grid_handler.go` dividindo os endpoints por tipo de recurso (`AUDIO`/`VIDEO`)
  - [x] Adaptar `parseGridForm` para coletar dados independentes de formulário para os 8 dias
- [x] Interface Frontend (Templates)
  - [x] Criar template `web/templates/grids_audio.html` para grades de áudio (8 abas de dias, replicação e gráfico comparativo semanal)
  - [x] Criar template `web/templates/grids_video.html` para grades de vídeo (mesmo layout adaptado)
  - [x] Modificar navegação lateral (Sidebar) nos templates de regionais, filiais e tenants
- [x] Verificação e Testes
  - [x] Atualizar suíte de testes em `internal/repository/grid_repository_test.go` para verificar os 8 dias (incluindo feriado)
  - [x] Executar testes no Linux `dreams` via SSH e garantir 100% PASS
