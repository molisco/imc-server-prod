# Walkthrough: Transições de Áudio Suaves (Fades) e Ajustes de Reinicialização

Nesta etapa, ajustamos as transições de áudio no player local para garantir que comerciais e músicas não cortem de forma brusca (sílabas cortadas), além de resolver um problema onde comerciais eram pulados após reiniciar o player. Também implementamos o descarte inteligente do buffer do Liquidsoap durante o processo de sincronia de tempo (*catchup*) e a otimização da regra de segurança para pré-carregamento.

## Alterações Realizadas

### 1. Ajustes do Liquidsoap (Transição de Áudio)

#### [player.liq.tmpl](file:///c:/imc-audio-local/player.liq.tmpl)
* Envelopamos a fonte dinâmica em um `buffer(source)` para isolar sua execução em uma thread de clock dedicada.
* Isso permitiu usar o operador `crossfade` com segurança (`fade_in=0.2`, `fade_out=0.1`, `duration=0.3`), prevenindo completamente o deadlock de clock que acontecia no driver Windows WMM (`output.ao`).
* A transição agora possui sobreposição suave, evitando o corte seco de sílabas no final dos áudios.

```liquidsoap
source = request.dynamic(id="go_source", get_next)
source = buffer(source)
source = crossfade(fade_in=0.2, fade_out=0.1, duration=0.3, source)
source = mksafe(source)
```

---

### 2. Ajustes de Banco de Dados (Comerciais Pulados)

#### [db.go](file:///c:/imc-audio-local/db.go)
* Criada a função `ResetQueuedSlots()`, que redefine slots que ficaram no estado `'QUEUED'` de volta para `'PENDING'` quando o processo inicia.

#### [main.go](file:///c:/imc-audio-local/main.go)
* Chamada a nova função `ResetQueuedSlots()` logo na inicialização, corrigindo o problema de pulo de comerciais decorrente do reinício do player.

---

### 3. Melhoria no Auto-Catchup: Descarte de Faixas Pré-Carregadas (Flush & Skip)

#### [player.liq.tmpl](file:///c:/imc-audio-local/player.liq.tmpl)
* Criado o novo endpoint harbor HTTP `/flush_and_skip`. Esse endpoint executa internamente o comando do Liquidsoap `go_source.flush_and_skip` para descartar qualquer faixa que esteja pré-carregada no buffer de reprodução de forma imediata e interromper a faixa atual.

#### [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* Modificamos o monitor de sincronia (`StartCatchupMonitor`) para chamar `/flush_and_skip` em vez de `/skip` ao forçar a entrada de comerciais.
* Isso garante que, se o player precisar pular uma música por conta de atraso acumulado, ele pulará direto para os comerciais sem reproduzir a faixa subsequente que já havia sido pré-carregada pelo Liquidsoap.

---

### 4. Otimização da Regra de Segurança para Lookahead (Pré-carregamento)

#### [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* **Problema:** A regra de segurança que garante pelo menos 3 minutos de música entre blocos comerciais estava sendo avaliada usando o tempo real atual (`time.Since`). Quando o Liquidsoap pedia a próxima música para o buffer em avanço (lookahead), a segurança era avaliada incorretamente como ativa (pois o comercial anterior tinha acabado de tocar poucos segundos atrás no relógio real). Isso forçava o agendamento de uma segunda música longa desnecessariamente e aumentava o atraso geral.
* **Ajuste:** Alteramos todas as validações de segurança em `GetNextTrack` and `selectNextMusicTrack` para calcular a separação com base no **horário planejado de início da faixa** (`targetPlayTime` ou `t`) em vez do relógio real atual:
  ```go
  targetPlayTime.Sub(lastCommercialPlayedAt) < 3*time.Minute
  ```
* **Resultado:** Agora, ao pré-carregar uma música que tocará no futuro, o sistema percebe que os 3 minutos de segurança já terão se passado quando ela tocar. Se houver atraso na grade, o scheduler agendará **apenas uma música** e deixará o bloco de comerciais engatilhado imediatamente na sequência no buffer do Liquidsoap, permitindo que a compensação do atraso ocorra de forma automática e preventiva.

---

# Walkthrough Suplementar: Correção de Erros 404 e Filtro de Arquivos Temporários (.master.mp3)

Nesta etapa suplementar, investigamos e resolvemos a falha em loop de download (Erros 404) relatada no player local para a faixa contendo `.master.mp3` no nome do arquivo.

## Diagnóstico do Problema

1. **Divergência de Arquivos**: O banco de dados do servidor possuía uma música cadastrada com o nome `[Intro Mix] DJ Bobo - Love Is All Around (Remix 2025).mp3.master.mp3`, porém esse arquivo temporário não existia mais no disco do servidor (apenas o arquivo `.mp3` original).
2. **Origem da Inserção**: O pipeline de masterização do servidor cria temporariamente arquivos `.master.mp3` antes de renomeá-los e substituir os originais. Se o processo do servidor sofrer uma interrupção ou falha durante essa renomeação, o arquivo `.master.mp3` permanece no disco. Em um escaneamento subsequente de pastas, a varredura (`strings.HasSuffix(name, ".mp3")`) acabava incluindo e cadastrando esses arquivos `.master.mp3` temporários como novas faixas na biblioteca.

## Solução Aplicada

### 1. Limpeza do Banco de Dados PostgreSQL do Servidor
* Desenvolvemos e executamos uma ferramenta em Go (`cmd/clean_db/main.go`) que conectou ao banco central no servidor e removeu o registro órfão que apontava para o arquivo `.master.mp3`.

### 2. Correção do Scanner do Servidor (Evitar Cadastros de Arquivos Temporários)
#### [music_style_handler.go](file:///P:/imc-server/internal/handlers/music_style_handler.go)
* Refatoramos o filtro da listagem de arquivos da varredura de pastas para ignorar qualquer arquivo que contenha `.master.` em seu nome, bem como arquivos ocultos ou de metadados do sistema (iniciados com `.` ou `._`).

### 3. Robustez na Exclusão de Arquivos Temporários (Prevenção)
#### [metadata.go](file:///P:/imc-server/internal/utils/metadata.go)
* Adicionamos rotinas de limpeza (`os.Remove(tempPath)`) nos caminhos de tratamento de erro do `MasterizeAudio` para garantir que o arquivo `.master.mp3` seja deletado caso a remoção do original ou a renomeação falhe.

### 4. Limpeza Física e Recompilação
* Removemos fisicamente o arquivo temporário órfão `John Cafferty.-Voice of americas sons.mp3.master.mp3` encontrado no diretório do servidor.
* Recompilamos o binário do servidor para Linux (`GOOS=linux GOARCH=amd64`) e reiniciamos o serviço central no servidor `192.168.15.250`.

## Resultados e Validação

* Executamos o sincronismo manual no player local enviando o comando `r` no console. O registro do arquivo `.master.mp3` foi deletado do banco de dados local.
* O loop de erro 404 foi interrompido imediatamente e o player voltou a baixar a biblioteca de músicas pendentes com sucesso (avançando rapidamente de 562 para 656 faixas baixadas e tocando áudios normalmente).

---

# Walkthrough Suplementar: Sincronização Restrita de Estilos Programados e Limpeza de Diretórios

Nesta etapa suplementar, ajustamos o sincronismo do servidor e do player para garantir que apenas os estilos musicais programados nas agendas da grade da filial sejam sincronizados e baixados. Também implementamos a exclusão física dos subdiretórios de estilos vazios/removidos do player local.

## Alterações Realizadas

### 1. Filtro de Estilos Ativos no Servidor (Download e Sincronismo)
#### [store_handler.go](file:///P:/imc-server/internal/handlers/store_handler.go)
* Refatoramos o endpoint `/api/v1/sync` para ler e analisar o campo `CustomStylesJSON` de todas as agendas associadas à grade de programação semanal da loja.
* Extraímos os nomes dos estilos e resolvemos seus IDs correspondentes.
* Alteramos as queries de busca de estilos e faixas de mídia para incluir a restrição `music_style_id = ANY($1)`. Isso impede que o servidor envie faixas de estilos não programados (como `EURODANCE 80` no cenário relatado) para o player local.

### 2. Exclusão Física de Diretórios de Estilo Vazios no Cliente
#### [sync.go](file:///c:/imc-audio-local/sync.go)
* Implementamos a função auxiliar `cleanEmptyStyleDirs` que verifica todas as subpastas em `music/`. Se uma subpasta de estilo estiver vazia (com 0 arquivos) decorrente de exclusões de faixas não mais programadas, ela é excluída fisicamente do HD local (`os.Remove`).
* Adicionamos a execução desse cleanup logo após o commit bem-sucedido da transação de sincronização do player.

## Resultados e Validação

* Recompilamos o servidor central e o reiniciamos na máquina host.
* Recompilamos o player local (`go build -o imc-audio-local.exe`) e o reiniciamos.
* O player local sincronizou e removeu as referências e arquivos do estilo `EURODANCE 80` da base local.
* Os testes em disco confirmaram a remoção da pasta do estilo sob `music/` (`music/06fd824a-af88-4bd2-8b4a-5de8213d6b78` foi totalmente excluída do HD).

---

# Walkthrough Suplementar: Correção de Recálculo de SONIC para Reprocessamento Individual e Cues Manuais

Nesta etapa suplementar, corrigimos a regra de atribuição do selo `SONIC` no servidor central, garantindo que faixas com ajustes manuais de início/fim (`ManualCue = true`) tenham sua elegibilidade SONIC calculada com base nos pontos de execução finais salvos no banco de dados.

## Diagnóstico do Problema

1. **Problema de Cálculo**: Quando o usuário acionava a análise ou o reprocessamento de faixas de forma individual na interface administrativa, a flag `is_sonic` era atualizada a partir do objeto `meta.IsSonic` retornado por `ExtractMetadata`.
2. **Incompatibilidade com Cues Manuais**: O objeto `meta.IsSonic` utiliza a duração detectada automaticamente pelo FFmpeg (`meta.CueOut - meta.CueIn`). Se uma faixa tivesse `ManualCue = true`, os valores de `CueIn` e `CueOut` do banco eram mantidos intactos, mas a flag `is_sonic` era sobrescrita incorretamente com o valor automático.
3. **Exemplo**: A faixa `Queen - Bohemian Rhapsody.mp3` (duração de 354 segundos) possuía pontos de cue manuais (`CueIn: 3.07`, `CueOut: 344.47`), resultando em uma duração útil de execução de `341.4` segundos (> 5 minutos). No entanto, o FFmpeg detectava silêncio curto e definia a flag automática como `true`. O servidor aplicava essa flag incorretamente, permitindo que a faixa constasse como `is_sonic = true` e continuasse sendo sincronizada pelo player da filial mesmo violando a regra de 5 minutos.

## Solução Aplicada

### 1. Atualização do Cálculo de Elegibilidade SONIC no Servidor
#### [music_style_handler.go](file:///P:/imc-server/internal/handlers/music_style_handler.go)
* Atualizamos a rotina de varredura (scanning) e de reprocessamento individual de faixas para recalcular a flag `IsSonic` usando os valores finais de cue que de fato serão gravados:
  ```go
  track.IsSonic = (track.CueOut - track.CueIn) <= 300.0
  ```
* Adicionamos a mesma verificação e recálculo dinâmico nos endpoints de alteração manual de cue points (`HandleSaveMusicTrackCue`) e restauração automática (`HandleResetMusicTrackCue`).

#### [commercial_handler.go](file:///P:/imc-server/internal/handlers/commercial_handler.go)
* Replicamos a correção nas rotinas equivalentes de comerciais e eventos para garantir que qualquer alteração manual ou reprocessamento reavalie a flag `IsSonic` usando a diferença real de cue points:
  ```go
  c.IsSonic = (c.CueOut - c.CueIn) <= 300.0
  ```

### 2. Validação e Execução
* Recompilamos o servidor e reiniciamos a execução na máquina Linux do cliente.
* Criamos e executamos um script de teste que reproduziu o comportamento do endpoint de reprocessamento individual especificamente sobre a faixa de Bohemian Rhapsody.
* O registro no banco de dados do servidor foi atualizado com sucesso para `is_sonic = false`.
* Disparamos o sincronismo forçado no player local e comprovamos que a música deixou de ser transmitida e foi devidamente expurgada do HD da filial por não cumprir a regra de limite de tempo do selo SONIC.

---

# Walkthrough Suplementar: Rotatividade de Logs e Otimização do SQLite

Nesta etapa suplementar, avaliamos e implementamos soluções para gerenciar o crescimento dos logs e otimizar o uso do banco de dados SQLite local, evitando vazamentos de espaço em disco e garantindo alta performance de leitura/escrita.

## Diagnóstico do Problema

1. **Crescimento Exponencial de Logs de Terceiros**: A execução do motor Liquidsoap gerava arquivos de log (`imcAudio.log`) gigantescos (acumulando ~43 MB de logs em apenas 8 dias, projetando ~2 GB em um ano).
2. **Crescimento de Arquivo Local**: O arquivo de logs do player local (`client_player.log`) crescia de forma contínua com aberturas/fechamentos repetitivos de arquivos a cada linha gravada.
3. **Bloat do Banco de Dados SQLite**: As tabelas locais `local_logs` e `planned_commercial_slots` acumulavam dezenas de quaisquer linhas rapidamente (ex.: **42.500+ registros de log** e **14.000+ slots planejados** em apenas 8 dias), sem qualquer mecanismo de expurgo, causando aumento silencioso do tamanho do banco de dados.

## Soluções Aplicadas

### 1. Rotatividade Automática de Arquivos de Log (Log Rotation)
#### [db.go](file:///c:/imc-audio-local/db.go)
* Criamos a função `RotateLogFile(filePath string, maxSize int64)` que verifica se um arquivo excedeu o tamanho limite de 5 MB. Caso tenha excedido, ela rotaciona os backups mantendo até 3 arquivos históricos (`log.1`, `log.2`, `log.3`).
* Alteramos a função `LogLocal` para rotacionar o arquivo `client_player.log` dinamicamente em tempo de execução antes de cada inserção.

#### [main.go](file:///c:/imc-audio-local/main.go) e [watchdog.go](file:///c:/imc-audio-local/watchdog.go)
* Integramos a chamada de rotação do arquivo de logs do Liquidsoap (`cfg.LogPath`) antes de cada inicialização/reinicialização do processo. Como o arquivo de log fica travado pelo Liquidsoap durante sua execução normal, a rotação ocorre com total segurança no exato momento em que o processo não está rodando (na inicialização e durante ações do watchdog).

### 2. Limpeza Periódica de Tabelas de Histórico e Logs no SQLite
#### [db.go](file:///c:/imc-audio-local/db.go)
* Refatoramos a função `CleanOldHistory()` para realizar uma limpeza abrangente e multi-tabela:
  * Pruning de `play_history` com registros de reprodução mais antigos que 48 horas.
  * Pruning de `planned_commercial_slots` antigos de distribuições de comerciais passadas (com mais de 48 horas).
  * Pruning da tabela `local_logs`, mantendo apenas os logs dos últimos 3 dias (72 horas) para diagnósticos rápidos no painel.

#### [sync.go](file:///c:/imc-audio-local/sync.go)
* Inserimos a execução de `CleanOldHistory()` de forma recorrente dentro do loop de sincronismo (`StartSyncLoop`). Desse modo, o banco é limpo automaticamente e de forma leve a cada 5 minutos, garantindo que o banco de dados SQLite nunca cresça indefinidamente.

## Resultados e Validação

1. **Limpeza do SQLite**: A tabela `local_logs` encolheu imediatamente de **42.561** para **32.668** registros (removendo 10 mil linhas obsoletas), e a tabela de slots caiu de **14.089** para **7.940** registros.
2. **Rotação Física**: O arquivo volumoso `imcAudio.log` de 43.1 MB foi rotacionado com sucesso para `imcAudio.log.1`, liberando o arquivo ativo que iniciou do zero (~13 KB).
3. **Performance**: Com o modo WAL (Write-Ahead Logging) do SQLite ativo e índices rápidos aplicados, as operações de leitura/escrita mantiveram latência de frações de milissegundo, eliminando riscos de contenção de acesso.

---

# Walkthrough Suplementar: Ajuste de Volume de Comerciais (Boost de 20%)

Nesta etapa suplementar, corrigimos a lógica de aumento de volume de blocos comerciais para corresponder ao requisito de 20% solicitado pelo usuário.

## Alterações Realizadas

### 1. Atualização do Script de Template do Liquidsoap
#### [player.liq.tmpl](file:///c:/imc-audio-local/player.liq.tmpl)
* Alteramos o fator multiplicador do volume dos comerciais na função `on_meta` de `1.1` (10%) para `1.2` (20%).
* Atualizamos o comentário descritivo do script para documentar a alteração:
  ```liquidsoap
  # Volume rules: increase by 20% on commercial blocks, standard on music
  def on_meta(m) =
    t = m["type"]
    id = m["id"]
    if t == "commercial" or t == "opening" or t == "closing" or t == "right_time" or t == "temperature" or t == "event" then
      var_volume.set(base_volume() * 1.2)
  ```

### 2. Reinicialização e Aplicação
* Reiniciamos a execução do player Go em background.
* O player Go gerou o arquivo de script `player.liq` atualizado com o fator `1.2` e iniciou o motor Liquidsoap aplicando imediatamente o ganho de 20% nas faixas do tipo `commercial`, `opening`, `closing`, `right_time`, `temperature` e `event`.

---

# Walkthrough Suplementar: Logs de Transições de Volume

Nesta etapa suplementar, incluímos no console do player local e nos logs em disco o registro exato de quando o volume do mixer é alterado ou restaurado ao seu estado original.

## Alterações Realizadas

### 1. Logs de Transições no Core do Scheduler
#### [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* Alteramos a rotina `MarkTrackAsPlaying(id)` para identificar a transição da faixa de áudio e logar no sistema a ação do mixer de volume com base no tipo de conteúdo executado:
  * Quando a faixa que entra em execução é do tipo **Música**, registramos a restauração para o volume padrão (`1.0`):
    ```go
    LogLocal("INFO", "[Volume] Restaurando volume do mixer para o padrão (1.0) para: %s", displayName)
    ```
  * Quando a faixa em execução é do tipo **Comerciais/Eventos**, registramos o ajuste do mixer para ganho de `+20%`:
    ```go
    LogLocal("INFO", "[Volume] Ajustando volume do mixer para +20%% (Comerciais/Eventos) para: %s", displayName)
    ```
* Para deixar os logs legíveis e úteis à TI, alteramos a consulta interna da transição para obter e imprimir o nome amigável de exibição da mídia (`displayName`) em vez de exibir apenas a UUID do arquivo.

## Resultados e Validação

* Recompilamos o player e reiniciamos sua execução.
* Os logs no arquivo `client_player.log` e na console de depuração agora registram explicitamente a transição de volumes:
  * Exemplo de log de música:
    `[2026-06-21 21:13:56] [INFO] [Volume] Restaurando volume do mixer para o padrão (1.0) para: 48 Crash`

---

# Walkthrough Suplementar: Sincronização Dinâmica de Cues e Metadados de Áudio

Nesta etapa suplementar, implementamos e validamos a detecção e aplicação de alterações nos pontos de cue (`cue_in`, `cue_out`), duração (`duration_seconds`) e ganho de áudio (`amplify`) de músicas e comerciais.

## Alterações Realizadas

### 1. Detecção de Alterações de Configuração Estendida
#### [sync.go](file:///c:/imc-audio-local/sync.go)
* Aprimoramos a função `checkIfConfigChanged()` para ler os metadados ativos (cues, duração e amplify) do banco SQLite local de comerciais (`commercials_and_events`) e músicas (`media_library`) e compará-los com o payload de sincronização vindo do servidor.
* Se houver qualquer divergência em qualquer faixa SONIC ativa (por exemplo, após um ajuste manual de cue points no servidor), a função retorna `true`. Isso sinaliza imediatamente ao scheduler local que ocorreu uma mudança e força a regeneração e pré-distribuição dos comerciais do dia atual e seguinte.

## Resultados e Validação

1. **Compilação e Execução**: O player local foi recompilado com sucesso e reiniciado.
2. **Atualização de Cues Manuais**: O banco de dados local SQLite agora sincroniza e armazena corretamente os novos pontos de cue alterados no painel do servidor.
3. **Distribuição Automática**: Ao detectar a discrepância de cues, o player local executou imediatamente a pré-distribuição de comerciais (`PreDistributeCommercials`), garantindo a geração de playlists/timelines com os tempos e pontos de corte corretos.

---

# Walkthrough Suplementar: Exibição do Bloco Planejado de Comerciais

Nesta etapa suplementar, corrigimos a exibição do identificador do bloco planejado na tabela "Execução Real" do painel de diagnósticos, garantindo que comerciais exibam seu horário planejado original mesmo se executados após a meia-noite (virada do dia) ou quando o horário do bloco não existir na grade de operação do dia atual.

## Diagnóstico do Problema

1. **Divergência de Horários por Transição**: Quando comerciais planejados para o final de um dia (ex: bloco de `23:58:00`) são reproduzidos logo após a meia-noite (ex: `00:00:51` do dia seguinte), eles são gravados no histórico com `planned_block_time = '23:58:00'` e `played_at` no novo dia.
2. **Falta de Associação na Interface**: O painel de diagnósticos consome a grade de blocos do dia carregado (hoje). Como a grade de blocos do novo dia não possui um bloco planejado correspondente a `'23:58:00'`, a função `getBlockIdForRecord()` falhava ao buscar uma correspondência e exibia `-`.

## Solução Aplicada

### 1. Ajuste na Lógica de Renderização do Bloco do Histórico
#### [web_dashboard.go](file:///c:/imc-audio-local/web_dashboard.go)
* Refatoramos a função JavaScript `getBlockIdForRecord(rec, blockStructures)`:
  * Adicionamos uma validação direta: caso o registro possua `rec.planned_block_time` definido, o sistema primeiro busca uma correspondência nas estruturas de bloco do dia carregado.
  * Se nenhuma correspondência com o número do bloco do dia for encontrada (caso comum em reproduções pós-meia-noite de blocos do dia anterior ou fora das horas úteis atuais), a função retorna o horário planejado original formatado (ex: `23:58`) ao invés de exibir `-`.

## Resultados e Validação

* O player local foi recompilado e reiniciado.
* Agora, todas as entradas de comerciais exibidas na aba "Execução Real" que ocorreram após a virada do dia ou fora da grade padrão mostram com precisão o horário do bloco planejado que originou aquela reprodução (ex: `23:58`), eliminando a exibição genérica de `-`.

---

# Walkthrough Suplementar: Restrição de Blocos por Porcentagem de Ofertas e Institucionais

Nesta etapa suplementar, implementamos a restrição correta das porcentagens de blocos comerciais para Ofertas (`PercentageOffer`) e Institucionais (`PercentageInstitutional`) durante o processo de distribuição antecipada de comerciais.

## Diagnóstico do Problema

1. **Ignorância das Porcentagens de Bloco**: Embora o player local sincronizasse as porcentagens (ex: `50%` para Institucionais e `50%` para Ofertas), o algoritmo de pré-distribuição antecipada (`PreDistributeCommercials` em `scheduler.go`) considerava todos os blocos do dia como elegíveis para qualquer tipo de mídia.
2. **Consequência**: Com isso, comerciais de Ofertas e Institucionais eram espalhados livremente por toda a grade, ignorando os limites definidos pelo usuário no painel.

## Solução Aplicada

### 1. Filtragem de Blocos Elegíveis na Distribuição Antecipada
#### [scheduler.go](file:///c:/imc-audio-local/scheduler.go)
* Alteramos a lógica de seleção de blocos em `PreDistributeCommercials`:
  * Para mídias do tipo **Oferta** (`OFFER`), a lista de blocos elegíveis é filtrada mantendo apenas aqueles nos quais `shouldPlayForPercentage(idx, blockConfig.PercentageOffer)` é verdadeiro.
  * Para mídias do tipo **Institucional** (`INSTITUTIONAL`), filtramos para manter apenas blocos nos quais `shouldPlayForPercentage(idx, blockConfig.PercentageInstitutional)` é verdadeiro.
  * Comerciais padrão (`COMMERCIAL`) continuam sendo elegíveis para 100% dos blocos.

## Resultados e Validação

1. **Sincronismo e Regeneração**: O player foi compilado e reiniciado. Ao acionar a sincronização POST no endpoint `/api/sync`, a distribuição foi recalculada respeitando as regras.
2. **Resultados em Banco (Segunda-feira - 50% / 50%)**:
   * O total de blocos da tarde possuía 54 posições planejadas.
   * Exatamente **27 blocos** foram planejados contendo Institucionais (exatamente 50%).
   * Exatamente **27 blocos** foram planejados contendo Ofertas (exatamente 50%).
   * A distribuição intercalou os blocos elegíveis de maneira uniforme (um sim, um não).
3. **Resultados em Banco (Terça-feira - 0% / 0%)**:
   * Terça-feira possuía 0% para ambas as configurações. A distribuição gerou exatamente **0 slots planejados** de Ofertas e Institucionais, mantendo apenas comerciais gerais. Conclusão perfeita.
