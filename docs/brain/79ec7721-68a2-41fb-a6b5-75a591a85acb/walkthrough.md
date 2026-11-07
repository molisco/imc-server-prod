# Walkthrough - PoP Logging, Execução Real Tab, Auto-Catchup

## Resumo das Alterações

Três funcionalidades foram implementadas nesta sessão:

---

## 1. Proof of Play (PoP) Logging

**Objetivo:** Registrar de forma estruturada cada item reproduzido para auditoria e análise.

### Arquivos Modificados

#### [db.go](file:///c:/imc-audio-local/db.go)
- Adicionadas colunas `duration_seconds` e `skipped_by_catchup` via `ALTER TABLE` na migração do banco
- Criado índice `idx_play_history_track_id` para consultas rápidas por track_id

#### [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
- `recordPlayHistory()` agora grava `duration_seconds` junto com cada registro
- Permite rastrear duração real de cada item reproduzido

---

## 2. Aba "Execução Real (Hoje)" no Diagnóstico

**Objetivo:** Visualizar em tempo real o histórico de tudo que foi efetivamente tocado pelo player.

### Arquivos Modificados

#### [web_dashboard.go](file:///c:/imc-audio-local/web_dashboard.go)

**Estruturas adicionadas:**
- `PlayHistoryRecord` struct com campos: track_id, track_type, display_name, artist, played_at, duration_seconds, skipped_by_catchup
- `PlayHistory` e `CurrentDesvioSecs` adicionados ao `DiagnosticResponse`

**API (`handleAPIDiagnostics`):**
- Query de `play_history` para o dia atual com ORDER BY played_at ASC
- Cálculo do desvio atual via `calcCurrentDesvio()`

**UI (HTML/JS):**
- Nova aba "Execução Real (Hoje)" no nav de tabs
- Tabela com colunas: Horário, Tipo, Título, Artista, Duração, Status
- Tipos coloridos conforme padrão (Música=azul, Comercial=amarelo, etc.)
- Badge de desvio em tempo real (verde=sincronizado, vermelho=atrasado)
- Contador de totais (músicas, comerciais, puladas por auto-catchup)
- Itens pulados por catchup aparecem com fundo vermelho e badge "⏩ Pulada"
- Auto-scroll para o item mais recente

---

## 3. Auto-Catchup Monitor

**Objetivo:** Detectar automaticamente quando o player está atrasado ≥3 minutos e pular a música atual (se já tocou ≥50%) para recuperar o sincronismo.

### Arquivos Modificados

#### [watchdog.go](file:///c:/imc-audio-local/watchdog.go)
- Adicionado `elapsed_seconds` à resposta do endpoint `/getmeta` do Liquidsoap
- Valor sanitizado (NaN/Infinite/negativo → 0.0)

#### [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
- **`StartCatchupMonitor()`**: Goroutine que a cada 15s:
  1. Verifica se está no horário de operação
  2. Consulta metadata do Liquidsoap (`/getmeta`)
  3. Só age em tracks de música (nunca pula comerciais)
  4. Verifica se elapsed ≥ 50% da duração total
  5. Calcula desvio via `calcCurrentDesvio()`
  6. Se desvio ≥ 180s (3 min), envia skip via `/skip`
  7. Marca o registro no `play_history` como `skipped_by_catchup = 1`
  8. Cooldown de 10s para evitar double-skip

- **`calcCurrentDesvio()`**: Compara tempo decorrido do relógio desde o início da programação vs. soma de durations de tudo que foi reproduzido hoje

#### [main.go](file:///c:/imc-audio-local/main.go)
- Adicionado `StartCatchupMonitor()` como step 6 na inicialização

---

## Verificação

| Teste | Resultado |
|-------|-----------|
| `go build ./...` | ✅ Compilação sem erros |
| `go run .` | ✅ Player iniciado com sucesso |
| Nova aba visível em `/diagnostics` | ✅ "Execução Real (Hoje)" |
| Catchup monitor logs | ✅ "[Catchup] Auto-catchup monitor started" |
