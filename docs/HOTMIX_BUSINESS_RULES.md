# Regras de Negócio - Integração Hotmix Streaming & Playout Local

Este documento descreve o funcionamento técnico e as regras de negócio para a integração das rádios por streaming Hotmix com o player de áudio local e o painel central de gerenciamento.

---

## 1. Resolução de Redirecionamento HTTP 302 dos Streams Hotmix

* **Problema:** As URLs base fornecidas pela API do Hotmix (ex: `https://streaming.hotmixradio.com/hotmix-shopping-vibes-en-pre`) realizam redirecionamentos HTTP 302 temporários para subdomínios e parâmetros dinâmicos (ex: `https://streaming.hotmixradio.com/hotmixpro-shopping-vibes-pre`).
* **Regra:** Como decodificadores como Liquidsoap / FFmpeg não seguem redirecionamentos HTTP 302 durante conexões contínuas de streaming, todas as URLs de rádio **devem ser previamente resolvidas via solicitação HTTP com `allow_redirects=True`** antes de serem gravadas no banco de dados e repassadas ao `input.http` do Liquidsoap.

---

## 2. Mapeamento Automático de Estações no Painel Central (`imc-server`)

* **Regra:** Ao cadastrar ou editar uma filial no painel administrativo (`/dashboard/stores/update` ou `/create`), o usuário seleciona o nome da estação no campo `external_stream_name` (ex: *Gentle Luxury*, *Cool & Calm*).
* O servidor `imc-server` consulta o dicionário oficial de estações `hotmixStationURLs` e salva automaticamente a URL de stream direta resolvida em `external_stream_url` no banco de dados PostgreSQL.

### Estações Mapeadas:
| Estação (Nome) | URL Direta de Transmissão (200 OK) |
| :--- | :--- |
| **Best-of** | `https://streaming.hotmixradio.com/hotmixpro-best-of-pre` |
| **Cocktail & Dinner Groove** | `https://streaming.hotmixradio.com/hotmixpro-cocktail-and-dinner-pre` |
| **Cool & Calm** | `https://streaming.hotmixradio.com/hotmixpro-cool-and-calm-pre` |
| **Cozy & Coffee** | `https://streaming.hotmixradio.com/hotmixpro-cozy-coffee-shop-pre` |
| **Gentle Luxury** | `https://streaming.hotmixradio.com/hotmixpro-gentle-luxury-pre` |
| **Just Lounge** | `https://streaming.hotmixradio.com/hotmixpro-just-lounge-pre` |
| **On the Rocks** | `https://streaming.hotmixradio.com/hotmixpro-on-the-rocks-pre` |
| **Shopping Vibes** | `https://streaming.hotmixradio.com/hotmixpro-shopping-vibes-pre` |
| **Spa Garden** | `https://streaming.hotmixradio.com/hotmixpro-spa-garden-pre` |
| **Workout** | `https://streaming.hotmixradio.com/hotmixpro-workout-pre` |

---

## 3. Detecção e Atualização Instantânea no Player Local (`imc-audio-local`)

* **Regra:** Durante a sincronização periódica (ou acionada via API/Painel) com o servidor central (`/api/v1/sync`), o comparador de configurações local (`checkIfConfigChanged`) verifica se a rádio streaming mudou (`external_stream_url` ou `external_stream_name`).
* Se houver alteração:
  1. Atualiza a tabela local `store_info` no SQLite.
  2. Regenera o script do player `/app/player.liq`.
  3. Reinicia o processo do Liquidsoap em menos de 5 segundos, garantindo a troca de áudio sem travamentos.

---

## 4. Transição Suave de Comerciais (Fade Down + Silêncio de 0.5s)

* **Problema:** Ao disparar um bloco comercial sobre o streaming da rádio, o áudio inicial da vinheta/comercial podia ser cortado nos primeiros milissegundos devido à troca abrupta de fontes.
* **Regra:** No arquivo `player.liq.tmpl`:
  * A fila de comerciais recebe um trecho de silêncio inicial de 0,5s (`blank(duration=0.5)`).
  * O volume da música de fundo (streaming) sofre um *fade down* para 0% em 0,5s (`smooth_add(duration=0.5, p=0.0)`).
  * O comercial inicia a execução perfeitamente do segundo `0.0s` sem qualquer corte de voz.

---

## 5. Duração Limite do Bloco Comercial

* **Regra:** Na ausência de valor explícito no formulário de edição da grade, a duração limite dos comerciais no bloco é derivada automaticamente multiplicando o intervalo em minutos por 60 segundos (`limitSec = intervalMin * 60`).
