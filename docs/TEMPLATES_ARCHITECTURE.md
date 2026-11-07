# Arquitetura de Templates Especializados do Playout (Liquidsoap 2.4.0)

Este documento define formalmente a arquitetura da **Matriz de Templates 2×3** para o mecanismo de playout do **IMC (Indoor Media Control)**.

---

## 1. Visão Geral da Matriz de Operação

O sistema opera com separação estrita de responsabilidades entre **Local de Reprodução** (Ambiente de Saída) e **Fonte de Áudio Primária**, gerando scripts Liquidsoap enxutos, dedicados e com zero regressões cruzadas.

```
+--------------------------+----------------------------------------+---------------------------------------+
| Fonte de Áudio / Modo    | SERVIDOR (Web Player / Docker)         | MÁQUINA LOCAL (Hardware / PDV / Loja) |
+--------------------------+----------------------------------------+---------------------------------------+
| ACERVO LOCAL (100% MP3)  | Perfil A1.a: server_acervo.liq.tmpl    | Perfil A2.a: local_acervo.liq.tmpl    |
| HÍBRIDO (Acervo + Stream)| Perfil A1.b: server_hybrid.liq.tmpl    | Perfil A2.b: local_hybrid.liq.tmpl    |
| STREAMING RELAY (100%)   | Perfil A1.c: server_streaming.liq.tmpl | Perfil A2.c: local_streaming.liq.tmpl |
+--------------------------+----------------------------------------+---------------------------------------+
```

---

## 2. Detalhamento dos Perfis e Templates

### A. Perfis de Servidor (`SERVER_MODE = true`)
As saídas de áudio no servidor são roteadas exclusivamente para o **Icecast** (OGG Vorbis Mono 64 kbps ou MP3 128 kbps), alimentando o Web Player com baixíssima latência (< 1s) e consumo de CPU otimizado (eliminando encoders HLS redundantes). Drivers de placa de som física (PulseAudio/AO) são omitidos.

1. **`server_acervo.liq.tmpl` (Perfil A1.a)**:
   * **Uso**: Rádios com acervo musical local MP3 (Ex: *Matriz 01*).
   * **Entradas**: `request.dynamic` (chamadas para Go em `/next`) + `request.queue` (comerciais).
   * **Ducking**: `smooth_add(p=0.03)` para avisos instantâneos e placas de veículos.
   * **Isolamento**: Zero decodificadores ou threads de streaming externo (elimina `Avutil.Error`).

2. **`server_streaming.liq.tmpl` (Perfil A1.c)**:
   * **Uso**: Rádios retransmissoras de Web Rádio ao vivo (Ex: *Shopping Cidade Sorocaba*).
   * **Entradas**: `input.http` com URL direta ou fader dinâmico.
   * **Comerciais**: Injeção por sobreposição prioritária via `fallback(track_sensitive=false, [commercials_queue, stream_source])`.
   * **Ducking**: Atenuação a 3% para avisos e veículos sobre o áudio da web rádio.
   * **Transição**: Endpoints Harbor `/fade_to_stream` e `/set_stream_url`.

3. **`server_hybrid.liq.tmpl` (Perfil A1.b)**:
   * **Uso**: Rádios com grade mista (blocos de músicas locais e blocos de streaming, ou sobreposição manual).
   * **Operador**: `switch(track_sensitive=false, [ ({ active_stream_url() != "" }, stream_source), ({ true }, music_source) ])`.

---

### B. Perfis de Máquina Local / Loja Física (`SERVER_MODE = false`)
As saídas de áudio na máquina da loja tocam diretamente na **placa de som conectada ao amplificador/mesa da loja** (`output.pulseaudio` ou `output.ao`). Encoders Icecast/HLS são omitidos para poupar 100% de CPU.

4. **`local_acervo.liq.tmpl` (Perfil A2.a)**:
   * **Uso**: Lojas físicas com músicas encriptadas `.imc` em cache local.
   * **Funcionamento**: 100% offline, independente de internet após o download inicial.

5. **`local_streaming.liq.tmpl` (Perfil A2.c)**:
   * **Uso**: Lojas físicas que retransmitem web rádio direto na caixa de som.

6. **`local_hybrid.liq.tmpl` (Perfil A2.b)**:
   * **Uso**: Lojas físicas com grade híbrida alternando acervo local e web rádio.

---

## 3. Resolução Automática pelo Motor Go

O executável unificado `imc-audio-local` executa a função `DetectPlayoutProfile(cfg)` na inicialização e após cada ciclo de sincronização (`sync.go`):

```go
func DetectPlayoutProfile(cfg *Config) (profileName string, tmplContent string) {
    // 1. Inspeciona SERVER_MODE (Servidor vs Placa de Som)
    // 2. Inspeciona media_library e estilos de streaming no SQLite local
    // 3. Seleciona e compila o template .liq correspondente via text/template
}
```

### Chaveamento Dinâmico sem Recompilação:
* **No Servidor (Docker)**: A imagem base `imc-radio-base` contém todos os templates compilados. Alterar o perfil de uma rádio requer apenas restart do container (1 segundo), sem rebuild de imagem.
* **No Cliente Local (Loja)**: Ao detectar alteração na grade ou nos estilos pelo Sync, o Go gera o novo `player.liq` e reinicia o Liquidsoap em background.

---

## 4. Regras Universais de Injeção de Áudio (Todos os Templates)

Todos os 6 templates garantem rigorosamente:
1. **Comerciais Prioritários**: `request.queue(id="commercials_queue")` com boost de volume de +20%.
2. **Avisos e Placas de Veículos com Ducking Suave**: `warnings_source = request.queue(id="warnings_queue")` atenuando a música/stream para **3% (`p=0.03`)**.
3. **API Harbor HTTP**: Porta dedicada (`7000`, `7001`, `7002`, etc.) expondo `/getmeta`, `/skip`, `/push_warning`, `/push_commercial`.
4. **Normalização de Ganho e Cue-Points**: Respeito às tags `liq_amplify`, `liq_cue_in`, `liq_cue_out`.
