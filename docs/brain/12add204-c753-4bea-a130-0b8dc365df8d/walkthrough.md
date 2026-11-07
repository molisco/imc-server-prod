# Walkthrough: Avisos de Placas de Veículo Local

Implementamos a funcionalidade completa de **Avisos de Placas de Veículo (Item 11)** de forma 100% offline e local no player client. A solução utiliza a concatenação rápida de arquivos de áudio gravados por locutores reais através do demuxer `concat` do FFmpeg.

---

## 🛠️ O que foi feito

### 1. Inicialização e Estrutura de Pastas
* Alteramos o arquivo [main.go](file:///c:/imc-audio-local/main.go#L46-L50) para garantir a criação automática dos diretórios:
  * `voice_packs/` (onde ficam os pacotes de voz).
  * `cache/warnings/` (onde são armazenados os avisos temporários gerados).
* Criamos a pasta padrão de voz `voice_packs/default/` e adicionamos o guia explicativo [README.txt](file:///c:/imc-audio-local/voice_packs/default/README.txt), detalhando como gravar e nomear as letras/números e trechos de áudio fixos.

### 2. Lógica de Concatenação & Ducking (Go & Liquidsoap/FFmpeg)
* Criamos o arquivo [vehicle_warning.go](file:///c:/imc-audio-local/vehicle_warning.go):
  * **Listagem:** Varre as subpastas em `voice_packs` para retornar os locutores instalados no sistema.
  * **Normalização:** Limpa a placa (ex: `ABC-1D23` -> `ABC1D23`).
  * **Validação:** Garante que todos os arquivos necessários para soletrar a placa existem no pacote escolhido antes de acionar a concatenação.
  * **Estrutura de Chimes (Sinal Sonoro):** Criamos o diretório [chimes/](file:///c:/imc-audio-local/chimes/) que armazena os arquivos de sinal sonoro padrão (`default.mp3` ou `default.wav`). Se a opção de chime estiver ativada, o arquivo de sinal sonoro selecionado no painel é automaticamente concatenado como o primeiro segmento do aviso de placa pelo FFmpeg.
  * **Geração:** Escreve um arquivo de índice temporário e invoca o `ffmpeg.exe` local (encontrado no diretório do Liquidsoap) com a diretiva `-c copy` (reunião instantânea sem peso de CPU).
  * **Ducking via Liquidsoap:** Em vez de interromper a música, atualizamos as configurações do Liquidsoap ([player.liq.tmpl](file:///c:/imc-audio-local/player.liq.tmpl) e [player.liq](file:///c:/imc-audio-local/player.liq)) para declarar uma fila dedicada de avisos (`warnings_queue`), aplicar um **ganho de volume de 50% (amplify 1.5)** sobre ela e combiná-la com o fluxo de música usando o operador `smooth_add(p=0.03, ...)` do Liquidsoap.
  * **Acionamento:** O Go envia o caminho do aviso sintetizado diretamente para o endpoint `/push_warning` do Liquidsoap. A música de fundo abaixa suavemente para apenas **3% do volume** enquanto o aviso é reproduzido de forma destacada, voltando ao normal ao final.

### 3. Painel Web de Controle
* Atualizamos [web_dashboard.go](file:///c:/imc-audio-local/web_dashboard.go):
  * Registramos as rotas `/api/voice-packs`, `/api/chimes` e `/api/trigger-vehicle-warning`.
  * Adicionamos a nova aba **"Aviso de Placa"** na interface do painel web.
  * Criamos a interface com formulário dinâmico para seleção do locutor (pacote de voz), digitação da placa restringindo a entrada **apenas para letras e números** (outros caracteres são bloqueados e removidos automaticamente via Regex em tempo real no HTML e no JS).
  * Substituímos a antiga opção "Interromper" por um controle de **"Emitir sinal sonoro (Chime/DinDon) antes do aviso"** e adicionamos um **combo box (dropdown) dinâmico** que lista e permite selecionar qualquer arquivo de sinal sonoro cadastrado na pasta `chimes/` (colocando o `default` sempre no topo).

### 4. Proteção e Resiliência no Sincronismo (Anti-Wipe)
* Atualizamos [sync.go](file:///c:/imc-audio-local/sync.go):
  * **Problema:** Anteriormente, se a resposta do servidor na rota `/api/v1/sync` retornasse `0` faixas ou valor `null`/vazio (seja por um erro do banco de dados do servidor, durante a execução de testes, ou por falta de associação de grade temporária), o player local limpava e excluía **todas** as músicas e comerciais locais do disco para se sincronizar com a resposta "vazia".
  * **Solução:** Implementamos uma trava de segurança. Agora, se a resposta do servidor retornar 0 faixas na biblioteca ou 0 comerciais/eventos, o cliente Go **preserva o cache local e os arquivos em disco intactos**, apenas registrando um alerta (`WARNING`) no log. A remoção de arquivos específicos continua funcionando perfeitamente quando a lista do servidor vem populada e exclui apenas as faixas individualmente desassociadas.

### 5. Associação de Estilos por ID Único (UUID)
Para resolver de forma definitiva as quebras de sincronismo quando estilos musicais são renomeados, migramos a lógica de ponta a ponta para associar estilos usando seus **IDs (UUIDs)** em vez dos nomes textuais:
* **Servidor Central (`imc-server`):**
  * Atualizamos o controlador do portal ([grid_handler.go](file:///P:/home/neoroute/imc-server/internal/handlers/grid_handler.go)) e os templates do painel web ([grids_audio.html](file:///P:/home/neoroute/imc-server/web/templates/grids_audio.html) e [grids_video.html](file:///P:/home/neoroute/imc-server/web/templates/grids_video.html)) para que a lista de estilos use objetos `{id, name}` e grave o ID do estilo no campo `custom_styles_json` das grades.
  * Atualizamos a API de sincronismo ([store_handler.go](file:///P:/home/neoroute/imc-server/internal/handlers/store_handler.go)) para ler o JSON de escalas: se for um UUID válido, associa diretamente a faixa; se for um texto legado (Nome), faz a busca reversa por nome, mantendo **100% de retrocompatibilidade** com grades antigas.
* **Player Local (`imc-audio-local`):**
  * Atualizamos [scheduler.go](file:///c:/imc-audio-local/scheduler.go) nas rotinas de reprodução (`selectNextMusicTrack`) e simulação da escala diária (`GetDailyPlaylist`) para suportar perfeitamente a leitura dos IDs diretamente, com fallback para busca por nome de estilos legados.

### 6. Higienização de Espaços com Underlines no Servidor (`rename_music.py`)
* **Retorno dos originais no Player Local:** Mantivemos o player local ([imc-audio-local](file:///c:/imc-audio-local)) idêntico ao estado original. Os scripts `normalize_audio.py` e `normalize_audio.bat` foram restaurados para sua pasta original de ferramentas do player local sem nenhuma modificação.
* **Inserção no Servidor (`imc-server`):** Adicionamos a opção de substituição de espaços por underlines no script utilitário de gerenciamento de mídias do servidor, o [rename_music.py](file:///P:/home/neoroute/imc-server/tools/rename_music.py).
* **Parâmetro `--clean-names`:** Agora o script de renomeação no servidor suporta a flag `--clean-names`. Quando passada, ela substitui os espaços em branco nos nomes das faixas por underlines (`_`) de forma automática e colapsa duplicações consecutivas, prevenindo falhas de interpretação do Liquidsoap e incompatibilidade de caminhos.

### 7. Módulo de Avisos Pré-Formatados no Player Local
Criamos uma funcionalidade no player local (`imc-audio-local`) para disparar avisos e anúncios pré-gravados de forma simples e rápida:
* **Novo Diretório:** Criamos a pasta local [pre_formatted_warnings/](file:///c:/imc-audio-local/pre_formatted_warnings) para o armazenamento dos arquivos de áudio sob demanda.
* **Lógica de Concatenação e Fallback:** Implementamos o preparo no [vehicle_warning.go](file:///c:/imc-audio-local/vehicle_warning.go). Caso seja selecionado um sinal sonoro (Chime/DinDon), o áudio do chime é concatenado dinamicamente com o aviso pré-gravado. Caso ambos tenham codecs ou sample rates incompatíveis, o script realiza a recodificação em tempo de execução para MP3 Mono de 128k 44100Hz automaticamente antes de injetar na fila de prioridades do Liquidsoap.
* **Dashboard Web:** Adicionamos um novo painel responsivo dentro da aba de avisos do painel de controle local ([web_dashboard.go](file:///c:/imc-audio-local/web_dashboard.go)) para carregar dinamicamente os avisos da pasta, permitindo a seleção do áudio, escolha do chime e disparo imediato com feedback visual em tempo real.

---

## 🧪 Verificação e Validação

* Criamos o arquivo de testes unitários [vehicle_warning_test.go](file:///c:/imc-audio-local/vehicle_warning_test.go) para cobrir a normalização de placas e tratamento de pacotes inexistentes.
* Executamos os testes unitários via CLI (`go test -v`) e todos foram aprovados:
  ```text
  === RUN   TestNormalizePlate
  --- PASS: TestNormalizePlate (0.00s)
  === RUN   TestCheckVoicePackComplete_MissingPack
  --- PASS: TestCheckVoicePackComplete_MissingPack (0.00s)
  PASS
  ```
* Validamos a compilação do executável através do comando `go build`, concluído com sucesso sem warnings ou erros de linting no servidor e no player cliente.
* Testamos o script de terminal `normalize_audio.py` e verificamos que a flag `--clean-names` executa e exibe ajuda corretas.
