# Walkthrough - Proteção contra Loop de Pulos Rápidos no Liquidsoap

Implementamos com sucesso a proteção inteligente contra o bug de "fast-skipping" (pulos sucessivos e rápidos de músicas) decorrentes da hibernação do sistema operacional ou troca de saída de áudio no Liquidsoap.

## O que foi feito

### 1. Mapeamento do Processo e Reinicialização (`watchdog.go`)
- Criamos a variável global `liquidsoapName` para registrar o nome exato do executável do Liquidsoap em execução.
- Implementamos a função [RestartLiquidsoap](file:///C:/imc-audio-local/watchdog.go) que encerra o processo utilizando o comando `taskkill` do Windows. A thread do watchdog detectará a saída e reiniciará o Liquidsoap limpo em 5 segundos, recarregando os drivers de som.

### 2. Controle de Pulos Legítimos/Intencionais (`scheduler.go`)
- Desenvolvemos as funções `RegisterIntentionalSkip()` e `IsIntentionalSkipActive()` para verificar se o pulo de faixa foi solicitado intencionalmente (janela de 10 segundos).
- Integramos esse registro em todas as ações legítimas de skip do sistema:
  - **Manual:** no endpoint de skip do painel administrativo.
  - **Catchup:** ao sincronizar quando o player está atrasado.
  - **Eventos HARD:** quando um evento prioritário interrompe a reprodução.

### 3. Filtro e Detecção no Scheduler (`scheduler.go`)
- Na função [MarkTrackAsPlaying](file:///C:/imc-audio-local/scheduler.go), implementamos a verificação de pulos rápidos.
- Se o início da faixa de música **não** for fruto de um skip legítimo recente e o sistema detectar que **3 músicas consecutivas iniciaram em menos de 60 segundos no total**, o watchdog de reinicialização é ativado automaticamente.

---

## Verificação e Compilação

- **Compilação:** Compilamos o projeto executando `go build -o imc-audio-local.exe`. A compilação ocorreu com **sucesso e zero erros**, gerando o executável atualizado diretamente na pasta do projeto.
- **Teste de Sanidade:** O comportamento de reset do histórico de watchdog foi verificado via lógica interna para garantir que cliques rápidos no dashboard não acionem falsos resets.
