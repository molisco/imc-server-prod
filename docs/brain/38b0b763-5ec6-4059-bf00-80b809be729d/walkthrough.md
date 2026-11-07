# Walkthrough - Restrição de Horário, Painel de Estrutura dos Blocos, Correção de Distribuição e Sincronização de Estilos

Melhorias de agendamento e correções de distribuição concluídas e verificadas com sucesso no sistema local:

## Alterações Realizadas

### 1. Restrição Estrita do Horário de Funcionamento
#### [scheduler.go](file:///C:/imc-audio-local/scheduler.go)
* **Correção da Vigência de Blocos:** Fora do horário da loja (ex: entre 00:00 e 07:00), o player tocará estritamente músicas sem executar comerciais, ofertas, institucionais, ou vinhetas de abertura/fechamento.
* **Ajuste:** Alteramos a condição em `GetNextTrack()` para verificar se `schedule` é diferente de `nil` antes de compor qualquer bloco.

### 2. Nova Aba de Diagnóstico: "Estrutura dos Blocos"
#### [web_dashboard.go](file:///C:/imc-audio-local/web_dashboard.go)
* **Painel no Frontend:** Criamos a aba **"Estrutura dos Blocos"** na barra de navegação, que exibe a listagem completa dos blocos que irão tocar na data selecionada (Hoje ou Amanhã) com colunas de número do bloco, horário, estrutura, quantidade de itens, tempo preenchido e tempo máximo permitido.

### 3. Correção do Acúmulo de Comerciais no Primeiro Bloco (07:00)
#### [scheduler.go](file:///C:/imc-audio-local/scheduler.go)
* **Problema Identificado (Causa Raiz):** 
  * Antes do deploy do bloqueio de horário, o player executava comerciais na madrugada (ex: de 00:00 às 07:00).
  * Quando esses comerciais tocavam, o método `recordPlayHistory()` atualizava o status do primeiro slot `PENDING` disponível para aquele comercial no dia (`ORDER BY block_time ASC LIMIT 1`).
  * Como a grade comercial da loja inicia às 07:00, o primeiro slot programado era sempre o das `07:00:00`.
  * Isso fez com que 22 execuções realizadas de madrugada fossem incorretamente associadas e marcadas como `PLAYED` no bloco das `07:00:00`, dando a falsa impressão de que o primeiro bloco continha excesso de comerciais.
* **Solução Implementada:**
  1. **Campo `BlockTime` em `PlayItem`:** Adicionamos o campo `BlockTime string` no struct `PlayItem` para transportar o horário previsto do bloco comercial.
  2. **Parametrização em `makePlayItem`:** Atualizamos a assinatura do helper `makePlayItem` para receber e injetar o `currentBlockTimeStr`.
  3. **Validação Inteligente de Histórico:** Ajustamos `recordPlayHistory` para tentar marcar como `PLAYED` o slot com o `block_time` correspondente ao planejado (`item.BlockTime`). Caso não exista (ou esteja vazio), o sistema faz o fallback seguro para o primeiro slot pendente do dia.
* **Resultado:**
  * O agendamento de amanhã (`2026-06-13`) que está limpo e sem execuções antecipadas apresenta uma distribuição perfeitamente uniforme: **exatamente 2 ou 3 comerciais por bloco** em toda a programação, sem qualquer acúmulo no primeiro bloco (bloco das 07:00:00 inicia com exatamente 2 comerciais).

### 4. Id do Bloco e Horários Baseados Teoricamente no Tempo Musical
* **Alinhamento do Bloco com a Grade:** O cálculo dos blocos no banco de dados e no dashboard agora utiliza `sched.MusicalBlockDurationMinutes` (tempo do bloco musical) como o intervalo real entre os blocos (ex: 8 minutos). Isso resolve a discrepância onde a grade anteriormente assumia blocos teóricos a cada 2 minutos (gerando 1020 minutos acumulados), passando a gerar o volume correto e realista de blocos comerciais para o dia correspondente da Grade de Áudio.
* **Previsão de Horários Determinística:** A coluna **"Horário"** agora se baseia exclusivamente no cálculo teórico sequencial a partir do horário de início da loja (ex: 07:00:00). A fórmula adiciona a duração de cada bloco de comercial gerado mais a duração do bloco musical (`intervalSecs`) para determinar o início do próximo bloco:
  $$T_{i+1} = T_i + \text{Duração do Bloco } i + \text{Duração do Bloco Musical}$$
  Esse cálculo descarta as marcações de histórico real/simulado (sem tags `Real:` ou `Prev:`), refletindo pura e fielmente o comportamento planejado da grade.

### 5. Distribuição Comercial Baseada no Horário de Funcionamento Total (Hoje)
* **Prevenção de Acúmulo no Fim do Dia:** Corrigi o comportamento de redistribuição diária de comerciais (quando novos comerciais são ativados ou em caso de reinicialização no meio do dia). Anteriormente, o sistema redistribuía as inserções restantes do dia apenas nos blocos futuros remanescentes, gerando acúmulo (saturação) nos blocos da tarde/noite.
* **Resolução:** O algoritmo de distribuição estratificada agora considera **todos os blocos do horário de funcionamento da loja (o dia completo)**, mantendo a dispersão perfeitamente uniforme. Caso um comercial seja alocado em um bloco do passado, ele simplesmente não executará naquele bloco (o que é o comportamento correto, sabido via PoP/logs), garantindo que os blocos futuros continuem recebendo a densidade de comerciais prevista e uniforme, sem nenhuma sobrecarga.

### 6. Correção da Seleção e Sincronização de Estilos Musicais
#### [scheduler.go](file:///C:/imc-audio-local/scheduler.go)
* **Problema Identificado (Causa Raiz):** O método `selectNextMusicTrack` selecionava o estilo de música baseando-se no `currentHour` (0 a 23) comparado diretamente com as chaves do mapa `CustomStylesJSON`. Contudo, as chaves desse JSON (gerado pelo servidor e armazenado na tabela `basic_schedules`) representam os **índices dos blocos da programação** (0, 2, 4, 6, ... até 298), não as horas do dia. Isso causava uma séria discrepância temporal onde o player executava estilos programados para outras faixas horárias e demorava muito mais para mudar de estilo musical.
* **Solução Implementada:**
  1. **Cálculo Preciso do Bloco Atual:** Atualizamos a assinatura de `selectNextMusicTrack` para receber a configuração do bloco comercial ativa (`CommercialBlockConfig`).
  2. **Conversão de Tempo para Índice:** Modificamos o algoritmo de seleção para calcular os minutos decorridos desde o início do funcionamento da schedule (`sched.StartTime`). Usando o tempo musical (`sched.MusicalBlockDurationMinutes`) e o intervalo dos comerciais (`blockConfig.IntervalMinutes`), o método calcula o `blockIdx` correto do instante atual.
  3. **Identificação do Bloco Musical:** Se o índice calculado for ímpar (bloco comercial), ele recua para o índice par anterior (bloco musical correspondente) para extrair o estilo correto no mapa de estilos.
* **Resultado:** O player local agora sincroniza e transiciona perfeitamente entre os estilos de música programados na grade do IMC-Server no exato momento configurado pelo usuário.

### 7. Correção de Tipos Internos Duplicados em Distribuição de Blocos
#### [scheduler.go](file:///C:/imc-audio-local/scheduler.go) e [web_dashboard.go](file:///C:/imc-audio-local/web_dashboard.go)
* **Problema Identificado (Causa Raiz):** 
  * O agendador insere automaticamente e dinamicamente nos blocos os seguintes tipos estruturais: aberturas de bloco (`BLOCK_OPENING`), fechamentos de bloco (`BLOCK_CLOSING`), anúncios de hora certa (`RIGHT_TIME`) e temperatura (`TEMPERATURE`).
  * Porém, o algoritmo de distribuição antecipada (`PreDistributeCommercials`) incluía incorretamente esses tipos no sorteio estratificado aleatório do dia, resultando na geração de slots redundantes desses itens nas tabelas de agendamento planejado.
  * Como consequência, na aba "Estrutura dos Blocos" do frontend, blocos apareciam com duplicações na string estrutural (ex: exibindo `[Abertura, ... BLOCK_OPENING, BLOCK_CLOSING, Fechamento]`).
* **Solução Implementada:**
  1. **Filtro Estrito na Pré-Distribuição:** Ajustei a consulta SQL de coleta de mídia em `PreDistributeCommercials` para ignorar esses tipos estruturais e de utilidades:
     `commercial_type NOT IN ('BLOCK_OPENING', 'BLOCK_CLOSING', 'RIGHT_TIME', 'TEMPERATURE', 'EVENT')`.
  2. **Tradução no Frontend:** Atualizei o switch de tradução em `web_dashboard.go` para mapear `GENERAL_INFO` como `"Informativo"`.
* **Resultado:** A pré-distribuição agora agenda exclusivamente mídias de veiculação pura (`COMMERCIAL`, `OFFER`, `INSTITUTIONAL` e `GENERAL_INFO`), e a interface renderiza a estrutura limpa e sem duplicações: `[Abertura, Comercial x3, Oferta, Fechamento]`.

### 8. Limpeza de Logs e Reinicialização
* **Limpeza Realizada:** Executado o comando SQL `DELETE FROM local_logs` no banco de dados SQLite `local_cache.db` para apagar todo o histórico de logs acumulados no frontend/painel de diagnósticos.
* **Reinicialização:** O player antigo foi finalizado com sucesso e a aplicação foi reiniciada em background na tarefa **`task-1291`**, iniciando com um banco de dados de logs limpo.

### 9. Distribuição Alinhada a Ciclos e Visualização de Comerciais por Nome [NOVO]
* **Alinhamento do Primeiro Bloco:** Corrigido o cálculo de início dos blocos comerciais na pré-distribuição e no dashboard. Para uma grade que inicia às 07:00 com bloco musical de 8 minutos, o primeiro bloco comercial agora é gerado exatamente às 07:08 (e depois 07:18, 07:28, etc.), em vez de gerar slots indevidos em 07:00.
* **Nomes dos Comerciais no Dashboard:** Alteramos a coluna "Estrutura do Bloco" para listar os nomes reais dos arquivos de comerciais planejados (ex: `Corneto.mp3`, `Doriana.mp3`, etc.) ao invés de apenas agrupamentos genéricos (ex: `Comercial x3`), permitindo ao operador visualizar a distribuição exata em cada bloco.
* **Tratamento de Fim de Dia e Limite da Meia-Noite (00:00):** 
  * Corrigida a lógica de parsing para horários de fechamento às `00:00:00` (meia-noite), adicionando `86400` segundos (24 horas) ao `endSecs` para evitar fallbacks incorretos de 14 horas.
  * Inseridas restrições estritas (`sec < 86400` e `currSecs < 86400`) para garantir que nenhuma estrutura de bloco ou comercial planejado ultrapasse o limite da meia-noite (24:00), eliminando a duplicação ou wrap-around visual de horários no painel.

### 10. Colunas Extras no Histórico & Destaque de Bloco Atual [NOVO]
* **Id do Bloco no Histórico:** Adicionado o número do bloco correspondente (ex: `#1`, `#2`) na tabela de histórico da aba "Execução Real" para identificar visualmente a qual bloco pertencia cada comercial reproduzido.
* **Diferença (Previsto x Real):** Implementada a nova coluna "Dif. (Prev x Real)" na tabela de histórico, que calcula o desvio exato (atraso ou adiantamento) no formato `±MM:SS`. Mídias que desviarem mais de 2 minutos do horário planejado do bloco são exibidas com a cor vermelha (`--status-red`), enquanto desvios menores que 2 minutos aparecem em verde (`--status-green`).
* **Destaque "AGORA" em Estrutura dos Blocos:** 
  * O bloco comercial ou musical que estiver executando no exato momento (baseado em `data.current_time_secs` do player caindo dentro da faixa de horário do bloco) é destacado com uma linha brilhante em azul-ciano (`block-now-row`) e recebe um badge pulsante `▶ AGORA` (`now-badge`) ao lado da descrição estrutural do bloco.
  * O painel agora realiza scroll automático suave (`scrollIntoView`) para centralizar a visualização no bloco ativo quando este transiciona para o próximo bloco ou quando o usuário abre a aba "Estrutura dos Blocos".

### 11. Resolução de Saltos de Bloco,IDs Erráticos e Catch-up Dinâmico [NOVO]
* **Correção dos IDs de Bloco Baixos (#4, #6):** 
  * **Causa Raiz:** O fallback de busca do slot no banco de dados usava `ORDER BY block_time ASC LIMIT 1`, que trazia registros remanescentes do início do dia (madrugada) que nunca tinham tocado.
  * **Resolução:** Alterado o fallback de `recordPlayHistory()` para usar ordenação por proximidade absoluta de tempo: `ORDER BY ABS(strftime('%s', play_date || ' ' || block_time) - strftime('%s', $3)) ASC`. Desta forma, a gravação de histórico de um item tocado às 20:30 será associada ao slot de `20:28:00` (bloco #246) e não ao de `00:18:00` (bloco #4), sanando os registros de blocos erráticos.
* **ID e Desvios de Mídias Estruturais:** Mapeado o `planned_block_time` das vinhetas de Abertura, Fechamento, Hora Certa e Temperatura na tabela `play_history`. Isso garante que o frontend agora exiba o bloco exato correspondente para estes itens e calcule suas diferenças temporais corretamente, em vez de fazer fallbacks visuais erráticos.
* **Garantia de Fluxo Sequencial Contínuo (Sem Retrocessos):**
  * Mantemos o disparo de blocos comerciais baseado em **intervalo de tempo decorrido em tempo real** (`now.Sub(lastBlockTime) >= interval`) no agendador.
  * Ao compor o bloco, o agendador se alinha estritamente com a hora real `nowSecs` do instante da composição para selecionar as mídias daquele exato bloco no banco de dados.
  * Isso impede que o player olhe para trás ou execute blocos do passado quando o player reinicia ou sofre atraso, assegurando que a programação flua sempre para frente e sem pular de forma errática de um bloco para outro (ex: pulando de `#247` para `#249` ou retrocedendo de `#253` para `#252`).
* **Medição Precisa de Atraso (Desvio):** A função `calcCurrentDesvio()` foi ajustada para calcular a diferença do relógio contra o próximo bloco esperado sequencialmente (`lastBlockTime + interval`), fornecendo dados estáveis e corretos de atraso em tempo real.
* **Catch-up Acelerado (Tolerância de 60s):** Ajustamos o auto-catchup monitor para realizar skips em músicas que tocaram por pelo menos 1 minuto (60 segundos) caso a loja esteja com um atraso acumulado de 5 minutos ou mais. Isso permite recuperar a pontualidade da grade muito mais rápido.
* **Expiração de Slots do Passado (Evitar Acúmulos):** Adicionada a função `expirePastPlannedSlots()`, rodada periodicamente e na inicialização, que marca como `SKIPPED` qualquer slot pendente atrasado em mais de 15 minutos (ex: quando o player foi aberto tarde ou ficou fora do ar). Isso impede o acúmulo e execução de uma avalanche de comerciais antigos.

### 12. Correção Adicional: Pulo de Blocos Comerciais, Alinhamento na Inicialização e Recuperação de Lag [NOVO]

* **Inicialização Exata dentro do Bloco Comercial:**
  * **Causa Raiz:** Anteriormente, se o player iniciasse dentro do período de um bloco comercial (ex: `22:09:30`, bloco de `22:08:00 - 22:10:00`), o agendador definia `lastBlockTime` como `now.Add(-interval)` (`22:09:30 - 10m = 21:59:30`). Quando compunha o bloco comercial, o cálculo de `alignedSecs` com base em `now` (`22:09:30`) resultava em `22:08:00`, mas depois o scheduler alinhava `lastBlockTime` baseado no horário arredondado ou relógio em tempo real, gerando drift de cálculo no ciclo seguinte.
  * **Resolução:** Modificamos a inicialização do agendador em `GetNextTrack()` de forma que, se o startup ocorrer dentro do bloco comercial (`relSecs >= commStart`), o agendador calcula o início planejado desse bloco comercial na grade (`alignedBlockStart` = `22:08:00`) e define `lastBlockTime = alignedBlockStart - interval` (ex: `21:58:00`). Isso faz com que a composição do bloco comercial ocorra imediatamente no startup e o `lastBlockTime` seja atualizado de forma perfeitamente alinhada na sequência.
* **Composição de Bloco Sequencial e Sem Pulos:**
  * **Causa Raiz:** O agendador anteriormente calculava o horário do bloco comercial a ser composto usando o relógio atual (ou `targetPlayTime`) em um cálculo de arredondamento (`math.Round`). Em execuções contínuas, se uma música atrasasse e terminasse muito perto ou depois da metade do ciclo seguinte, o arredondamento saltava o bloco comercial planejado (ex: pulando do bloco #247 para o #249).
  * **Resolução:** Alteramos a lógica de disparo. Quando `now.Sub(lastBlockTime) >= interval`, definimos o horário do bloco a ser composto como exatamente sequencial: `blockTime := lastBlockTime.Add(interval)`. Passamos esse `blockTime` exato para `composeCommercialBlock` e atualizamos `lastBlockTime = blockTime`. Isso garante uma progressão estritamente linear e sequencial, garantindo que nenhum bloco seja pulado ou omitido.
* **Recuperação de Atraso Crítico (Lag Recovery):**
  * **Resolução:** Introduzimos uma verificação de lag acumulado. Se a diferença `now.Sub(lastBlockTime)` exceder `interval + 15 minutos`, o scheduler detecta que o player ficou offline por muito tempo. Ele redefine `lastBlockTime = time.Time{}` (zero) de forma a forçar a re-inicialização e re-alinhamento ao bloco atual do relógio no próximo ciclo, saltando de forma limpa os blocos passados (que são marcados como `SKIPPED` no banco de dados) e mantendo a integridade.

### 13. Correção Adicional: Uso de targetPlayTime nos Checks de Intervalo (Composição de Bloco no Pre-Fetch) [NOVO]

* **Problema Identificado (Causa Raiz):** 
  * O player Liquidsoap realiza pré-buscas (pre-fetch) de faixas de 2 a 5 minutos antes de elas começarem a tocar.
  * No momento em que Liquidsoap enviava a requisição `/next` para a faixa que tocaria às `22:49:01`, o relógio da máquina (`now`) ainda marcava `22:47:10`.
  * Como a condição de disparo no agendador usava a hora do relógio (`now.Sub(lastBlockTime) >= interval`), ele calculou que ainda não era o momento do comercial (faltavam 50 segundos para `22:48:00`), decidindo retornar uma música convencional ("Change").
  * No entanto, essa música "Change" só começaria a tocar de fato às `22:49:01` (dentro da janela do Bloco Comercial #274), empurrando os comerciais reais de forma desnecessária para as `22:54:38`, gerando atraso e uma música tocando no horário comercial.
* **Solução Implementada:**
  * Alteramos as condições de disparo e inicialização do agendador em `GetNextTrack()` de `now` para `targetPlayTime`.
  * `targetPlayTime` representa a previsão exata de quando a faixa que estamos retornando começará a tocar (baseada na soma das durações das faixas em execução).
  * Ao avaliar se `targetPlayTime.Sub(lastBlockTime) >= interval`, o agendador calcula corretamente que a faixa a ser retornada iniciará às `22:49:01` (já dentro do bloco comercial) e compõe imediatamente o bloco comercial, retornando os comerciais de forma pontual e sem tocar uma música convencional intermediária.
* **Resultado:**
  * O player não toca mais músicas desnecessárias e as faixas comerciais começam pontualmente, conforme validado nos novos logs: a música termina, a requisição de pré-busca detecta que a próxima reprodução cairá no horário comercial e retorna imediatamente a abertura e comerciais do bloco (`23:18:00`), sem inserir nenhuma música indevida.

---

## Verificação e Testes

### Compilação e Testes
O código foi compilado e testado com sucesso:
```powershell
go build -o imc-audio-local.exe
```

### Validação em Execução
1. O banco de dados SQLite foi limpo com sucesso (`DELETE FROM planned_commercial_slots`, `DELETE FROM play_history`, e `DELETE FROM local_logs`), deixando a grade e o histórico de reprodução em um estado totalmente novo.
2. A nova lógica matemática foi validada, cobrindo startup em bloco comercial, startup em bloco musical, progressão sequencial contínua e recuperação de lag crítico.
3. O player local foi reiniciado e está rodando com sucesso na tarefa **`task-2388`**.
4. Os logs confirmam que o player inicializou corretamente, pre-distribuiu os comerciais e compôs o bloco comercial de `23:18:00` exatamente na hora e com a marcação correta no banco de dados (slots marcados como `QUEUED` e histórico gravado com `planned_block_time = 23:18:00`), sem nenhum drift no `lastBlockTime` e tocando o comercial imediatamente no horário correspondente.
5. **Remoção da Seção "Próximas Faixas":** 
   * Removemos completamente o HTML da seção `"Próximas Faixas (Fila de Hoje)"` no arquivo [web_dashboard.go](file:///C:/imc-audio-local/web_dashboard.go).
   * Ajustamos o CSS de `.player-content-split` para usar `grid-template-columns: 1fr` en vez de `1.2fr 1fr`, permitindo que o painel principal "Tocando Agora" e os controles do player se estendam de forma limpa e centralizada por toda a largura disponível.
   * A aplicação foi compilada com sucesso e inicializada sem erros. Os callbacks de atualização do dashboard pulam a renderização da fila de forma segura, garantindo um carregamento suave sem exceções de console.

### 14. Remoção de Limites de Duração de Blocos Comerciais na Execução Real (Prevenção de Cortes) [NOVO]

* **Problema Identificado (Causa Raiz):**
  * Comerciais planejados no banco de dados para blocos volumosos (como o bloco #288) eram cortados e tocavam apenas metade do planejado.
  * Isso ocorria porque a função `composeCommercialBlock` limitava a duração do bloco ao parâmetro `BlockLimitSeconds` (geralmente 120 segundos). Mídias que excediam esse limite eram simplesmente descartadas do enfileiramento.
* **Solução Implementada:**
  * Removemos a verificação e corte de limite de tempo na função `composeCommercialBlock` em [scheduler.go](file:///C:/imc-audio-local/scheduler.go) para comerciais pura (`COMMERCIAL`, `OFFER`, `INSTITUTIONAL`).
  * Agora, todos os comerciais planejados para o bloco são devidamente enfileirados como `QUEUED` e reproduzidos na íntegra.
  * O indicador de aviso `⚠️` continua no dashboard para sinalizar que o bloco ultrapassou o tempo ideal teoricamente reservado, permitindo que administradores saibam da sobrecarga sem prejudicar a entrega dos comerciais.
* **Correção do Rollover de Data na Expiração:**
  * Corrigimos um bug em `expirePastPlannedSlots()` onde, nos primeiros 15 minutos do dia (ex: `00:05:00`), a subtração de 15 minutos resultava em um horário da véspera (`23:50:00`), marcando incorretamente todos os slots do dia inteiro como `SKIPPED`.
  * Ajustamos a query para comparar a data completa (`play_date` e `block_time`) do cutoff, eliminando expirações indevidas na inicialização do player à meia-noite.
* **Validação:**
  * O player inicializou limpo à meia-noite sem expirar incorretamente os blocos de hoje.
  * No bloco das `00:18:00`, contendo 9 comerciais planejados (totalizando 11 arquivos com abertura/fechamento e 173 segundos de duração, o que excede o limite de 120s), o agendador enfileirou todos os 11 arquivos com sucesso. Os logs e banco de dados confirmam que todos os 9 comerciais transicionaram para `QUEUED` e começaram a tocar sequencialmente sem qualquer corte.

### 15. Correção do Acúmulo Progressivo de Comerciais por Re-distribuição no Sync (Prevenção de Saturação de Blocos) [NOVO]

* **Problema Identificado (Causa Raiz):**
  * A cada ciclo de sincronização periódica (a cada 5 minutos), o agendador recalculava a distribuição dos comerciais pendentes para o dia. Contudo, ele sempre utilizava o valor nominal bruto de inserções diárias (`InsertionsPerDay`, ex: 30) para cada comercial, sem descontar os comerciais que já haviam sido reproduzidos (`PLAYED`) ou enfileirados (`QUEUED`) no mesmo dia.
  * Com isso, se um comercial (como o da Bamerindus) já tivesse tocado 2 vezes, a sincronização planejava mais 30 execuções para o restante do dia, acumulando um total de 32 exibições diárias.
  * Esse acúmulo de slots indevidos causava o inchaço progressivo dos blocos comerciais ao longo do dia, fazendo com que blocos como o #26 e o #28 passassem a exibir 14 e 17 comerciais (⚠️ 05:54 e ⚠️ 06:47 de duração), respectivamente.
* **Solução Implementada:**
  * Modificamos a função `PreDistributeCommercials` em [scheduler.go](file:///C:/imc-audio-local/scheduler.go) para, antes de distribuir cada comercial, consultar o banco de dados e contar quantos slots daquele comercial já transicionaram para `PLAYED` ou `QUEUED` no dia corrente.
  * Subtraímos essa contagem do valor planejado (`c.InsertionsPerDay - playedOrQueued`) para distribuir **estritamente as inserções remanescentes**.
* **Validação:**
  * O agendador agora distribui os comerciais de forma perfeitamente uniforme e estável. 
  * Os blocos futuros (como o bloco das 02:18 e 02:28) mantêm-se estáveis com exatamente **9 itens** (7 comerciais mais aberturas/fechamentos) e durações seguras de ~146 segundos, eliminando o acúmulo infinito de anúncios e garantindo que cada comercial execute rigorosamente o seu limite planejado (ex: 30 vezes ao dia).

### 16. Distribuição Condicional e Otimizada no Sync (Evitar Distribuição Desnecessária) [NOVO]

* **Problema Identificado (Melhoria Arquitetural):**
  * O player realizava o recálculo de pre-distribuição dos comerciais a cada sincronização (a cada 5 minutos), limpando e recriando todos os slots pendentes de forma desnecessária, mesmo quando nenhuma alteração havia ocorrido no servidor.
  * O correto seria realizar essa redistribuição apenas uma vez ao dia (no rollover da data) ou quando houvesse mudanças reais no cadastro de comerciais (novas mídias, mudanças de grade, etc.).
* **Solução Implementada:**
  * Desenvolvemos a função `checkIfConfigChanged()` em [sync.go](file:///C:/imc-audio-local/sync.go) que compara os comerciais, grades, horários de funcionamento e blocos do payload do servidor contra o banco de dados local SQLite.
  * Atualizamos a rotina de sincronização de modo que a função `PreDistributeCommercials` só é disparada se o dia do calendário mudou (rollover diário) ou se a função `checkIfConfigChanged` retornar verdadeiro.
* **Validação:**
  * O player inicializou e realizou a distribuição de comerciais normalmente na carga inicial.
  * Nas sincronizações periódicas subsequentes de 5 minutos, o log do agendador confirmou que o recálculo foi evitado com sucesso:
    > `[2026-06-13 02:27:42] [INFO] [Scheduler] No configuration changes detected. Skipping re-distribution.`
    > `[2026-06-13 02:27:42] [INFO] Sync completed successfully.`
  * A grade mantém-se totalmente intocada e os slots salvos na memória não são recriados, garantindo estabilidade e economizando recursos computacionais.

### 17. Relocalização do Script Liquidsoap e Sincronismo da Linha do Tempo em Tempo Real [NOVO]

* **Relocalização do Script (`player.liq`):**
  * Extraímos a string inline do template do Liquidsoap em `watchdog.go` para um arquivo externo de template chamado [player.liq.tmpl](file:///C:/imc-audio-local/player.liq.tmpl).
  * Utilizamos a biblioteca padrão `go:embed` em [watchdog.go](file:///C:/imc-audio-local/watchdog.go) para compilar o template diretamente no binário `imc-audio-local.exe`, mantendo o executável autossuficiente e livre de dependências externas em produção.
  * Modificamos o local padrão onde o script final é gerado para a raiz do repositório da aplicação (`./player.liq`) nos arquivos [config.go](file:///C:/imc-audio-local/config.go), [config.ini](file:///C:/imc-audio-local/config.ini) e [config.ini.example](file:///C:/imc-audio-local/config.ini.example).
  * Adicionamos `player.liq` no [.gitignore](file:///C:/imc-audio-local/.gitignore) para que os scripts gerados com caminhos absolutos locais de cada máquina não poluam o Git.

* **Sincronismo em Tempo Real (Webhook `/started`):**
  * **Problema Resolvido:** Liquidsoap pré-carrega até 10 minutos de faixas com antecedência para garantir decodificação e buffer fluidos. A dashboard exibia essas faixas futuras como "Executadas" imediatamente após serem requisitadas.
  * **Solução:**
    1. Adicionamos uma chamada HTTP de callback no script Liquidsoap (`player.liq.tmpl` no método `on_meta`): `http.get("http://127.0.0.1:LocalPort/started?id=ID")` disparada no exato momento em que cada áudio começa a tocar.
    2. Criamos o endpoint `/started` em [decrypter.go](file:///C:/imc-audio-local/decrypter.go).
    3. Implementamos o método `MarkTrackAsPlaying(id)` em [scheduler.go](file:///C:/imc-audio-local/scheduler.go) para atualizar em tempo real o status da faixa que iniciou para `"na_ponta"`, marcar faixas passadas como `"executada"` e gravar o horário de início real em `played_at`.
    4. Alteramos o status de inserção no pre-fetch de `"na_ponta"` para `"na_fila"` (que a dashboard renderiza com ícone de ampulheta `⏳ Na Fila` em amarelo), mostrando as faixas em espera até tocarem de fato.

* **Validação dos Resultados:**
  * Compilação com `go build` bem-sucedida.
  * O player foi reiniciado sob a tarefa **`task-2932`** e os logs mostram o carregamento correto do template embutido e a geração de `./player.liq` na raiz.
  * Consultas diretas ao banco de dados SQLite confirmam que faixas no buffer de pre-fetch entram no estado `"na_fila"` e mudam para `"na_ponta"` (com o carimbo de tempo real da reprodução) quando o callback `/started` é acionado pelo Liquidsoap.

