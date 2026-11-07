# Playout Streaming com HLS e Sincronismo Automático de Metadados

Este plano descreve a substituição da transmissão em tempo real Icecast (MP3/Opus) por um barramento em **HLS (HTTP Live Streaming)** de baixa latência. Isso nos permitirá embarcar metadados sincronizados (tags ID3) diretamente nos blocos de áudio e ler essa informação no JavaScript do navegador do operador, garantindo sincronismo automático e absoluto entre o áudio que sai nas caixas e as informações na tela (sem necessidade de ajuste manual de delay).

---

## User Review Required

> [!IMPORTANT]
> **Compatibilidade do Codec**: O HLS utilizará o codec **AAC**, altamente otimizado para áudio, com consumo de internet idêntico ao Opus (ex: 64kbps ou 96kbps) e qualidade indistinguível.
> 
> **Fim da Dependência de Porta do Icecast**: A transmissão de áudio passará a ser servida pela própria porta do player local (`5003`), eliminando a necessidade de expor ou acessar a porta `8000` do Icecast para audição em tempo real na loja.

---

## Proposed Changes

### Componente: Playout Player Client (`imc-audio-local`)

#### [MODIFY] [player.liq.tmpl](file:///P:/home/neoroute/imc-audio-local/player.liq.tmpl)
- Substituiremos a saída `output.icecast` pela diretiva `output.hls` do Liquidsoap:
```liquidsoap
# HLS Output Configuration
# segment_duration=2.0 seconds for low latency
enc = %ffmpeg(format="mpegts", %audio(codec="aac", bitrate=96000))

output.hls(
  playlist="playlist.m3u8",
  segment_duration=2.0,
  segment_temp_name="temp-%Y-%m-%d-%H-%M-%S.ts",
  [("audio", enc)],
  directory="/app/cache/hls",
  source
)
```

#### [MODIFY] [web_dashboard.go](file:///P:/home/neoroute/imc-audio-local/web_dashboard.go)
1. **Serviço de Arquivos Estáticos HLS**:
   - Exporemos o diretório `/app/cache/hls` via HTTP sob a rota `/hls/` na API Go:
   ```go
   mux.Handle("/hls/", http.StripPrefix("/hls/", http.FileServer(http.Dir("/app/cache/hls"))))
   ```
2. **Integração do Player com `hls.js`**:
   - Integraremos a biblioteca open-source `hls.js` no dashboard HTML.
   - Quando o operador clicar em "Play", o browser carregará `/hls/playlist.m3u8`.
3. **Sincronismo via Eventos de Metadados HLS**:
   - Escutaremos os eventos de metadados temporizados da faixa de áudio (`Hls.Events.FRAG_PARSING_METADATA`).
   - O player disparará a atualização do Gauge visual (título, artista, progresso) no exato instante em que o trecho de áudio com o ID3 correspondente for reproduzido nas caixas, garantindo sincronismo automático impecável.

---

## O que mais conseguimos além do sincronismo?

Utilizando HLS e o leitor de metadados integrado no browser, podemos trafegar dados ricos diretamente no áudio, tais como:
1. **Identificadores Únicos de Transmissão (IDs de POP)**: Para auditoria em tempo real pelo navegador.
2. **Tipo de Mídia (Música / Comercial)**: Para disparar animações e efeitos visuais na tela exatamente quando um comercial começa a tocar nas caixas de som da loja.
3. **Indicador de Chime / Vinheta**: Permitindo exibir avisos visuais customizados para o público no momento em que uma vinheta institucional entra em execução.

---

## Verification Plan

### Automated Tests
- Validar se a sintaxe do template Liquidsoap está correta compilando o player local.

### Manual Verification
1. Abrir a tela do painel do operador na porta `5003`.
2. Iniciar a reprodução do HLS.
3. Confirmar que a troca de faixa visual ocorre exatamente no mesmo segundo em que a música muda nos alto-falantes do computador.
