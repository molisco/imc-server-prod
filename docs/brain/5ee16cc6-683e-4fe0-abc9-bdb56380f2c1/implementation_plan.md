# Plano de Implementação: Correção e Padronização do Comportamento Fora do Horário (Standby / Silêncio)

Implementação da interrupção automática de reprodução e transição para **Standby Silencioso** (`SILENCE`) no horário de término (`end_time`) e retomada no horário de início (`start_time`), abrangendo todos os perfis de playout (**Acervo**, **Streaming Relay** e **Híbrido**) no `imc-audio-local` e na interface Web Player do `imc-server`.

---

## 1. Contexto e Diagnóstico

### A. O que foi diagnosticado no Player Local (`imc-player-lab`):
* No player local com acervo, o cálculo de standby possuía uma tolerância residual (`graceEndSeconds = endSeconds + 30*60`) que postergava o corte por 30 minutos.
* Essa tolerância já foi removida e testada com sucesso no ambiente lab.

### B. O que foi diagnosticado na Rádio Sorocaba (Servidor / Perfil Streaming):
1. **Prioridade de Checks no `scheduler.go`:**
   * Em `scheduler.go`, a verificação de streaming ativo (`getActiveStreamForTime` e `CurrentConfig.ExternalStreamURL`) ocorria **antes** da verificação `IsStoreInStandby(targetPlayTime)`.
   * Além disso, `getActiveStreamForTime(t)` recorria aos dados globais da loja (`store_info.external_stream_url`) como fallback mesmo quando fora do expediente (`sched == nil`), fazendo com que o streaming permanecesse ativo 24 horas por dia.
2. **Desativação do Relay Liquidsoap fora do horário:**
   * No perfil `SERVER_STREAMING` / `LOCAL_STREAMING`, o Liquidsoap executa o stream via `input.http`. A rotina `StartExternalStreamScheduler` precisa enviar o comando `/set_stream_url?url=none` para desarmar o relay e entrar em `blank()` (silêncio digital 0 dB) às 23:00, e disparar `/fade_to_stream?url=...` às 08:00 para reabrir a transmissão suavemente.
3. **Web Player (`imc-server`):**
   * O payload da API `/api/status` e a renderização inicial do Web Player precisam ocultar a caixa de streaming quando a loja estiver em `is_standby: true`.

---

## 2. Mudanças Propostas

### Repositório `imc-audio-local`

#### [MODIFY] [scheduler.go](file:///home/paulo-assis/imc-audio-local/scheduler.go)
1. **`getActiveStreamForTime(t time.Time)`**:
   * Adicionar verificação inicial de `IsStoreInStandby(t)`: se `isStandby && offPolicy == "SILENCE"`, retornar imediatamente `"", ""` para impedir que a URL de streaming persista fora do horário.
   * No fallback onde `sched == nil`, verificar se a loja está em standby antes de consultar `store_info`.
2. **`GetNextTrack()`**:
   * Mover o check `IsStoreInStandby(targetPlayTime)` para o topo da função, antes de qualquer verificação de streaming ou música local.
3. **`StartExternalStreamScheduler()`**:
   * Garantir que, ao detectar entrada em standby (`activeStreamURL == ""`), comande `/set_stream_url?url=none` na porta Harbor do Liquidsoap para cessar a transmissão externa e manter o playout em 0 dB.
   * Ao retornar o horário de expediente (`activeStreamURL != ""`), acionar `/fade_to_stream` com a URL ativa para restabelecer a web rádio com transição suave.

#### [MODIFY] [main.go](file:///home/paulo-assis/imc-audio-local/main.go)
* Ao iniciar o processo, verificar se `IsStoreInStandby(time.Now())` é verdadeiro com política `SILENCE`. Se for, garantir que `cfg.ExternalStreamURL` inicialize vazio para que o Liquidsoap suba em estado de silêncio/standby (`blank()`).

#### [MODIFY] [web_dashboard.go](file:///home/paulo-assis/imc-audio-local/web_dashboard.go)
* Em `handleAPIStatus`, quando a loja estiver em standby (`isStandby && offPolicy == "SILENCE"`), forçar `currentTrack.Title = "Sem reprodução"`, `currentTrack.Type = "STANDBY"` e zerar tempos.

#### [MODIFY] [scheduler_test.go](file:///home/paulo-assis/imc-audio-local/scheduler_test.go)
* Adicionar testes unitários cobrindo o comportamento de `getActiveStreamForTime` e `GetNextTrack` durante o horário de expediente e durante o standby/silêncio.

---

### Repositório `imc-server`

#### [MODIFY] [customer_player_handler.go](file:///home/paulo-assis/imc-server/internal/handlers/customer_player_handler.go)
* Em `HandleCustomerPlayerPage`, consultar o status do player local (`proxyStoreRequest(store, "/api/status", "GET", nil)`) e, se em standby, pré-preencher os dados da página com `InitialTitle = "🌙 Standby (Fora do Horário)"` e `InitialArtist = "Retorna às HH:MM:SS"`.
* Em `HandlePlayerNowPlayingAPI`, assegurar que `is_streaming: false` quando `is_standby: true`.

#### [MODIFY] [customer_player.html](file:///home/paulo-assis/imc-server/web/templates/customer_player.html)
* Ocultar a caixa `#streaming-box` e o botão de pular faixa quando `data.is_standby` for verdadeiro.

---

## 3. Plano de Verificação

### Testes Automatizados
1. Executar os testes unitários em `imc-audio-local`:
   ```bash
   cd /home/paulo-assis/imc-audio-local && go test -v ./...
   ```

### Testes Manuais & Validação Local
1. Compilar o novo binário de `imc-audio-local`.
2. Copiar o binário para `/home/paulo-assis/imc-player-lab/imc-audio-local`.
3. Reiniciar a instância do laboratório (`imc-player-lab`).
4. Testar a transição fora do horário (ex: simular grade terminando às 23:00 e iniciando às 08:00):
   * Consultar `http://127.0.0.1:5010/api/status` e verificar `is_standby: true`, `standby_reason: "SILENCE"`, `next_open_time: "08:00:00"`.
   * Verificar se o endpoint Harbor `/getmeta` retorna silêncio/blank sem erros.
5. Testar o Web Player no `imc-server` via túnel ou porta local:
   * Validar exibição da mensagem `🌙 Standby (Fora do Horário) - Retorna às 08:00:00`.
