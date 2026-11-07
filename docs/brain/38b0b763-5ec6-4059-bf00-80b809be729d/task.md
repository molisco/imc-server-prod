# Tarefas - Relocalização do Script Liquidsoap e Correção do Sincronismo da Linha do Tempo

- [x] Criar o arquivo de template `player.liq.tmpl` com as configurações do Liquidsoap e a chamada HTTP de callback `/started` ([player.liq.tmpl](file:///C:/imc-audio-local/player.liq.tmpl))
- [x] Modificar `watchdog.go` para importar `embed` e carregar o template usando `//go:embed player.liq.tmpl` ([watchdog.go](file:///C:/imc-audio-local/watchdog.go))
- [x] Modificar `decrypter.go` para adicionar o endpoint `/started` que recebe o id da faixa e chama `MarkTrackAsPlaying` ([decrypter.go](file:///C:/imc-audio-local/decrypter.go))
- [x] Modificar `scheduler.go` para:
  - [x] Alterar o status padrão de inserção de `"na_ponta"` para `"na_fila"` e não remover o `"na_ponta"` anterior no pre-fetch
  - [x] Adicionar a função `MarkTrackAsPlaying(id string) error` que atualiza os estados do SQLite em tempo real
- [x] Atualizar os arquivos de configuração:
  - [x] Modificar `config.go` para mudar o default do `ScriptPath` de `C:\liquidsoap-2.4.0-win64\player.liq` para `./player.liq` ([config.go](file:///C:/imc-audio-local/config.go))
  - [x] Modificar `config.ini` local para usar `ScriptPath=./player.liq` ([config.ini](file:///C:/imc-audio-local/config.ini))
  - [x] Modificar `config.ini.example` para usar `ScriptPath=./player.liq` ([config.ini.example](file:///C:/imc-audio-local/config.ini.example))
- [x] Atualizar o `.gitignore` para ignorar o arquivo gerado `./player.liq` ([.gitignore](file:///C:/imc-audio-local/.gitignore))
- [x] Compilar, iniciar o player e validar as alterações:
  - [x] Parar qualquer instância anterior do daemon
  - [x] Compilar com `go build`
  - [x] Iniciar o daemon em background
  - [x] Verificar a geração de `./player.liq`
  - [x] Abrir a dashboard e monitorar o status em tempo real (transição de `Na Fila` para `Na Ponta` e `Executada`)
  - [x] Commit e Push dos arquivos no Git
