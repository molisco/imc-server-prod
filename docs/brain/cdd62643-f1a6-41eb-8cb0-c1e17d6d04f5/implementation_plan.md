# Implementação do Pipeline audio_analise & Fluxo de Aprovação Direta

## Objetivo
Implementar o diretório de trabalho `audio_analise/`, o status formal de catálogo `Em Análise` e o fluxo onde o áudio é fisicamente normalizado e cortado antes da inspeção humana em `audio_analise/`. Ao clicar em "Aprovar", o sistema apenas promove o arquivo pronto para `audio_out/` e atualiza o status para `Aprovado` no SQLite sem necessidade de reprocessamento.

## Mudanças Propostas

### 1. Camada de Dados (SQLite)
#### [MODIFY] [database.py](file:///home/paulo-assis/imc-music-studio/studio/core/database.py)
- Suporte explícito ao status `'Em Análise'` nas queries e índices.
- Atualizar `get_stats()` para contabilizar faixas em `'Em Análise'`.
- Implementar `promote_to_approved(file_path, target_out_dir)`: move/copia o arquivo tratado de `audio_analise/` para `audio_out/`, atualiza paths e grava status `Aprovado`.

### 2. Backend API
#### [MODIFY] [server.py](file:///home/paulo-assis/imc-music-studio/studio/ui/server.py)
- Garantir existência do diretório `audio_analise/` e expor em `/api/config` e nos atalhos de `/api/browse-directories`.
- Criar endpoint `/api/approve-track` para promoção direta sem reprocessamento DSP.
- Atualizar `/api/process-track` e reprocessamento: quando chamado para uma faixa de `audio_in/`, salvar o arquivo tratado em `audio_analise/` e definir status como `Em Análise`.
- Atualizar contadores de estatísticas globais para incluir `in_analysis`.

### 3. Interface Web & Estilos
#### [MODIFY] [index.html](file:///home/paulo-assis/imc-music-studio/studio/ui/templates/index.html)
- Adicionar atalho de pasta rápida `audio_analise` na barra de pastas.
- Adicionar badge e contador de status `Em Análise` no topo da tela.
- Adicionar pílula de filtro `Em Análise` na lista lateral de faixas.
- Ajustar botão "Exportar MP3" para "Processar p/ Análise" quando a faixa estiver em `audio_in`.

#### [MODIFY] [style.css](file:///home/paulo-assis/imc-music-studio/studio/ui/static/style.css)
- Adicionar estilos visuais para `.status-analysis` e `.stat-blue`.

#### [MODIFY] [app.js](file:///home/paulo-assis/imc-music-studio/studio/ui/static/app.js)
- No clique de `btnApprove` ("Aprovar & Próxima Faixa"): chamar `/api/approve-track`, atualizando status instantaneamente sem re-encoding.
- Renderizar badge `Em Análise` nas faixas da lista.
- Suporte ao filtro `in_analysis` nas pílulas de filtro.

## Plano de Verificação
- Executar testes unitários atualizados em `tests/test_ui_api.py` e `tests/test_reprocess.py`.
- Validar via script de teste o ciclo de vida completo: `audio_in` -> `audio_analise` (Em Análise) -> aprovação para `audio_out` (Aprovado).
