# Walkthrough - Playout Streaming HLS, Sincronismo Automático, Timezone, Transições de Áudio e Deploy Geral

Nesta atualização, implementamos a geração do fluxo em **HLS (HTTP Live Streaming)** de baixa latência em paralelo com a transmissão Icecast tradicional. Também integramos a biblioteca `hls.js` no painel de controle local da rádio (porta `5003`), corrigimos um bug crítico de fuso horário (Timezone) que afetava a rotação de artistas, ativamos a transição profissional cruzada (crossfade) de faixas e realizamos o deploy dessa nova versão em **todas as lojas ativas do servidor**.

---

## 🛠️ Mudanças Realizadas

### 1. Liquidsoap (`player.liq.tmpl`)
- Mantivemos o canal Icecast ativo para auditoria administrativa geral do servidor.
- Adicionamos a geração paralela de HLS no container (`output.file.hls`).
- Usamos o codificador **AAC mono (`channels=1`)** para compatibilidade exata com o sinal de áudio `audio_final`.
- **Transições Profissionais (Crossfade)**: Adicionamos o operador nativo `crossfade` ao fluxo dinâmico de áudio (`raw_source`), fazendo com que o final de uma música se misture suavemente com o início da próxima por 3 segundos.
- **Filtro de Duração (`minimum=32.0`)**: Configuramos o limite mínimo de duração para o crossfade em **32 segundos**. Isso garante que anúncios institucionais, vinhetas e comerciais curtos (geralmente menores ou iguais a 30 segundos) toquem na íntegra de forma limpa (corte seco) sem fades ou sobreposição de voz, enquanto músicas normais ganham o crossfade suave.
- **Corte de Silêncio Automático**: Na versão v2.2.4 do Liquidsoap, a lógica do antigo `cue_cut` foi integrada nativamente na decodificação de requisições anotadas. O player agora respeita automaticamente as marcações de `liq_cue_in` e `liq_cue_out` geradas pela análise de áudio (SONIC).

### 2. Servidor Go (`main.go`, `scheduler.go` & `web_dashboard.go`)
- Criamos a pasta `/app/hls` com permissões completas de escrita para o usuário do container `liquidsoap`.
- Mapeamos a rota `/hls/` no multiplexador HTTP do Go para expor a playlist e os segmentos `.ts` gerados.
- **Correção de Timezone (Fuso Horário)**: Atualizamos todas as consultas do SQLite que usavam `datetime('now')` para usar `datetime('now', 'localtime')`. Isso sincronizou as verificações do banco de dados (que salvava em horário local da loja) com o horário real do container, destravando a **regra de 45 minutos de separação de artista** e a regra de espaçamento de comerciais repetidos.

### 3. Painel Front-End (`web_dashboard.go`)
- Carregamos a biblioteca `hls.js` no cabeçalho do HTML.
- Modificamos o botão redondo de **Play** do player para usar a API do `hls.js`, caindo de volta para a tag nativa de HLS no Safari/iOS.
- Automatizamos a seleção de atraso de buffer para **6 segundos** (tempo fixo e estável de buffer do HLS) ao acionar a escuta, garantindo que o Gauge de progresso e nomes das músicas fiquem alinhados perfeitamente com os alto-falantes sem intervenção do operador.

---

## 🚀 Deploy Geral e Lojas Atualizadas

Como as instâncias das Lojas 01 e 02 são gerenciadas de forma declarativa via Docker Compose (enquanto os shopping centers dinâmicos são criados por UUID na base de dados), efetuamos o deploy em cascata para cobrir **100% dos players ativos no servidor**:

1. **Shopping Cidade Sorocaba (Playout Dinâmico)**: Recriado e reiniciado com sucesso na porta `7003`.
2. **Matriz 01 - Loja 001 (Compose-Managed)**: Recriado e reiniciado com a nova imagem na porta `7001` (painel em `5001`).
3. **Matriz 01 - Loja 002 (Compose-Managed)**: Recriado e reiniciado com a nova imagem na porta `7002` (painel em `5002`).

---

## 🧪 Resultados dos Testes de Validação

1. **Geração dos Segmentos no Container**:
   - Os arquivos `playlist.m3u8`, `audio.m3u8` e os pedaços `audio_X.ts` estão sendo salvos com sucesso em `/app/hls` em todos os containers ativos.
2. **Serviço HTTP de Arquivos**:
   - Uma requisição `curl` para `http://localhost:5003/hls/playlist.m3u8` retornou com sucesso o código de status **`200 OK`** e o Content-Type correto **`audio/x-mpegurl`**.
3. **Estabilidade de Execução e Transições**:
   - Todas as instâncias do Liquidsoap estão ativas e estáveis. O log do player não apresenta erros e as transições de áudio cruzadas agora ocorrem nativamente respeitando o limite mínimo de 32 segundos.
