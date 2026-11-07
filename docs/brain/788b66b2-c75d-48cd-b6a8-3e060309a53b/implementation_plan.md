# Plano de Implementação - Proteção contra Loop de Pulos do Liquidsoap (Revisado)

Durante a hibernação ou alteração do dispositivo de áudio (fone de ouvido / HDMI), o Liquidsoap pode entrar em um loop rápido de pulo de faixas porque o hardware de saída de áudio para de fornecer um clock de sincronização (o PortAudio consome as amostras instantaneamente).

Propomos implementar uma detecção inteligente e reinicialização automática do Liquidsoap quando este comportamento for identificado, **desconsiderando pulos legítimos acionados pelo usuário (botão Skip) ou rotinas automáticas de sincronização/eventos**.

---

## Mudanças Propostas

### 1. Registro de Pulos Intencionais (`scheduler.go`)
- Adicionar uma variável global de controle para registrar o timestamp do último pulo legítimo.
- Criar funções `RegisterIntentionalSkip()` e `IsIntentionalSkipActive()` para ignorar a proteção de watchdog temporariamente (janela de 10 segundos) após um pulo legítimo ser solicitado.

### 2. Integração com Comandos de Skip
Registrar o pulo legítimo em todas as três origens de skip do sistema:
- **Dashboard (Manual):** Em `handleAPISkip` (`web_dashboard.go`).
- **Auto-Catchup (Automático por atraso):** Em `StartCatchupMonitor` (`scheduler.go`).
- **Interrupção de Eventos (Automático por agenda):** Em `StartHardEventMonitor` (`scheduler.go`).

### 3. Monitoramento de Pulos Rápidos no Scheduler (`scheduler.go`)
- Se uma música começar e o watchdog detectar que um skip legítimo foi registrado nos últimos 10 segundos, o histórico de pulos é resetado.
- Caso contrário, monitoramos os inícios de músicas normais. Se **3 músicas consecutivas** iniciarem em um intervalo menor que **60 segundos** no total, o watchdog de reinicialização é acionado.

### 4. Reinicialização no Watchdog (`watchdog.go`)
- Implementar a função `RestartLiquidsoap()` para forçar o fechamento do Liquidsoap (`taskkill`). A rotina principal de monitoramento irá reiniciá-lo automaticamente 5 segundos depois.

---

## Detalhes das Alterações

### [MODIFY] [watchdog.go](file:///C:/imc-audio-local/watchdog.go)

Adicionar a variável de controle e a função de reinicialização:

```go
var (
	liquidsoapName string
)

// RestartLiquidsoap forcefully terminates the Liquidsoap process so the watchdog loop restarts it
func RestartLiquidsoap() {
	name := liquidsoapName
	if name == "" {
		name = "liquidsoap.exe"
	}
	LogLocal("WARNING", "[Watchdog] Forcefully restarting Liquidsoap due to detected audio loop/skip bug...")
	_ = exec.Command("taskkill", "/F", "/IM", name).Run()
}
```

Atualizar o início do watchdog para gravar o nome do processo:

```go
func StartLiquidsoapWatchdog(cfg *Config) {
	// Terminate any existing orphaned Liquidsoap process before starting a new one
	liqName := filepath.Base(cfg.LiquidsoapPath)
	liquidsoapName = liqName
	LogLocal("INFO", "Ensuring no previous instances of %s are running...", liqName)
	_ = exec.Command("taskkill", "/F", "/IM", liqName).Run()
...
```

---

### [MODIFY] [scheduler.go](file:///C:/imc-audio-local/scheduler.go)

Adicionar controle de pulos intencionais e histórico:

```go
var (
	recentMusicStarts       []time.Time
	lastIntentionalSkipTime time.Time
	intentionalSkipMutex    sync.Mutex
)

func RegisterIntentionalSkip() {
	intentionalSkipMutex.Lock()
	lastIntentionalSkipTime = time.Now()
	intentionalSkipMutex.Unlock()
}

func IsIntentionalSkipActive() bool {
	intentionalSkipMutex.Lock()
	defer intentionalSkipMutex.Unlock()
	return !lastIntentionalSkipTime.IsZero() && time.Since(lastIntentionalSkipTime) < 10*time.Second
}
```

Acrescentar `RegisterIntentionalSkip()` no monitor de Catchup e Hard Event (onde enviam `/flush_and_skip`).

Na função `MarkTrackAsPlaying(id string)`, substituir a lógica de música pelo watchdog inteligente:

```go
	} else {
		LogLocal("INFO", "[Volume] Restaurando volume do mixer para o padrão (1.0) para: %s", displayName)
		
		// Watchdog de proteção contra loop rápido de pulo (hibernação/bug de áudio)
		if IsIntentionalSkipActive() {
			recentMusicStarts = nil // Limpa o histórico pois o pulo foi intencional
		} else {
			recentMusicStarts = append(recentMusicStarts, now)
			if len(recentMusicStarts) > 3 {
				recentMusicStarts = recentMusicStarts[1:]
			}
			if len(recentMusicStarts) == 3 {
				timeSpan := recentMusicStarts[2].Sub(recentMusicStarts[0])
				if timeSpan < 60*time.Second {
					recentMusicStarts = nil // Reseta para evitar múltiplos disparos
					LogLocal("WARNING", "[Watchdog] Rate of music track starts is too high (3 music tracks in %v). Liquidsoap might be in a fast-skipping loop. Restarting...", timeSpan)
					go RestartLiquidsoap()
				}
			}
		}
	}
```

---

### [MODIFY] [web_dashboard.go](file:///C:/imc-audio-local/web_dashboard.go)

Registrar o pulo manual em `handleAPISkip`:

```go
func handleAPISkip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	LogLocal("INFO", "Web Dashboard: Skip current track command triggered")
	RegisterIntentionalSkip()
	client := &http.Client{Timeout: 500 * time.Millisecond}
...
```

---

## Plano de Verificação

### Testes Manuais
1. Compilar o executável com `go build`.
2. Rodar a aplicação.
3. Testar o botão **Skip** no dashboard várias vezes consecutivas (com intervalos de 2 a 3 segundos) para confirmar que o Liquidsoap **não** é reiniciado (o histórico é resetado a cada skip manual).
4. Induzir 3 pulos rápidos sem clicar no botão de skip para certificar-se de que o watchdog reinicia o Liquidsoap após a 3ª música iniciar rápido demais.
