# Plano de Implementação - Relocalização do Script Liquidsoap e Correção do Sincronismo da Linha do Tempo

Este plano visa realizar duas melhorias integradas no player local:
1. **Relocalização do Script Liquidsoap:** Mover o script do Liquidsoap (`player.liq`) e seu template de geração (`player.liq.tmpl`) para dentro da estrutura do repositório (`C:\imc-audio-local`), controlando o template no Git.
2. **Correção do Sincronismo (Timeline real-time):** Corrigir a divergência onde músicas/comerciais pré-carregados pelo Liquidsoap apareciam como "Executados" no futuro. Faremos isso introduzindo um callback em tempo real (`/started`) do Liquidsoap para o servidor Go no exato momento em que o áudio começa a tocar.

---

## User Review Required

Nenhuma alteração de impacto crítico ou quebra de compatibilidade está prevista. A autossuficiência do executável final via `go:embed` é preservada.

> [!NOTE]
> * **Interface Visual:** O front-end da dashboard já tem suporte visual para exibir itens na fila como `⏳ Na Fila`. Com esta mudança, as músicas e comerciais pré-carregados aparecerão com esse status em amarelo na linha do tempo, e mudarão para verde `✓ Executada` ou ciano `▶ Na Ponta` exatamente em tempo real.
> * **Git:** O arquivo gerado final (`player.liq`) continuará sendo ignorado no `.gitignore` por conter caminhos absolutos locais, mas seu template (`player.liq.tmpl`) será controlado no repositório.

---

## Proposed Changes

### [Component: Servidor de Decodificação e Métricas (Go)]

#### [NEW] [player.liq.tmpl](file:///C:/imc-audio-local/player.liq.tmpl)
* Contém a configuração exata do Liquidsoap.
* **Ajuste:** No método `on_meta(m)`, adicionaremos uma chamada HTTP para notificar o Go quando a faixa realmente começar:
  ```liq
  id = m["id"]
  if id != "" then
    ignore(http.get("http://127.0.0.1:{{.LocalPort}}/started?id=#{id}"))
  end
  ```

#### [MODIFY] [watchdog.go](file:///C:/imc-audio-local/watchdog.go)
* Importar o pacote `embed`.
* Substituir a string constante `liqTemplate` pela diretiva `//go:embed player.liq.tmpl` para embutir o template no executável final.
* Remover a string constante longa inline de `GenerateLiquidsoapScript`.

#### [MODIFY] [decrypter.go](file:///C:/imc-audio-local/decrypter.go)
* Adicionar a rota HTTP `/started` que recebe o `id` da faixa que começou a tocar no Liquidsoap e invoca `MarkTrackAsPlaying(id)`.

#### [MODIFY] [scheduler.go](file:///C:/imc-audio-local/scheduler.go)
* **Status de Inserção:** Em `recordPlayHistory`, alterar o status de inserção padrão de `"na_ponta"` para `"na_fila"`. Remover a query que marcava o item anterior como executado na hora da requisição (isso agora será feito apenas quando a música realmente começar).
* **Novo Método:** Adicionar `MarkTrackAsPlaying(id string) error` que:
  1. Identifica o registro no histórico (`play_history`) correspondente ao ID que está iniciando.
  2. Atualiza os registros mais antigos que ainda estão pendentes para `"executada"`.
  3. Atualiza o status do registro atual para `"na_ponta"` e define a coluna `played_at` com o horário atual real (`time.Now()`).

#### [MODIFY] [config.go](file:///C:/imc-audio-local/config.go)
* Atualizar o fallback interno padrão de `ScriptPath` de `C:\liquidsoap-2.4.0-win64\player.liq` para `./player.liq`.

#### [MODIFY] [config.ini](file:///C:/imc-audio-local/config.ini)
* Alterar o caminho do script gerado para `./player.liq`.

#### [MODIFY] [config.ini.example](file:///C:/imc-audio-local/config.ini.example)
* Alterar a documentação do exemplo do caminho do script para `./player.liq`.

#### [MODIFY] [.gitignore](file:///C:/imc-audio-local/.gitignore)
* Adicionar `player.liq` para garantir que o script gerado localmente não seja enviado ao repositório.

---

## Verification Plan

### Automated Tests
* Compilar a aplicação para validar a sintaxe de Go:
  ```bash
  go build -o imc-audio-local.exe
  ```

### Manual Verification
1. **Verificar Geração de Arquivo:** Iniciar o player e garantir que o `./player.liq` é gerado na raiz do diretório do repositório.
2. **Verificar Comportamento da Linha do Tempo:** 
   * Abrir a dashboard web no navegador.
   * Acompanhar a aba **Execução Real**.
   * Verificar se as faixas futuras pré-carregadas aparecem corretamente com o status `⏳ Na Fila` (amarelo) com o horário estimado em que vão tocar.
   * Verificar se, ao iniciar o áudio, o status da faixa atual muda para `▶ Na Ponta` (ciano) e o horário (`played_at`) é atualizado para o segundo real em que ela iniciou, movendo a faixa anterior para `✓ Executada` (verde).
3. **Validar Git:** Rodar `git status` para garantir que o repositório está limpo e apenas as alterações pretendidas e o novo arquivo `.tmpl` estão rastreados.
