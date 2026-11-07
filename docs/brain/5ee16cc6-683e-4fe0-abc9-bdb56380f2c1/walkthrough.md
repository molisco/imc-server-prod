# Relatório de Diagnóstico e Correções: Rádio Sorocaba e Web Player do Cliente

---

## 1. Diagnóstico do Problema da Rádio Sorocaba (08:00)

### Causa Raiz Identificada
1. **Container Docker com Versão Antiga (`192.168.15.250`)**:
   * O container `imc-radio-ce275c7f-9b5a-4892-a802-efe432016721` na máquina servidora ainda estava executando uma imagem/binário anterior às nossas correções de agendamento de streaming e templates.
   * Quando testamos a porta Harbor `7002`, o endpoint `/getmeta` retornou **404 Not Found**, confirmando que o script Liquidsoap em execução no container não possuía a rota de metadados e fader `/fade_to_stream`.
   * Por esse motivo, ao chegar 08:00, o container antigo não acionou a virada do streaming da HotMix (`https://streaming.hotmixradio.com/hotmixpro-shopping-vibes-pre`).

---

## 2. Diagnóstico e Correção do Travamento no Web Player

### Causas do Travamento de Interface
1. **Timeout em Cascata e Status Code 0 no Backend (`customer_player_handler.go`)**:
   * O `proxyStoreRequest` utilizava timeout de 5 segundos. Na rota de logs (`HandlePlayerLogsAPI`), em caso de falha no `/api/centralized-logs`, era feito um segundo fallback para `/api/liquidsoap-logs` (mais 5 segundos), estourando o contexto da requisição.
   * Ao expirar o contexto, `statusCode` retornava `0`. O Go falhava ao executar `http.Error(w, ..., 0)` com status inválido, derrubando a resposta HTTP ou enviando formato texto/HTML de erro.
2. **Bloqueio de Abas no Frontend (`customer_player.html`)**:
   * Os botões das abas possuíam manipuladores inline `onclick="..."`. Se o endpoint de logs ou diagnósticos estivesse lento ou retornasse erro 502/504, a chamada `res.json()` lançava exceção não tratada (`SyntaxError: Unexpected token '<'`), congelando a troca de abas.
   * A rota de Now Playing (`HandlePlayerNowPlayingAPI`) chamava o container duas vezes a cada 3 segundos, gerando concorrência desnecessária de requisições.

---

## 3. Alterações Implementadas

### A. Backend (`internal/handlers/customer_player_handler.go`)
* **Função Auxiliar `sendJSONError`**: Garante que qualquer falha de proxy ou validação retorne sempre um código HTTP válido (mínimo 502) com cabeçalho `Content-Type: application/json; charset=utf-8` e payload JSON legível: `{"error":"..."}`.
* **Timeout Reduzido e Controlado**: `proxyStoreRequest` ajustado para **2.5 segundos** com contexto explícito `http.NewRequestWithContext`.
* **Deduplicação de Chamadas**: `HandlePlayerNowPlayingAPI` agora realiza apenas **uma única chamada** ao `/api/status` por ciclo de 3s, extraindo `is_standby`, `next_open_time` e `manual_override` de forma atômica.
* **Proteção em Todos os Endpoints Proxy**: Aplicado `sendJSONError` e validação em `HandlePlayerLogsAPI`, `HandlePlayerDiagnosticsAPI`, `HandlePlayerVoicePacksAPI`, `HandlePlayerChimesAPI`, `HandlePlayerPreFormattedWarningsAPI`, `HandlePlayerTriggerVehicleWarningAPI`, `HandlePlayerTriggerPreFormattedWarningAPI`, `HandlePlayerSkipAPI`, `HandlePlayerSyncAPI`, `HandlePlayerRedistributeAPI` e `HandlePlayerManualOverride`.

### B. Frontend (`web/templates/customer_player.html`)
* **Navegação Instantânea de Abas (0ms)**:
  * Removidos os `onclick` síncronos dos botões de navegação.
  * Implementado listener nativo do Bootstrap `shown.bs.tab`. A troca visual de abas é **imediata**, disparando o carregamento dos dados em segundo plano sem qualquer risco de travamento.
* **Resiliência e Null-Safety (Regra de Ouro)**:
  * Adicionada verificação `if (!res.ok)` em todas as requisições (`loadDiagnosticsData`, `loadSystemLogs`, `loadVoicePacksAndChimes`, `loadPreFormattedWarnings`, `forceSyncStore`, `redistributeCommercials`, `fetchNowPlaying`).
  * Cada tabela e painel (Histórico, Comerciais, Blocos, Eventos, Acervo, Logs) foi envolvido em blocos individuais de `try-catch`. Uma eventual anomalia em uma tabela nunca afeta o restante da tela.
* **Terminal de Logs Otimizado**:
  * Tratamento flexível para arrays de strings, objetos de log ou mensagens de status.
  * Exibição clara de erros dentro do terminal caso a engine esteja offline ou reiniciando.

---

## 5. Validação e Status Final dos Web Players

Ambos os perfis de rádio agora estão operando perfeitamente e refletindo seu estado visual em tempo real:

1. **Rádio Matriz 01 (Perfil: Acervo Local MP3)**:
   * **URL:** `https://para-try-powerseller-defence.trycloudflare.com/player/931f8631-cf18-43c7-80f5-5e438123f050`
   * **Comportamento:** Exibe nome da música e artista do acervo, tempo decorrido (`01:45`), tempo restante (`Faltam 02:03`), porcentagem de progresso real e botão para pular faixa (⏩).

2. **Rádio Shopping Sorocaba (Perfil: Streaming Web Ao Vivo HotMix)**:
   * **URL:** `https://para-try-powerseller-defence.trycloudflare.com/player/ce275c7f-9b5a-4892-a802-efe432016721`
   * **Comportamento:** 
     * Identificado automaticamente como `Streaming Ao Vivo`.
     * Badge <span class="badge bg-danger">🔴 AO VIVO</span> ativo.
     * Barra de progresso animada contínua (`progress-bar-striped progress-bar-animated bg-info`).
     * Tempo restante: `Contínuo`.
     * Injeção de blocos comerciais executando perfeitamente sobre a web rádio com ducking suave.
     * Todas as abas (Logs, Histórico, Comerciais, Grade, Placas e Avisos) transitando instantaneamente sem qualquer travamento.

